package workspace_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"isobox/internal/workspace"
)

func TestCreateRepositoryClonesCleanSource(t *testing.T) {
	source := initGitRepo(t)

	ws, err := workspace.CreateRepository(source)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}
	defer ws.Close()

	readme, err := os.ReadFile(filepath.Join(ws.Root(), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("cloned README = %q, want original", string(readme))
	}
}

func TestCreateRepositoryRecordsSourceCommit(t *testing.T) {
	source := initGitRepo(t)

	commit, err := workspace.HeadCommit(source)
	if err != nil {
		t.Fatalf("HeadCommit failed: %v", err)
	}
	if commit == "" {
		t.Fatal("HeadCommit returned empty commit")
	}

	ws, err := workspace.CreateRepository(source)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}
	defer ws.Close()

	if ws.SourceCommit() != commit {
		t.Fatalf("workspace source_commit = %q, want %q", ws.SourceCommit(), commit)
	}
}

func TestCreateRepositoryRejectsDirtySource(t *testing.T) {
	source := initGitRepo(t)
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("uncommitted\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := workspace.CreateRepository(source)
	if ws != nil {
		ws.Close()
	}
	if !errors.Is(err, workspace.ErrDirtyWorkspaceSource) {
		t.Fatalf("CreateRepository error = %v, want ErrDirtyWorkspaceSource", err)
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Fatalf("error does not explain dirty source: %v", err)
	}
}

func TestRepositoryWorkspaceIsolatesWritesFromSource(t *testing.T) {
	source := initGitRepo(t)

	ws, err := workspace.CreateRepository(source)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}
	defer ws.Close()

	if err := os.WriteFile(filepath.Join(ws.Root(), "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	sourceReadme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(sourceReadme) != "original\n" {
		t.Fatalf("source README was modified: %q", string(sourceReadme))
	}
}

func TestRepositoryWorkspaceDiffCapturesChanges(t *testing.T) {
	source := initGitRepo(t)

	ws, err := workspace.CreateRepository(source)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}
	defer ws.Close()

	if err := os.WriteFile(filepath.Join(ws.Root(), "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff, err := ws.Diff()
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if !strings.Contains(diff, "-original") || !strings.Contains(diff, "+changed") {
		t.Fatalf("diff does not describe workspace change:\n%s", diff)
	}
}

func TestRepositoryWorkspaceCloseRemovesPrivateCopy(t *testing.T) {
	source := initGitRepo(t)

	ws, err := workspace.CreateRepository(source)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	root := ws.Root()
	if err := ws.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("workspace root still exists after Close: %s", root)
	}

	if err := ws.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "README.md")
	run(t, dir, "git", "commit", "-m", "initial")
	return dir
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
