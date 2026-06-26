package preflight

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// validDefaultBody is the canonical first-milestone project policy used as
// the baseline for unit tests. Individual tests override specific lines to
// exercise the preflight checks.
const validDefaultBody = `api_version: isobox.dev/v1alpha1
kind: ProjectPolicy
tool_call:
  enabled: true
runtime_backend: bubblewrap
development_environment:
  path_mode: backend_default
workspace_source:
  kind: project_root
network:
  default: deny
filesystem:
  expose_workspace: true
credentials:
  default: deny
preflight:
  rules:
    - reject_dirty_workspace_source
promotion:
  mode: manual
`

func TestRunRejectsDirectoryWithoutProjectPolicy(t *testing.T) {
	dir := initRepo(t)

	err := Run(dir)
	if err == nil {
		t.Fatal("Run unexpectedly succeeded without a project policy")
	}
	if !strings.Contains(err.Error(), "isobox init") {
		t.Fatalf("error does not direct the user to `isobox init`: %v", err)
	}
	if !strings.Contains(err.Error(), "no project policy") {
		t.Fatalf("error does not mention the missing project policy: %v", err)
	}
}

func TestRunRejectsWhenToolCallIsDisabled(t *testing.T) {
	dir := initRepo(t)
	writeRawPolicy(t, dir, strings.Replace(validDefaultBody, "  enabled: true", "  enabled: false", 1))

	err := Run(dir)
	if err == nil {
		t.Fatal("Run unexpectedly succeeded with tool_call.enabled=false")
	}
	if !strings.Contains(err.Error(), "tool_call.enabled=false") {
		t.Fatalf("error does not mention the disabled tool-call flag: %v", err)
	}
}

func TestRunRejectsDirtyTrustedRepository(t *testing.T) {
	dir := initRepo(t)
	writeRawPolicy(t, dir, validDefaultBody)

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("uncommitted\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Run(dir)
	if err == nil {
		t.Fatal("Run unexpectedly succeeded with a dirty trusted repository")
	}
	if !strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("error does not mention the dirty state: %v", err)
	}
	if !strings.Contains(err.Error(), "no dirty-source override") {
		t.Fatalf("error does not document the missing dirty-source override: %v", err)
	}
}

func TestRunRejectsUntrackedNonIgnoredFiles(t *testing.T) {
	dir := initRepo(t)
	writeRawPolicy(t, dir, validDefaultBody)

	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Run(dir)
	if err == nil {
		t.Fatal("Run unexpectedly succeeded with an untracked file")
	}
	if !strings.Contains(err.Error(), "scratch.txt") {
		t.Fatalf("error does not name the offending untracked file: %v", err)
	}
}

func TestRunRejectsWhenBubblewrapMissing(t *testing.T) {
	dir := initRepo(t)
	writeRawPolicy(t, dir, validDefaultBody)

	isolated := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(isolated, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", isolated)

	err = Run(dir)
	if err == nil {
		t.Fatal("Run unexpectedly succeeded without bubblewrap on PATH")
	}
	if !strings.Contains(err.Error(), "bubblewrap") && !strings.Contains(err.Error(), "bwrap") {
		t.Fatalf("error does not name the missing capability: %v", err)
	}
}

func TestRunRejectsUnsupportedPolicyShape(t *testing.T) {
	cases := []struct {
		name         string
		override     string
		from         string
		wantField    string
		wantSentinel string
	}{
		{
			name:         "non-bubblewrap runtime backend",
			override:     "host_process",
			from:         "runtime_backend: bubblewrap",
			wantField:    "runtime_backend",
			wantSentinel: "bubblewrap",
		},
		{
			name:         "credentials default not deny",
			override:     "scoped",
			from:         "credentials:\n  default: deny",
			wantField:    "credentials.default",
			wantSentinel: "deny",
		},
		{
			name:         "promotion mode not manual",
			override:     "automatic",
			from:         "promotion:\n  mode: manual",
			wantField:    "promotion.mode",
			wantSentinel: "manual",
		},
		{
			name:         "workspace source kind not project_root",
			override:     "subdirectory",
			from:         "workspace_source:\n  kind: project_root",
			wantField:    "workspace_source.kind",
			wantSentinel: "project_root",
		},
		{
			name:         "path mode not backend_default",
			override:     "inherit",
			from:         "development_environment:\n  path_mode: backend_default",
			wantField:    "development_environment.path_mode",
			wantSentinel: "backend_default",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := initRepo(t)
			body := overrideValue(validDefaultBody, tc.from, tc.override)
			writeRawPolicy(t, dir, body)

			err := Run(dir)
			if err == nil {
				t.Fatalf("Run unexpectedly succeeded: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantField) {
				t.Fatalf("error does not name the offending field %q: %v", tc.wantField, err)
			}
			if !strings.Contains(err.Error(), tc.wantSentinel) {
				t.Fatalf("error does not mention the supported value %q: %v", tc.wantSentinel, err)
			}
		})
	}
}

func TestRunStopsAtFirstFailure(t *testing.T) {
	dir := initRepo(t)
	// First failure is missing policy: Run reports the missing-policy
	// failure and never reaches the dirty-repo or bubblewrap checks.
	// Writing scratch.txt would be a second-failure noise; we deliberately
	// keep the repo clean here.
	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Run(dir)
	if err == nil {
		t.Fatal("Run unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "no project policy") {
		t.Fatalf("error is not the missing-policy failure (the first check should win): %v", err)
	}
	if strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("error includes later-check noise; Run should stop at the first failure: %v", err)
	}
}

// overrideValue replaces the value at the end of the first occurrence of
// originalLine with newValue. The match is anchored to the end of the
// originalLine so the caller can pass a multi-line fragment whose last
// line carries the value to override. The replacement preserves the
// originalLine's key and indentation.
func overrideValue(body, originalLine, newValue string) string {
	idx := strings.Index(body, originalLine)
	if idx < 0 {
		return body
	}
	end := idx + len(originalLine)
	// Find the last segment of originalLine to compute the value boundary.
	lastColon := strings.LastIndex(originalLine, ":")
	if lastColon < 0 {
		return body
	}
	// Everything from the colon to the end of originalLine is the value
	// being replaced.
	return body[:idx+lastColon+1] + " " + newValue + body[end:]
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	} {
		runGit(t, dir, args...)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial")
	return dir
}

func writeRawPolicy(t *testing.T, dir, body string) {
	t.Helper()

	configDir := filepath.Join(dir, ".isobox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".isobox")
	runGit(t, dir, "commit", "-m", "add project policy")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
