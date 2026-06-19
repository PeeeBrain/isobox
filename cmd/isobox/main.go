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
	"time"

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
}

type workspaceInfo struct {
	Retention string `json:"retention"`
	Path      string `json:"path,omitempty"`
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
	source          string
	records         string
	retainWorkspace bool
	command         []string
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
		case "--retain-workspace":
			opts.retainWorkspace = true
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

	record := taskRecord{
		SchemaVersion: taskRecordSchemaVersion,
		ID:            id,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		EffectivePolicy: effectivePolicy{
			SchemaVersion:    "v1",
			WorkspaceSource:  opts.source,
			WorkloadCommand:  opts.command,
			RuntimeBackend:   backend.Name(),
			RetentionDefault: retention,
			Limitations:      backend.Limitations(),
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

func newID() (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "task-" + hex.EncodeToString(bytes[:]), nil
}
