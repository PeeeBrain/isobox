package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runDoctorFromDirWithEnv behaves like runDoctorFromDir but lets the
// caller add KEY=VALUE entries to the child process environment. The
// global doctor checks consult the process PATH to resolve git, bwrap,
// and isobox; tests use this helper to simulate a PATH that does not
// match the host.
func runDoctorFromDirWithEnv(t *testing.T, dir string, extraEnv []string, args ...string) doctorRunResult {
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
	cmd.Env = append(os.Environ(), extraEnv...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return doctorRunResult{combined: stdout.String() + stderr.String(), stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func TestDoctorReportsGitOnPathAsOK(t *testing.T) {
	result := runDoctorFromDir(t, t.TempDir())
	if result.err != nil {
		t.Fatalf("isobox doctor failed: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "[ok] git is on PATH") {
		t.Errorf("isobox doctor does not report git-on-path as ok:\n%s", result.combined)
	}
}

func TestDoctorReportsGitMissingAsErrorWithConsequenceAndFix(t *testing.T) {
	// Empty isolated directory means no binary is on PATH, including
	// git. The doctor command itself does not need git when no path
	// argument is given, so this is a safe way to exercise the
	// missing-git error path.
	isolated := buildIsolatedPath(t)

	result := runDoctorFromDirWithEnv(t, t.TempDir(), []string{"PATH=" + isolated})
	if result.err == nil {
		t.Fatalf("isobox doctor unexpectedly succeeded without git on PATH:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "[error] git is not on PATH") {
		t.Errorf("isobox doctor does not report missing git as error:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "consequence:") {
		t.Errorf("isobox doctor does not include consequence text for missing git:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "fix:") {
		t.Errorf("isobox doctor does not include fix text for missing git:\n%s", result.combined)
	}
}

func TestDoctorReportsBwrapOnPathAsOK(t *testing.T) {
	// The bwrap-on-path check uses exec.LookPath, which only
	// requires an executable file by that name. The doctor does not
	// invoke bwrap. The test therefore builds a fake bwrap in a
	// temp directory and prepends that directory to PATH, so the
	// test does not depend on bubblewrap being installed on the
	// host.
	binPath := buildIsobox(t)
	binDir := filepath.Dir(binPath)
	fakeBwrap := writeFakeExecutable(t, "bwrap")
	path := filepath.Dir(fakeBwrap) + string(os.PathListSeparator) + binDir + string(os.PathListSeparator) + os.Getenv("PATH")

	result := runDoctorFromDirWithEnv(t, t.TempDir(), []string{"PATH=" + path})
	if result.err != nil {
		t.Fatalf("isobox doctor failed: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "[ok] bubblewrap (bwrap) is on PATH") {
		t.Errorf("isobox doctor does not report bwrap-on-path as ok:\n%s", result.combined)
	}
}

func TestDoctorReportsBwrapMissingAsWarningNamingToolCallSandbox(t *testing.T) {
	// The test builds a fully controlled PATH that contains only
	// git and the test's own isobox binary; bwrap is intentionally
	// absent. The host PATH is not prepended, so a real bwrap
	// installed on the host cannot satisfy the check. The git
	// binary is copied from the host because it is a system
	// dependency isobox itself relies on, and rewriting it as a fake
	// would be over-scoping the test.
	binPath := buildIsobox(t)
	binDir := filepath.Dir(binPath)
	hostGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("test setup requires git on host PATH: %v", err)
	}
	gitLink := filepath.Join(binDir, "git")
	if err := os.Symlink(hostGit, gitLink); err != nil {
		data, err := os.ReadFile(hostGit)
		if err != nil {
			t.Fatalf("read git binary: %v", err)
		}
		if err := os.WriteFile(gitLink, data, 0o755); err != nil {
			t.Fatalf("write git binary: %v", err)
		}
	}
	// PATH contains only the controlled directory. The host PATH is
	// deliberately excluded so a real bwrap installed on the host
	// cannot satisfy the check.
	result := runDoctorFromDirWithEnv(t, t.TempDir(), []string{"PATH=" + binDir})
	if result.err != nil {
		t.Fatalf("isobox doctor should exit 0 with only warning findings: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "[warning] bubblewrap (bwrap) is not on PATH") {
		t.Errorf("isobox doctor does not report missing bwrap as warning:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "Tool-Call") {
		t.Errorf("isobox doctor does not name Tool-Call Sandboxes in bwrap warning:\n%s", result.combined)
	}
}

func TestDoctorReportsIsoboxOnPathAsOK(t *testing.T) {
	// Build a PATH that contains git, bwrap, and the freshly built
	// isobox binary so all three on-path checks resolve.
	binPath := buildIsobox(t)
	binDir := filepath.Dir(binPath)
	extended, cleanup := extendPath(t, binDir)
	defer cleanup()

	result := runDoctorFromDirWithEnv(t, t.TempDir(), []string{"PATH=" + extended})
	if result.err != nil {
		t.Fatalf("isobox doctor failed: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "[ok] isobox is on PATH") {
		t.Errorf("isobox doctor does not report isobox-on-path as ok:\n%s", result.combined)
	}
}

func TestDoctorReportsIsoboxMissingAsWarning(t *testing.T) {
	// A PATH that contains git and a fake bwrap but no isobox will
	// leave the isobox check unable to find the binary by name. The
	// fake bwrap is built in a temp dir so the test does not depend
	// on bubblewrap being installed on the host.
	fakeBwrap := writeFakeExecutable(t, "bwrap")
	isolated := buildIsolatedPath(t, "git")
	path := isolated + string(os.PathListSeparator) + filepath.Dir(fakeBwrap)

	result := runDoctorFromDirWithEnv(t, t.TempDir(), []string{"PATH=" + path})
	if result.err != nil {
		t.Fatalf("isobox doctor should exit 0 with only warning findings: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "[warning] isobox is not on PATH") {
		t.Errorf("isobox doctor does not report missing isobox as warning:\n%s", result.combined)
	}
}

func TestDoctorReportsDuplicateIsoboxAsWarning(t *testing.T) {
	// Build isobox into two separate temp directories and put both
	// directories on PATH so the duplicates check sees two entries.
	// The PATH also needs git and bwrap so the only warning is the
	// duplicates finding; otherwise the test would race against the
	// missing-git error finding.
	binPath := buildIsobox(t)
	binDir := filepath.Dir(binPath)

	other := t.TempDir()
	otherBin := filepath.Join(other, "isobox")
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read isobox binary: %v", err)
	}
	if err := os.WriteFile(otherBin, data, 0o755); err != nil {
		t.Fatalf("write second isobox: %v", err)
	}

	// Build a PATH that contains both isobox directories, plus a
	// fake bwrap (the doctor only checks for presence) so the
	// missing-bwrap warning does not race with the duplicates
	// finding. git is supplied by prepending the host PATH last so
	// the missing-git error does not fire.
	fakeBwrap := writeFakeExecutable(t, "bwrap")

	path := binDir + string(os.PathListSeparator) + other + string(os.PathListSeparator) + filepath.Dir(fakeBwrap) + string(os.PathListSeparator) + os.Getenv("PATH")
	result := runDoctorFromDirWithEnv(t, t.TempDir(), []string{"PATH=" + path})
	if result.err != nil {
		t.Fatalf("isobox doctor should exit 0 with only warning findings: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "[warning] multiple isobox binaries on PATH") {
		t.Errorf("isobox doctor does not report duplicate isobox as warning:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "consequence:") || !strings.Contains(result.combined, otherBin) {
		t.Errorf("isobox doctor does not list the duplicate binary in the consequence text:\n%s", result.combined)
	}
}

func TestDoctorReportsVersionMetadataAsOKForDevBuild(t *testing.T) {
	// The doctor binary built by the test harness uses the default
	// "dev" version; a `dev` version must be reported as ok, not as a
	// warning or update-eligibility error. The test puts a fake
	// bwrap on PATH so the missing-bwrap warning does not drown out
	// the version finding.
	binPath := buildIsobox(t)
	binDir := filepath.Dir(binPath)
	fakeBwrap := writeFakeExecutable(t, "bwrap")
	path := filepath.Dir(fakeBwrap) + string(os.PathListSeparator) + binDir + string(os.PathListSeparator) + os.Getenv("PATH")

	result := runDoctorFromDirWithEnv(t, t.TempDir(), []string{"PATH=" + path})
	if result.err != nil {
		t.Fatalf("isobox doctor failed: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "[ok] isobox version") {
		t.Errorf("isobox doctor does not report version as ok for a dev build:\n%s", result.combined)
	}
	if strings.Contains(result.combined, "update") && strings.Contains(result.combined, "warning") {
		t.Errorf("isobox doctor must not imply update eligibility from a dev version:\n%s", result.combined)
	}
}

func TestDoctorDoesNotCallTheNetwork(t *testing.T) {
	// Structural assertion: the global checks must never hit the
	// network. The fastest observable guarantee is that the doctor
	// command runs to completion with PATH manipulated to remove
	// every host tool except git, which is the only one isobox cannot
	// do without.
	isolated := buildIsolatedPath(t, "git")
	result := runDoctorFromDirWithEnv(t, t.TempDir(), []string{"PATH=" + isolated})
	if result.err != nil {
		t.Fatalf("isobox doctor should run with git only on PATH: %v\n%s", result.err, result.combined)
	}
}

// buildIsolatedPath creates a temp directory containing only the
// requested binary names (looked up from the host) and returns the
// directory as a single-entry PATH. The returned PATH has no other
// directories, so any binary not named in `keep` will not be
// resolvable.
func buildIsolatedPath(t *testing.T, keep ...string) string {
	t.Helper()

	isolated := t.TempDir()
	for _, name := range keep {
		src, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("test setup requires %s on host PATH: %v", name, err)
		}
		dst := filepath.Join(isolated, name)
		if err := os.Symlink(src, dst); err != nil {
			data, err := os.ReadFile(src)
			if err != nil {
				t.Fatalf("read %s binary: %v", name, err)
			}
			if err := os.WriteFile(dst, data, 0o755); err != nil {
				t.Fatalf("write %s binary: %v", name, err)
			}
		}
	}
	return isolated
}

// writeFakeExecutable creates an empty file with the given name in a
// temp directory and returns the file's absolute path. The file has
// the execute bit set so exec.LookPath will resolve it on PATH. The
// doctor global checks call LookPath only to test for presence; the
// fake binary is never executed.
func writeFakeExecutable(t *testing.T, name string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake %s: %v", name, err)
	}
	return path
}

// extendPath returns a PATH that includes the host PATH plus an extra
// directory. The cleanup function restores nothing — the test will
// discard the temp directory on completion.
func extendPath(t *testing.T, extra string) (string, func()) {
	t.Helper()
	hostPath := os.Getenv("PATH")
	combined := extra + string(os.PathListSeparator) + hostPath
	return combined, func() {}
}
