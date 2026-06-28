package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// updateRunResult is the result of invoking the update command via
// the built binary. The combined output, process error, and exit
// code are captured so tests can assert on the full externally
// visible shape.
type updateRunResult struct {
	combined string
	stdout   string
	stderr   string
	err      error
}

// runUpdateWithBinaryEnv runs the supplied isobox binary with `update
// --check`, setting ISOBOX_UPDATE_CLIENT to the test-only client and
// appending the supplied extra environment entries. The helper
// exists so the integration tests can pin the running binary's
// version (via the -ldflags override) and verify that the update
// check reports the correct current version.
func runUpdateWithBinaryEnv(t *testing.T, dir string, binPath string, client string, extraEnv []string) updateRunResult {
	t.Helper()

	cmd := exec.Command(binPath, "update", "--check")
	cmd.Dir = dir
	env := append(os.Environ(), "ISOBOX_UPDATE_CLIENT="+client)
	env = append(env, extraEnv...)
	cmd.Env = env
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return updateRunResult{combined: stdout.String() + stderr.String(), stdout: stdout.String(), stderr: stderr.String(), err: err}
}

// fakeClientRelease is the wire format the test-only update client
// emits. The struct mirrors the release metadata the update package
// consumes, so the production JSON parser can decode it unchanged.
type fakeClientRelease struct {
	TagName     string    `json:"tag_name"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
}

// writeFakeUpdateClient writes a tiny Go program that returns the
// supplied releases when invoked as `client list`. The helper
// compiles the program and returns the absolute path to the
// compiled binary so it can be set as ISOBOX_UPDATE_CLIENT for the
// integration test.
func writeFakeUpdateClient(t *testing.T, releases []fakeClientRelease) string {
	t.Helper()

	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	releasesJSON, err := json.Marshal(releases)
	if err != nil {
		t.Fatalf("marshal fake releases: %v", err)
	}

	// Build the source via fmt.Sprintf with %s only for the JSON
	// payload so the rest of the program reads as plain Go. The
	// struct tags use Go's raw string delimiter (backtick) which is
	// safe to embed in an interpreted string.
	prog := fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type release struct {
	TagName     string `+"`json:\"tag_name\"`"+`
	Prerelease  bool   `+"`json:\"prerelease\"`"+`
	Draft       bool   `+"`json:\"draft\"`"+`
}

var data = %q

func main() {
	if len(os.Args) < 2 || os.Args[1] != "list" {
		fmt.Fprintln(os.Stderr, "fake client: expected list subcommand")
		os.Exit(2)
	}
	var rs []release
	if err := json.Unmarshal([]byte(data), &rs); err != nil {
		fmt.Fprintf(os.Stderr, "fake client: parse releases: %%v\n", err)
		os.Exit(2)
	}
	out, _ := json.Marshal(rs)
	fmt.Print(string(out))
}
`, string(releasesJSON))
	if err := os.WriteFile(src, []byte(prog), 0o644); err != nil {
		t.Fatalf("write fake client: %v", err)
	}
	bin := filepath.Join(dir, "fake-client")
	build := exec.Command("go", "build", "-o", bin, src)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake client: %v\n%s", err, out)
	}
	return bin
}

func TestUpdateCheckReportsUpToDateWhenAlreadyLatest(t *testing.T) {
	client := writeFakeUpdateClient(t, []fakeClientRelease{
		{TagName: "v0.1.1"},
		{TagName: "v0.1.0"},
	})
	binPath := buildIsoboxWithVersion(t, "v0.1.1")
	binDir := filepath.Dir(binPath)

	result := runUpdateWithBinaryEnv(t, t.TempDir(), binPath, client, []string{
		"PATH=" + extendPathFor(t, binDir),
	})
	if result.err != nil {
		t.Fatalf("isobox update --check exited with error: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "up-to-date") {
		t.Errorf("isobox update --check does not report up-to-date status:\n%s", result.combined)
	}
}

func TestUpdateCheckReportsBehindWhenNewerReleaseExists(t *testing.T) {
	client := writeFakeUpdateClient(t, []fakeClientRelease{
		{TagName: "v0.2.0"},
		{TagName: "v0.1.1"},
	})
	binPath := buildIsoboxWithVersion(t, "v0.1.1")
	binDir := filepath.Dir(binPath)

	result := runUpdateWithBinaryEnv(t, t.TempDir(), binPath, client, []string{
		"PATH=" + extendPathFor(t, binDir),
	})
	if result.err != nil {
		t.Fatalf("isobox update --check exited with error: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "current:") || !strings.Contains(result.combined, "v0.1.1") {
		t.Errorf("isobox update --check does not report current version:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "latest:") || !strings.Contains(result.combined, "v0.2.0") {
		t.Errorf("isobox update --check does not report latest version:\n%s", result.combined)
	}
}

func TestUpdateCheckRefusesDevBuildWithActionableMessage(t *testing.T) {
	client := writeFakeUpdateClient(t, []fakeClientRelease{
		{TagName: "v0.1.0"},
	})
	// No --ldflags, so the binary defaults to "dev".
	binPath := buildIsobox(t)
	binDir := filepath.Dir(binPath)

	result := runUpdateWithBinaryEnv(t, t.TempDir(), binPath, client, []string{
		"PATH=" + extendPathFor(t, binDir),
	})
	if result.err == nil {
		t.Fatalf("isobox update --check unexpectedly succeeded for a dev build:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "dev") {
		t.Errorf("isobox update --check does not mention dev in its refusal:\n%s", result.combined)
	}
}

func TestUpdateCheckReportsSelectedUpdateTarget(t *testing.T) {
	client := writeFakeUpdateClient(t, []fakeClientRelease{
		{TagName: "v0.1.0"},
	})
	binPath := buildIsoboxWithVersion(t, "v0.1.0")
	binDir := filepath.Dir(binPath)

	result := runUpdateWithBinaryEnv(t, t.TempDir(), binPath, client, []string{
		"PATH=" + extendPathFor(t, binDir),
	})
	if result.err != nil {
		t.Fatalf("isobox update --check exited with error: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "target:") {
		t.Errorf("isobox update --check does not report the selected Update Target:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, binPath) {
		t.Errorf("isobox update --check does not name the active isobox path %q:\n%s", binPath, result.combined)
	}
}

func TestUpdateCheckWarnsAboutDuplicateIsoboxOnPath(t *testing.T) {
	client := writeFakeUpdateClient(t, []fakeClientRelease{
		{TagName: "v0.1.0"},
	})
	binPath := buildIsoboxWithVersion(t, "v0.1.0")
	binDir := filepath.Dir(binPath)

	otherDir := t.TempDir()
	otherBin := filepath.Join(otherDir, "isobox")
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read isobox: %v", err)
	}
	if err := os.WriteFile(otherBin, data, 0o755); err != nil {
		t.Fatalf("write second isobox: %v", err)
	}

	path := binDir + string(os.PathListSeparator) + otherDir
	result := runUpdateWithBinaryEnv(t, t.TempDir(), binPath, client, []string{"PATH=" + path})
	if result.err != nil {
		t.Fatalf("isobox update --check should exit 0 on warning: %v\n%s", result.err, result.combined)
	}
	if !strings.Contains(result.combined, "additional isobox binaries") {
		t.Errorf("isobox update --check does not warn about duplicate isobox binaries:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, otherBin) {
		t.Errorf("isobox update --check does not name the duplicate isobox path %q:\n%s", otherBin, result.combined)
	}
}

// TestUpdateCheckRefusesPackageManagedUpdateTarget exercises the
// managed-path refusal end-to-end. The full allow/deny matrix for
// every managed prefix is covered by the unit suite in
// internal/update/target_test.go; this integration test only
// confirms the CLI wire-up by pointing PATH at a fake isobox
// binary placed under a known managed prefix.
//
// The test places a symlink under /opt, which is a known managed
// prefix in production code. On systems where /opt is read-only
// without elevated privileges the test is skipped; the unit tests
// already cover the matrix and the CLI wire-up is a single line
// (`update.CheckManagedTarget(target.Path)`).
func TestUpdateCheckRefusesPackageManagedUpdateTarget(t *testing.T) {
	binPath := buildIsoboxWithVersion(t, "v0.1.0")

	target := "/opt/isobox-update-test-" + filepath.Base(t.TempDir())
	if err := os.Symlink(binPath, target); err != nil {
		t.Skipf("test requires symlink under /opt: %v", err)
	}
	defer os.Remove(target)

	client := writeFakeUpdateClient(t, []fakeClientRelease{
		{TagName: "v0.1.0"},
	})
	result := runUpdateWithBinaryEnv(t, t.TempDir(), binPath, client, []string{
		"PATH=" + filepath.Dir(target) + string(os.PathListSeparator) + os.Getenv("PATH"),
	})
	if result.err == nil {
		t.Fatalf("isobox update --check unexpectedly succeeded with a managed target:\n%s", result.combined)
	}
	if !strings.Contains(result.combined, "package manager") && !strings.Contains(result.combined, "managed") {
		t.Errorf("isobox update --check does not explain the managed-path refusal:\n%s", result.combined)
	}
}

// buildIsoboxWithVersion compiles isobox with the supplied -X
// version override so the test can pin a tagged version for the
// update check flow. The commit/date variables keep their default
// values because the update check does not read them.
func buildIsoboxWithVersion(t *testing.T, version string) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "isobox")
	ldflags := "-X main.version=" + version
	build := exec.Command("go", "build", "-ldflags", ldflags, "-o", binPath, ".")
	build.Dir = filepath.Join("..", "..", "cmd", "isobox")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build isobox with version %q: %v\n%s", version, err, out)
	}
	return binPath
}

// extendPathFor returns PATH that contains the given extra directory
// followed by the host PATH. The temp directory is cleaned up
// automatically by t.TempDir inside the helper.
func extendPathFor(t *testing.T, extra string) string {
	t.Helper()
	return extra + string(os.PathListSeparator) + os.Getenv("PATH")
}
