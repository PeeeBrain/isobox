package promotion

import (
	"strings"
	"testing"
)

// diffFor builds a minimal but realistic unified Git diff for a single file.
// mode may be "" (modified), "added", "added-exec", "deleted", or "binary".
func diffFor(path, mode, content string) string {
	var b strings.Builder
	b.WriteString("diff --git a/" + path + " b/" + path + "\n")
	switch mode {
	case "added":
		b.WriteString("new file mode 100644\n")
		b.WriteString("index 0000000..1111111\n")
		b.WriteString("--- /dev/null\n")
		b.WriteString("+++ b/" + path + "\n")
	case "added-exec":
		b.WriteString("new file mode 100755\n")
		b.WriteString("index 0000000..1111111\n")
		b.WriteString("--- /dev/null\n")
		b.WriteString("+++ b/" + path + "\n")
	case "deleted":
		b.WriteString("deleted file mode 100644\n")
		b.WriteString("index 1111111..0000000\n")
		b.WriteString("--- a/" + path + "\n")
		b.WriteString("+++ /dev/null\n")
	case "binary":
		b.WriteString("new file mode 100644\n")
		b.WriteString("index 0000000..1111111\n")
		b.WriteString("Binary files /dev/null and b/" + path + " differ\n")
		return b.String()
	default:
		b.WriteString("index 1111111..2222222 100644\n")
		b.WriteString("--- a/" + path + "\n")
		b.WriteString("+++ b/" + path + "\n")
	}
	if content == "" {
		return b.String()
	}
	b.WriteString("@@ -1,1 +1,1 @@\n")
	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		b.WriteString(line + "\n")
	}
	return b.String()
}

func TestGenerateReportEmptyDiffHasNoChangedFiles(t *testing.T) {
	report := GenerateReport("")
	if report.SchemaVersion != ReportSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", report.SchemaVersion, ReportSchemaVersion)
	}
	if len(report.ChangedFiles) != 0 {
		t.Fatalf("changed_files = %d, want 0", len(report.ChangedFiles))
	}
	if len(report.HighRisk) != 0 {
		t.Fatalf("high_risk = %d, want 0", len(report.HighRisk))
	}
}

func TestGenerateReportRecordsOrdinarySourceChangeWithoutHighRisk(t *testing.T) {
	diff := diffFor("src/main.go", "modified", "-old\n+new\n")
	report := GenerateReport(diff)

	if len(report.ChangedFiles) != 1 {
		t.Fatalf("changed_files = %d, want 1", len(report.ChangedFiles))
	}
	c := report.ChangedFiles[0]
	if c.Path != "src/main.go" {
		t.Fatalf("path = %q, want src/main.go", c.Path)
	}
	if c.Status != "modified" {
		t.Fatalf("status = %q, want modified", c.Status)
	}
	if c.AddedLines != 1 || c.RemovedLines != 1 {
		t.Fatalf("added/removed = %d/%d, want 1/1", c.AddedLines, c.RemovedLines)
	}
	if len(c.Categories) != 0 {
		t.Fatalf("ordinary source file flagged as high-risk: %#v", c.Categories)
	}
	if len(report.HighRisk) != 0 {
		t.Fatalf("high_risk = %d, want 0 for ordinary change", len(report.HighRisk))
	}
}

func TestGenerateReportFlagsAddedScriptByExtension(t *testing.T) {
	diff := diffFor("build.sh", "added", "+echo build\n")
	report := GenerateReport(diff)

	c := report.ChangedFiles[0]
	if c.Status != "added" {
		t.Fatalf("status = %q, want added", c.Status)
	}
	if !containsCategory(c.Categories, CategoryScript) {
		t.Fatalf("script category not flagged: %#v", c.Categories)
	}
	if !hasHighRisk(report, CategoryScript, "build.sh") {
		t.Fatalf("high-risk script not grouped: %#v", report.HighRisk)
	}
}

func TestGenerateReportFlagsAddedExecutableScriptByMode(t *testing.T) {
	diff := diffFor("runme", "added-exec", "+./runme\n")
	report := GenerateReport(diff)

	c := report.ChangedFiles[0]
	if !containsCategory(c.Categories, CategoryScript) {
		t.Fatalf("executable new file not flagged as script: %#v", c.Categories)
	}
}

func TestGenerateReportFlagsGitHook(t *testing.T) {
	diff := diffFor(".husky/pre-commit", "added", "+#!/bin/sh\n")
	report := GenerateReport(diff)

	c := report.ChangedFiles[0]
	if !containsCategory(c.Categories, CategoryHook) {
		t.Fatalf("hook category not flagged: %#v", c.Categories)
	}
	if !hasHighRisk(report, CategoryHook, ".husky/pre-commit") {
		t.Fatalf("high-risk hook not grouped: %#v", report.HighRisk)
	}
}

func TestGenerateReportFlagsDependencyManifest(t *testing.T) {
	for _, manifest := range []string{"package.json", "go.mod", "Cargo.lock", "package-lock.json"} {
		t.Run(manifest, func(t *testing.T) {
			diff := diffFor(manifest, "modified", "-{}\n+{ \"name\": \"app\" }\n")
			report := GenerateReport(diff)

			c := report.ChangedFiles[0]
			if !containsCategory(c.Categories, CategoryDependencyManifest) {
				t.Fatalf("dependency manifest not flagged: %#v", c.Categories)
			}
			if !hasHighRisk(report, CategoryDependencyManifest, manifest) {
				t.Fatalf("high-risk dependency manifest not grouped: %#v", report.HighRisk)
			}
		})
	}
}

func TestGenerateReportFlagsCIWorkflow(t *testing.T) {
	for _, path := range []string{".github/workflows/ci.yml", ".gitlab-ci.yml", "Jenkinsfile"} {
		t.Run(path, func(t *testing.T) {
			diff := diffFor(path, "modified", "-old: ci\n+new: ci\n")
			report := GenerateReport(diff)

			c := report.ChangedFiles[0]
			if !containsCategory(c.Categories, CategoryCIWorkflow) {
				t.Fatalf("CI workflow not flagged: %#v", c.Categories)
			}
			if !hasHighRisk(report, CategoryCIWorkflow, path) {
				t.Fatalf("high-risk CI workflow not grouped: %#v", report.HighRisk)
			}
		})
	}
}

func TestGenerateReportFlagsLargeFileByChangedLines(t *testing.T) {
	var content strings.Builder
	content.WriteString("@@ -1,1 +1,600 @@\n")
	for i := 0; i < 600; i++ {
		content.WriteString("+generated line\n")
	}
	diff := diffFor("generated/report.txt", "added", strings.TrimRight(content.String(), "\n"))
	report := GenerateReport(diff)

	c := report.ChangedFiles[0]
	if !containsCategory(c.Categories, CategoryLargeFile) {
		t.Fatalf("large file not flagged: added=%d, categories=%#v", c.AddedLines, c.Categories)
	}
	if !hasHighRisk(report, CategoryLargeFile, "generated/report.txt") {
		t.Fatalf("high-risk large file not grouped: %#v", report.HighRisk)
	}
}

func TestGenerateReportFlagsBinaryChange(t *testing.T) {
	diff := diffFor("assets/logo.png", "binary", "")
	report := GenerateReport(diff)

	c := report.ChangedFiles[0]
	if !containsCategory(c.Categories, CategoryBinary) {
		t.Fatalf("binary change not flagged: %#v", c.Categories)
	}
	if !hasHighRisk(report, CategoryBinary, "assets/logo.png") {
		t.Fatalf("high-risk binary not grouped: %#v", report.HighRisk)
	}
}

func TestGenerateReportDetectsDeletionStatus(t *testing.T) {
	diff := diffFor("old/removed.go", "deleted", "-removed\n")
	report := GenerateReport(diff)

	c := report.ChangedFiles[0]
	if c.Status != "deleted" {
		t.Fatalf("status = %q, want deleted", c.Status)
	}
	if c.Path != "old/removed.go" {
		t.Fatalf("path = %q, want old/removed.go (deleted path)", c.Path)
	}
	if c.RemovedLines != 1 {
		t.Fatalf("removed lines = %d, want 1", c.RemovedLines)
	}
}

func TestGenerateReportGroupsMultipleCategoriesInStableOrder(t *testing.T) {
	diff := diffFor("package.json", "modified", "-{}\n+{}\n") + "\n" +
		diffFor("scripts/build.sh", "added-exec", "+echo hi\n") + "\n" +
		diffFor("src/main.go", "modified", "-old\n+new\n")

	report := GenerateReport(diff)

	if len(report.ChangedFiles) != 3 {
		t.Fatalf("changed_files = %d, want 3", len(report.ChangedFiles))
	}

	wantOrder := []string{CategoryScript, CategoryDependencyManifest}
	if len(report.HighRisk) < len(wantOrder) {
		t.Fatalf("high_risk = %d, want at least %d", len(report.HighRisk), len(wantOrder))
	}
	for i, want := range wantOrder {
		if report.HighRisk[i].Category != want {
			t.Fatalf("high_risk[%d].category = %q, want %q (full: %#v)", i, report.HighRisk[i].Category, want, report.HighRisk)
		}
	}
}

func TestSummarizeListsChangedFilesAndHighRisk(t *testing.T) {
	diff := diffFor("build.sh", "added-exec", "+echo hi\n")
	report := GenerateReport(diff)

	out := report.Summarize()
	if !strings.Contains(out, "promotion report:") {
		t.Fatalf("summary missing header:\n%s", out)
	}
	if !strings.Contains(out, "build.sh") {
		t.Fatalf("summary missing changed file:\n%s", out)
	}
	if !strings.Contains(out, CategoryScript) {
		t.Fatalf("summary missing high-risk category:\n%s", out)
	}
}

func TestSummarizeEmptyReport(t *testing.T) {
	out := (&Report{}).Summarize()
	if !strings.Contains(out, "no changed files") {
		t.Fatalf("empty summary should state no changed files:\n%s", out)
	}
}

func containsCategory(cats []string, want string) bool {
	for _, c := range cats {
		if c == want {
			return true
		}
	}
	return false
}

func hasHighRisk(report *Report, category, path string) bool {
	for _, hr := range report.HighRisk {
		if hr.Category != category {
			continue
		}
		for _, p := range hr.Paths {
			if p == path {
				return true
			}
		}
	}
	return false
}
