package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRecordAcceptsValidV1TaskRecord(t *testing.T) {
	dir := writeRecordFixture(t, validTaskRecord())

	loaded, err := loadRecord(dir)
	if err != nil {
		t.Fatalf("loadRecord failed for valid v1 record: %v", err)
	}

	if loaded.SchemaVersion != taskRecordSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", loaded.SchemaVersion, taskRecordSchemaVersion)
	}
	if loaded.ID != "task-abc" {
		t.Fatalf("id = %q, want task-abc", loaded.ID)
	}
	if loaded.EffectivePolicy.WorkspaceSource != "/tmp/source" {
		t.Fatalf("workspace_source = %q, want /tmp/source", loaded.EffectivePolicy.WorkspaceSource)
	}
}

func TestLoadRecordRejectsMissingRecordFile(t *testing.T) {
	dir := t.TempDir()

	_, err := loadRecord(dir)
	if err == nil {
		t.Fatal("loadRecord succeeded when record.json is missing")
	}
	if !strings.Contains(err.Error(), "read task record") {
		t.Fatalf("error does not mention reading record: %v", err)
	}
}

func TestLoadRecordRejectsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "record.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadRecord(dir)
	if err == nil {
		t.Fatal("loadRecord accepted malformed JSON")
	}
	if !strings.Contains(err.Error(), "parse task record") {
		t.Fatalf("error does not indicate parse failure: %v", err)
	}
}

func TestLoadRecordRejectsMissingSchemaVersion(t *testing.T) {
	record := validTaskRecord()
	record.SchemaVersion = ""
	dir := writeRecordFixture(t, record)

	_, err := loadRecord(dir)
	if err == nil {
		t.Fatal("loadRecord accepted record missing schema_version")
	}
	if !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("error does not mention schema_version: %v", err)
	}
}

func TestLoadRecordRejectsUnsupportedSchemaVersion(t *testing.T) {
	record := validTaskRecord()
	record.SchemaVersion = "v999"
	dir := writeRecordFixture(t, record)

	_, err := loadRecord(dir)
	if err == nil {
		t.Fatal("loadRecord accepted record with unsupported schema_version")
	}
	if !strings.Contains(err.Error(), "v999") {
		t.Fatalf("error does not mention the unsupported version: %v", err)
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error does not indicate the version is unsupported: %v", err)
	}
}

func TestLoadRecordRejectsMissingRequiredFields(t *testing.T) {
	cases := []struct {
		name         string
		mutate       func(*taskRecord)
		wantFragment string
	}{
		{
			name:         "missing id",
			mutate:       func(r *taskRecord) { r.ID = "" },
			wantFragment: "id",
		},
		{
			name:         "missing created_at",
			mutate:       func(r *taskRecord) { r.CreatedAt = "" },
			wantFragment: "created_at",
		},
		{
			name:         "missing effective_policy schema_version",
			mutate:       func(r *taskRecord) { r.EffectivePolicy.SchemaVersion = "" },
			wantFragment: "effective_policy",
		},
		{
			name:         "missing effective_policy workspace_source",
			mutate:       func(r *taskRecord) { r.EffectivePolicy.WorkspaceSource = "" },
			wantFragment: "workspace_source",
		},
		{
			name:         "missing effective_policy workload_command",
			mutate:       func(r *taskRecord) { r.EffectivePolicy.WorkloadCommand = nil },
			wantFragment: "workload_command",
		},
		{
			name:         "missing effective_policy runtime_backend",
			mutate:       func(r *taskRecord) { r.EffectivePolicy.RuntimeBackend = "" },
			wantFragment: "runtime_backend",
		},
		{
			name:         "missing effective_policy retention_default",
			mutate:       func(r *taskRecord) { r.EffectivePolicy.RetentionDefault = "" },
			wantFragment: "retention_default",
		},
		{
			name:         "missing outcome type",
			mutate:       func(r *taskRecord) { r.Outcome.Type = "" },
			wantFragment: "outcome",
		},
		{
			name:         "unknown outcome type",
			mutate:       func(r *taskRecord) { r.Outcome.Type = "catastrophic_failure" },
			wantFragment: "outcome",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := validTaskRecord()
			tc.mutate(&record)
			dir := writeRecordFixture(t, record)

			_, err := loadRecord(dir)
			if err == nil {
				t.Fatalf("loadRecord accepted record: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantFragment) {
				t.Fatalf("error does not mention %q: %v", tc.wantFragment, err)
			}
		})
	}
}

func validTaskRecord() taskRecord {
	return taskRecord{
		SchemaVersion: taskRecordSchemaVersion,
		ID:            "task-abc",
		CreatedAt:     "2026-06-20T00:00:00Z",
		EffectivePolicy: effectivePolicy{
			SchemaVersion:    "v1",
			WorkspaceSource:  "/tmp/source",
			WorkloadCommand:  []string{"sh", "-c", "true"},
			RuntimeBackend:   "host-process",
			RetentionDefault: "disposable",
		},
		Workspace: workspaceInfo{
			SourceType:   "repository",
			SourceCommit: "abc123",
			Retention:    "disposable",
		},
		Result:  taskResult{Diff: "diff content"},
		Outcome: taskAttemptOutcome{Type: outcomeSuccess},
	}
}

func writeRecordFixture(t *testing.T, record taskRecord) string {
	t.Helper()
	dir := t.TempDir()
	recordDir := filepath.Join(dir, "task-abc")
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
