package runtimebackend

import (
	"bytes"
	"context"
	"errors"
	"os/exec"

	"isobox/internal/policy"
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

// ResourceEnforcement reports that the host backend does not enforce resource
// limits in this milestone. The returned report clearly records each limit
// category as not_enforced so the Task Record does not overstate containment.
func (h *Host) ResourceEnforcement() policy.ResourceEnforcement {
	categories := []struct {
		name   string
		detail string
	}{
		{"time", "the host-process backend does not enforce time limits in this milestone"},
		{"output_size", "the host-process backend does not enforce output size limits in this milestone"},
		{"cpu", "the host-process backend does not enforce CPU limits in this milestone"},
		{"memory", "the host-process backend does not enforce memory limits in this milestone"},
		{"process", "the host-process backend does not enforce process limits in this milestone"},
		{"disk", "the host-process backend does not enforce disk limits in this milestone"},
		{"file_descriptors", "the host-process backend does not enforce file descriptor limits in this milestone"},
	}

	limits := make([]policy.ResourceLimitEnforcement, len(categories))
	for i, c := range categories {
		limits[i] = policy.ResourceLimitEnforcement{
			Name:   c.name,
			Status: policy.NotEnforced,
			Detail: c.detail,
		}
	}

	return policy.ResourceEnforcement{
		RuntimeBackend: h.Name(),
		Limits:         limits,
	}
}

// NetworkEnforcement reports that the host backend does not enforce the
// network policy in this milestone. The default-deny intent and any allow
// rules are recorded as not_enforced so the Task Record does not overstate
// network containment. The host backend preserves the current user's
// privileges, environment, and filesystem access, including network access.
func (h *Host) NetworkEnforcement() policy.NetworkEnforcement {
	return policy.NetworkEnforcement{
		RuntimeBackend: h.Name(),
		Rules: []policy.NetworkEnforcementRule{
			{
				Aspect: "default_deny",
				Status: policy.NotEnforced,
				Detail: "the host-process backend does not enforce the deny-by-default network policy in this milestone; workloads retain host network access",
			},
			{
				Aspect: "allow_rules",
				Status: policy.NotEnforced,
				Detail: "the host-process backend does not enforce network allow rules in this milestone; workloads retain host network access",
			},
		},
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
