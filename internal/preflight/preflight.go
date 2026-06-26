// Package preflight implements the preflight boundary for Tool-Call Sandbox
// creation.
//
// Every Cooperative Tool Call runs through Run before a Sandbox is created.
// A preflight failure is an isobox infrastructure error: it never creates a
// Sandbox, never runs the Workload Command, and never claims to have done
// either. The first Tool-Call milestone exposes a fixed set of named
// preflight checks; user-authored command matching is a later feature.
package preflight

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"isobox/internal/projectpolicy"
)

// Failure wraps a single preflight rejection with a short, user-facing
// reason. Callers should surface Reason as the preflight error message.
type Failure struct {
	// Reason is the human-readable reason the preflight rejected the
	// cooperative tool call. It never includes wrapped-command output.
	Reason string
}

func (f *Failure) Error() string { return f.Reason }

// ErrProjectPolicyMissing is returned when no .isobox/config.yaml is found
// inside the project Git repository. The Tool-Call Sandbox requires project
// policy so the first-milestone checks can run; the failure message tells
// the user how to create one.
var ErrProjectPolicyMissing = errors.New("project policy is required for cooperative tool calls")

// Run executes the full preflight sequence for a cooperative tool call
// rooted at the given directory. It returns the first failure encountered.
// Checks run in this order so the user always sees the first reason to fix:
//
//  1. project policy exists at the Git repository root
//  2. tool_call.enabled is true
//  3. first-milestone policy shape is honored (runtime_backend, path_mode,
//     workspace_source.kind, credentials.default, promotion.mode)
//  4. trusted repository has no tracked modifications or untracked
//     non-ignored files (no dirty-source override in the first milestone)
//  5. bubblewrap (bwrap) is on PATH
//
// Validating the declarative policy before the runtime environment lets
// policy-shape failures surface even on machines where bubblewrap is not
// yet installed.
func Run(dir string) error {
	policy, err := projectpolicy.Load(dir)
	if err != nil {
		return wrapMissingPolicy(err, dir)
	}

	if !policy.ToolCall.Enabled {
		return failuref("project policy disables tool-call (tool_call.enabled=false); set tool_call.enabled=true in %s to allow cooperative tool calls", projectPolicyPath(dir))
	}

	if err := assertFirstMilestonePolicyShape(policy); err != nil {
		return err
	}

	if err := assertCleanTrustedRepo(dir); err != nil {
		return failuref("%s; the first Tool-Call milestone has no dirty-source override, commit or stash the changes before invoking isobox tool", err.Error())
	}

	if err := assertBubblewrapAvailable(); err != nil {
		return err
	}

	return nil
}

// wrapMissingPolicy converts a projectpolicy.Load error into a preflight
// failure that points the user at `isobox init`. The original error is
// preserved as a prefix so the user sees both the underlying reason and
// the recovery command.
func wrapMissingPolicy(err error, dir string) error {
	configPath := filepath.Join(dir, ".isobox", "config.yaml")
	if root, gerr := gitTopLevel(dir); gerr == nil {
		configPath = filepath.Join(root, ".isobox", "config.yaml")
	}
	return failuref("no project policy at %s; run `isobox init` to create one (cause: %s)", configPath, err.Error())
}

func projectPolicyPath(dir string) string {
	if root, err := gitTopLevel(dir); err == nil {
		return filepath.Join(root, ".isobox", "config.yaml")
	}
	return filepath.Join(dir, ".isobox", "config.yaml")
}

func failuref(format string, args ...any) *Failure {
	return &Failure{Reason: "isobox tool preflight: " + fmt.Sprintf(format, args...)}
}

func assertCleanTrustedRepo(dir string) error {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("inspect trusted repository: %w", err)
	}
	entries := dirtyEntries(string(output))
	if len(entries) == 0 {
		return nil
	}
	return fmt.Errorf("trusted repository has uncommitted changes: %s", strings.Join(entries, ", "))
}

func assertBubblewrapAvailable() error {
	if _, err := exec.LookPath("bwrap"); err != nil {
		return failuref("bubblewrap (bwrap) is not on PATH; install bubblewrap or add it to PATH before invoking isobox tool")
	}
	return nil
}

func assertFirstMilestonePolicyShape(policy projectpolicy.ProjectPolicy) error {
	checks := []struct {
		field     string
		got       string
		want      string
		suggested string
	}{
		{"runtime_backend", policy.RuntimeBackend, projectpolicy.RuntimeBackendBubblewrap, "set runtime_backend: bubblewrap in .isobox/config.yaml"},
		{"development_environment.path_mode", policy.DevelopmentEnv.PathMode, projectpolicy.PathModeBackendDefault, "set development_environment.path_mode: backend_default in .isobox/config.yaml"},
		{"workspace_source.kind", policy.WorkspaceSource.Kind, projectpolicy.WorkspaceSourceProjectRoot, "set workspace_source.kind: project_root in .isobox/config.yaml"},
		{"credentials.default", policy.Credentials.Default, projectpolicy.CredentialsDefaultDeny, "set credentials.default: deny in .isobox/config.yaml"},
		{"promotion.mode", policy.Promotion.Mode, projectpolicy.PromotionModeManual, "set promotion.mode: manual in .isobox/config.yaml"},
	}
	for _, c := range checks {
		if c.got != c.want {
			return failuref("project policy %s=%q is not supported in the first Tool-Call milestone (only %s is supported); %s", c.field, c.got, c.want, c.suggested)
		}
	}
	return nil
}

func dirtyEntries(porcelain string) []string {
	var out []string
	for _, line := range strings.Split(porcelain, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(line))
	}
	return out
}

func gitTopLevel(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", errors.New("not inside a Git repository")
	}
	return root, nil
}
