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

func TestRunRecordsDefaultDenyNetworkPolicyAndHostLimitations(t *testing.T) {
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

	if record.EffectivePolicy.Network.Default != "deny" {
		t.Fatalf("network policy default = %q, want deny", record.EffectivePolicy.Network.Default)
	}
	if len(record.EffectivePolicy.Network.Allow) != 0 {
		t.Fatalf("network policy allow = %d rules, want 0", len(record.EffectivePolicy.Network.Allow))
	}

	if record.EffectivePolicy.NetworkEnforcement.RuntimeBackend != "host-process" {
		t.Fatalf("network enforcement runtime_backend = %q, want host-process", record.EffectivePolicy.NetworkEnforcement.RuntimeBackend)
	}
	if len(record.EffectivePolicy.NetworkEnforcement.Rules) == 0 {
		t.Fatal("network enforcement rules not recorded")
	}

	wantAspects := map[string]bool{"default_deny": false, "allow_rules": false}
	for _, r := range record.EffectivePolicy.NetworkEnforcement.Rules {
		matched, ok := wantAspects[r.Aspect]
		if !ok {
			t.Fatalf("unexpected network enforcement aspect: %q", r.Aspect)
		}
		if matched {
			t.Fatalf("network enforcement aspect %q recorded more than once", r.Aspect)
		}
		wantAspects[r.Aspect] = true
		if r.Status != "not_enforced" {
			t.Fatalf("%s enforcement status = %q, want not_enforced", r.Aspect, r.Status)
		}
		if !strings.Contains(r.Detail, "does not enforce") {
			t.Fatalf("%s enforcement detail does not document non-enforcement: %q", r.Aspect, r.Detail)
		}
		if !strings.Contains(r.Detail, "host network access") {
			t.Fatalf("%s enforcement detail does not record that workloads retain host network access: %q", r.Aspect, r.Detail)
		}
	}
	for aspect, seen := range wantAspects {
		if !seen {
			t.Fatalf("missing network enforcement aspect: %q", aspect)
		}
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

func TestRunRecordsRepositoryWorkspaceSourceCommit(t *testing.T) {
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

	if record.Workspace.SourceType != "repository" {
		t.Fatalf("workspace source_type = %q, want repository", record.Workspace.SourceType)
	}
	wantCommit := headCommit(t, source)
	if record.Workspace.SourceCommit != wantCommit {
		t.Fatalf("workspace source_commit = %q, want %q", record.Workspace.SourceCommit, wantCommit)
	}
}

func TestPromoteRejectsEmptyDiff(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()
	recordDir := writePromotableRecord(t, source, records, "")

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox promote succeeded for empty diff:\n%s", output)
	}
	if !strings.Contains(string(output), "no diff to promote") {
		t.Fatalf("error does not indicate empty diff:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected promotion: %q", readme)
	}
}

func TestPromoteAllowsWorkloadCommandExitOutcome(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()
	record := validPromotableRecord(t, source)
	record["result"] = map[string]any{"diff": readmeChangedDiff()}
	record["outcome"] = map[string]any{"type": "workload_command_exit", "exit_code": 7}
	recordDir := writeRecordMap(t, records, "task-failed", record)

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox promote rejected workload_command_exit outcome:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "changed\n" {
		t.Fatalf("Workspace Source was not promoted: %q", readme)
	}
}

func TestPromoteRejectsPreparationFailureOutcome(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()
	record := validPromotableRecord(t, source)
	record["outcome"] = map[string]any{"type": "preparation_failure", "error": "no workspace"}
	recordDir := writeRecordMap(t, records, "task-failed", record)

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox promote succeeded for preparation_failure outcome:\n%s", output)
	}
	out := string(output)
	if !strings.Contains(out, "preparation_failure") {
		t.Fatalf("error does not mention outcome type:\n%s", output)
	}
	if !strings.Contains(out, "only successful tasks or workload-command exits can be promoted") {
		t.Fatalf("error does not explain promotion outcome requirement:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected promotion: %q", readme)
	}
}

func TestPromoteRejectsNonRepositoryWorkspaceSourceType(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()
	record := validPromotableRecord(t, source)
	record["workspace"] = map[string]any{
		"source_type":   "directory",
		"source_commit": headCommit(t, source),
		"retention":     "disposable",
	}
	recordDir := writeRecordMap(t, records, "task-directory", record)

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox promote succeeded for directory workspace:\n%s", output)
	}
	if !strings.Contains(string(output), "only supported for Repository Workspace") {
		t.Fatalf("error does not explain repository workspace requirement:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected promotion: %q", readme)
	}
}

func TestPromoteRejectsStaleWorkspaceSource(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()
	recordDir := writePromotableRecord(t, source, records, "diff content")

	if err := os.WriteFile(filepath.Join(source, "stale.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, source, "git", "add", "stale.md")
	run(t, source, "git", "commit", "-m", "advance source")

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox promote succeeded for stale workspace source:\n%s", output)
	}
	if !strings.Contains(string(output), "trusted repository has changed") {
		t.Fatalf("error does not indicate stale source:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected promotion: %q", readme)
	}
}

func TestPromoteRejectsMissingSourceCommit(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()
	record := validPromotableRecord(t, source)
	workspace := record["workspace"].(map[string]any)
	workspace["source_commit"] = ""
	recordDir := writeRecordMap(t, records, "task-missing-commit", record)

	promote := exec.Command("go", "run", ".", "promote", recordDir)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := promote.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox promote succeeded for record missing source_commit:\n%s", output)
	}
	if !strings.Contains(string(output), "missing Workspace Source commit") {
		t.Fatalf("error does not indicate missing source_commit:\n%s", output)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected promotion: %q", readme)
	}
}

func TestRunRecordsDeclaredReuseInputsInEffectivePolicy(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--reuse-input", "host_binary=/usr/local/bin/codex",
		"--reuse-input", "path=/home/user/.codex",
		"--reuse-input", "env_var=ANTHROPIC_API_KEY",
		"--reuse-input", "credential_ref=keychain://anthropic",
		"--reuse-input", "local_integration=filesystem-mcp",
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

	if len(record.EffectivePolicy.ReuseInputs) != 5 {
		t.Fatalf("reuse_inputs = %d, want 5: %#v", len(record.EffectivePolicy.ReuseInputs), record.EffectivePolicy.ReuseInputs)
	}

	want := []struct {
		kind, value string
	}{
		{"host_binary", "/usr/local/bin/codex"},
		{"path", "/home/user/.codex"},
		{"env_var", "ANTHROPIC_API_KEY"},
		{"credential_ref", "keychain://anthropic"},
		{"local_integration", "filesystem-mcp"},
	}
	for i, w := range want {
		if record.EffectivePolicy.ReuseInputs[i].Kind != w.kind {
			t.Fatalf("reuse_inputs[%d].kind = %q, want %q", i, record.EffectivePolicy.ReuseInputs[i].Kind, w.kind)
		}
		if record.EffectivePolicy.ReuseInputs[i].Value != w.value {
			t.Fatalf("reuse_inputs[%d].value = %q, want %q", i, record.EffectivePolicy.ReuseInputs[i].Value, w.value)
		}
	}

	var foundReuseLimitation bool
	for _, l := range record.EffectivePolicy.Limitations {
		if strings.Contains(l, "host-agent-reuse") && strings.Contains(l, "5 explicit Reuse Input") {
			foundReuseLimitation = true
			break
		}
	}
	if !foundReuseLimitation {
		t.Fatalf("limitations do not make Host Agent Reuse exposure visible: %#v", record.EffectivePolicy.Limitations)
	}
}

func TestRunRecordsNoReuseInputsByDefaultWithoutBroadInheritance(t *testing.T) {
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

	if len(record.EffectivePolicy.ReuseInputs) != 0 {
		t.Fatalf("default reuse_inputs = %d, want 0 (no silent host inheritance): %#v", len(record.EffectivePolicy.ReuseInputs), record.EffectivePolicy.ReuseInputs)
	}
	for _, l := range record.EffectivePolicy.Limitations {
		if strings.Contains(l, "host-agent-reuse") {
			t.Fatalf("default limitations must not claim Host Agent Reuse exposure: %q", l)
		}
		if strings.Contains(l, "inherited") || strings.Contains(l, "inherit") {
			t.Fatalf("default limitations must not imply broad host inheritance: %q", l)
		}
	}
}

func TestRunRejectsReuseInputWithUnknownKind(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--reuse-input", "home_directory=/home/user",
		"--",
		"sh", "-c", "true",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox run succeeded for unsupported reuse input kind:\n%s", output)
	}
	if !strings.Contains(string(output), "unsupported reuse input kind") {
		t.Fatalf("error does not reject unsupported reuse input kind:\n%s", output)
	}

	entries, err := os.ReadDir(records)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("task record written for rejected run: %d entries", len(entries))
	}
}

func TestRunRejectsReuseInputMissingValue(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--reuse-input", "host_binary=",
		"--",
		"sh", "-c", "true",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox run succeeded for empty reuse input value:\n%s", output)
	}
	if !strings.Contains(string(output), "empty value") {
		t.Fatalf("error does not reject empty reuse input value:\n%s", output)
	}
}

func TestRunRejectsReuseInputMissingEquals(t *testing.T) {
	source := initGitRepo(t)
	records := t.TempDir()

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--reuse-input", "host_binary",
		"--",
		"sh", "-c", "true",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox run succeeded for malformed reuse input:\n%s", output)
	}
	if !strings.Contains(string(output), "kind=value") {
		t.Fatalf("error does not explain kind=value format:\n%s", output)
	}
}

func TestRunGeneratesPromotionReportWithHighRiskCategories(t *testing.T) {
	source := initGitRepoWithTrackedHighRiskFiles(t)
	records := t.TempDir()

	// The Workload Command modifies pre-existing tracked files spanning each
	// high-risk category plus an ordinary source file. The Repository Workspace
	// diff captures modifications to tracked files, so each category is
	// detectable from the captured Task Result.
	ordinary := "printf 'new\n' >> README.md"
	script := "printf 'echo run\n' >> scripts/build.sh"
	hook := "printf '#!/bin/sh\necho hook\n' > .husky/pre-commit"
	manifest := "printf '{\"name\":\"app\"}\n' > package.json"
	ci := "printf 'on: [push]\n' > .github/workflows/ci.yml"
	large := "yes generated | head -n 600 > report.txt"
	binary := "printf '\\x89PNG\\r\\n\\x1a\\n\\x00\\x00\\x00' > assets/logo.png"
	workload := strings.Join([]string{ordinary, script, hook, manifest, ci, large, binary}, "; ")

	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", workload,
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)

	if record.PromotionReport == nil {
		t.Fatalf("task record missing promotion_report")
	}
	if record.PromotionReport.SchemaVersion != "v1" {
		t.Fatalf("promotion_report schema_version = %q, want v1", record.PromotionReport.SchemaVersion)
	}

	changedByPath := map[string]fileChangeView{}
	for _, c := range record.PromotionReport.ChangedFiles {
		changedByPath[c.Path] = c
	}

	wantChanged := []string{
		"README.md",
		"scripts/build.sh",
		".husky/pre-commit",
		"package.json",
		".github/workflows/ci.yml",
		"report.txt",
		"assets/logo.png",
	}
	for _, p := range wantChanged {
		if _, ok := changedByPath[p]; !ok {
			t.Fatalf("promotion_report changed_files missing %q: %#v", p, record.PromotionReport.ChangedFiles)
		}
	}

	if c, ok := changedByPath["README.md"]; ok && len(c.Categories) != 0 {
		t.Fatalf("ordinary README.md change flagged high-risk: %#v", c.Categories)
	}

	haveCategory := func(category string) bool {
		for _, hr := range record.PromotionReport.HighRisk {
			if hr.Category == category {
				return true
			}
		}
		return false
	}
	for _, category := range []string{
		"script", "hook", "dependency_manifest", "ci_workflow", "large_file", "binary",
	} {
		if !haveCategory(category) {
			t.Fatalf("promotion_report high_risk missing %q: %#v", category, record.PromotionReport.HighRisk)
		}
	}
}

func TestPromotionReportIsInformationalAndPromotionStaysExplicit(t *testing.T) {
	source := initGitRepoWithTrackedHighRiskFiles(t)
	records := t.TempDir()

	// A high-risk change (modifying a dependency manifest) should still be
	// promotable on explicit request; the report flags risk but never gates
	// Promotion. The user remains the review gate.
	cmd := exec.Command(
		"go", "run", ".",
		"run",
		"--source", source,
		"--records", records,
		"--",
		"sh", "-c", "printf '{\"name\":\"app\"}\n' > package.json",
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox run failed: %v\n%s", err, output)
	}

	recordPath := onlyTaskRecord(t, records)
	record := readRecord(t, recordPath)
	if record.PromotionReport == nil {
		t.Fatalf("task record missing promotion_report")
	}
	if !highRiskHasCategory(record.PromotionReport.HighRisk, "dependency_manifest") {
		t.Fatalf("promotion_report did not flag high-risk dependency manifest: %#v", record.PromotionReport.HighRisk)
	}

	promote := exec.Command("go", "run", ".", "promote", recordPath)
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	promoteOutput, err := promote.CombinedOutput()
	if err != nil {
		t.Fatalf("explicit promote of high-risk result failed: %v\n%s", err, promoteOutput)
	}
	if !strings.Contains(string(promoteOutput), "promotion report:") {
		t.Fatalf("promote did not print the promotion report:\n%s", promoteOutput)
	}
	if !strings.Contains(string(promoteOutput), "package.json") {
		t.Fatalf("promote report did not list the changed file:\n%s", promoteOutput)
	}

	promoted, err := os.ReadFile(filepath.Join(source, "package.json"))
	if err != nil {
		t.Fatalf("read promoted package.json: %v", err)
	}
	if string(promoted) != "{\"name\":\"app\"}\n" {
		t.Fatalf("high-risk change was not applied by explicit promotion: %q", promoted)
	}
}

func highRiskHasCategory(highRisk []highRiskView, category string) bool {
	for _, hr := range highRisk {
		if hr.Category == category {
			return true
		}
	}
	return false
}

func initGitRepoWithTrackedHighRiskFiles(t *testing.T) string {
	t.Helper()

	dir := initGitRepo(t)
	// Pre-create tracked files for each high-risk category so a Workload
	// Command can modify them and the Repository Workspace diff captures
	// the change.
	mustWrite(t, dir, "package.json", []byte("{}\n"))
	mustWrite(t, dir, "scripts/build.sh", []byte("#!/bin/sh\necho build\n"))
	mustWrite(t, dir, ".husky/pre-commit", []byte("#!/bin/sh\n"))
	mustWrite(t, dir, ".github/workflows/ci.yml", []byte("on: []\n"))
	mustWrite(t, dir, "report.txt", []byte("line\n"))
	// A tracked binary file (NUL bytes so Git treats it as binary).
	mustWrite(t, dir, "assets/logo.png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x01})
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "add tracked high-risk files")
	return dir
}

func mustWrite(t *testing.T, dir, rel string, content []byte) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func headCommit(t *testing.T, dir string) string {
	t.Helper()

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse failed: %v\n%s", err, output)
	}
	return strings.TrimSpace(string(output))
}

func validPromotableRecord(t *testing.T, source string) map[string]any {
	t.Helper()

	return map[string]any{
		"schema_version": "v1",
		"id":             "task-promote",
		"created_at":     "2026-06-20T00:00:00Z",
		"effective_policy": map[string]any{
			"schema_version":    "v1",
			"workspace_source":  source,
			"workload_command":  []string{"sh", "-c", "true"},
			"runtime_backend":   "host-process",
			"retention_default": "disposable",
		},
		"workspace": map[string]any{
			"source_type":   "repository",
			"source_commit": headCommit(t, source),
			"retention":     "disposable",
		},
		"result":  map[string]any{"diff": "diff content"},
		"outcome": map[string]any{"type": "success"},
	}
}

func writePromotableRecord(t *testing.T, source, records, diff string) string {
	t.Helper()

	record := validPromotableRecord(t, source)
	record["result"] = map[string]any{"diff": diff}
	return writeRecordMap(t, records, "task-promote", record)
}

func readmeChangedDiff() string {
	return "diff --git a/README.md b/README.md\n" +
		"index 3be9c81..5ea2ed4 100644\n" +
		"--- a/README.md\n" +
		"+++ b/README.md\n" +
		"@@ -1 +1 @@\n" +
		"-original\n" +
		"+changed\n"
}

func writeRecordMap(t *testing.T, records, id string, record map[string]any) string {
	t.Helper()

	recordDir := filepath.Join(records, id)
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		t.Fatal(err)
	}
	recordBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recordDir, "record.json"), append(recordBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return recordDir
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
		Network struct {
			Default string `json:"default"`
			Allow   []struct {
				Origin     string `json:"origin"`
				PathPrefix string `json:"path_prefix"`
				Method     string `json:"method"`
			} `json:"allow"`
		} `json:"network"`
		NetworkEnforcement struct {
			RuntimeBackend string `json:"runtime_backend"`
			Rules          []struct {
				Aspect string `json:"aspect"`
				Status string `json:"status"`
				Detail string `json:"detail"`
			} `json:"rules"`
		} `json:"network_enforcement"`
		ReuseInputs []struct {
			Kind  string `json:"kind"`
			Value string `json:"value"`
		} `json:"reuse_inputs"`
		Limitations []string `json:"limitations"`
	} `json:"effective_policy"`
	Workspace struct {
		SourceType   string `json:"source_type"`
		SourceCommit string `json:"source_commit"`
		Retention    string `json:"retention"`
		Path         string `json:"path"`
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
	PromotionReport *promotionReportView `json:"promotion_report,omitempty"`
}

type promotionReportView struct {
	SchemaVersion string           `json:"schema_version"`
	ChangedFiles  []fileChangeView `json:"changed_files"`
	HighRisk      []highRiskView   `json:"high_risk"`
}

type fileChangeView struct {
	Path         string   `json:"path"`
	Status       string   `json:"status"`
	Categories   []string `json:"categories,omitempty"`
	AddedLines   int      `json:"added_lines"`
	RemovedLines int      `json:"removed_lines"`
}

type highRiskView struct {
	Category string   `json:"category"`
	Paths    []string `json:"paths"`
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
