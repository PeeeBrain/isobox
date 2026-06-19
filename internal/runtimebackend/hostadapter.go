package runtimebackend

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// Host is a Runtime Backend that executes Workload Commands as normal host
// processes. It preserves the current user's privileges, environment, and
// filesystem access, so it does not provide strong isolation. It exists to
// route the existing execution behavior through the Backend contract while
// stronger backends are introduced later.
type Host struct{}

// NewHost returns a host Runtime Backend.
func NewHost() *Host {
	return &Host{}
}

// Name returns the identifier recorded in the Effective Policy.
func (h *Host) Name() string {
	return "host-process"
}

// Limitations describes the lower-assurance nature of the host backend.
func (h *Host) Limitations() []string {
	return []string{
		"host-process: workload executes as a normal host process with the current user's privileges, environment, and filesystem access; this Runtime Backend does not provide strong isolation",
	}
}

// Run executes the requested command in the requested working directory,
// capturing stdout, stderr, and the exit status. A non-zero exit status is
// returned in RunResult without an error; an error is returned only when the
// command cannot be started or when execution cannot be observed.
func (h *Host) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if len(req.Command) == 0 {
		return RunResult{}, errors.New("workload command is required")
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, req.Command[0], req.Command[1:]...)
	cmd.Dir = req.Workdir
	cmd.Stdin = req.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return RunResult{
			ExitStatus: 0,
			Stdout:     stdout.String(),
			Stderr:     stderr.String(),
		}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return RunResult{
			ExitStatus: exitErr.ExitCode(),
			Stdout:     stdout.String(),
			Stderr:     stderr.String(),
		}, nil
	}

	return RunResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, err
}
