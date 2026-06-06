package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCreatesTaskResultFromPrivateWorkspace(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf changed > README.md; printf task-output; printf task-error >&2",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("source repository was modified: %q", readme)
	}

	recordPath := onlyTaskRecord(t, records)
	recordBytes, err := os.ReadFile(filepath.Join(recordPath, "record.json"))
	if err != nil {
		t.Fatal(err)
	}

	var record struct {
		EffectivePolicy struct {
			WorkspaceSource string   `json:"workspace_source"`
			WorkloadCommand []string `json:"workload_command"`
		} `json:"effective_policy"`
		Result struct {
			ExitStatus int    `json:"exit_status"`
			Stdout     string `json:"stdout"`
			Stderr     string `json:"stderr"`
			Diff       string `json:"diff"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		t.Fatal(err)
	}

	if record.EffectivePolicy.WorkspaceSource != source {
		t.Fatalf("workspace source not captured in effective policy: %q", record.EffectivePolicy.WorkspaceSource)
	}
	if strings.Join(record.EffectivePolicy.WorkloadCommand, " ") != "sh -c printf changed > README.md; printf task-output; printf task-error >&2" {
		t.Fatalf("workload command not captured in effective policy: %#v", record.EffectivePolicy.WorkloadCommand)
	}
	if record.Result.ExitStatus != 0 {
		t.Fatalf("exit status = %d, want 0", record.Result.ExitStatus)
	}
	if record.Result.Stdout != "task-output" {
		t.Fatalf("stdout = %q", record.Result.Stdout)
	}
	if record.Result.Stderr != "task-error" {
		t.Fatalf("stderr = %q", record.Result.Stderr)
	}
	if !strings.Contains(record.Result.Diff, "-original") || !strings.Contains(record.Result.Diff, "+changed") {
		t.Fatalf("diff does not describe workspace change:\n%s", record.Result.Diff)
	}
}

func TestPromoteAppliesReviewedTaskResultToWorkspaceSource(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf promoted > README.md",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	recordPath := onlyTaskRecord(t, records)
	promote := exec.Command("go", "run", ".", "promote", recordPath)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err = promote.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox promote failed: %v\n%s", err, output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "promoted" {
		t.Fatalf("source repository was not promoted: %q", readme)
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

func onlyTaskRecord(t *testing.T, records string) string {
	t.Helper()

	entries, err := os.ReadDir(records)
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(records, entry.Name()))
		}
	}
	if len(dirs) != 1 {
		t.Fatalf("task record dirs = %d, want 1", len(dirs))
	}
	return dirs[0]
}
