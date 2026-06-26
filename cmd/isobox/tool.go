package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"isobox/internal/policy"
	"isobox/internal/preflight"
	"isobox/internal/promotion"
	"isobox/internal/runtimebackend"
	"isobox/internal/workspace"
)

// toolCmd runs `isobox tool -- <command>` to enter a Tool-Call Sandbox.
func toolCmd(args []string) error {
	cmd, err := parseToolCommand(args)
	if err != nil {
		return err
	}

	startDir, err := os.Getwd()
	if err != nil {
		return err
	}
	projectRoot, err := gitTopLevelForTool(startDir)
	if err != nil {
		return fmt.Errorf("isobox tool: locate project root: %w", err)
	}

	if err := preflight.Run(startDir); err != nil {
		return err
	}

	ws, err := workspace.CreateRepository(projectRoot)
	if err != nil {
		return fmt.Errorf("create Repository Workspace: %w", err)
	}
	defer ws.Close()

	rel, err := filepath.Rel(projectRoot, startDir)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("isobox tool: current directory is outside project root")
	}
	workdir := filepath.Join(ws.Root(), rel)

	backend := runtimebackend.NewBubblewrap()
	id, err := newID()
	if err != nil {
		return err
	}
	record := taskRecord{
		SchemaVersion: taskRecordSchemaVersion,
		ID:            id,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		EffectivePolicy: effectivePolicy{
			SchemaVersion:       "v1",
			WorkspaceSource:     projectRoot,
			WorkloadCommand:     cmd,
			RuntimeBackend:      backend.Name(),
			RetentionDefault:    "disposable",
			ResourceLimits:      policy.ResolveResourceLimits(policy.DefaultResourceLimits()),
			ResourceEnforcement: backend.ResourceEnforcement(),
			Network:             policy.ResolveNetworkPolicy(policy.DefaultNetworkPolicy()),
			NetworkEnforcement:  backend.NetworkEnforcement(),
			Limitations:         backend.Limitations(),
		},
		Workspace: workspaceInfo{SourceType: "repository", SourceCommit: ws.SourceCommit(), Retention: "disposable"},
	}

	fmt.Fprintf(os.Stderr, "isobox task %s: starting tool call\n", id)
	result, launchErr := backend.Run(context.Background(), runtimebackend.RunRequest{
		WorkspaceRoot: ws.Root(),
		Workdir:       workdir,
		Command:       cmd,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	})
	if launchErr != nil {
		record.Outcome = taskAttemptOutcome{Type: outcomeLaunchFailure, Error: launchErr.Error()}
		_ = writeArtifactBackedRecord(filepath.Join(projectRoot, ".isobox", "tasks"), record)
		return fmt.Errorf("launch workload command: %w", launchErr)
	}
	record.Result = taskResult{ExitStatus: result.ExitStatus, Stdout: result.Stdout, Stderr: result.Stderr}
	diff, err := ws.Diff()
	if err != nil {
		record.Outcome = taskAttemptOutcome{Type: outcomeResultCaptureFailure, Error: err.Error()}
		_ = writeArtifactBackedRecord(filepath.Join(projectRoot, ".isobox", "tasks"), record)
		return fmt.Errorf("capture task result diff: %w", err)
	}
	record.Result.Diff = diff
	record.PromotionReport = promotion.GenerateReport(diff)
	if result.ExitStatus != 0 {
		record.Outcome = taskAttemptOutcome{Type: outcomeWorkloadCommandExit, ExitCode: result.ExitStatus}
		if err := writeArtifactBackedRecord(filepath.Join(projectRoot, ".isobox", "tasks"), record); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "isobox task %s: completed outcome=%s exit_code=%d\n", id, record.Outcome.Type, result.ExitStatus)
		return commandExitError{code: result.ExitStatus}
	}
	record.Outcome = taskAttemptOutcome{Type: outcomeSuccess}
	if err := writeArtifactBackedRecord(filepath.Join(projectRoot, ".isobox", "tasks"), record); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "isobox task %s: completed outcome=%s\n", id, record.Outcome.Type)
	return nil
}

func gitTopLevelForTool(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("not inside a Git repository")
	}
	return root, nil
}

// parseToolCommand extracts the Workload Command from `isobox tool -- <cmd>`.
// The command portion is required so a Cooperative Tool Call that lacks a
// workload is rejected before any preflight work runs.
func parseToolCommand(args []string) ([]string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			cmd := args[i+1:]
			if len(cmd) == 0 {
				return nil, fmt.Errorf("isobox tool: workload command is required after `--`")
			}
			return cmd, nil
		}
	}
	return nil, fmt.Errorf("isobox tool: usage: isobox tool -- <command> [args...]")
}
