package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"isobox/internal/policy"
	"isobox/internal/promotion"
	"isobox/internal/runtimebackend"
	"isobox/internal/workspace"
)

type taskRecord struct {
	SchemaVersion   string             `json:"schema_version"`
	ID              string             `json:"id"`
	CreatedAt       string             `json:"created_at"`
	EffectivePolicy effectivePolicy    `json:"effective_policy"`
	Workspace       workspaceInfo      `json:"workspace"`
	Result          taskResult         `json:"result"`
	Outcome         taskAttemptOutcome `json:"outcome"`
	// PromotionReport is the initial Promotion Report generated from the
	// Task Result diff. It is informational and never gates Promotion; it is
	// captured here so review can focus on high-risk changes before explicit
	// Promotion. It is omitted when no diff was captured.
	PromotionReport *promotion.Report `json:"promotion_report,omitempty"`
}

type workspaceInfo struct {
	SourceType   string `json:"source_type"`
	SourceCommit string `json:"source_commit"`
	Retention    string `json:"retention"`
	Path         string `json:"path,omitempty"`
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
	SchemaVersion       string                     `json:"schema_version"`
	WorkspaceSource     string                     `json:"workspace_source"`
	WorkloadCommand     []string                   `json:"workload_command"`
	RuntimeBackend      string                     `json:"runtime_backend"`
	RetentionDefault    string                     `json:"retention_default"`
	ResourceLimits      policy.ResourceLimits      `json:"resource_limits"`
	ResourceEnforcement policy.ResourceEnforcement `json:"resource_enforcement"`
	Network             policy.NetworkPolicy       `json:"network"`
	NetworkEnforcement  policy.NetworkEnforcement  `json:"network_enforcement"`
	ReuseInputs         []policy.ReuseInput        `json:"reuse_inputs"`
	Limitations         []string                   `json:"limitations"`
}

type taskResult struct {
	ExitStatus int    `json:"exit_status,omitempty"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Diff       string `json:"diff"`
}

type runOptions struct {
	source          string
	records         string
	retainWorkspace bool
	command         []string
	reuseInputs     []policy.ReuseInput
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		var exitErr commandExitError
		if errors.As(err, &exitErr) {
			if exitErr.err != nil {
				fmt.Fprintln(os.Stderr, exitErr.err)
			}
			os.Exit(exitErr.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type commandExitError struct {
	code int
	err  error
}

func (e commandExitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("workload command exited with status %d", e.code)
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: isobox <init|run|promote>")
	}

	switch args[0] {
	case "init":
		return initCmd(args[1:])
	case "run":
		opts, err := parseRun(args[1:])
		if err != nil {
			return err
		}
		return runTask(opts)
	case "tool":
		return toolCmd(args[1:])
	case "promote":
		return promote(args[1:])
	default:
		return errors.New("usage: isobox <init|run|promote>")
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
		case "--retain-workspace":
			opts.retainWorkspace = true
		case "--reuse-input":
			i++
			if i >= len(args) {
				return opts, errors.New("--reuse-input requires a kind=value declaration")
			}
			input, err := parseReuseInput(args[i])
			if err != nil {
				return opts, err
			}
			opts.reuseInputs = append(opts.reuseInputs, input)
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

	backend := runtimebackend.NewHost()
	retention := "disposable"
	if opts.retainWorkspace {
		retention = "retained"
	}

	sandboxPolicy := policy.SandboxPolicy{
		ResourceLimits: policy.DefaultResourceLimits(),
		Network:        policy.DefaultNetworkPolicy(),
		ReuseInputs:    opts.reuseInputs,
	}

	resolvedReuseInputs, err := policy.ResolveReuseInputs(sandboxPolicy.ReuseInputs)
	if err != nil {
		return fmt.Errorf("resolve reuse inputs: %w", err)
	}

	limitations := backend.Limitations()
	if len(resolvedReuseInputs) > 0 {
		limitations = append(limitations, policy.ReuseInputsLimitation(resolvedReuseInputs))
	}

	record := taskRecord{
		SchemaVersion: taskRecordSchemaVersion,
		ID:            id,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		EffectivePolicy: effectivePolicy{
			SchemaVersion:       "v1",
			WorkspaceSource:     opts.source,
			WorkloadCommand:     opts.command,
			RuntimeBackend:      backend.Name(),
			RetentionDefault:    retention,
			ResourceLimits:      policy.ResolveResourceLimits(sandboxPolicy.ResourceLimits),
			ResourceEnforcement: backend.ResourceEnforcement(),
			Network:             policy.ResolveNetworkPolicy(sandboxPolicy.Network),
			NetworkEnforcement:  backend.NetworkEnforcement(),
			ReuseInputs:         resolvedReuseInputs,
			Limitations:         limitations,
		},
		Workspace: workspaceInfo{Retention: retention},
	}

	ws, err := workspace.CreateRepository(opts.source)
	if err != nil {
		if errors.Is(err, workspace.ErrDirtyWorkspaceSource) {
			record.Outcome = taskAttemptOutcome{Type: outcomePreparationFailure, Error: err.Error()}
			if werr := writeRecord(opts.records, record); werr != nil {
				return werr
			}
			return err
		}
		record.Outcome = taskAttemptOutcome{Type: outcomePreparationFailure, Error: err.Error()}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("create Repository Workspace: %w", err)
	}
	defer ws.Close()

	record.Workspace.SourceType = "repository"
	record.Workspace.SourceCommit = ws.SourceCommit()

	if opts.retainWorkspace {
		record.Workspace.Path = ws.Retain()
	}
	defer reportWorkspace(record.Workspace)

	result, launchErr := backend.Run(context.Background(), runtimebackend.RunRequest{
		Workdir: ws.Root(),
		Command: opts.command,
	})
	if launchErr != nil {
		record.Result = taskResult{Stdout: result.Stdout, Stderr: result.Stderr}
		record.Outcome = taskAttemptOutcome{Type: outcomeLaunchFailure, Error: launchErr.Error()}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("launch workload command: %w", launchErr)
	}
	record.Result = taskResult{
		ExitStatus: result.ExitStatus,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
	}

	diff, err := ws.Diff()
	if err != nil {
		record.Outcome = taskAttemptOutcome{Type: outcomeResultCaptureFailure, Error: err.Error()}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("capture task result diff: %w", err)
	}
	record.Result.Diff = diff
	record.PromotionReport = promotion.GenerateReport(diff)

	if result.ExitStatus != 0 {
		record.Outcome = taskAttemptOutcome{Type: outcomeWorkloadCommandExit, ExitCode: result.ExitStatus}
		if werr := writeRecord(opts.records, record); werr != nil {
			return werr
		}
		return fmt.Errorf("workload command exited with status %d", result.ExitStatus)
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

	record, err := loadRecord(args[0])
	if err != nil {
		return err
	}

	if record.Outcome.Type != outcomeSuccess && record.Outcome.Type != outcomeWorkloadCommandExit {
		return fmt.Errorf("cannot promote task %q: task outcome is %q, only successful tasks or workload-command exits can be promoted", record.ID, record.Outcome.Type)
	}

	if record.Workspace.SourceType != "repository" {
		return fmt.Errorf("cannot promote task %q: promotion is only supported for Repository Workspace results", record.ID)
	}

	if record.Workspace.SourceCommit == "" {
		return fmt.Errorf("cannot promote task %q: task record is missing Workspace Source commit", record.ID)
	}

	currentCommit, err := workspace.HeadCommit(record.EffectivePolicy.WorkspaceSource)
	if err != nil {
		return fmt.Errorf("cannot promote task %q: inspect trusted repository: %w", record.ID, err)
	}
	if currentCommit != record.Workspace.SourceCommit {
		return fmt.Errorf("cannot promote task %q: trusted repository has changed since the task was recorded", record.ID)
	}

	if record.Result.Diff == "" {
		return fmt.Errorf("cannot promote task %q: task result has no diff to promote", record.ID)
	}

	// The Promotion Report is informational: it focuses review on high-risk
	// changes but never gates or auto-applies Promotion. The user remains the
	// review gate by running `isobox promote` explicitly.
	if record.PromotionReport != nil {
		fmt.Print(record.PromotionReport.Summarize())
	}

	cmd := command(record.EffectivePolicy.WorkspaceSource, "git", "apply", "--whitespace=nowarn")
	cmd.Stdin = bytes.NewBufferString(record.Result.Diff)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("promote task %q: git apply failed: %w", record.ID, err)
	}
	return nil
}

func command(dir, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd
}

func reportWorkspace(ws workspaceInfo) {
	if ws.Retention == "retained" {
		fmt.Printf("workspace retained at: %s\n", ws.Path)
		return
	}
	fmt.Println("workspace disposed")
}

func parseReuseInput(spec string) (policy.ReuseInput, error) {
	kind, value, found := strings.Cut(spec, "=")
	if !found {
		return policy.ReuseInput{}, fmt.Errorf("--reuse-input requires a kind=value declaration, got %q", spec)
	}
	if err := policy.ValidateReuseInputKind(kind); err != nil {
		return policy.ReuseInput{}, err
	}
	if value == "" {
		return policy.ReuseInput{}, fmt.Errorf("--reuse-input %q has empty value", kind)
	}
	return policy.ReuseInput{Kind: policy.ReuseInputKind(kind), Value: value}, nil
}

func newID() (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "task-" + hex.EncodeToString(bytes[:]), nil
}
