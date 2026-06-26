package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"isobox/internal/projectpolicy"
)

func TestInitCreatesProjectConfigInGitRepository(t *testing.T) {
	source := initGitRepo(t)

	cmd := exec.Command("go", "run", ".", "init", source)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox init failed: %v\n%s", err, output)
	}

	configPath := filepath.Join(source, ".isobox", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("isobox init did not create project config: %v", err)
	}
}

func TestInitRejectsDirectoryOutsideGitRepository(t *testing.T) {
	source := t.TempDir()

	cmd := exec.Command("go", "run", ".", "init", source)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox init unexpectedly succeeded outside a Git repository\n%s", output)
	}

	if _, err := os.Stat(filepath.Join(source, ".isobox")); !os.IsNotExist(err) {
		t.Fatalf("isobox init created .isobox despite failing: %v", err)
	}
}

func TestInitRejectsWhenProjectConfigAlreadyExists(t *testing.T) {
	source := initGitRepo(t)

	existing := filepath.Join(source, ".isobox", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatalf("seed config dir: %v", err)
	}
	if err := os.WriteFile(existing, []byte("api_version: isobox.dev/v1alpha1\nkind: ProjectPolicy\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	cmd := exec.Command("go", "run", ".", "init", source)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("isobox init unexpectedly succeeded with an existing project config\n%s", output)
	}

	contents, readErr := os.ReadFile(existing)
	if readErr != nil {
		t.Fatalf("read existing config: %v", readErr)
	}
	if string(contents) != "api_version: isobox.dev/v1alpha1\nkind: ProjectPolicy\n" {
		t.Fatalf("isobox init modified existing config: %q", contents)
	}
}

func TestInitPlacesConfigAtGitRootWhenInvokedFromSubdirectory(t *testing.T) {
	source := initGitRepo(t)
	subdir := filepath.Join(source, "src", "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	cmd := exec.Command("go", "run", ".", "init", subdir)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox init failed: %v\n%s", err, output)
	}

	rootConfig := filepath.Join(source, ".isobox", "config.yaml")
	if _, err := os.Stat(rootConfig); err != nil {
		t.Fatalf("isobox init did not create config at Git root %s: %v", rootConfig, err)
	}

	subdirConfig := filepath.Join(subdir, ".isobox", "config.yaml")
	if _, err := os.Stat(subdirConfig); !os.IsNotExist(err) {
		t.Fatalf("isobox init created config at subdir %s; the policy must live at the Git root: %v", subdirConfig, err)
	}

	rootGitignore := filepath.Join(source, ".gitignore")
	contents, err := os.ReadFile(rootGitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(contents), ".isobox/tasks/") {
		t.Fatalf(".gitignore at Git root missing .isobox/tasks/ entry: %q", contents)
	}
}

func TestInitIgnoresIsoboxTasksDirectory(t *testing.T) {
	source := initGitRepo(t)

	cmd := exec.Command("go", "run", ".", "init", source)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox init failed: %v\n%s", err, output)
	}

	gitignorePath := filepath.Join(source, ".gitignore")
	contents, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}

	lines := strings.Split(string(contents), "\n")
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == ".isobox/tasks/" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf(".gitignore missing .isobox/tasks/ entry; got %q", contents)
	}
}

func TestInitDoesNotModifyReadmeOrAgentInstructionFiles(t *testing.T) {
	source := initGitRepo(t)

	agentsContent := []byte("# project agent instructions\n")
	claudeContent := []byte("# project claude instructions\n")
	if err := os.WriteFile(filepath.Join(source, "AGENTS.md"), agentsContent, 0o644); err != nil {
		t.Fatalf("seed AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "CLAUDE.md"), claudeContent, 0o644); err != nil {
		t.Fatalf("seed CLAUDE.md: %v", err)
	}

	cmd := exec.Command("go", "run", ".", "init", source)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox init failed: %v\n%s", err, output)
	}

	assertUnchanged := func(name string, want []byte) {
		t.Helper()
		got, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s was modified by isobox init\nwant: %q\ngot:  %q", name, want, got)
		}
	}

	assertUnchanged("README.md", []byte("original\n"))
	assertUnchanged("AGENTS.md", agentsContent)
	assertUnchanged("CLAUDE.md", claudeContent)
}

func TestInitGeneratesRestrictiveProjectPolicyDefaults(t *testing.T) {
	source := initGitRepo(t)

	cmd := exec.Command("go", "run", ".", "init", source)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox init failed: %v\n%s", err, output)
	}

	configPath := filepath.Join(source, ".isobox", "config.yaml")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read project policy: %v", err)
	}

	policy, err := projectpolicy.Parse(raw)
	if err != nil {
		t.Fatalf("parse project policy: %v\n%s", err, raw)
	}

	if policy.Network.Default != projectpolicy.NetworkDefaultDeny {
		t.Fatalf("network default = %q, want %q", policy.Network.Default, projectpolicy.NetworkDefaultDeny)
	}
	if policy.Credentials.Default != projectpolicy.CredentialsDefaultDeny {
		t.Fatalf("credentials default = %q, want %q", policy.Credentials.Default, projectpolicy.CredentialsDefaultDeny)
	}
	if policy.Promotion.Mode != projectpolicy.PromotionModeManual {
		t.Fatalf("promotion mode = %q, want %q", policy.Promotion.Mode, projectpolicy.PromotionModeManual)
	}
	if policy.RuntimeBackend != projectpolicy.RuntimeBackendBubblewrap {
		t.Fatalf("runtime backend = %q, want %q", policy.RuntimeBackend, projectpolicy.RuntimeBackendBubblewrap)
	}
	if policy.WorkspaceSource.Kind != projectpolicy.WorkspaceSourceProjectRoot {
		t.Fatalf("workspace source kind = %q, want %q", policy.WorkspaceSource.Kind, projectpolicy.WorkspaceSourceProjectRoot)
	}
	if !policy.ToolCall.Enabled {
		t.Fatalf("tool_call.enabled = false, want true (the init command is the supported entry point for tool-call policy)")
	}
	if !policy.Filesystem.ExposeWorkspace {
		t.Fatalf("filesystem.expose_workspace = false, want true")
	}

	wantPreflight := []string{projectpolicy.PreflightRejectDirtyWorkspaceSource}
	if len(policy.Preflight.Rules) != len(wantPreflight) || policy.Preflight.Rules[0] != wantPreflight[0] {
		t.Fatalf("preflight.rules = %#v, want %#v", policy.Preflight.Rules, wantPreflight)
	}
}

func TestInitGeneratesProjectPolicyWithSparseConsequenceComments(t *testing.T) {
	source := initGitRepo(t)

	cmd := exec.Command("go", "run", ".", "init", source)
	cmd.Dir = filepath.Join("..", "..", "cmd", "isobox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isobox init failed: %v\n%s", err, output)
	}

	raw, err := os.ReadFile(filepath.Join(source, ".isobox", "config.yaml"))
	if err != nil {
		t.Fatalf("read project policy: %v", err)
	}

	text := string(raw)
	if _, err := projectpolicy.Parse(raw); err != nil {
		t.Fatalf("parse project policy: %v\n%s", err, text)
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		t.Fatal("empty project policy")
	}

	var commentLines []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			commentLines = append(commentLines, line)
		}
	}
	yamlLines := len(lines) - len(commentLines)
	if len(commentLines) == 0 {
		t.Fatalf("project policy has no comment lines; sparse comments are required to explain consequences:\n%s", text)
	}
	if len(commentLines) > yamlLines*2 {
		t.Fatalf("project policy has %d comment lines for %d yaml lines; comments must remain sparse:\n%s", len(commentLines), yamlLines, text)
	}

	combined := strings.ToLower(strings.Join(commentLines, "\n"))
	consequenceTopics := []string{"network", "credential", "promotion"}
	for _, topic := range consequenceTopics {
		if !strings.Contains(combined, topic) {
			t.Fatalf("project policy comments do not mention %q consequences:\n%s", topic, text)
		}
	}
}
