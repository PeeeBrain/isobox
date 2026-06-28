package docs_test

import (
	"os"
	"strings"
	"testing"
)

// The README is the user-facing documentation of the policy-shaped host backend
// workflow (issue #14). These tests guard the documented acceptance criteria so
// the safety-boundary thesis and the intent-vs-enforcement distinction are not
// silently lost from the docs. They assert user-facing statements, not private
// implementation details.

func TestReadmeExplainsAgentSafetyBoundaryThesis(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"isobox is not a safer shell",
		"safety boundary for Agent autonomy",
		"disposable Workspace",
		"Task Record",
		"Task Result",
		"Promotion",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document %q (Agent safety boundary thesis / current-milestone concepts)", phrase)
		}
	}
}

func TestReadmeDescribesPolicyShapedHostBackendLowerAssurance(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"host Runtime Backend",
		"does not provide strong isolation",
		"Effective Policy",
		"Sandbox Policy",
		"Runtime Backend",
		"lower-assurance",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not describe the policy-shaped host backend and lower-assurance limitations: missing %q", phrase)
		}
	}
}

func TestReadmeDistinguishesIntentFromEnforcement(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"Policy intent versus enforcement",
		"intent",
		"enforcement status",
		"not enforced",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not distinguish policy intent from enforcement: missing %q", phrase)
		}
	}

	// The intent-vs-enforcement distinction must call out the recorded policy
	// categories by name.
	for _, category := range []string{"Network", "Resource limits", "Reuse Inputs"} {
		if !strings.Contains(readme, category) {
			t.Errorf("README intent-vs-enforcement section does not mention recorded policy category %q", category)
		}
	}
}

func TestReadmeDoesNotImplyStrongSandboxIsolation(t *testing.T) {
	readme := normalize(readReadme(t))

	// The docs must explicitly disclaim strong isolation for the host backend.
	if !strings.Contains(readme, "does not provide strong isolation") {
		t.Errorf("README must state the host Runtime Backend does not provide strong isolation")
	}

	// It must not claim that isobox currently provides strong sandbox isolation.
	for _, forbidden := range []string{
		"isobox provides strong sandbox isolation",
		"isobox provides a security sandbox",
		"strong isolation is enforced",
	} {
		if strings.Contains(readme, forbidden) {
			t.Errorf("README implies strong isolation with forbidden phrase %q", forbidden)
		}
	}
}

func TestReadmeDocumentsHostAgentReuseExplicitInputs(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"Host Agent Reuse",
		"Reuse Inputs are always explicit",
		"never silently inherits broad host state",
		"lowers isolation assurance",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document Host Agent Reuse explicit Reuse Inputs: missing %q", phrase)
		}
	}
}

func TestCooperativeSafeModeSkillDocumentsSafetyRules(t *testing.T) {
	skill := normalize(readFile(t, "skills/isobox-agent-guide/SKILL.md"))

	wantPhrases := []string{
		"Cooperative Safe Mode",
		"route shell actions through isobox tool by default",
		"Direct Shell Escape",
		"fresh human approval",
		"creates no isobox Task Record",
		"does not make an isobox containment claim",
		"Promotion Approval",
		"isobox promote --yes",
		"specific Task Result",
		"Stop and report the exact reason",
		"missing project policy",
		"tool calls are disabled",
		"dirty trusted repository",
		"bubblewrap is missing",
		"unsupported policy",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(skill, phrase) {
			t.Errorf("Cooperative Safe Mode skill does not document %q", phrase)
		}
	}
}

func TestCooperativeSafeModeSkillDoesNotOverclaimTrackingOrContainment(t *testing.T) {
	skill := normalize(readFile(t, "skills/isobox-agent-guide/SKILL.md"))

	forbiddenPhrases := []string{
		"isobox tracks all shell actions",
		"isobox tracks every shell action",
		"isobox contains all shell actions",
		"isobox contains every shell action",
		"Direct Shell Escape creates an isobox Task Record",
		"Direct Shell Escape is contained by isobox",
		"direct shell calls are tracked by isobox",
		"direct shell calls are contained by isobox",
	}
	for _, phrase := range forbiddenPhrases {
		if strings.Contains(skill, phrase) {
			t.Errorf("Cooperative Safe Mode skill overclaims tracking or containment with forbidden phrase %q", phrase)
		}
	}
}

func TestReadmeDocumentsToolCallSandboxWorkflow(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"isobox init",
		"isobox tool -- <command>",
		"Cooperative Safe Mode",
		"bubblewrap",
		"host_process is insufficient for this workflow",
		".isobox/tasks",
		"Task Artifacts",
		"stdout",
		"stderr",
		"patch data",
		"untracked file",
		"Agent Feedback",
		"isobox promote .isobox/tasks/",
		"ADR 0003",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document Tool-Call Sandbox workflow acceptance point %q", phrase)
		}
	}
}

func TestReadmeDocumentsDirectShellEscapeBoundary(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"Direct Shell Escape",
		"creates no Task Record",
		"receives no containment claim",
		"conversation-level human approval",
		"cooperative routing",
		"enforced shell interception",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document the direct shell / cooperative boundary: missing %q", phrase)
		}
	}

	for _, forbidden := range []string{
		"isobox intercepts direct shell calls",
		"direct shell calls are contained by isobox",
		"Cooperative Safe Mode enforces shell interception",
	} {
		if strings.Contains(readme, forbidden) {
			t.Errorf("README overclaims shell interception with forbidden phrase %q", forbidden)
		}
	}
}

func TestReadmeDocumentsDoctorCommand(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"isobox doctor",
		"Doctor Finding",
		"Doctor Check",
		"read-only",
		"status 1 only when",
		"ok",
		"warning",
		"error",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document isobox doctor point %q", phrase)
		}
	}
}

func TestReadmeDocumentsUpdateCheckCommand(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"isobox update --check",
		"Update Target",
		"GitHub Releases API",
		"stable release",
		"dev build",
		"package-manager-managed",
		"${HOME}/.local/bin",
		"/usr/local/bin",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document isobox update --check point %q", phrase)
		}
	}

	forbidden := []string{
		"isobox update --check downloads and replaces",
		"isobox update runs install.sh",
	}
	for _, phrase := range forbidden {
		if strings.Contains(readme, phrase) {
			t.Errorf("README overclaims isobox update --check behavior with %q", phrase)
		}
	}
}

func TestReadmeDocumentsGlobalDoctorChecks(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"git on PATH",
		"bubblewrap (bwrap)",
		"Tool-Call Sandbox",
		"isobox on PATH",
		"multiple isobox binaries on PATH",
		"call the network",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document the global doctor checks point %q", phrase)
		}
	}
}

func TestReadmeDocumentsRicherHelpSurface(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"isobox --help",
		"isobox <command> --help",
		"Workload Command",
		"per-command usage",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document the richer help surface point %q", phrase)
		}
	}
}

func TestReadmeDocumentsToolCallMilestoneBehavior(t *testing.T) {
	readme := normalize(readReadme(t))

	wantPhrases := []string{
		"Tool-Call Sandbox",
		"isobox init",
		"isobox tool --",
		"Preflight Rules",
		"bubblewrap",
		"stores the Task Record in the project-owned Project Task Store under .isobox/tasks/",
		"tracked changes and reviewable untracked files",
		"explicit Promotion",
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(readme, phrase) {
			t.Errorf("README does not document Tool-Call Sandbox behavior: missing %q", phrase)
		}
	}

	forbidden := []string{
		"New untracked files are not included",
		"new untracked files are not yet captured",
	}
	for _, phrase := range forbidden {
		if strings.Contains(readme, phrase) {
			t.Errorf("README still contradicts untracked result capture with %q", phrase)
		}
	}
}

// readReadme reads the README at the repository root.
func readReadme(t *testing.T) string {
	t.Helper()
	return readFile(t, "README.md")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(bytes)
}

// normalize collapses whitespace (including newlines) to single spaces and strips
// common Markdown emphasis so phrase matching survives line wrapping and bold.
func normalize(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
