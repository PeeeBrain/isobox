package doctor_test

import (
	"strings"
	"testing"

	"isobox/internal/doctor"
)

func TestSeverityString(t *testing.T) {
	cases := []struct {
		sev  doctor.Severity
		want string
	}{
		{doctor.SeverityOK, "ok"},
		{doctor.SeverityWarning, "warning"},
		{doctor.SeverityError, "error"},
	}
	for _, c := range cases {
		if got := c.sev.String(); got != c.want {
			t.Errorf("Severity(%d).String() = %q, want %q", c.sev, got, c.want)
		}
	}
}

func TestSeverityHighestReturnsErrorOverWarningOverOK(t *testing.T) {
	cases := []struct {
		name string
		in   []doctor.Severity
		want doctor.Severity
	}{
		{"no findings", nil, doctor.SeverityOK},
		{"only ok", []doctor.Severity{doctor.SeverityOK}, doctor.SeverityOK},
		{"warning only", []doctor.Severity{doctor.SeverityWarning}, doctor.SeverityWarning},
		{"error only", []doctor.Severity{doctor.SeverityError}, doctor.SeverityError},
		{"ok plus warning", []doctor.Severity{doctor.SeverityOK, doctor.SeverityWarning}, doctor.SeverityWarning},
		{"ok plus error", []doctor.Severity{doctor.SeverityOK, doctor.SeverityError}, doctor.SeverityError},
		{"warning plus error", []doctor.Severity{doctor.SeverityWarning, doctor.SeverityError}, doctor.SeverityError},
		{"all three", []doctor.Severity{doctor.SeverityOK, doctor.SeverityWarning, doctor.SeverityError}, doctor.SeverityError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := doctor.Highest(c.in); got != c.want {
				t.Errorf("Highest(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestFindingsExitCodeReturnsOneOnlyOnError(t *testing.T) {
	cases := []struct {
		name     string
		in       []doctor.Severity
		wantCode int
	}{
		{"empty findings", nil, 0},
		{"only ok", []doctor.Severity{doctor.SeverityOK}, 0},
		{"only warning", []doctor.Severity{doctor.SeverityWarning}, 0},
		{"ok plus warning", []doctor.Severity{doctor.SeverityOK, doctor.SeverityWarning}, 0},
		{"any error", []doctor.Severity{doctor.SeverityOK, doctor.SeverityError}, 1},
		{"error only", []doctor.Severity{doctor.SeverityError}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := doctor.ExitCode(c.in); got != c.wantCode {
				t.Errorf("ExitCode(%v) = %d, want %d", c.in, got, c.wantCode)
			}
		})
	}
}

func TestCheckOKProducesOKFinding(t *testing.T) {
	check := doctor.OK("version", "isobox version", "v0.1.1")

	if check.ID != "version" {
		t.Fatalf("Check.ID = %q, want version", check.ID)
	}
	if check.Severity != doctor.SeverityOK {
		t.Fatalf("Check severity = %q, want ok", check.Severity)
	}
	if check.Message != "v0.1.1" {
		t.Fatalf("Check message = %q, want v0.1.1", check.Message)
	}
}

func TestCheckWarningProducesWarningFinding(t *testing.T) {
	check := doctor.Warning("bubblewrap", "bubblewrap (bwrap) is not on PATH", "Tool-Call Sandboxes are unavailable", "install bubblewrap to enable Tool-Call Sandboxes")

	if check.Severity != doctor.SeverityWarning {
		t.Fatalf("Check severity = %q, want warning", check.Severity)
	}
	if check.Consequence == "" || check.Fix == "" {
		t.Fatalf("warning check missing consequence/fix: %+v", check)
	}
}

func TestCheckErrorProducesErrorFinding(t *testing.T) {
	check := doctor.Error("git", "git is not on PATH", "no isobox workflow can run", "install git before using isobox")

	if check.Severity != doctor.SeverityError {
		t.Fatalf("Check severity = %q, want error", check.Severity)
	}
	if check.Consequence == "" || check.Fix == "" {
		t.Fatalf("error check missing consequence/fix: %+v", check)
	}
}

func TestReportGroupsFindingsByScope(t *testing.T) {
	checks := []doctor.Check{
		doctor.OK("version", "isobox version", "v0.1.1"),
		doctor.OK("path", "isobox on PATH", "/usr/local/bin/isobox"),
		doctor.Warning("bubblewrap", "bubblewrap missing", "Tool-Call Sandboxes are unavailable", "install bubblewrap"),
		doctor.OK("project-policy", "project policy present", "/p/.isobox/config.yaml"),
	}
	report := doctor.NewReport("version", "abc123", "/p", checks)

	if report.ScopeFor("version") != doctor.ScopeGlobal {
		t.Errorf("version check scope = %q, want global", report.ScopeFor("version"))
	}
	if report.ScopeFor("path") != doctor.ScopeGlobal {
		t.Errorf("path check scope = %q, want global", report.ScopeFor("path"))
	}
	if report.ScopeFor("bubblewrap") != doctor.ScopeGlobal {
		t.Errorf("bubblewrap check scope = %q, want global", report.ScopeFor("bubblewrap"))
	}
	if report.ScopeFor("project-policy") != doctor.ScopeProject {
		t.Errorf("project-policy check scope = %q, want project", report.ScopeFor("project-policy"))
	}
}

func TestReportExitCodeReflectsHighestFinding(t *testing.T) {
	warning := doctor.NewReport("v", "c", "", []doctor.Check{doctor.Warning("bubblewrap", "missing", "no Tool-Call Sandbox", "install bubblewrap")})
	if got := warning.ExitCode(); got != 0 {
		t.Errorf("warning-only exit code = %d, want 0", got)
	}

	errored := doctor.NewReport("v", "c", "", []doctor.Check{doctor.Error("git", "missing", "no isobox workflow can run", "install git")})
	if got := errored.ExitCode(); got != 1 {
		t.Errorf("error exit code = %d, want 1", got)
	}
}

func TestReportFormatRendersGlobalAndProjectSections(t *testing.T) {
	checks := []doctor.Check{
		doctor.OK("version", "isobox version", "v0.1.1"),
		doctor.Warning("bubblewrap", "bubblewrap missing", "Tool-Call Sandboxes are unavailable", "install bubblewrap to enable Tool-Call Sandboxes"),
		doctor.OK("project-policy", "project policy present", "/p/.isobox/config.yaml"),
	}
	report := doctor.NewReport("v0.1.1", "abc1234", "/p", checks)

	text := report.Format()

	for _, want := range []string{
		"v0.1.1",
		"abc1234",
		"isobox doctor",
		"Global checks",
		"Project checks",
		"isobox version",
		"v0.1.1",
		"bubblewrap missing",
		"project policy present",
		"ok",
		"warning",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("report output missing %q:\n%s", want, text)
		}
	}

	// The format must not invent a project path when none was supplied.
	noProject := doctor.NewReport("v0.1.1", "abc1234", "", checks).Format()
	if strings.Contains(noProject, "Project checks") {
		t.Errorf("report without a project path should not include a project section:\n%s", noProject)
	}
}

func TestReportFormatRendersConsequenceAndFixForNonOKFindings(t *testing.T) {
	checks := []doctor.Check{
		doctor.Warning("bubblewrap", "bubblewrap missing", "Tool-Call Sandboxes are unavailable", "install bubblewrap"),
		doctor.Error("git", "git missing", "no isobox workflow can run", "install git"),
	}
	report := doctor.NewReport("v", "c", "", checks)
	text := report.Format()

	for _, want := range []string{
		"Tool-Call Sandboxes are unavailable",
		"install bubblewrap",
		"no isobox workflow can run",
		"install git",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("report output missing consequence/fix %q:\n%s", want, text)
		}
	}
}
