// Package projectpolicy implements the project-owned Tool-Call Sandbox policy.
//
// A project policy lives at .isobox/config.yaml inside a Git repository and
// declares how cooperative tool calls (isobox tool) are governed for that
// project. The init command generates a restrictive default with sparse
// comments that explain the security consequences of loosening each section.
package projectpolicy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// APIVersion is the api_version recorded in generated project policy files.
const APIVersion = "isobox.dev/v1alpha1"

// Kind is the value of the kind field recorded in generated project policy files.
const Kind = "ProjectPolicy"

// ProjectPolicy is the project-owned Tool-Call Sandbox policy.
type ProjectPolicy struct {
	APIVersion      string                `yaml:"api_version"`
	Kind            string                `yaml:"kind"`
	ToolCall        ToolCallConfig        `yaml:"tool_call"`
	RuntimeBackend  string                `yaml:"runtime_backend"`
	DevelopmentEnv  DevelopmentEnvConfig  `yaml:"development_environment"`
	WorkspaceSource WorkspaceSourceConfig `yaml:"workspace_source"`
	Network         NetworkConfig         `yaml:"network"`
	Filesystem      FilesystemConfig      `yaml:"filesystem"`
	Credentials     CredentialsConfig     `yaml:"credentials"`
	Preflight       PreflightConfig       `yaml:"preflight"`
	Promotion       PromotionConfig       `yaml:"promotion"`
}

// ToolCallConfig controls whether isobox tool is enabled for the project.
type ToolCallConfig struct {
	// Enabled reports whether cooperative tool calls are accepted for the
	// project. When false, isobox tool rejects the call even if the same
	// caller is the cooperative agent.
	Enabled bool `yaml:"enabled"`
}

// DevelopmentEnvConfig describes the Development Environment available inside
// the Sandbox for cooperative tool calls.
type DevelopmentEnvConfig struct {
	// PathMode controls how PATH is shaped inside the Sandbox. The first
	// milestone supports only `backend_default`, which exposes the runtime
	// backend's default PATH and no other ambient environment state.
	PathMode string `yaml:"path_mode"`
}

// WorkspaceSourceConfig describes where the Workspace Source for cooperative
// tool calls comes from.
type WorkspaceSourceConfig struct {
	// Kind identifies the Workspace Source kind. The first milestone
	// supports only `project_root`, which uses the Git repository root
	// containing this policy file as the trusted source.
	Kind string `yaml:"kind"`
}

// NetworkConfig captures network-access intent for cooperative tool calls.
type NetworkConfig struct {
	// Default is the resolved network access mode. The first milestone
	// supports `deny` (deny-by-default, no allow rules) and `inherited`
	// (use the runtime backend's ordinary network access). Host/domain
	// allowlists are not supported.
	Default string               `yaml:"default"`
	Allow   []NetworkAllowConfig `yaml:"allow"`
}

// NetworkAllowConfig captures unsupported first-milestone allowlist entries
// so preflight can reject them with a truthful policy error.
type NetworkAllowConfig struct {
	Host   string `yaml:"host"`
	Domain string `yaml:"domain"`
}

// FilesystemConfig describes the Filesystem Policy for cooperative tool calls.
type FilesystemConfig struct {
	// ExposeWorkspace reports that the sandbox sees the private Repository
	// Workspace. The first milestone exposes only the Workspace; trusted
	// repository and host home are not exposed by the bubblewrap backend.
	ExposeWorkspace bool `yaml:"expose_workspace"`
}

// CredentialsConfig captures credential-access intent for cooperative tool calls.
type CredentialsConfig struct {
	// Default is the resolved credential access mode. The first milestone
	// supports only `deny`; credential brokering is a later explicit
	// feature and is never silently enabled.
	Default string `yaml:"default"`
}

// PreflightConfig describes the named Preflight Rules evaluated before a
// cooperative tool call enters the Sandbox.
type PreflightConfig struct {
	// Rules lists the named built-in preflight checks to apply. The first
	// milestone supports only built-in named checks; user-authored command
	// matching is out of scope.
	Rules []string `yaml:"rules"`
}

// PromotionConfig describes how Promotion is approved for the project.
type PromotionConfig struct {
	// Mode reports how Promotion is gated. The first milestone supports
	// only `manual` (human-confirmed, with `--yes` for explicit non-
	// interactive use after fresh human approval).
	Mode string `yaml:"mode"`
}

// Path mode values for DevelopmentEnvConfig.
const (
	PathModeBackendDefault = "backend_default"
)

// WorkspaceSource kinds.
const (
	WorkspaceSourceProjectRoot = "project_root"
)

// Network default values.
const (
	NetworkDefaultDeny      = "deny"
	NetworkDefaultInherited = "inherited"
)

// Credentials default values.
const (
	CredentialsDefaultDeny = "deny"
)

// Promotion modes.
const (
	PromotionModeManual = "manual"
)

// Preflight rule names. The first milestone supports named built-in checks
// rather than user-authored command matching.
const (
	PreflightRejectDirtyWorkspaceSource = "reject_dirty_workspace_source"
)

// Runtime backend names supported by the first-milestone Tool-Call Sandbox.
const (
	RuntimeBackendBubblewrap = "bubblewrap"
)

// Default returns the restrictive default project policy generated by
// `isobox init`. Each section is intentionally closed by default; the user
// must explicitly loosen the policy to allow broader access.
func Default() ProjectPolicy {
	return ProjectPolicy{
		APIVersion: APIVersion,
		Kind:       Kind,
		ToolCall: ToolCallConfig{
			Enabled: true,
		},
		RuntimeBackend: RuntimeBackendBubblewrap,
		DevelopmentEnv: DevelopmentEnvConfig{
			PathMode: PathModeBackendDefault,
		},
		WorkspaceSource: WorkspaceSourceConfig{
			Kind: WorkspaceSourceProjectRoot,
		},
		Network: NetworkConfig{
			Default: NetworkDefaultDeny,
		},
		Filesystem: FilesystemConfig{
			ExposeWorkspace: true,
		},
		Credentials: CredentialsConfig{
			Default: CredentialsDefaultDeny,
		},
		Preflight: PreflightConfig{
			Rules: []string{PreflightRejectDirtyWorkspaceSource},
		},
		Promotion: PromotionConfig{
			Mode: PromotionModeManual,
		},
	}
}

// Render returns the project policy as a YAML document with sparse,
// consequence-focused comments. The comments explain what changes when the
// user loosens each section, so the policy file is understandable without
// reading external documentation.
func (p ProjectPolicy) Render() (string, error) {
	body, err := yaml.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal project policy: %w", err)
	}
	return commentHeader + annotateSections(string(body)), nil
}

// annotateSections walks a rendered YAML document and inserts sparse,
// consequence-focused comments above the major sections. Comments are placed
// immediately above the first key of each section and explain what changes
// when the user loosens the defaults.
func annotateSections(body string) string {
	// yaml.v3 renders nested sections with 4-space indentation. We key off
	// the top-level keys and insert a single comment line above each.
	annotations := map[string]string{
		"tool_call:":               "# tool_call.enabled=false rejects cooperative tool calls for this project.",
		"runtime_backend:":         "# runtime_backend=bubblewrap is required for the first Tool-Call milestone; host_process is not enough.",
		"development_environment:": "# path_mode=backend_default exposes only the backend's default PATH; no ambient environment is inherited.",
		"workspace_source:":        "# workspace_source.kind=project_root uses this policy's Git repository root as the trusted source.",
		"network:":                 "# network.default=deny blocks outbound access; switching to inherited re-exposes the host network.",
		"filesystem:":              "# filesystem.expose_workspace=true is the only mount; trusted repo and host home stay outside the sandbox.",
		"credentials:":             "# credentials.default=deny refuses secret exposure; only loosening to a scoped credential mode can grant access.",
		"preflight:":               "# preflight.rules gates risky actions before sandboxing; removing reject_dirty_workspace_source re-enables dirty source snapshots.",
		"promotion:":               "# promotion.mode=manual requires human review; switching away from manual weakens the review gate.",
	}

	lines := strings.Split(body, "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if note, ok := annotations[trimmed]; ok && !strings.HasPrefix(trimmed, "-") {
			out = append(out, note)
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

const commentHeader = `# isobox project policy.
# Generated by ` + "`isobox init`" + `. Edit by hand to loosen the defaults; isobox
# will not modify this file again. The Tool-Call Sandbox uses the Git
# repository root that contains this file as the trusted Workspace Source.
`

// Parse decodes a project policy from YAML.
func Parse(data []byte) (ProjectPolicy, error) {
	var p ProjectPolicy
	if len(data) == 0 {
		return p, errors.New("project policy is empty")
	}
	if err := yaml.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("parse project policy: %w", err)
	}
	return p, nil
}

// Load reads the project policy for the given directory. The first milestone
// walks upward from start until it finds a `.isobox/config.yaml` at a Git
// repository root; a non-Git location or a `.isobox` directory that is not at
// the repository root is rejected so the discovery boundary matches
// Promotion's workspace semantics.
func Load(start string) (ProjectPolicy, error) {
	var p ProjectPolicy

	abs, err := filepath.Abs(start)
	if err != nil {
		return p, fmt.Errorf("resolve %s: %w", start, err)
	}

	repoRoot, err := gitTopLevel(abs)
	if err != nil {
		return p, err
	}

	configPath := filepath.Join(repoRoot, ".isobox", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return p, fmt.Errorf("no project policy at %s; run `isobox init` to create one", configPath)
		}
		return p, fmt.Errorf("inspect %s: %w", configPath, err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return p, fmt.Errorf("read %s: %w", configPath, err)
	}

	policy, err := Parse(data)
	if err != nil {
		return p, fmt.Errorf("parse %s: %w", configPath, err)
	}
	return policy, nil
}

// gitTopLevel reports the absolute path of the Git repository containing the
// given directory, or an error if the directory is not inside a working tree.
func gitTopLevel(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s is not inside a Git repository: %w", dir, err)
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", fmt.Errorf("%s is not inside a Git repository", dir)
	}
	return root, nil
}
