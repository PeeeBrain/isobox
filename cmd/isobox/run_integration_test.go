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
	record := readRecord(t, recordPath)

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
	if record.Outcome.Type != "success" {
		t.Fatalf("outcome = %q, want success", record.Outcome.Type)
	}
}

func TestRunCapturesSchemaVersionedEffectivePolicy(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf changed > README.md",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	recordPath := onlyTaskRecord(t, records)
	recordBytes, err := os.ReadFile(filepath.Join(recordPath, "record.json"))
	if err != nil {
		t.Fatal(err)
	}

	var record struct {
		EffectivePolicy struct {
			SchemaVersion    string   `json:"schema_version"`
			WorkspaceSource  string   `json:"workspace_source"`
			WorkloadCommand  []string `json:"workload_command"`
			RuntimeBackend   string   `json:"runtime_backend"`
			RetentionDefault string   `json:"retention_default"`
			Limitations      []string `json:"limitations"`
		} `json:"effective_policy"`
	}
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		t.Fatal(err)
	}

	if record.EffectivePolicy.SchemaVersion != "v1" {
		t.Fatalf("effective policy schema version = %q, want v1", record.EffectivePolicy.SchemaVersion)
	}
	if record.EffectivePolicy.WorkspaceSource != source {
		t.Fatalf("workspace source = %q, want %q", record.EffectivePolicy.WorkspaceSource, source)
	}
	if strings.Join(record.EffectivePolicy.WorkloadCommand, " ") != "sh -c printf changed > README.md" {
		t.Fatalf("workload command = %#v", record.EffectivePolicy.WorkloadCommand)
	}
	if record.EffectivePolicy.RuntimeBackend != "host-process" {
		t.Fatalf("runtime backend = %q, want host-process", record.EffectivePolicy.RuntimeBackend)
	}
	if record.EffectivePolicy.RetentionDefault != "disposable" {
		t.Fatalf("retention default = %q, want disposable", record.EffectivePolicy.RetentionDefault)
	}
	if len(record.EffectivePolicy.Limitations) == 0 {
		t.Fatalf("limitations not recorded")
	}
	var foundHostProcessLimitation bool
	for _, limitation := range record.EffectivePolicy.Limitations {
		if strings.Contains(limitation, "host-process") {
			foundHostProcessLimitation = true
			break
		}
	}
	if !foundHostProcessLimitation {
		t.Fatalf("limitations do not document host-process lower assurance: %#v", record.EffectivePolicy.Limitations)
	}
}

func TestRunRecordsResourcePolicyAndEnforcement(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf changed > README.md",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)

	if record.EffectivePolicy.ResourceLimits.MaxDurationSeconds != 0 {
		t.Fatalf("default max_duration_seconds = %d, want 0", record.EffectivePolicy.ResourceLimits.MaxDurationSeconds)
	}
	if record.EffectivePolicy.ResourceLimits.MaxOutputBytes != 0 {
		t.Fatalf("default max_output_bytes = %d, want 0", record.EffectivePolicy.ResourceLimits.MaxOutputBytes)
	}
	if record.EffectivePolicy.ResourceLimits.MaxCPUCores != 0 {
		t.Fatalf("default max_cpu_cores = %d, want 0", record.EffectivePolicy.ResourceLimits.MaxCPUCores)
	}
	if record.EffectivePolicy.ResourceLimits.MaxMemoryBytes != 0 {
		t.Fatalf("default max_memory_bytes = %d, want 0", record.EffectivePolicy.ResourceLimits.MaxMemoryBytes)
	}
	if record.EffectivePolicy.ResourceLimits.MaxProcesses != 0 {
		t.Fatalf("default max_processes = %d, want 0", record.EffectivePolicy.ResourceLimits.MaxProcesses)
	}
	if record.EffectivePolicy.ResourceLimits.MaxDiskBytes != 0 {
		t.Fatalf("default max_disk_bytes = %d, want 0", record.EffectivePolicy.ResourceLimits.MaxDiskBytes)
	}
	if record.EffectivePolicy.ResourceLimits.MaxFileDescriptors != 0 {
		t.Fatalf("default max_file_descriptors = %d, want 0", record.EffectivePolicy.ResourceLimits.MaxFileDescriptors)
	}

	if record.EffectivePolicy.ResourceEnforcement.RuntimeBackend != "host-process" {
		t.Fatalf("resource enforcement runtime_backend = %q, want host-process", record.EffectivePolicy.ResourceEnforcement.RuntimeBackend)
	}
	if len(record.EffectivePolicy.ResourceEnforcement.Limits) == 0 {
		t.Fatal("resource enforcement limits not recorded")
	}

	for _, l := range record.EffectivePolicy.ResourceEnforcement.Limits {
		if l.Status != "not_enforced" {
			t.Fatalf("%s enforcement status = %q, want not_enforced", l.Name, l.Status)
		}
		if !strings.Contains(l.Detail, "does not enforce") {
			t.Fatalf("%s enforcement detail does not document non-enforcement: %q", l.Name, l.Detail)
		}
	}

	if !strings.Contains(string(output), "workspace disposed") {
		t.Fatalf("CLI output does not announce disposable workspace:\n%s", output)
	}
}

func TestRunRecordsEffectivePolicyWhenWorkloadCommandFails(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf changed > README.md; exit 7",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox run succeeded for failing workload command:\n%s", output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)

	if record.EffectivePolicy.SchemaVersion != "v1" {
		t.Fatalf("effective policy schema version = %q, want v1", record.EffectivePolicy.SchemaVersion)
	}
	if record.EffectivePolicy.RuntimeBackend != "host-process" {
		t.Fatalf("runtime backend = %q, want host-process", record.EffectivePolicy.RuntimeBackend)
	}
	if record.EffectivePolicy.RetentionDefault != "disposable" {
		t.Fatalf("retention default = %q, want disposable", record.EffectivePolicy.RetentionDefault)
	}
	if len(record.EffectivePolicy.Limitations) == 0 {
		t.Fatalf("limitations not recorded")
	}
	if record.Result.ExitStatus != 7 {
		t.Fatalf("exit status = %d, want 7", record.Result.ExitStatus)
	}
	if record.Outcome.Type != "workload_command_exit" {
		t.Fatalf("outcome = %q, want workload_command_exit", record.Outcome.Type)
	}
	if record.Outcome.ExitCode != 7 {
		t.Fatalf("outcome exit code = %d, want 7", record.Outcome.ExitCode)
	}
}

func TestRunRejectsWorkspaceSourceWithUncommittedChanges(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("uncommitted\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf should-not-run > README.md",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox run succeeded with uncommitted Workspace Source changes:\n%s", output)
	}
	if !strings.Contains(string(output), "Workspace Source has uncommitted changes; commit them before running isobox") {
		t.Fatalf("error does not explain committed content requirement:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "uncommitted\n" {
		t.Fatalf("Workspace Source was modified: %q", readme)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)
	if record.Outcome.Type != "preparation_failure" {
		t.Fatalf("outcome = %q, want preparation_failure", record.Outcome.Type)
	}
	if record.Outcome.Error == "" {
		t.Fatal("preparation failure record missing error")
	}
}

func TestRunRecordsLaunchFailure(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"this-binary-does-not-exist",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox run succeeded for missing workload command:\n%s", output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)
	if record.Outcome.Type != "launch_failure" {
		t.Fatalf("outcome = %q, want launch_failure", record.Outcome.Type)
	}
	if record.Outcome.Error == "" {
		t.Fatal("launch failure record missing error")
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified: %q", readme)
	}
}

func TestRunRecordsResultCaptureFailure(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "rm -rf .git",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox run succeeded when result capture failed:\n%s", output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)
	if record.Outcome.Type != "result_capture_failure" {
		t.Fatalf("outcome = %q, want result_capture_failure", record.Outcome.Type)
	}
	if record.Outcome.Error == "" {
		t.Fatal("result-capture failure record missing error")
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified: %q", readme)
	}
}

func TestRunRetainsWorkspaceWhenRequested(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--retain-workspace",
		"--",
		"sh", "-c", "printf changed > README.md",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)

	if record.Workspace.Retention != "retained" {
		t.Fatalf("workspace retention = %q, want retained", record.Workspace.Retention)
	}
	if record.EffectivePolicy.RetentionDefault != "retained" {
		t.Fatalf("effective policy retention default = %q, want retained", record.EffectivePolicy.RetentionDefault)
	}
	if record.Workspace.Path == "" {
		t.Fatal("retained workspace path not recorded")
	}
	if _, err := os.Stat(record.Workspace.Path); err != nil {
		t.Fatalf("retained workspace path does not exist: %v", err)
	}

	readme, err := os.ReadFile(filepath.Join(record.Workspace.Path, "README.md"))
	if err != nil {
		t.Fatalf("read retained workspace README: %v", err)
	}
	if string(readme) != "changed" {
		t.Fatalf("retained workspace README = %q, want changed", readme)
	}

	if !strings.Contains(string(output), "workspace retained at:") {
		t.Fatalf("CLI output does not announce retained workspace:\n%s", output)
	}
	if !strings.Contains(string(output), record.Workspace.Path) {
		t.Fatalf("CLI output does not include retained workspace path:\n%s", output)
	}
}

func TestRunRetainsWorkspaceOnWorkloadFailure(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--retain-workspace",
		"--",
		"sh", "-c", "printf changed > README.md; exit 7",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox run succeeded for failing workload command:\n%s", output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)

	if record.Outcome.Type != "workload_command_exit" {
		t.Fatalf("outcome = %q, want workload_command_exit", record.Outcome.Type)
	}
	if record.Workspace.Retention != "retained" {
		t.Fatalf("workspace retention = %q, want retained", record.Workspace.Retention)
	}
	if record.Workspace.Path == "" {
		t.Fatal("retained workspace path not recorded for failed workload")
	}
	if _, err := os.Stat(record.Workspace.Path); err != nil {
		t.Fatalf("retained workspace path does not exist after failure: %v", err)
	}

	readme, err := os.ReadFile(filepath.Join(record.Workspace.Path, "README.md"))
	if err != nil {
		t.Fatalf("read retained workspace README: %v", err)
	}
	if string(readme) != "changed" {
		t.Fatalf("retained workspace README = %q, want changed", readme)
	}

	if !strings.Contains(string(output), "workspace retained at:") {
		t.Fatalf("CLI output does not announce retained workspace:\n%s", output)
	}
}

func TestRunCleansUpWorkspaceByDefault(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()
	tmpDir := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf changed > README.md",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	cmd.Env = append(os.Environ(), "TMPDIR="+tmpDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)

	if record.Workspace.Retention != "disposable" {
		t.Fatalf("workspace retention = %q, want disposable", record.Workspace.Retention)
	}
	if record.Workspace.Path != "" {
		t.Fatalf("disposable workspace path = %q, want empty", record.Workspace.Path)
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "isobox-workspace-") {
			t.Fatalf("workspace was not cleaned up: %s", entry.Name())
		}
	}

	if !strings.Contains(string(output), "workspace disposed") {
		t.Fatalf("CLI output does not announce disposable workspace:\n%s", output)
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

func TestRunWritesSchemaVersionedTaskRecord(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf changed > README.md",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)
	if record.SchemaVersion != "v1" {
		t.Fatalf("task record schema_version = %q, want v1", record.SchemaVersion)
	}
	if record.EffectivePolicy.SchemaVersion != "v1" {
		t.Fatalf("effective policy schema_version = %q, want v1", record.EffectivePolicy.SchemaVersion)
	}
}

func TestPromoteRejectsMalformedTaskRecord(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	recordDir := filepath.Join(records, "task-malformed")
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recordDir, "record.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox promote succeeded for malformed record:\n%s", output)
	}
	if !strings.Contains(string(output), "parse task record") {
		t.Fatalf("error does not indicate parse failure:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected promotion: %q", readme)
	}
}

func TestPromoteRejectsTaskRecordMissingRequiredFields(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	recordDir := filepath.Join(records, "task-incomplete")
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		t.Fatal(err)
	}
	record := map[string]any{
		"schema_version": "v1",
		"created_at":     "2026-06-20T00:00:00Z",
		"effective_policy": map[string]any{
			"schema_version":    "v1",
			"workspace_source":  source,
			"workload_command":  []string{"sh", "-c", "true"},
			"runtime_backend":   "host-process",
			"retention_default": "disposable",
		},
		"result":  map[string]any{"diff": "should-not-apply"},
		"outcome": map[string]any{"type": "success"},
	}
	recordBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recordDir, "record.json"), append(recordBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox promote succeeded for record missing required fields:\n%s", output)
	}
	if !strings.Contains(string(output), "id") {
		t.Fatalf("error does not mention the missing id field:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected promotion: %q", readme)
	}
}

func TestPromoteRejectsUnsupportedTaskRecordSchemaVersion(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	recordDir := filepath.Join(records, "task-future")
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		t.Fatal(err)
	}
	record := map[string]any{
		"schema_version": "v999",
		"id":             "task-future",
		"created_at":     "2026-06-20T00:00:00Z",
		"effective_policy": map[string]any{
			"schema_version":    "v1",
			"workspace_source":  source,
			"workload_command":  []string{"sh", "-c", "true"},
			"runtime_backend":   "host-process",
			"retention_default": "disposable",
		},
		"result":  map[string]any{"diff": "should-not-apply"},
		"outcome": map[string]any{"type": "success"},
	}
	recordBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recordDir, "record.json"), append(recordBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox promote succeeded for record with unsupported schema_version:\n%s", output)
	}
	out := string(output)
	if !strings.Contains(out, "v999") {
		t.Fatalf("error does not mention the unsupported schema_version:\n%s", output)
	}
	if !strings.Contains(out, "unsupported") {
		t.Fatalf("error does not indicate the schema_version is unsupported:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected promotion: %q", readme)
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

func readRecord(t *testing.T, recordDir string) recordView {
	t.Helper()

	recordBytes, err := os.ReadFile(filepath.Join(recordDir, "record.json"))
	if err != nil {
		t.Fatal(err)
	}

	var record recordView
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		t.Fatal(err)
	}
	return record
}

type recordView struct {
	SchemaVersion   string `json:"schema_version"`
	EffectivePolicy struct {
		SchemaVersion    string   `json:"schema_version"`
		WorkspaceSource  string   `json:"workspace_source"`
		WorkloadCommand  []string `json:"workload_command"`
		RuntimeBackend   string   `json:"runtime_backend"`
		RetentionDefault string   `json:"retention_default"`
		ResourceLimits   struct {
			MaxDurationSeconds int64 `json:"max_duration_seconds"`
			MaxOutputBytes     int64 `json:"max_output_bytes"`
			MaxCPUCores        int64 `json:"max_cpu_cores"`
			MaxMemoryBytes     int64 `json:"max_memory_bytes"`
			MaxProcesses       int64 `json:"max_processes"`
			MaxDiskBytes       int64 `json:"max_disk_bytes"`
			MaxFileDescriptors int64 `json:"max_file_descriptors"`
		} `json:"resource_limits"`
		ResourceEnforcement struct {
			RuntimeBackend string `json:"runtime_backend"`
			Limits         []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Detail string `json:"detail"`
			} `json:"limits"`
		} `json:"resource_enforcement"`
		Limitations []string `json:"limitations"`
	} `json:"effective_policy"`
	Workspace struct {
		Retention string `json:"retention"`
		Path      string `json:"path"`
	} `json:"workspace"`
	Result struct {
		ExitStatus int    `json:"exit_status"`
		Stdout     string `json:"stdout"`
		Stderr     string `json:"stderr"`
		Diff       string `json:"diff"`
	} `json:"result"`
	Outcome struct {
		Type     string `json:"type"`
		ExitCode int    `json:"exit_code"`
		Error    string `json:"error"`
	} `json:"outcome"`
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
