package runtimebackend_test

import (
	"context"
	"strings"
	"testing"

	"isobox/internal/policy"
	"isobox/internal/runtimebackend"
)

func TestHostBackendRunsWorkloadCommand(t *testing.T) {
	backend := runtimebackend.NewHost()
	result, err := backend.Run(context.Background(), runtimebackend.RunRequest{
		Workdir: t.TempDir(),
		Command: []string{"sh", "-c", "printf out; printf err >&2"},
	})
	if err != nil {
		t.Fatalf("host backend run failed: %v", err)
	}
	if result.ExitStatus != 0 {
		t.Fatalf("exit status = %d, want 0", result.ExitStatus)
	}
	if result.Stdout != "out" {
		t.Fatalf("stdout = %q, want out", result.Stdout)
	}
	if result.Stderr != "err" {
		t.Fatalf("stderr = %q, want err", result.Stderr)
	}
}

func TestHostBackendCapturesNonZeroExit(t *testing.T) {
	backend := runtimebackend.NewHost()
	result, err := backend.Run(context.Background(), runtimebackend.RunRequest{
		Workdir: t.TempDir(),
		Command: []string{"sh", "-c", "printf out; printf err >&2; exit 7"},
	})
	if err != nil {
		t.Fatalf("host backend returned error for non-zero exit: %v", err)
	}
	if result.ExitStatus != 7 {
		t.Fatalf("exit status = %d, want 7", result.ExitStatus)
	}
	if result.Stdout != "out" {
		t.Fatalf("stdout = %q, want out", result.Stdout)
	}
	if result.Stderr != "err" {
		t.Fatalf("stderr = %q, want err", result.Stderr)
	}
}

func TestHostBackendReportsLaunchFailure(t *testing.T) {
	backend := runtimebackend.NewHost()
	_, err := backend.Run(context.Background(), runtimebackend.RunRequest{
		Workdir: t.TempDir(),
		Command: []string{"this-binary-does-not-exist"},
	})
	if err == nil {
		t.Fatal("host backend returned nil error for missing executable")
	}
}

func TestHostBackendRunsInRequestedWorkdir(t *testing.T) {
	workdir := t.TempDir()
	backend := runtimebackend.NewHost()
	result, err := backend.Run(context.Background(), runtimebackend.RunRequest{
		Workdir: workdir,
		Command: []string{"pwd"},
	})
	if err != nil {
		t.Fatalf("host backend run failed: %v", err)
	}
	trimmed := strings.TrimSpace(result.Stdout)
	if trimmed != workdir {
		t.Fatalf("working directory = %q, want %q", trimmed, workdir)
	}
}

func TestHostBackendDocumentsLowerAssurance(t *testing.T) {
	backend := runtimebackend.NewHost()
	if backend.Name() != "host-process" {
		t.Fatalf("backend name = %q, want host-process", backend.Name())
	}

	limitations := backend.Limitations()
	if len(limitations) == 0 {
		t.Fatal("host backend returned no limitations")
	}

	var found bool
	for _, l := range limitations {
		if strings.Contains(l, "host-process") && strings.Contains(l, "does not provide strong isolation") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("limitations do not document lower-assurance host backend: %#v", limitations)
	}
}

func TestHostBackendReportsResourceLimitsNotEnforced(t *testing.T) {
	backend := runtimebackend.NewHost()
	enforcement := backend.ResourceEnforcement()

	if enforcement.RuntimeBackend != "host-process" {
		t.Fatalf("enforcement runtime_backend = %q, want host-process", enforcement.RuntimeBackend)
	}

	wantCategories := map[string]policy.EnforcementStatus{
		"time":             policy.NotEnforced,
		"output_size":      policy.NotEnforced,
		"cpu":              policy.NotEnforced,
		"memory":           policy.NotEnforced,
		"process":          policy.NotEnforced,
		"disk":             policy.NotEnforced,
		"file_descriptors": policy.NotEnforced,
	}

	if len(enforcement.Limits) != len(wantCategories) {
		t.Fatalf("enforcement limits = %d, want %d: %#v", len(enforcement.Limits), len(wantCategories), enforcement.Limits)
	}

	seen := make(map[string]bool)
	for _, l := range enforcement.Limits {
		want, ok := wantCategories[l.Name]
		if !ok {
			t.Fatalf("unexpected resource limit category: %q", l.Name)
		}
		if l.Status != want {
			t.Fatalf("%s status = %q, want %q", l.Name, l.Status, want)
		}
		if !strings.Contains(l.Detail, "does not enforce") {
			t.Fatalf("%s detail does not state non-enforcement: %q", l.Name, l.Detail)
		}
		seen[l.Name] = true
	}

	for name := range wantCategories {
		if !seen[name] {
			t.Fatalf("missing resource limit category: %q", name)
		}
	}
}
