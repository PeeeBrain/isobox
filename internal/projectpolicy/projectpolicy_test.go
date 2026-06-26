package projectpolicy_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"isobox/internal/projectpolicy"
)

func TestLoadFindsConfigAtProjectRoot(t *testing.T) {
	dir := t.TempDir()

	gitInit(t, dir)

	isoboxDir := filepath.Join(dir, ".isobox")
	if err := os.MkdirAll(isoboxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rendered, err := projectpolicy.Default().Render()
	if err != nil {
		t.Fatalf("render default policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(isoboxDir, "config.yaml"), []byte(rendered), 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	policy, err := projectpolicy.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if policy.APIVersion != projectpolicy.APIVersion {
		t.Fatalf("api_version = %q, want %q", policy.APIVersion, projectpolicy.APIVersion)
	}
	if policy.Kind != projectpolicy.Kind {
		t.Fatalf("kind = %q, want %q", policy.Kind, projectpolicy.Kind)
	}
	if policy.RuntimeBackend != projectpolicy.RuntimeBackendBubblewrap {
		t.Fatalf("runtime_backend = %q, want %q", policy.RuntimeBackend, projectpolicy.RuntimeBackendBubblewrap)
	}
}

func TestLoadWalksUpwardFromSubdirectory(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	isoboxDir := filepath.Join(dir, ".isobox")
	if err := os.MkdirAll(isoboxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rendered, err := projectpolicy.Default().Render()
	if err != nil {
		t.Fatalf("render default policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(isoboxDir, "config.yaml"), []byte(rendered), 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	nested := filepath.Join(dir, "src", "pkg", "internal")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	policy, err := projectpolicy.Load(nested)
	if err != nil {
		t.Fatalf("Load from subdirectory failed: %v", err)
	}
	if policy.APIVersion != projectpolicy.APIVersion {
		t.Fatalf("api_version = %q, want %q", policy.APIVersion, projectpolicy.APIVersion)
	}
}

func TestLoadErrorsWhenConfigMissing(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	_, err := projectpolicy.Load(dir)
	if err == nil {
		t.Fatalf("Load unexpectedly succeeded without a project policy")
	}
	if !strings.Contains(err.Error(), "isobox init") {
		t.Fatalf("error does not direct the user to `isobox init`: %v", err)
	}
}

func TestLoadIgnoresProjectPolicyOutsideGitRoot(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	subdir := filepath.Join(dir, "nested")
	if err := os.MkdirAll(filepath.Join(subdir, ".isobox"), 0o755); err != nil {
		t.Fatal(err)
	}
	rendered, err := projectpolicy.Default().Render()
	if err != nil {
		t.Fatalf("render default policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, ".isobox", "config.yaml"), []byte(rendered), 0o644); err != nil {
		t.Fatalf("write misplaced policy: %v", err)
	}

	_, err = projectpolicy.Load(subdir)
	if err == nil {
		t.Fatalf("Load unexpectedly succeeded with project policy outside Git root")
	}
	if !strings.Contains(err.Error(), "isobox init") {
		t.Fatalf("error does not direct the user to `isobox init`: %v", err)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()

	runGit(t, dir, "init")
	for _, args := range [][]string{
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	} {
		runGit(t, dir, args...)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
