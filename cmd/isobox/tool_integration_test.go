package main_test

import (
	"encoding/json"
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

func TestToolCreatesArtifactBackedTaskRecord(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	output := runToolFromDir(t, source, "sh", "-c", "printf out; printf err >&2; printf changed > README.md")
	if output.err != nil {
		t.Fatalf("isobox tool failed:\n%s", output.combined)
	}
	if output.stdout != "out" || !strings.Contains(output.stderr, "err") {
		t.Fatalf("agent feedback stdout=%q stderr=%q, want live stdout/stderr", output.stdout, output.stderr)
	}
	if !strings.Contains(output.stderr, "starting tool call") || !strings.Contains(output.stderr, "completed outcome=success") {
		t.Fatalf("stderr does not contain task metadata prelude and summary: %q", output.stderr)
	}
	if !strings.Contains(output.stderr, "policy network=deny") ||
		!strings.Contains(output.stderr, "network_enforcement=not_enforced") ||
		!strings.Contains(output.stderr, "credentials=deny") ||
		!strings.Contains(output.stderr, "credential_enforcement=enforced") {
		t.Fatalf("stderr does not report policy intent and backend enforcement status: %q", output.stderr)
	}

	taskDirs, err := filepath.Glob(filepath.Join(source, ".isobox", "tasks", "task-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(taskDirs) != 1 {
		t.Fatalf("task records = %d, want 1 under project .isobox/tasks", len(taskDirs))
	}

	recordBytes, err := os.ReadFile(filepath.Join(taskDirs[0], "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		t.Fatal(err)
	}
	result, ok := record["result"].(map[string]any)
	if !ok {
		t.Fatalf("record result missing or wrong shape: %s", recordBytes)
	}
	for _, field := range []string{"stdout", "stderr", "diff"} {
		if _, ok := result[field]; ok {
			t.Fatalf("record.json contains inline result field %q instead of only artifact references:\n%s", field, recordBytes)
		}
	}
	recordText := string(recordBytes)
	for _, rel := range []string{"artifacts/stdout.txt", "artifacts/stderr.txt", "artifacts/diff.patch"} {
		if _, err := os.Stat(filepath.Join(taskDirs[0], rel)); err != nil {
			t.Fatalf("missing task artifact %s: %v", rel, err)
		}
		if !strings.Contains(recordText, rel) {
			t.Fatalf("record.json does not reference artifact %s:\n%s", rel, recordText)
		}
	}
}

func TestToolWorkflowFromInitCapturesAndPromotesTrackedAndUntrackedResults(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)

	init := exec.Command("go", "run", ".", "init", source)
	init.Dir = filepath.Join("..", "..", "cmd", "isobox")
	if output, err := init.CombinedOutput(); err != nil {
		t.Fatalf("isobox init failed: %v\n%s", err, output)
	}
	runInDir(t, source, "git", "add", ".isobox", ".gitignore")
	runInDir(t, source, "git", "commit", "-m", "initialize isobox policy")

	output := runToolFromDir(t, source, "sh", "-c", "printf tracked > README.md; mkdir -p notes; printf untracked > notes/result.txt; printf agent-feedback; printf agent-note >&2")
	if output.err != nil {
		t.Fatalf("isobox tool failed:\n%s", output.combined)
	}
	if output.stdout != "agent-feedback" {
		t.Fatalf("agent feedback stdout = %q, want wrapped command stdout", output.stdout)
	}
	if !strings.Contains(output.stderr, "agent-note") {
		t.Fatalf("agent feedback stderr = %q, want wrapped command stderr", output.stderr)
	}
	if !strings.Contains(output.stderr, "starting tool call") || !strings.Contains(output.stderr, "completed outcome=success") {
		t.Fatalf("agent feedback stderr does not include task lifecycle:\n%s", output.stderr)
	}

	readme, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "original\n" {
		t.Fatalf("trusted repository changed before promotion: %q", readme)
	}
	if _, err := os.Stat(filepath.Join(source, "notes", "result.txt")); !os.IsNotExist(err) {
		t.Fatalf("untracked task result leaked into trusted repository before promotion: %v", err)
	}

	taskDirs, err := filepath.Glob(filepath.Join(source, ".isobox", "tasks", "task-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(taskDirs) != 1 {
		t.Fatalf("task records = %d, want 1 under project .isobox/tasks", len(taskDirs))
	}
	recordBytes, err := os.ReadFile(filepath.Join(taskDirs[0], "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	recordText := string(recordBytes)
	for _, rel := range []string{"artifacts/stdout.txt", "artifacts/stderr.txt", "artifacts/diff.patch"} {
		if _, err := os.Stat(filepath.Join(taskDirs[0], rel)); err != nil {
			t.Fatalf("missing task artifact %s: %v", rel, err)
		}
		if !strings.Contains(recordText, rel) {
			t.Fatalf("record.json does not reference artifact %s:\n%s", rel, recordText)
		}
	}
	diffBytes, err := os.ReadFile(filepath.Join(taskDirs[0], "artifacts", "diff.patch"))
	if err != nil {
		t.Fatal(err)
	}
	diff := string(diffBytes)
	for _, want := range []string{"README.md", "notes/result.txt", "+tracked", "+untracked"} {
		if !strings.Contains(diff, want) {
			t.Fatalf("captured diff missing %q:\n%s", want, diff)
		}
	}

	promote := exec.Command("go", "run", ".", "promote", "--yes", taskDirs[0])
	promote.Dir = filepath.Join("..", "..", "cmd", "isobox")
	if output, err := promote.CombinedOutput(); err != nil {
		t.Fatalf("isobox promote failed: %v\n%s", err, output)
	}

	readme, err = os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readme) != "tracked" {
		t.Fatalf("tracked result was not promoted: %q", readme)
	}
	result, err := os.ReadFile(filepath.Join(source, "notes", "result.txt"))
	if err != nil {
		t.Fatalf("untracked result was not promoted: %v", err)
	}
	if string(result) != "untracked" {
		t.Fatalf("promoted untracked result = %q, want untracked", result)
	}
	promotedRecordBytes, err := os.ReadFile(filepath.Join(taskDirs[0], "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(promotedRecordBytes), `"mode": "explicit_non_interactive"`) {
		t.Fatalf("promoted task record does not capture explicit non-interactive confirmation:\n%s", promotedRecordBytes)
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
	if strings.TrimSpace(output.stdout) != "/workspace/sub/dir" {
		t.Fatalf("working directory = %q, want /workspace/sub/dir", strings.TrimSpace(output.stdout))
	}
}

func TestToolDoesNotExposeCredentialEnvironment(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	output := runToolFromDirWithEnv(t, source, []string{"GITHUB_TOKEN=super-secret"}, "sh", "-c", "printf '%s' \"${GITHUB_TOKEN-unset}:$PATH\"")
	if output.err != nil {
		t.Fatalf("isobox tool failed:\n%s", output.combined)
	}
	if strings.Contains(output.combined, "super-secret") {
		t.Fatalf("sandbox exposed host credential environment:\n%s", output.combined)
	}
	if !strings.HasPrefix(output.stdout, "unset:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin") {
		t.Fatalf("sandbox did not use backend-default environment:\nstdout=%s\nstderr=%s", output.stdout, output.stderr)
	}
}

func TestToolCommandReceivesOnlyBackendDefaultPath(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	output := runToolFromDirWithEnv(t, source, []string{
		"ISOBOX_TEST_AMBIENT=ambient",
		"HOME=/host/home",
	}, "env")
	if output.err != nil {
		t.Fatalf("isobox tool failed:\n%s", output.combined)
	}
	if strings.TrimSpace(output.stdout) != "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" {
		t.Fatalf("sandbox environment = %q, want only backend-default PATH", output.stdout)
	}
}

func TestToolRecordsCredentialPolicyAndEnforcement(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	output := runToolFromDirWithEnv(t, source, []string{"GITHUB_TOKEN=super-secret"}, "sh", "-c", "true")
	if output.err != nil {
		t.Fatalf("isobox tool failed:\n%s", output.combined)
	}

	recordText := readOnlyToolRecordText(t, source)
	if !strings.Contains(recordText, `"credentials"`) || !strings.Contains(recordText, `"default": "deny"`) {
		t.Fatalf("task record does not record credential deny intent:\n%s", recordText)
	}
	if !strings.Contains(recordText, `"credential_enforcement"`) || !strings.Contains(recordText, `"status": "enforced"`) {
		t.Fatalf("task record does not record credential enforcement status:\n%s", recordText)
	}
	if !strings.Contains(recordText, "no credential material was exposed") {
		t.Fatalf("task record does not state that no credentials were exposed:\n%s", recordText)
	}
	if strings.Contains(recordText, "super-secret") {
		t.Fatalf("task record exposed credential material:\n%s", recordText)
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
	if output.stdout != "out" {
		t.Fatalf("stdout = %q, want out; stderr=%q", output.stdout, output.stderr)
	}
}

func TestToolSurfacesPrivateDependencyFailureAsCommandOutput(t *testing.T) {
	skipIfBubblewrapUnavailable(t)
	source := initGitRepo(t)
	writeProjectPolicy(t, source, projectPolicyYAML{toolCallEnabled: true})

	output := runToolFromDir(t, source, "sh", "-c", "printf 'fatal: could not read Username for private.example\\n' >&2; exit 128")
	if output.err == nil {
		t.Fatalf("isobox tool unexpectedly succeeded:\n%s", output.combined)
	}
	exitErr, ok := output.err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T %v, want exec.ExitError", output.err, output.err)
	}
	if exitErr.ExitCode() != 128 {
		t.Fatalf("exit code = %d, want 128; output:\n%s", exitErr.ExitCode(), output.combined)
	}
	if !strings.Contains(output.stderr, "fatal: could not read Username for private.example") {
		t.Fatalf("wrapped command stderr was not surfaced:\n%s", output.combined)
	}
	if strings.Contains(output.combined, "launch workload command") || strings.Contains(output.combined, "bubblewrap setup failed") {
		t.Fatalf("private dependency failure was diagnosed as isobox infrastructure:\n%s", output.combined)
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

func TestToolNetworkPolicyAcceptsDenyAndInheritedOnly(t *testing.T) {
	cases := []struct {
		name           string
		networkDefault string
		extraNetwork   []string
		wantErr        bool
	}{
		{name: "deny", networkDefault: "deny"},
		{name: "inherited", networkDefault: "inherited"},
		{
			name:           "host allowlist",
			networkDefault: "deny",
			extraNetwork:   []string{"  allow:", "    - host: github.com"},
			wantErr:        true,
		},
		{
			name:           "domain allowlist",
			networkDefault: "deny",
			extraNetwork:   []string{"  allow:", "    - domain: example.com"},
			wantErr:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := initGitRepo(t)
			writeProjectPolicy(t, source, projectPolicyYAML{
				toolCallEnabled: true,
				networkDefault:  tc.networkDefault,
				extraNetwork:    tc.extraNetwork,
			})

			output := runToolFromDir(t, source, "sh", "-c", "true")
			if tc.wantErr {
				if output.err == nil {
					t.Fatalf("isobox tool unexpectedly accepted unsupported network policy:\n%s", output.combined)
				}
				if !strings.Contains(output.combined, "network.allow") {
					t.Fatalf("network allowlist rejection does not name network.allow:\n%s", output.combined)
				}
				return
			}
			if strings.Contains(output.combined, "bubblewrap (bwrap) is not on PATH") {
				t.Skipf("bwrap unavailable; skipping supported network policy execution case: %s", output.combined)
			}
			if output.err != nil {
				t.Fatalf("isobox tool rejected supported network policy:\n%s", output.combined)
			}
			recordText := readOnlyToolRecordText(t, source)
			if !strings.Contains(recordText, `"network"`) || !strings.Contains(recordText, `"default": "`+tc.networkDefault+`"`) {
				t.Fatalf("task record does not record network default %q:\n%s", tc.networkDefault, recordText)
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
	extraNetwork       []string
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
		strings.Join(p.extraNetwork, "\n"),
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

func readOnlyToolRecordText(t *testing.T, source string) string {
	t.Helper()

	taskRoot := filepath.Join(source, ".isobox", "tasks")
	entries, err := os.ReadDir(taskRoot)
	if err != nil {
		t.Fatalf("read task root: %v", err)
	}
	var taskDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			taskDirs = append(taskDirs, entry.Name())
		}
	}
	if len(taskDirs) != 1 {
		t.Fatalf("task record dirs = %v, want exactly one", taskDirs)
	}
	recordBytes, err := os.ReadFile(filepath.Join(taskRoot, taskDirs[0], "record.json"))
	if err != nil {
		t.Fatalf("read record.json: %v", err)
	}
	return string(recordBytes)
}

type toolRunResult struct {
	combined string
	stdout   string
	stderr   string
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
	var stdout, stderr strings.Builder
	tool.Stdout = &stdout
	tool.Stderr = &stderr
	err := tool.Run()
	return toolRunResult{combined: stdout.String() + stderr.String(), stdout: stdout.String(), stderr: stderr.String(), err: err}
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
