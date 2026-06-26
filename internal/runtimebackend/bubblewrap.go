package runtimebackend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"isobox/internal/policy"
)

// Bubblewrap is a Runtime Backend that runs a Workload Command inside a
// bubblewrap filesystem boundary. The Repository Workspace is exposed at the
// stable internal path /workspace.
type Bubblewrap struct{}

func NewBubblewrap() *Bubblewrap { return &Bubblewrap{} }

func (b *Bubblewrap) Name() string { return "bubblewrap" }

func (b *Bubblewrap) Limitations() []string {
	return []string{
		"bubblewrap: first milestone provides filesystem containment for the Workspace but records network and resource controls as not enforced",
	}
}

func (b *Bubblewrap) ResourceEnforcement() policy.ResourceEnforcement {
	return policy.ResourceEnforcement{RuntimeBackend: b.Name(), Limits: []policy.ResourceLimitEnforcement{
		{Name: "time", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce time limits in this milestone"},
		{Name: "output_size", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce output size limits in this milestone"},
		{Name: "cpu", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce CPU limits in this milestone"},
		{Name: "memory", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce memory limits in this milestone"},
		{Name: "process", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce process limits in this milestone"},
		{Name: "disk", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce disk limits in this milestone"},
		{Name: "file_descriptors", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce file descriptor limits in this milestone"},
	}}
}

func (b *Bubblewrap) NetworkEnforcement() policy.NetworkEnforcement {
	return policy.NetworkEnforcement{RuntimeBackend: b.Name(), Rules: []policy.NetworkEnforcementRule{
		{Aspect: "default_deny", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce the deny-by-default network policy in this milestone"},
		{Aspect: "allow_rules", Status: policy.NotEnforced, Detail: "the bubblewrap backend does not enforce network allow rules in this milestone"},
	}}
}

func (b *Bubblewrap) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if len(req.Command) == 0 {
		return RunResult{}, errors.New("workload command is required")
	}
	workspaceRoot, err := filepath.Abs(req.WorkspaceRoot)
	if err != nil || workspaceRoot == "" {
		workspaceRoot, err = filepath.Abs(req.Workdir)
		if err != nil {
			return RunResult{}, fmt.Errorf("resolve workspace root: %w", err)
		}
	}
	workdir, err := filepath.Abs(req.Workdir)
	if err != nil {
		return RunResult{}, fmt.Errorf("resolve workdir: %w", err)
	}
	rel, err := filepath.Rel(workspaceRoot, workdir)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return RunResult{}, fmt.Errorf("workdir %q is not inside workspace root %q", req.Workdir, workspaceRoot)
	}
	internalWorkdir := "/workspace"
	if rel != "." {
		internalWorkdir = "/workspace/" + filepath.ToSlash(rel)
	}

	args := []string{
		"--die-with-parent",
		"--unshare-pid",
		"--clearenv",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
		"--ro-bind-try", "/usr", "/usr",
		"--ro-bind-try", "/bin", "/bin",
		"--ro-bind-try", "/lib", "/lib",
		"--ro-bind-try", "/lib64", "/lib64",
		"--ro-bind-try", "/etc", "/etc",
		"--bind", workspaceRoot, "/workspace",
		"--setenv", "HOME", "/tmp",
		"--setenv", "PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"--chdir", internalWorkdir,
		"--",
	}
	args = append(args, req.Command...)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "bwrap", args...)
	cmd.Env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	cmd.Stdin = req.Stdin
	cmd.Stdout = &stdout
	if req.Stdout != nil {
		cmd.Stdout = io.MultiWriter(&stdout, req.Stdout)
	}
	cmd.Stderr = &stderr
	if req.Stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, req.Stderr)
	}
	err = cmd.Run()
	result := RunResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitStatus = exitErr.ExitCode()
		if strings.Contains(result.Stderr, "bwrap:") || strings.Contains(result.Stderr, "bubblewrap:") {
			return result, fmt.Errorf("bubblewrap setup failed: %s", strings.TrimSpace(result.Stderr))
		}
		return result, nil
	}
	return result, err
}
