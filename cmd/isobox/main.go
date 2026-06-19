package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type taskRecord struct {
	ID              string             `json:"id"`
	CreatedAt       string             `json:"created_at"`
	EffectivePolicy effectivePolicy    `json:"effective_policy"`
	Result          taskResult         `json:"result"`
	Outcome         taskAttemptOutcome `json:"outcome"`
}

type taskAttemptOutcome struct {
	Type     outcomeType `json:"type"`
	ExitCode int         `json:"exit_code,omitempty"`
	Error    string      `json:"error,omitempty"`
}

type outcomeType string

const (
	outcomeSuccess              outcomeType = "success"
	outcomePreparationFailure               = "preparation_failure"
	outcomeLaunchFailure                    = "launch_failure"
	outcomeWorkloadCommandExit              = "workload_command_exit"
	outcomeResultCaptureFailure             = "result_capture_failure"
)

type effectivePolicy struct {
	SchemaVersion    string   `json:"schema_version"`
	WorkspaceSource  string   `json:"workspace_source"`
	WorkloadCommand  []string `json:"workload_command"`
	RuntimeBackend   string   `json:"runtime_backend"`
	RetentionDefault string   `json:"retention_default"`
	Limitations      []string `json:"limitations"`
}

type taskResult struct {
	ExitStatus int    `json:"exit_status,omitempty"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Diff       string `json:"diff"`
}

type runOptions struct {
	source  string
	records string
	command []string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: isobox <run|promote>")
	}

	switch args[0] {
	case "run":
		opts, err := parseRun(args[1:])
		if err != nil {
			return err
		}
		return runTask(opts)
	case "promote":
		return promote(args[1:])
	default:
		return errors.New("usage: isobox <run|promote>")
	}
}

func parseRun(args []string) (runOptions, error) {
	opts := runOptions{records: ".isobox/tasks"}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			i++
			if i >= len(args) {
				return opts, errors.New("--source requires a path")
			}
			opts.source = args[i]
		case "--records":
			i++
			if i >= len(args) {
				return opts, errors.New("--records requires a path")
			}
			opts.records = args[i]
		case "--":
			opts.command = args[i+1:]
			i = len(args)
		default:
			return opts, fmt.Errorf("unknown argument: %s", args[i])
		}
	}

	if opts.source == "" {
		return opts, errors.New("--source is required")
	}
	if len(opts.command) == 0 {
		return opts, errors.New("workload command is required")
	}

	absSource, err := filepath.Abs(opts.source)
	if err != nil {
		return opts, err
	}
	opts.source = absSource

	absRecords, err := filepath.Abs(opts.records)
	if err != nil {
		return opts, err
	}
	opts.records = absRecords

	return opts, nil
}

func runTask(opts runOptions) error {
	id, err := newID()
	if err != nil {
		return err
	}

	record := taskRecord{
		ID:        id,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		EffectivePolicy: effectivePolicy{
			SchemaVersion:    "v1",
			WorkspaceSource:  opts.source,
			WorkloadCommand:  opts.command,
			RuntimeBackend:   "host-process",
			RetentionDefault: "disposable",
			Limitations: []string{
				"host-process: workload executes as a normal host process with the current user's privileges and filesystem access",
			},
		},
	}

	status, err := command(opts.source, "git", "status", "--porcelain").Output()
	if err != nil {
		record.Outcome = taskAttemptOutcome{Type: outcomePreparationFailure, Error: err.Error()}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("inspect Workspace Source: %w", err)
	}
	if len(status) != 0 {
		msg := "Workspace Source has uncommitted changes; commit them before running isobox"
		record.Outcome = taskAttemptOutcome{Type: outcomePreparationFailure, Error: msg}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return errors.New(msg)
	}

	workspaceRoot, err := os.MkdirTemp("", "isobox-workspace-*")
	if err != nil {
		record.Outcome = taskAttemptOutcome{Type: outcomePreparationFailure, Error: err.Error()}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return err
	}
	defer os.RemoveAll(workspaceRoot)

	workspace := filepath.Join(workspaceRoot, "workspace")
	if err := command("", "git", "clone", "--quiet", opts.source, workspace).Run(); err != nil {
		record.Outcome = taskAttemptOutcome{Type: outcomePreparationFailure, Error: err.Error()}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("create private workspace: %w", err)
	}

	stdout, stderr, exitStatus, launchErr := runWorkload(workspace, opts.command)
	if launchErr != nil {
		record.Result = taskResult{Stdout: stdout, Stderr: stderr}
		record.Outcome = taskAttemptOutcome{Type: outcomeLaunchFailure, Error: launchErr.Error()}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("launch workload command: %w", launchErr)
	}
	record.Result = taskResult{
		ExitStatus: exitStatus,
		Stdout:     stdout,
		Stderr:     stderr,
	}

	var diff bytes.Buffer
	diffCmd := command(workspace, "git", "diff", "--no-ext-diff")
	diffCmd.Stdout = &diff
	diffCmd.Stderr = os.Stderr
	if err := diffCmd.Run(); err != nil {
		record.Outcome = taskAttemptOutcome{Type: outcomeResultCaptureFailure, Error: err.Error()}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("capture task result diff: %w", err)
	}
	record.Result.Diff = diff.String()

	if exitStatus != 0 {
		record.Outcome = taskAttemptOutcome{Type: outcomeWorkloadCommandExit, ExitCode: exitStatus}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("workload command exited with status %d", exitStatus)
	}

	record.Outcome = taskAttemptOutcome{Type: outcomeSuccess}
	return writeRecord(opts.records, record)
}

func writeRecord(recordsDir string, record taskRecord) error {
	recordDir := filepath.Join(recordsDir, record.ID)
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		return err
	}
	recordBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(recordDir, "record.json"), append(recordBytes, '\n'), 0o644)
}

func promote(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: isobox promote <task-record-dir>")
	}

	recordBytes, err := os.ReadFile(filepath.Join(args[0], "record.json"))
	if err != nil {
		return err
	}

	var record taskRecord
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		return err
	}
	if record.Result.Diff == "" {
		return errors.New("task result has no diff to promote")
	}

	cmd := command(record.EffectivePolicy.WorkspaceSource, "git", "apply", "--whitespace=nowarn")
	cmd.Stdin = bytes.NewBufferString(record.Result.Diff)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("promote task result: %w", err)
	}
	return nil
}

func runWorkload(workspace string, workload []string) (string, string, int, error) {
	var stdout, stderr bytes.Buffer
	cmd := command(workspace, workload[0], workload[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdout.String(), stderr.String(), exitErr.ExitCode(), nil
	}
	return stdout.String(), stderr.String(), 0, err
}

func command(dir, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd
}

func newID() (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "task-" + hex.EncodeToString(bytes[:]), nil
}
