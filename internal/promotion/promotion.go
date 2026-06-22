// Package promotion generates the initial Promotion Report for a Task Result.
//
// A Promotion Report summarizes the changed files in a Repository Workspace
// Task Result and flags high-risk categories that deserve extra review before
// explicit Promotion. The report is informational: it never gates or auto-applies
// Promotion. The user remains the review gate.
//
// The report is generated from the captured Git diff. File size and executability
// are inferred from what the diff exposes (new-file mode, changed-line counts,
// binary markers), so detection is best-effort and conservative: a category is
// flagged only where the diff makes it detectable.
package promotion

import (
	"path/filepath"
	"strings"
)

// ReportSchemaVersion is the schema version of the Promotion Report captured in
// the Task Record.
const ReportSchemaVersion = "v1"

// Category names for high-risk changed files. They are stable string constants
// recorded in the Task Record.
const (
	CategoryScript             = "script"
	CategoryHook               = "hook"
	CategoryDependencyManifest = "dependency_manifest"
	CategoryCIWorkflow         = "ci_workflow"
	CategoryLargeFile          = "large_file"
	CategoryBinary             = "binary"
)

// largeFileChangedLineThreshold is the heuristic threshold above which a single
// file's change is treated as a large-file review burden. It is a proxy for file
// size, since the diff only exposes changed lines, not total file size.
const largeFileChangedLineThreshold = 500

// Report is the structured Promotion Report for a Task Result.
//
// ChangedFiles lists every file the diff touches, in diff order, with its
// change status and any high-risk categories that apply to it. HighRisk groups
// the same categories by name with the affected paths, so reviewers can see at
// a glance which categories deserve extra attention before Promotion.
type Report struct {
	SchemaVersion string       `json:"schema_version"`
	ChangedFiles  []FileChange `json:"changed_files"`
	HighRisk      []HighRisk   `json:"high_risk"`
}

// FileChange describes one changed file in a Task Result diff.
type FileChange struct {
	// Path is the repository-relative path of the changed file. For deletions
	// it is the path that was removed.
	Path         string   `json:"path"`
	Status       string   `json:"status"`
	Categories   []string `json:"categories,omitempty"`
	AddedLines   int      `json:"added_lines"`
	RemovedLines int      `json:"removed_lines"`

	// executable records that the diff exposes an executable file mode. It is
	// not serialized; it only feeds high-risk categorization.
	executable bool `json:"-"`
}

// HighRisk groups the paths that fall under one high-risk category.
type HighRisk struct {
	Category string   `json:"category"`
	Paths    []string `json:"paths"`
}

// GenerateReport parses a unified Git diff and produces the initial Promotion
// Report for the Task Result. An empty diff produces an empty report with the
// current schema version.
func GenerateReport(diff string) *Report {
	report := &Report{SchemaVersion: ReportSchemaVersion}
	if strings.TrimSpace(diff) == "" {
		return report
	}

	sections := splitDiffSections(diff)
	for _, section := range sections {
		change := parseFileChange(section)
		if change.Path == "" {
			continue
		}
		report.ChangedFiles = append(report.ChangedFiles, change)
	}

	report.HighRisk = groupHighRisk(report.ChangedFiles)
	return report
}

// Summarize renders a short human-readable summary of the report for the
// Promotion command. It is intended to focus review, not to replace the
// structured record.
func (r *Report) Summarize() string {
	var b strings.Builder
	if r == nil || len(r.ChangedFiles) == 0 {
		b.WriteString("promotion report: no changed files\n")
		return b.String()
	}
	b.WriteString("promotion report:\n")
	b.WriteString("  changed files:\n")
	for _, c := range r.ChangedFiles {
		line := "    " + c.Status + " " + c.Path
		if len(c.Categories) > 0 {
			line += " [" + strings.Join(c.Categories, ", ") + "]"
		}
		b.WriteString(line + "\n")
	}
	if len(r.HighRisk) == 0 {
		b.WriteString("  high-risk: none\n")
	} else {
		b.WriteString("  high-risk:\n")
		for _, hr := range r.HighRisk {
			b.WriteString("    " + hr.Category + ": " + strings.Join(hr.Paths, ", ") + "\n")
		}
	}
	return b.String()
}

// splitDiffSections splits a unified Git diff into per-file sections. Each
// section begins at a "diff --git " line and includes all lines up to the next
// "diff --git " line or the end of the diff.
func splitDiffSections(diff string) []string {
	var sections []string
	var current []string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			if len(current) > 0 {
				sections = append(sections, strings.Join(current, "\n"))
			}
			current = []string{line}
			continue
		}
		if len(current) > 0 {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		sections = append(sections, strings.Join(current, "\n"))
	}
	return sections
}

// parseFileChange parses a single per-file diff section into a FileChange.
func parseFileChange(section string) FileChange {
	change := FileChange{Status: "modified"}

	inHunk := false
	for _, line := range strings.Split(section, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			ap, bp := parseDiffGitPaths(line)
			change.Path = pickPath(ap, bp)
		case strings.HasPrefix(line, "new file mode 100755"):
			change.Status = "added"
			change.executable = true
		case strings.HasPrefix(line, "new file mode "):
			change.Status = "added"
		case strings.HasPrefix(line, "deleted file mode "):
			change.Status = "deleted"
		case strings.HasPrefix(line, "new mode 100755"):
			change.executable = true
		case strings.HasPrefix(line, "Binary files ") || strings.HasPrefix(line, "GIT binary patch"):
			change.Categories = appendUnique(change.Categories, CategoryBinary)
		case strings.HasPrefix(line, "@@"):
			inHunk = true
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			// Header lines are not content; do not count.
		case inHunk && strings.HasPrefix(line, "+"):
			change.AddedLines++
		case inHunk && strings.HasPrefix(line, "-"):
			change.RemovedLines++
		}
	}

	change.Categories = appendUnique(change.Categories, categorize(change)...)
	return change
}

// parseDiffGitPaths extracts the a/ and b/ paths from a "diff --git a/X b/Y"
// header line.
func parseDiffGitPaths(line string) (aPath, bPath string) {
	rest := strings.TrimPrefix(line, "diff --git ")
	if strings.HasPrefix(rest, "a/") {
		rest = rest[len("a/"):]
	}
	idx := strings.LastIndex(rest, " b/")
	if idx < 0 {
		return "", ""
	}
	return rest[:idx], rest[idx+len(" b/"):]
}

// pickPath chooses the representative path for a change. For deletions the b
// side is /dev/null, so the a/ path is the removed file. Otherwise the b/ path
// is the file as it exists in the Task Result.
func pickPath(aPath, bPath string) string {
	if bPath != "" && bPath != "/dev/null" {
		return bPath
	}
	if aPath != "" && aPath != "/dev/null" {
		return aPath
	}
	return ""
}

// categorize returns the high-risk categories that apply to a changed file
// based on its path, status, and change size.
func categorize(c FileChange) []string {
	var cats []string
	base := filepath.Base(c.Path)

	if isBinary(c) {
		cats = append(cats, CategoryBinary)
	}
	if isScript(c, base) {
		cats = append(cats, CategoryScript)
	}
	if isHook(c.Path, base) {
		cats = append(cats, CategoryHook)
	}
	if isDependencyManifest(base) {
		cats = append(cats, CategoryDependencyManifest)
	}
	if isCIWorkflow(c.Path, base) {
		cats = append(cats, CategoryCIWorkflow)
	}
	if isLargeFile(c) {
		cats = append(cats, CategoryLargeFile)
	}
	return cats
}

func isBinary(c FileChange) bool {
	for _, cat := range c.Categories {
		if cat == CategoryBinary {
			return true
		}
	}
	return false
}

func isScript(c FileChange, base string) bool {
	switch filepath.Ext(base) {
	case ".sh", ".bash", ".zsh", ".ps1", ".bat", ".cmd":
		return true
	}
	if strings.HasPrefix(c.Path, "scripts/") {
		return true
	}
	// A newly added executable file is a script-like review risk.
	if c.executable {
		return true
	}
	return false
}

func isHook(path, base string) bool {
	for _, prefix := range []string{".git/hooks/", ".githooks/", ".husky/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	switch base {
	case "pre-commit", "pre-push", "pre-rebase", "pre-applypatch", "pre-merge",
		"post-merge", "post-commit", "commit-msg", "prepare-commit-msg":
		return true
	}
	return false
}

func isDependencyManifest(base string) bool {
	return dependencyManifests[base]
}

func isCIWorkflow(path, base string) bool {
	for _, prefix := range []string{".github/workflows/", ".gitlab/", ".circleci/", ".buildkite/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	switch base {
	case ".gitlab-ci.yml", ".travis.yml", "azure-pipelines.yml",
		"Jenkinsfile", "bitbucket-pipelines.yml":
		return true
	}
	return false
}

func isLargeFile(c FileChange) bool {
	return c.AddedLines+c.RemovedLines > largeFileChangedLineThreshold
}

// groupHighRisk builds the HighRisk list from the per-file categories, keeping
// a stable category order and diff order within each category.
func groupHighRisk(changes []FileChange) []HighRisk {
	order := []string{
		CategoryScript,
		CategoryHook,
		CategoryDependencyManifest,
		CategoryCIWorkflow,
		CategoryLargeFile,
		CategoryBinary,
	}
	pathsByCategory := map[string][]string{}
	for _, c := range changes {
		for _, cat := range c.Categories {
			pathsByCategory[cat] = append(pathsByCategory[cat], c.Path)
		}
	}
	var grouped []HighRisk
	for _, cat := range order {
		if paths, ok := pathsByCategory[cat]; ok {
			grouped = append(grouped, HighRisk{Category: cat, Paths: paths})
		}
	}
	return grouped
}

func appendUnique(existing []string, values ...string) []string {
	for _, v := range values {
		dup := false
		for _, e := range existing {
			if e == v {
				dup = true
				break
			}
		}
		if !dup {
			existing = append(existing, v)
		}
	}
	return existing
}

var dependencyManifests = map[string]bool{
	"package.json":             true,
	"package-lock.json":        true,
	"yarn.lock":                true,
	"pnpm-lock.yaml":           true,
	"npm-shrinkwrap.json":      true,
	"go.mod":                   true,
	"go.sum":                   true,
	"requirements.txt":         true,
	"Pipfile":                  true,
	"Pipfile.lock":             true,
	"pyproject.toml":           true,
	"poetry.lock":              true,
	"uv.lock":                  true,
	"Cargo.toml":               true,
	"Cargo.lock":               true,
	"Gemfile":                  true,
	"Gemfile.lock":             true,
	"composer.json":            true,
	"composer.lock":            true,
	"pom.xml":                  true,
	"build.gradle":             true,
	"build.gradle.kts":         true,
	"settings.gradle":          true,
	"packages.config":          true,
	"Directory.Packages.props": true,
}
