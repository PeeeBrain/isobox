package runtimebackend

import (
	"context"
	"io"

	"isobox/internal/policy"
)

// Backend executes a Workload Command inside a Sandbox.
//
// A Backend is a Runtime Backend: the isolation provider that creates and
// enforces the low-level boundary for a Sandbox. Implementations may range
// from lower-assurance host processes to stronger isolation providers.
type Backend interface {
	// Name returns the identifier recorded in the Effective Policy.
	Name() string

	// Limitations returns human-readable statements about the assurance and
	// enforcement limits of this backend. These are recorded in the Effective
	// Policy so the Task Record never implies stronger isolation than the
	// backend provides.
	Limitations() []string

	// ResourceEnforcement returns a structured report describing which
	// resource limits this backend enforces, partially enforces, or does not
	// enforce. This is recorded in the Effective Policy alongside the resolved
	// resource limits.
	ResourceEnforcement() policy.ResourceEnforcement

	// NetworkEnforcement returns a structured report describing how this
	// backend enforces the network policy. This is recorded in the Effective
	// Policy alongside the resolved network policy so the Task Record never
	// implies stronger network isolation than the backend provides.
	NetworkEnforcement() policy.NetworkEnforcement

	// Run executes the requested command and returns its captured output and
	// exit status. A non-zero exit status is returned in the result without an
	// error; an error is returned only when the command could not be launched
	// or when execution could not be observed.
	Run(ctx context.Context, req RunRequest) (RunResult, error)
}

// RunRequest configures a single Workload Command execution.
type RunRequest struct {
	// WorkspaceRoot is the host path for the Workspace root. Backends that expose
	// a stable internal Workspace path use this as the source for that mount.
	// When empty, Workdir is treated as the Workspace root for compatibility.
	WorkspaceRoot string
	Workdir       string
	Command       []string
	Stdin         io.Reader
}

// RunResult captures the observable output of a Workload Command execution.
type RunResult struct {
	ExitStatus int
	Stdout     string
	Stderr     string
}
