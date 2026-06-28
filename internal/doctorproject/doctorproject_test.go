package doctorproject_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"isobox/internal/doctor"
	"isobox/internal/doctorproject"
	"isobox/internal/projectpolicy"
)

func TestChecksSkipOutsideGitWithoutError(t *testing.T) {
	root, checks := doctorproject.Checks(t.TempDir(), true, true)
	if root != "" || len(checks) != 0 {
		t.Fatalf("outside git root=%q checks=%v, want skipped", root, checks)
	}
}

func TestChecksWarnMissingPolicyAtGitRootFromSubdirectory(t *testing.T) {
	repo := gitRepo(t)
	sub := filepath.Join(repo, "a", "b")
	must(t, os.MkdirAll(sub, 0o755))

	root, checks := doctorproject.Checks(sub, true, true)
	if root != repo {
		t.Fatalf("root=%q, want %q", root, repo)
	}
	got := find(checks, "project-policy")
	if got == nil || got.Severity != doctor.SeverityWarning || !strings.Contains(got.Fix, "isobox init") {
		t.Fatalf("missing policy finding=%+v, want init warning", got)
	}
	if find(checks, "project-gitignore") != nil || find(checks, "project-task-store") != nil {
		t.Fatalf("missing policy should not emit downstream findings: %v", checks)
	}
}

func TestChecksAccumulatePolicyCompatibilityFindings(t *testing.T) {
	repo := gitRepo(t)
	must(t, os.MkdirAll(filepath.Join(repo, ".isobox"), 0o755))
	body := []byte(`api_version: isobox.dev/v1alpha1
kind: ProjectPolicy
tool_call:
  enabled: true
runtime_backend: host_process
development_environment:
  path_mode: inherited
workspace_source:
  kind: directory
network:
  default: allow
  allow:
    - host: example.com
filesystem:
  expose_workspace: true
credentials:
  default: inherited
preflight:
  rules: []
promotion:
  mode: auto
`)
	must(t, os.WriteFile(filepath.Join(repo, ".isobox", "config.yaml"), body, 0o644))

	_, checks := doctorproject.Checks(repo, true, true)
	for _, id := range []string{"project-policy-runtime_backend", "project-policy-development_environment-path_mode", "project-policy-workspace_source-kind", "project-policy-network-default", "project-policy-network-allow", "project-policy-credentials-default", "project-policy-promotion-mode"} {
		if got := find(checks, id); got == nil || got.Severity != doctor.SeverityError {
			t.Fatalf("missing compatibility error %s in %v", id, checks)
		}
	}
}

func TestChecksDoNotCreateTaskStore(t *testing.T) {
	repo := gitRepo(t)
	must(t, os.MkdirAll(filepath.Join(repo, ".isobox"), 0o755))
	p, err := projectpolicy.Default().Render()
	must(t, err)
	must(t, os.WriteFile(filepath.Join(repo, ".isobox", "config.yaml"), []byte(p), 0o644))

	_, _ = doctorproject.Checks(repo, true, true)
	if _, err := os.Stat(filepath.Join(repo, ".isobox", "tasks")); !os.IsNotExist(err) {
		t.Fatalf("doctor created .isobox/tasks or unexpected stat error: %v", err)
	}
}

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	return dir
}

func find(checks []doctor.Check, id string) *doctor.Check {
	for i := range checks {
		if checks[i].ID == id {
			return &checks[i]
		}
	}
	return nil
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
