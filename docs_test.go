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

// readReadme reads the README at the repository root.
func readReadme(t *testing.T) string {
	t.Helper()
	bytes, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
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
