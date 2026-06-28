// Package doctor implements the read-only diagnostic surface for isobox.
//
// A Doctor Check inspects a host or project condition without mutating any
// state. Each Check produces a Doctor Finding classified with severity
// ok, warning, or error. A doctor Report groups Findings by scope
// (global checks vs project checks) and renders them as a human-readable
// summary whose exit code is 0 unless any Finding has severity error.
//
// The package is intentionally minimal: it models the Finding shape,
// aggregates severity, and renders the report. Issue #49 is the first
// vertical slice; richer checks (project policy, bwrap, git, task store
// readiness) are added in follow-up slices.
package doctor

import (
	"fmt"
	"sort"
	"strings"
)

// Severity classifies a Doctor Finding.
type Severity int

const (
	// SeverityOK means isobox is ready for the checked condition.
	SeverityOK Severity = iota
	// SeverityWarning means isobox can still run, but some workflow may
	// be unavailable or blocked until the condition changes.
	SeverityWarning
	// SeverityError means a required readiness condition failed.
	SeverityError
)

// String returns the canonical lower-case name of the severity.
func (s Severity) String() string {
	switch s {
	case SeverityOK:
		return "ok"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// Highest returns the most severe severity in the given list. An empty
// list resolves to SeverityOK so a doctor run with no findings exits 0.
func Highest(in []Severity) Severity {
	worst := SeverityOK
	for _, s := range in {
		if s > worst {
			worst = s
		}
	}
	return worst
}

// ExitCode maps a list of severities to a process exit code: 0 unless any
// severity is SeverityError, in which case 1. Warnings never break the
// exit code so normal development workflows are not interrupted by
// advisory findings.
func ExitCode(in []Severity) int {
	if Highest(in) == SeverityError {
		return 1
	}
	return 0
}

// Scope groups Doctor Checks by where they apply. Global checks run on
// every invocation regardless of the target directory. Project checks run
// only when the target directory is inside a Git repository.
type Scope int

const (
	// ScopeGlobal is for host-wide checks such as PATH and git.
	ScopeGlobal Scope = iota
	// ScopeProject is for checks that depend on a target project, such as
	// the presence of .isobox/config.yaml.
	ScopeProject
)

// String returns the canonical lower-case name of the scope.
func (s Scope) String() string {
	switch s {
	case ScopeGlobal:
		return "global"
	case ScopeProject:
		return "project"
	default:
		return "unknown"
	}
}

// CheckScope reports the Scope for a given Check ID. Known project checks
// are project-scoped; everything else is global. The lookup is
// deliberately small and explicit so the grouping is easy to audit.
func CheckScope(id string) Scope {
	if strings.HasPrefix(id, "project-") {
		return ScopeProject
	}
	return ScopeGlobal
}

// Check is a single read-only readiness probe and its Finding.
//
// Consequence and Fix are populated for non-ok severities so a user
// reading the report can see both what is broken and what to do next.
// The strings remain free of the wrapped-command output to keep the
// report stable across runs.
type Check struct {
	// ID is the stable identifier for the Check (e.g. "version", "git").
	ID string
	// Severity is the resulting Doctor Finding severity.
	Severity Severity
	// Title is the human-readable one-line summary of the Check.
	Title string
	// Message is the optional detail line that follows Title. For ok
	// findings, it usually reports the observed value (e.g. "v0.1.1").
	Message string
	// Consequence is what changes because of the finding. Empty for ok.
	Consequence string
	// Fix is the human-actionable next step. Empty for ok.
	Fix string
}

// OK returns an ok-severity Check with the given ID, title, and observed
// value. Use OK for successful checks so the resulting Finding carries
// only what a user needs to read at a glance.
func OK(id, title, message string) Check {
	return Check{ID: id, Severity: SeverityOK, Title: title, Message: message}
}

// Warning returns a warning-severity Check with the given ID, title,
// consequence, and fix. Warning findings never break the exit code; they
// surface conditions that block a specific workflow but leave isobox
// itself usable.
func Warning(id, title, consequence, fix string) Check {
	return Check{ID: id, Severity: SeverityWarning, Title: title, Consequence: consequence, Fix: fix}
}

// Error returns an error-severity Check with the given ID, title,
// consequence, and fix. Any Error finding causes the doctor command to
// exit with status 1.
func Error(id, title, consequence, fix string) Check {
	return Check{ID: id, Severity: SeverityError, Title: title, Consequence: consequence, Fix: fix}
}

// Report is the grouped collection of Doctor Checks for a single
// `isobox doctor` invocation. It carries the version metadata, the
// target project path (if any), and the slice of Checks performed.
type Report struct {
	Version     string
	Commit      string
	ProjectPath string
	Checks      []Check
}

// NewReport builds a Report from the supplied metadata and Checks. The
// ProjectPath may be empty when `isobox doctor` is invoked without a path
// argument or with a directory that is not inside a Git repository.
func NewReport(version, commit, projectPath string, checks []Check) Report {
	return Report{Version: version, Commit: commit, ProjectPath: projectPath, Checks: checks}
}

// ExitCode returns 0 unless any of the report's Checks has severity error.
func (r Report) ExitCode() int {
	severities := make([]Severity, 0, len(r.Checks))
	for _, c := range r.Checks {
		severities = append(severities, c.Severity)
	}
	return ExitCode(severities)
}

// ScopeFor returns the scope (global or project) the given Check ID
// belongs to. This is a convenience for the renderer so the caller does
// not have to walk the Checks slice to discover the scope of an ID.
func (r Report) ScopeFor(id string) Scope {
	return CheckScope(id)
}

// Format renders the Report as a human-readable, grouped summary. The
// format is intentionally stable because integration tests assert on it.
//
// The output has four regions when a project path is set:
//
//	isobox doctor
//	  version: <version>  commit: <commit>
//	  target:  <project path>
//
//	Global checks
//	  [ok|warning|error] <title>: <message>
//	  ...
//
//	Project checks
//	  [ok|warning|error] <title>: <message>
//	  ...
//
// Non-ok findings include the consequence and fix as additional indented
// lines so the user has both what is wrong and what to do.
func (r Report) Format() string {
	var b strings.Builder

	b.WriteString("isobox doctor\n")
	if r.Version != "" {
		fmt.Fprintf(&b, "  version: %s", r.Version)
		if r.Commit != "" {
			fmt.Fprintf(&b, "  commit: %s", r.Commit)
		}
		b.WriteString("\n")
	}
	if r.ProjectPath != "" {
		fmt.Fprintf(&b, "  target:  %s\n", r.ProjectPath)
	}
	b.WriteString("\n")

	globals, projects := r.splitByScope()
	writeSection(&b, "Global checks", globals)
	if r.ProjectPath != "" {
		writeSection(&b, "Project checks", projects)
	}
	return b.String()
}

func (r Report) splitByScope() (globals []Check, projects []Check) {
	for _, c := range r.Checks {
		if r.ScopeFor(c.ID) == ScopeProject {
			projects = append(projects, c)
		} else {
			globals = append(globals, c)
		}
	}
	sortChecks(globals)
	sortChecks(projects)
	return
}

// sortChecks orders Checks by Severity (error first, then warning, then
// ok) and then by ID so the report ordering is stable across runs and
// easy to assert against in tests.
func sortChecks(in []Check) {
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].Severity != in[j].Severity {
			return in[i].Severity > in[j].Severity
		}
		return in[i].ID < in[j].ID
	})
}

func writeSection(b *strings.Builder, title string, checks []Check) {
	b.WriteString(title + "\n")
	if len(checks) == 0 {
		b.WriteString("  (none)\n\n")
		return
	}
	for _, c := range checks {
		writeCheck(b, c)
	}
	b.WriteString("\n")
}

func writeCheck(b *strings.Builder, c Check) {
	if c.Message != "" {
		fmt.Fprintf(b, "  [%s] %s: %s\n", c.Severity, c.Title, c.Message)
	} else {
		fmt.Fprintf(b, "  [%s] %s\n", c.Severity, c.Title)
	}
	if c.Severity == SeverityOK {
		return
	}
	if c.Consequence != "" {
		fmt.Fprintf(b, "         consequence: %s\n", c.Consequence)
	}
	if c.Fix != "" {
		fmt.Fprintf(b, "         fix: %s\n", c.Fix)
	}
}
