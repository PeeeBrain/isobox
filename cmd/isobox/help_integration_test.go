package main_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// helpTextFor runs `isobox <args...>` and returns the combined stdout/stderr
// text. Integration tests for the help surface exec the built binary so the
// assertions cover the externally visible CLI shape.
func helpTextFor(t *testing.T, args ...string) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "isobox")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = filepath.Join("..", "..", "cmd", "isobox")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build isobox: %v\n%s", err, out)
	}

	cmd := exec.Command(binPath, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// A usage error is expected for unknown commands; the assertion
		// should still hold on the printed text. Non-zero exits do not
		// affect the combined output capture here.
		_ = err
	}
	return stdout.String() + stderr.String()
}

// buildIsobox compiles the isobox binary into a temp directory and returns
// its path. Help integration tests share this helper so the build cost is
// paid once per test rather than once per subtest.
func buildIsobox(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "isobox")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = filepath.Join("..", "..", "cmd", "isobox")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build isobox: %v\n%s", err, out)
	}
	return binPath
}

func TestHelpTopLevelListsAllCommandsWithShortPurposes(t *testing.T) {
	binPath := buildIsobox(t)

	for _, flag := range []string{"--help", "-h", "help"} {
		t.Run(flag, func(t *testing.T) {
			out, err := exec.Command(binPath, flag).CombinedOutput()
			if err != nil {
				t.Fatalf("isobox %s exited with error: %v\n%s", flag, err, out)
			}
			text := string(out)

			// Each top-level command must be named in the help text.
			for _, want := range []string{"init", "run", "tool", "promote", "version", "doctor"} {
				if !strings.Contains(text, want) {
					t.Errorf("isobox %s does not list command %q:\n%s", flag, want, text)
				}
			}

			// The richer help must explain what isobox is, not just print a
			// terse usage line.
			if !strings.Contains(strings.ToLower(text), "isobox") {
				t.Errorf("isobox %s does not explain what isobox is:\n%s", flag, text)
			}

			// Help must use the project glossary terms the issue calls out.
			for _, term := range []string{"Task", "Workspace", "Sandbox", "Task Record", "Task Result", "Promotion", "Workload Command"} {
				if !strings.Contains(text, term) {
					t.Errorf("isobox %s does not use glossary term %q:\n%s", flag, term, text)
				}
			}
		})
	}
}

func TestHelpPerCommandPrintsUsageAndExamples(t *testing.T) {
	binPath := buildIsobox(t)

	commands := []struct {
		name      string
		mustHave  []string
		mustNotBe string
	}{
		{
			name:     "init",
			mustHave: []string{"isobox init", "Usage:", "Examples:", ".isobox"},
		},
		{
			name:     "run",
			mustHave: []string{"isobox run", "Usage:", "Examples:", "--source", "--"},
		},
		{
			name:     "tool",
			mustHave: []string{"isobox tool", "Usage:", "Examples:", "--", "bubblewrap"},
		},
		{
			name:     "promote",
			mustHave: []string{"isobox promote", "Usage:", "Examples:", "Task Record", "--yes"},
		},
		{
			name:     "version",
			mustHave: []string{"isobox version", "Usage:"},
		},
		{
			name:     "doctor",
			mustHave: []string{"isobox doctor", "Usage:", "Examples:", "Doctor Finding"},
		},
	}

	for _, c := range commands {
		t.Run(c.name, func(t *testing.T) {
			out, err := exec.Command(binPath, c.name, "--help").CombinedOutput()
			if err != nil {
				t.Fatalf("isobox %s --help exited with error: %v\n%s", c.name, err, out)
			}
			text := string(out)
			for _, want := range c.mustHave {
				if !strings.Contains(text, want) {
					t.Errorf("isobox %s --help missing %q:\n%s", c.name, want, text)
				}
			}
			// Per-command help must not be the terse single-line usage that
			// the old help system produced.
			if strings.TrimSpace(text) == "usage: isobox <init|run|tool|promote|version|doctor>" {
				t.Errorf("isobox %s --help returned only the terse usage line", c.name)
			}
		})
	}
}

func TestHelpUnknownCommandReturnsConciseActionableUsage(t *testing.T) {
	binPath := buildIsobox(t)

	out, err := exec.Command(binPath, "not-a-command").CombinedOutput()
	if err == nil {
		t.Fatalf("isobox not-a-command unexpectedly succeeded:\n%s", out)
	}
	text := string(out)

	// The error must remain concise: the user needs the next step, not a
	// full command catalog dumped into a usage failure.
	if !strings.Contains(text, "isobox --help") {
		t.Errorf("isobox <unknown> does not point the user to `isobox --help`:\n%s", text)
	}
	for _, term := range []string{"Task", "Workspace", "Sandbox", "Task Record", "Task Result", "Promotion", "Workload Command"} {
		if strings.Contains(text, term) {
			t.Errorf("isobox <unknown> leaks glossary terms into the concise usage:\n%s", text)
		}
	}
}
