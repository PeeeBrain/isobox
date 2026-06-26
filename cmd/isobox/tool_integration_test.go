package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolRejectsToolCallWhenProjectConfigMissing(t *testing.T) {
	source := initGitRepo(t)

	output := runToolFromDir(t, source, "sh", "-c", "true")
	if output.err == nil {
		t.Fatalf("isobox tool unexpectedly succeeded without project config:\n%s", output.combined)
	}

	out := output.combined
	if !strings.Contains(out, "isobox init") {
		t.Fatalf("isobox tool does not direct the user to `isobox init`:\n%s", out)
	}
	if !strings.Contains(out, "no project policy") {
		t.Fatalf("isobox tool does not clearly explain the missing project policy:\n%s", out)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected tool call: %q", readme)
	}
}

func TestToolRejectsToolCallWhenProjectPolicyDisablesIt(t *testing.T) {
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: false})

	output := runToolFromDir(t, source, "sh", "-c", "true")
	if output.err == nil {
		t.Fatalf("isobox tool unexpectedly succeeded with tool_call.enabled=false:\n%s", output.combined)
	}

	out := output.combined
	if !strings.Contains(out, "tool_call.enabled=false") {
		t.Fatalf("isobox tool does not explain that project policy disables tool-call:\n%s", out)
	}
	if !strings.Contains(out, "preflight") {
		t.Fatalf("isobox tool does not mark the rejection as a preflight failure:\n%s", out)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("Workspace Source was modified by rejected tool call: %q", readme)
	}
}

func TestToolRejectsToolCallWhenTrustedRepoHasTrackedModifications(t *testing.T) {
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("uncommitted edit\n"), 0o644); err != nil {
		t.Fatalf("dirty tracked file: %v", err)
	}

	output := runToolFromDir(t, source, "sh", "-c", "true")
	if output.err == nil {
		t.Fatalf("isobox tool unexpectedly succeeded with a dirty trusted repository:\n%s", output.combined)
	}

	out := output.combined
	if !strings.Contains(out, "preflight") {
		t.Fatalf("isobox tool does not mark the rejection as a preflight failure:\n%s", out)
	}
	if !strings.Contains(out, "uncommitted") && !strings.Contains(out, "tracked") {
		t.Fatalf("isobox tool does not explain that tracked modifications are the reason for rejection:\n%s", out)
	}
	if !strings.Contains(out, "no dirty-source override") && !strings.Contains(out, "without --allow-dirty") {
		t.Fatalf("isobox tool does not document that the first milestone has no dirty-source override:\n%s", out)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "uncommitted edit\n" {
		t.Fatalf("Workspace Source was modified by rejected tool call: %q", readme)
	}
}

func TestToolRejectsToolCallWhenTrustedRepoHasUntrackedNonIgnoredFiles(t *testing.T) {
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	if err := os.WriteFile(filepath.Join(source, "scratch.txt"), []byte("scratch\n"), 0o644); err != nil {
		t.Fatalf("create untracked file: %v", err)
	}

	output := runToolFromDir(t, source, "sh", "-c", "true")
	if output.err == nil {
		t.Fatalf("isobox tool unexpectedly succeeded with untracked files in trusted repository:\n%s", output.combined)
	}

	out := output.combined
	if !strings.Contains(out, "preflight") {
		t.Fatalf("isobox tool does not mark the rejection as a preflight failure:\n%s", out)
	}
	if !strings.Contains(out, "scratch.txt") {
		t.Fatalf("isobox tool does not name the offending untracked file:\n%s", out)
	}
	if !strings.Contains(out, "no dirty-source override") && !strings.Contains(out, "without --allow-dirty") {
		t.Fatalf("isobox tool does not document that the first milestone has no dirty-source override:\n%s", out)
	}

	scratch, err := os.ReadFile(filepath.Join(source, "scratch.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(scratch) != "scratch\n" {
		t.Fatalf("Workspace Source was modified by rejected tool call: %q", scratch)
	}
}

func TestToolRejectsToolCallWhenBubblewrapUnavailable(t *testing.T) {
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	// Build a PATH that contains `git` (so project policy discovery works)
	// but excludes whatever directory hosts `bwrap` on the test machine.
	// This makes the preflight see a missing bwrap without depending on the
	// test environment's package manager.
	pathWithoutBwrap := buildPathWithoutBwrap(t)

	output := runToolFromDirWithEnv(t, source, []string{"PATH=" + pathWithoutBwrap}, "sh", "-c", "true")
	if output.err == nil {
		t.Fatalf("isobox tool unexpectedly succeeded without bubblewrap on PATH:\n%s", output.combined)
	}

	out := output.combined
	if !strings.Contains(out, "preflight") {
		t.Fatalf("isobox tool does not mark the rejection as a preflight failure:\n%s", out)
	}
	if !strings.Contains(out, "bubblewrap") && !strings.Contains(out, "bwrap") {
		t.Fatalf("isobox tool does not name bubblewrap as the missing capability:\n%s", out)
	}
	if !strings.Contains(out, "install") && !strings.Contains(out, "PATH") {
		t.Fatalf("isobox tool does not tell the user how to recover:\n%s", out)
	}
}

func TestToolRunsCommandInBubblewrapWorkspace(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	output := runToolFromDir(t, source, "sh", "-c", "printf sandbox > README.md; pwd")
	if output.err != nil {
		t.Fatalf("isobox tool failed:\n%s", output.combined)
	}
	if !strings.Contains(output.combined, "/workspace") {
		t.Fatalf("wrapped command did not see stable /workspace path:\n%s", output.combined)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("trusted repository was modified by wrapped command: %q", readme)
	}
}

func TestToolPreservesRelativeWorkingDirectoryInBubblewrapWorkspace(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})
	nested := filepath.Join(source, "sub", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, ".keep"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runInDir(t, source, "git", "add", ".")
	runInDir(t, source, "git", "commit", "-m", "add nested dir")

	output := runToolFromDir(t, nested, "pwd")
	if output.err != nil {
		t.Fatalf("isobox tool failed:\n%s", output.combined)
	}
	if strings.TrimSpace(output.combined) != "/workspace/sub/dir" {
		t.Fatalf("working directory = %q, want /workspace/sub/dir", strings.TrimSpace(output.combined))
	}
}

func TestToolReturnsWrappedCommandExitCode(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	output := runToolFromDir(t, source, "sh", "-c", "printf out; exit 7")
	if output.err == nil {
		t.Fatalf("isobox tool unexpectedly succeeded:\n%s", output.combined)
	}
	exitErr, ok := output.err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T %v, want exec.ExitError", output.err, output.err)
	}
	if exitErr.ExitCode() != 7 {
		t.Fatalf("exit code = %d, want 7; output:\n%s", exitErr.ExitCode(), output.combined)
	}
	if output.combined != "out" {
		t.Fatalf("stdout = %q, want out", output.combined)
	}
}

func skipIfBubblewrapUnavailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bubblewrap-dependent test skipped: bwrap not on PATH")
	}
}

func TestToolRejectsToolCallWhenPolicyShapeIsUnsupportedInFirstMilestone(t *testing.T) {
	cases := []struct {
		name           string
		policy         projectPolicyYAML
		wantField      string
		wantSentinel   string
		wantSuggestion string
	}{
		{
			name:           "runtime_backend is not bubblewrap",
			policy:         projectPolicyYAML{toolCallEnabled: true, runtimeBackend: "host_process"},
			wantField:      "runtime_backend",
			wantSentinel:   "bubblewrap",
			wantSuggestion: "set runtime_backend: bubblewrap",
		},
		{
			name:           "credentials default is not deny",
			policy:         projectPolicyYAML{toolCallEnabled: true, credentialsDefault: "scoped"},
			wantField:      "credentials.default",
			wantSentinel:   "deny",
			wantSuggestion: "set credentials.default: deny",
		},
		{
			name:           "promotion mode is not manual",
			policy:         projectPolicyYAML{toolCallEnabled: true, promotionMode: "automatic"},
			wantField:      "promotion.mode",
			wantSentinel:   "manual",
			wantSuggestion: "set promotion.mode: manual",
		},
		{
			name:           "workspace source kind is not project_root",
			policy:         projectPolicyYAML{toolCallEnabled: true, workspaceKind: "subdirectory"},
			wantField:      "workspace_source.kind",
			wantSentinel:   "project_root",
			wantSuggestion: "set workspace_source.kind: project_root",
		},
		{
			name:           "development environment path mode is not backend_default",
			policy:         projectPolicyYAML{toolCallEnabled: true, pathMode: "inherit"},
			wantField:      "development_environment.path_mode",
			wantSentinel:   "backend_default",
			wantSuggestion: "set development_environment.path_mode: backend_default",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := initGitRepo(t)
			writeProjectPolicy(t, source, tc.policy)

			output := runToolFromDir(t, source, "sh", "-c", "true")
			if output.err == nil {
				t.Fatalf("isobox tool unexpectedly succeeded with unsupported policy %s:\n%s", tc.name, output.combined)
			}

			out := output.combined
			if !strings.Contains(out, "preflight") {
				t.Fatalf("isobox tool does not mark the rejection as a preflight failure:\n%s", out)
			}
			if !strings.Contains(out, tc.wantField) {
				t.Fatalf("isobox tool does not name the offending field %q:\n%s", tc.wantField, out)
			}
			if !strings.Contains(out, tc.wantSentinel) {
				t.Fatalf("isobox tool does not mention the supported value %q:\n%s", tc.wantSentinel, out)
			}
			if !strings.Contains(out, tc.wantSuggestion) {
				t.Fatalf("isobox tool does not suggest %q:\n%s", tc.wantSuggestion, out)
			}
		})
	}
}

// buildPathWithoutBwrap returns a PATH string that keeps `git` available so
// project policy discovery works, but excludes `bwrap` regardless of the dev
// machine. A throwaway directory containing only `git` is created and used
// as the entire PATH so neither bwrap's directory nor any other host
// tooling can satisfy the bubblewrap lookup.
func buildPathWithoutBwrap(t *testing.T) string {
	t.Helper()

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("test setup requires git on host PATH: %v", err)
	}
	gitDir := filepath.Dir(gitPath)

	isolated := t.TempDir()
	gitLink := filepath.Join(isolated, "git")
	if err := os.Symlink(gitPath, gitLink); err != nil {
		// Fall back to copying the binary if symlinks are not available.
		data, err := os.ReadFile(gitPath)
		if err != nil {
			t.Fatalf("read git binary: %v", err)
		}
		if err := os.WriteFile(gitLink, data, 0o755); err != nil {
			t.Fatalf("write git binary: %v", err)
		}
	}
	_ = gitDir
	return isolated
}

type projectPolicyYAML struct {
	toolCallEnabled    bool
	runtimeBackend     string
	pathMode           string
	workspaceKind      string
	networkDefault     string
	filesystemExpose   bool
	credentialsDefault string
	preflightRules     []string
	promotionMode      string
}

// writeProjectPolicy writes a hand-crafted project policy at the Git root
// and commits it so subsequent preflight checks (such as the bubblewrap
// availability check) remain reachable from a clean repository state.
// Individual preflight behaviors are exercised by toggling one section at a
// time. Defaults match the restrictive policy `isobox init` would generate.
func writeProjectPolicy(t *testing.T, source string, p projectPolicyYAML) {
	t.Helper()

	if p.runtimeBackend == "" {
		p.runtimeBackend = "bubblewrap"
	}
	if p.pathMode == "" {
		p.pathMode = "backend_default"
	}
	if p.workspaceKind == "" {
		p.workspaceKind = "project_root"
	}
	if p.networkDefault == "" {
		p.networkDefault = "deny"
	}
	if !p.filesystemExpose {
		p.filesystemExpose = true
	}
	if p.credentialsDefault == "" {
		p.credentialsDefault = "deny"
	}
	if len(p.preflightRules) == 0 {
		p.preflightRules = []string{"reject_dirty_workspace_source"}
	}
	if p.promotionMode == "" {
		p.promotionMode = "manual"
	}

	var rules strings.Builder
	for _, r := range p.preflightRules {
		rules.WriteString("    - " + r + "\n")
	}

	policy := strings.Join([]string{
		"api_version: isobox.dev/v1alpha1",
		"kind: ProjectPolicy",
		"tool_call:",
		"  enabled: " + boolString(p.toolCallEnabled),
		"runtime_backend: " + p.runtimeBackend,
		"development_environment:",
		"  path_mode: " + p.pathMode,
		"workspace_source:",
		"  kind: " + p.workspaceKind,
		"network:",
		"  default: " + p.networkDefault,
		"filesystem:",
		"  expose_workspace: " + boolString(p.filesystemExpose),
		"credentials:",
		"  default: " + p.credentialsDefault,
		"preflight:",
		"  rules:",
		rules.String(),
		"promotion:",
		"  mode: " + p.promotionMode,
	}, "\n") + "\n"

	configDir := filepath.Join(source, ".isobox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create .isobox: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(policy), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	runInDir(t, source, "git", "add", ".isobox")
	runInDir(t, source, "git", "commit", "-m", "add project policy")
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

type toolRunResult struct {
	combined string
	err      error
}

// runToolFromDir builds the isobox binary, runs `isobox tool -- <args>` from
// the given working directory, and returns the combined output and process
// error. The tool command discovers the project policy from the current
// directory, so the test must run it from a directory inside the project
// Git repository.
func runToolFromDir(t *testing.T, dir string, cmdArgs ...string) toolRunResult {
	t.Helper()
	return runToolFromDirWithEnv(t, dir, nil, cmdArgs...)
}

// runToolFromDirWithEnv behaves like runToolFromDir but lets the caller add
// KEY=VALUE entries to the child process environment. Used to simulate a
// system PATH without bubblewrap for preflight coverage.
func runToolFromDirWithEnv(t *testing.T, dir string, extraEnv []string, cmdArgs ...string) toolRunResult {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "isobox")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = filepath.Join("..", "..", "cmd", "isobox")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build isobox: %v\n%s", err, out)
	}

	tool := exec.Command(binPath, append([]string{"tool", "--"}, cmdArgs...)...)
	tool.Dir = dir
	tool.Env = append(os.Environ(), extraEnv...)
	combined, err := tool.CombinedOutput()
	return toolRunResult{combined: string(combined), err: err}
}

func TestToolPreflightFailuresDoNotClaimWrappedCommandExecution(t *testing.T) {
	// Each scenario triggers a different preflight failure and asserts that
	// the output does not announce a Task identity or completion summary.
	// A preflight failure is an isobox infrastructure error and must be
	// distinct from a wrapped command result.
	scenarios := []struct {
		name        string
		setup       func(t *testing.T) string
		wantExitOne bool
	}{
		{
			name: "missing project policy",
			setup: func(t *testing.T) string {
				return initGitRepo(t)
			},
			wantExitOne: true,
		},
		{
			name: "tool-call disabled in project policy",
			setup: func(t *testing.T) string {
				source := initGitRepo(t)
				writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: false})
				return source
			},
			wantExitOne: true,
		},
		{
			name: "unsupported runtime backend",
			setup: func(t *testing.T) string {
				source := initGitRepo(t)
				writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true, runtimeBackend: "host_process"})
				return source
			},
			wantExitOne: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			source := sc.setup(t)
			output := runToolFromDir(t, source, "sh", "-c", "true")

			if sc.wantExitOne && output.err == nil {
				t.Fatalf("isobox tool unexpectedly succeeded in preflight scenario %q:\n%s", sc.name, output.combined)
			}
			// A preflight failure must always be reported as a non-zero
			// exit; isobox infrastructure failures are status 1, not the
			// wrapped command's exit code (which never ran).
			if !sc.wantExitOne && output.err != nil {
				t.Fatalf("isobox tool unexpectedly failed in non-failure scenario %q:\n%s", sc.name, output.combined)
			}

			out := output.combined
			if strings.Contains(out, "task-") {
				t.Fatalf("preflight failure must not announce a Task ID (it never ran a task):\n%s", out)
			}
			if strings.Contains(out, "workspace disposed") {
				t.Fatalf("preflight failure must not announce workspace disposal:\n%s", out)
			}
			if strings.Contains(out, "workspace retained") {
				t.Fatalf("preflight failure must not announce a retained workspace:\n%s", out)
			}
			if !strings.Contains(out, "preflight") {
				t.Fatalf("preflight failure must mark the rejection as preflight, not as a wrapped command result:\n%s", out)
			}
		})
	}
}

// runInDir runs a host command in the given directory and fails the test on
// error. Used for test setup work like committing the generated policy.
func runInDir(t *testing.T, dir, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v in %s: %v\n%s", name, args, dir, err, out)
	}
}
