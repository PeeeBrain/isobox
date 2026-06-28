package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// doctorRunResult is the result of invoking the doctor command via the
// built binary. The combined output, process error, and exit code are
// captured so tests can assert on the full externally visible shape.
type doctorRunResult struct {
	combined string
	stdout   string
	stderr   string
	err      error
}

// runDoctorFromDir builds the isobox binary, runs `isobox doctor [args]`
// from the given working directory, and returns the captured result. The
// build is paid once per invocation; the tests are simple enough that
// the per-test cost stays in the sub-second range.
func runDoctorFromDir(t *testing.T, dir string, args ...string) doctorRunResult {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "isobox")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = filepath.Join("..", "..", "cmd", "isobox")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build isobox: %v\n%s", err, out)
	}

	cmdArgs := append([]string{"doctor"}, args...)
	cmd := exec.Command(binPath, cmdArgs...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return doctorRunResult{combined: stdout.String() + stderr.String(), stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func TestDoctorCommandIsAcceptedByCLI(t *testing.T) {
	result := runDoctorFromDir(t, t.TempDir())
	if result.err != nil {
		t.Fatalf("isobox doctor unexpectedly failed: %v\n%s", result.err, result.combined)
	}
}

func TestDoctorReportsVersionMetadataAsOK(t *testing.T) {
	result := runDoctorFromDir(t, t.TempDir())

	if !strings.Contains(result.combined, "isobox doctor") {
		t.Errorf("isobox doctor output missing header:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "version:") {
		t.Errorf("isobox doctor does not report version metadata:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "[ok]") {
		t.Errorf("isobox doctor does not report the version check as ok:\n%s", result.combined)
	}
}

func TestDoctorExitsZeroOnOnlyOKFindings(t *testing.T) {
	result := runDoctorFromDir(t, t.TempDir())
	if result.err != nil {
		t.Fatalf("isobox doctor exited with error on ok findings: %v\n%s", result.err, result.combined)
	}
}

func TestDoctorRejectsMoreThanOnePathArgument(t *testing.T) {
	result := runDoctorFromDir(t, t.TempDir(), t.TempDir(), t.TempDir())
	if result.err == nil {
		t.Fatalf("isobox doctor with two paths unexpectedly succeeded:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "usage:") {
		t.Errorf("isobox doctor with two paths does not return a usage error:\n%s", result.combined)
	}
}

func TestDoctorRejectsMissingPathArgument(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	result := runDoctorFromDir(t, t.TempDir(), missing)
	if result.err == nil {
		t.Fatalf("isobox doctor with missing path unexpectedly succeeded:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, missing) {
		t.Errorf("isobox doctor with missing path does not mention the failing path:\n%s", result.combined)
	}
}

func TestDoctorRejectsNonDirectoryPathArgument(t *testing.T) {
	tmp := t.TempDir()
	notDir := filepath.Join(tmp, "regular-file")
	if err := os.WriteFile(notDir, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := runDoctorFromDir(t, t.TempDir(), notDir)
	if result.err == nil {
		t.Fatalf("isobox doctor with non-directory path unexpectedly succeeded:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "not a directory") {
		t.Errorf("isobox doctor with non-directory path does not explain the failure:\n%s", result.combined)
	}
}

func TestDoctorAcceptsOneExistingDirectoryPath(t *testing.T) {
	target := initGitRepo(t)
	result := runDoctorFromDir(t, t.TempDir(), target)
	if result.err != nil {
		t.Fatalf("isobox doctor with one valid directory path failed: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "Project checks") {
		t.Errorf("isobox doctor with a Git project does not show the project section:\n%s", result.combined)
	}
}

func TestDoctorOutputIsReadOnly(t *testing.T) {
	target := initGitRepo(t)
	beforeEntries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	beforeGitignore, gitignoreErr := os.ReadFile(filepath.Join(target, ".gitignore"))
	beforeIsbox, _ := os.Stat(filepath.Join(target, ".isobox"))

	result := runDoctorFromDir(t, t.TempDir(), target)
	if result.err != nil {
		t.Fatalf("isobox doctor failed: %v\n%s", result.err, result.combined)
	}

	afterEntries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeEntries) != len(afterEntries) {
		t.Errorf("isobox doctor created or removed top-level entries; before=%d after=%d", len(beforeEntries), len(afterEntries))
	}

	if gitignoreErr == nil {
		afterGitignore, err := os.ReadFile(filepath.Join(target, ".gitignore"))
		if err != nil {
			t.Fatal(err)
		}
		if string(beforeGitignore) != string(afterGitignore) {
			t.Errorf("isobox doctor modified .gitignore")
		}
	}

	afterIsbox, afterIsboxErr := os.Stat(filepath.Join(target, ".isobox"))
	if beforeIsbox == nil && afterIsboxErr == nil {
		t.Errorf("isobox doctor created a .isobox directory in the target project")
	}
	if beforeIsbox != nil && afterIsboxErr == nil && beforeIsbox.IsDir() != afterIsbox.IsDir() {
		t.Errorf("isobox doctor changed .isobox type")
	}
}

func TestDoctorExitCodeMapsFindingsToStatus(t *testing.T) {
	// We cannot directly trigger an Error-severity finding from the
	// environment without instrumenting the doctor, so this test exercises
	// the exit-code wiring by running doctor in a directory where global
	// checks should all be ok and verifying the exit code is 0. The
	// package-level TestFindingsExitCodeReturnsOneOnlyOnError covers the
	// aggregation logic for error findings; the CLI test covers the
	// shell-out path.
	result := runDoctorFromDir(t, t.TempDir())
	if result.err != nil {
		t.Fatalf("isobox doctor with ok findings exited non-zero: %v\n%s", result.err, result.combined)
	}
}

func TestDoctorHelpIsAvailableFromCLIDispatch(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "isobox")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = filepath.Join("..", "..", "cmd", "isobox")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build isobox: %v\n%s", err, out)
	}
	out, err := exec.Command(binPath, "doctor", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("isobox doctor --help failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "Usage:") {
		t.Errorf("isobox doctor --help does not include a Usage section:\n%s", text)
	}
	if !strings.Contains(text, "Doctor Finding") {
		t.Errorf("isobox doctor --help does not use the glossary term Doctor Finding:\n%s", text)
	}
}
