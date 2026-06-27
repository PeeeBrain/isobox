// Package policy implements the Sandbox Policy and Effective Policy models.
//
// A Sandbox Policy describes the capabilities and limits requested for a Task.
// The Effective Policy records the resolved values actually used, including
// backend-specific enforcement limitations, so the Task Record never implies
// stronger isolation than the selected Runtime Backend provides.
package policy

import "fmt"

// SandboxPolicy describes the capabilities and limits requested for a Task.
type SandboxPolicy struct {
	ResourceLimits ResourceLimits
	Network        NetworkPolicy
	Credentials    CredentialPolicy
	// ReuseInputs are the explicit host assets exposed to a Sandbox for Host
	// Agent Reuse. A nil or empty slice means no host assets are exposed; the
	// resolver never silently invents broad host inheritance.
	ReuseInputs []ReuseInput
}

// CredentialPolicy captures credential-access intent for a Sandbox Policy.
//
// The first milestone supports deny-only credential access. A zero value is
// resolved to deny so callers do not accidentally inherit ambient credentials.
type CredentialPolicy struct {
	Default string `json:"default,omitempty"`
}

// CredentialDefaultDeny is the resolved credential action for the first
// milestone: no credential material is exposed to the Sandbox.
const CredentialDefaultDeny = "deny"

// DefaultCredentialPolicy returns the resolved default credential policy.
func DefaultCredentialPolicy() CredentialPolicy {
	return CredentialPolicy{Default: CredentialDefaultDeny}
}

// ResolveCredentialPolicy merges requested credential policy with resolved
// defaults. The first milestone only supplies deny, but this keeps the public
// policy resolver shape aligned with other policy categories.
func ResolveCredentialPolicy(requested CredentialPolicy) CredentialPolicy {
	resolved := DefaultCredentialPolicy()
	if requested.Default != "" {
		resolved.Default = requested.Default
	}
	return resolved
}

// ResourceLimits captures resource-limit intent for a Sandbox Policy.
//
// A zero value means no explicit limit is requested for that category, which
// the default resolver interprets as the resolved default (typically no limit
// for the host-process backend in this milestone).
type ResourceLimits struct {
	MaxDurationSeconds int64 `json:"max_duration_seconds,omitempty"`
	MaxOutputBytes     int64 `json:"max_output_bytes,omitempty"`
	MaxCPUCores        int64 `json:"max_cpu_cores,omitempty"`
	MaxMemoryBytes     int64 `json:"max_memory_bytes,omitempty"`
	MaxProcesses       int64 `json:"max_processes,omitempty"`
	MaxDiskBytes       int64 `json:"max_disk_bytes,omitempty"`
	MaxFileDescriptors int64 `json:"max_file_descriptors,omitempty"`
}

// ReuseInputKind identifies the category of a host asset exposed for Host
// Agent Reuse.
type ReuseInputKind string

const (
	// ReuseInputHostBinary is a host-installed executable exposed to a Sandbox.
	ReuseInputHostBinary ReuseInputKind = "host_binary"
	// ReuseInputPath is a host filesystem path exposed to a Sandbox.
	ReuseInputPath ReuseInputKind = "path"
	// ReuseInputEnvVar is a host environment variable exposed to a Sandbox.
	ReuseInputEnvVar ReuseInputKind = "env_var"
	// ReuseInputCredentialRef is a reference to a host credential exposed to a
	// Sandbox. The reference itself is recorded, never the secret material.
	ReuseInputCredentialRef ReuseInputKind = "credential_ref"
	// ReuseInputLocalIntegration is a named local integration (for example an
	// MCP server or skill) exposed to a Sandbox.
	ReuseInputLocalIntegration ReuseInputKind = "local_integration"
)

var supportedReuseInputKinds = map[ReuseInputKind]struct{}{
	ReuseInputHostBinary:       {},
	ReuseInputPath:             {},
	ReuseInputEnvVar:           {},
	ReuseInputCredentialRef:    {},
	ReuseInputLocalIntegration: {},
}

// ReuseInput is a single host asset explicitly exposed to a Sandbox for Host
// Agent Reuse. Reuse Inputs are always explicit; isobox never silently
// inherits broad host state.
type ReuseInput struct {
	Kind  ReuseInputKind `json:"kind"`
	Value string         `json:"value"`
}

// ValidateReuseInputKind reports whether kind is a supported Reuse Input kind.
func ValidateReuseInputKind(kind string) error {
	if _, ok := supportedReuseInputKinds[ReuseInputKind(kind)]; !ok {
		return fmt.Errorf("unsupported reuse input kind %q", kind)
	}
	return nil
}

// ResolveReuseInputs validates the requested Reuse Inputs and returns the
// values recorded in the Effective Policy. Absence of requested Reuse Inputs
// resolves to an empty slice; the resolver never silently broadens host
// inheritance. Validation rejects unknown kinds or empty values so a Task
// Record can never carry an ambiguous Reuse Input.
func ResolveReuseInputs(requested []ReuseInput) ([]ReuseInput, error) {
	resolved := make([]ReuseInput, 0, len(requested))
	for i, input := range requested {
		if _, ok := supportedReuseInputKinds[input.Kind]; !ok {
			return nil, fmt.Errorf("reuse input %d has unsupported kind %q", i, input.Kind)
		}
		if input.Value == "" {
			return nil, fmt.Errorf("reuse input %d (%s) has empty value", i, input.Kind)
		}
		resolved = append(resolved, input)
	}
	return resolved, nil
}

// ReuseInputsLimitation returns a human-readable statement suitable for the
// Effective Policy limitations list when Reuse Inputs are configured. It makes
// Host Agent Reuse exposure visible in the Task Record and records the lowered
// isolation assurance that comes with reusing host assets.
func ReuseInputsLimitation(inputs []ReuseInput) string {
	return fmt.Sprintf("host-agent-reuse: Sandbox exposes %d explicit Reuse Input(s); Host Agent Reuse lowers isolation assurance compared with a more isolated Development Environment", len(inputs))
}

// DefaultResourceLimits returns the resolved default resource limits when a
// Sandbox Policy does not request explicit values.
func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{}
}

// ResolveResourceLimits merges requested resource limits with resolved defaults.
// Explicitly requested non-zero values override defaults.
func ResolveResourceLimits(requested ResourceLimits) ResourceLimits {
	resolved := DefaultResourceLimits()
	if requested.MaxDurationSeconds != 0 {
		resolved.MaxDurationSeconds = requested.MaxDurationSeconds
	}
	if requested.MaxOutputBytes != 0 {
		resolved.MaxOutputBytes = requested.MaxOutputBytes
	}
	if requested.MaxCPUCores != 0 {
		resolved.MaxCPUCores = requested.MaxCPUCores
	}
	if requested.MaxMemoryBytes != 0 {
		resolved.MaxMemoryBytes = requested.MaxMemoryBytes
	}
	if requested.MaxProcesses != 0 {
		resolved.MaxProcesses = requested.MaxProcesses
	}
	if requested.MaxDiskBytes != 0 {
		resolved.MaxDiskBytes = requested.MaxDiskBytes
	}
	if requested.MaxFileDescriptors != 0 {
		resolved.MaxFileDescriptors = requested.MaxFileDescriptors
	}
	return resolved
}

// EnforcementStatus describes how completely a Runtime Backend enforces a
// resource limit.
type EnforcementStatus string

const (
	Enforced          EnforcementStatus = "enforced"
	PartiallyEnforced                   = "partially_enforced"
	NotEnforced                         = "not_enforced"
)

// ResourceLimitEnforcement records the enforcement status for a single resource
// limit category.
type ResourceLimitEnforcement struct {
	Name   string            `json:"name"`
	Status EnforcementStatus `json:"status"`
	Detail string            `json:"detail,omitempty"`
}

// ResourceEnforcement records how a Runtime Backend enforces resource limits.
type ResourceEnforcement struct {
	RuntimeBackend string                     `json:"runtime_backend"`
	Limits         []ResourceLimitEnforcement `json:"limits"`
}

// LimitationStrings returns human-readable statements suitable for inclusion in
// an Effective Policy limitations list.
func (re ResourceEnforcement) LimitationStrings() []string {
	var out []string
	for _, l := range re.Limits {
		out = append(out, re.RuntimeBackend+": resource limit '"+l.Name+"' is "+string(l.Status)+"; "+l.Detail)
	}
	return out
}

// NetworkPolicy captures network-access intent for a Sandbox Policy.
//
// The default intent is deny-by-default: a Sandbox is not permitted to reach
// any network origin unless an explicit allow rule permits it. A zero value is
// resolved to the default-deny policy by the default resolver, so callers do
// not need to set Default explicitly to request deny-by-default behavior.
//
// This slice establishes the policy shape for future network enforcement; the
// host Runtime Backend does not enforce network limits in this milestone, and
// that limitation is recorded honestly in the Effective Policy.
type NetworkPolicy struct {
	Default string             `json:"default,omitempty"`
	Allow   []NetworkAllowRule `json:"allow,omitempty"`
}

// NetworkAllowRule describes a future network allow rule.
//
// Rules are matched by origin (host or origin identifier), URL path prefix,
// and request method. A zero-valued field in a rule matches any value for that
// dimension, so an allow rule with only an origin permits any path and method
// on that origin. Enforcement of these rules is a future capability.
type NetworkAllowRule struct {
	Origin     string `json:"origin,omitempty"`
	PathPrefix string `json:"path_prefix,omitempty"`
	Method     string `json:"method,omitempty"`
}

// NetworkDefaultDeny is the resolved default action when no allow rule matches.
const NetworkDefaultDeny = "deny"

// DefaultNetworkPolicy returns the resolved default network policy when a
// Sandbox Policy does not request explicit values. The default is deny-by-
// default with no allow rules.
func DefaultNetworkPolicy() NetworkPolicy {
	return NetworkPolicy{Default: NetworkDefaultDeny}
}

// ResolveNetworkPolicy merges requested network policy with resolved defaults.
// An empty Default is resolved to deny-by-default. Explicit allow rules are
// preserved unchanged.
func ResolveNetworkPolicy(requested NetworkPolicy) NetworkPolicy {
	resolved := DefaultNetworkPolicy()
	if requested.Default != "" {
		resolved.Default = requested.Default
	}
	resolved.Allow = append(resolved.Allow, requested.Allow...)
	return resolved
}

// NetworkEnforcementRule records the enforcement status for one aspect of the
// network policy.
type NetworkEnforcementRule struct {
	Aspect string            `json:"aspect"`
	Status EnforcementStatus `json:"status"`
	Detail string            `json:"detail,omitempty"`
}

// NetworkEnforcement records how a Runtime Backend enforces the network
// policy. This is recorded in the Effective Policy alongside the resolved
// network policy so the Task Record never implies stronger network isolation
// than the backend provides.
type NetworkEnforcement struct {
	RuntimeBackend string                   `json:"runtime_backend"`
	Rules          []NetworkEnforcementRule `json:"rules"`
}

// LimitationStrings returns human-readable statements suitable for inclusion in
// an Effective Policy limitations list.
func (ne NetworkEnforcement) LimitationStrings() []string {
	var out []string
	for _, r := range ne.Rules {
		out = append(out, ne.RuntimeBackend+": network policy '"+r.Aspect+"' is "+string(r.Status)+"; "+r.Detail)
	}
	return out
}

// CredentialEnforcementRule records the enforcement status for one aspect of
// the credential policy.
type CredentialEnforcementRule struct {
	Aspect string            `json:"aspect"`
	Status EnforcementStatus `json:"status"`
	Detail string            `json:"detail,omitempty"`
}

// CredentialEnforcement records how a Runtime Backend enforces credential
// exposure policy. This is recorded alongside the resolved credential policy
// so the Task Record can say what was intended and what actually happened.
type CredentialEnforcement struct {
	RuntimeBackend string                      `json:"runtime_backend"`
	Rules          []CredentialEnforcementRule `json:"rules"`
}

// LimitationStrings returns human-readable statements suitable for inclusion in
// an Effective Policy limitations list.
func (ce CredentialEnforcement) LimitationStrings() []string {
	var out []string
	for _, r := range ce.Rules {
		out = append(out, ce.RuntimeBackend+": credential policy '"+r.Aspect+"' is "+string(r.Status)+"; "+r.Detail)
	}
	return out
}
