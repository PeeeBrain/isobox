// Package policy implements the Sandbox Policy and Effective Policy models.
//
// A Sandbox Policy describes the capabilities and limits requested for a Task.
// The Effective Policy records the resolved values actually used, including
// backend-specific enforcement limitations, so the Task Record never implies
// stronger isolation than the selected Runtime Backend provides.
package policy

// SandboxPolicy describes the capabilities and limits requested for a Task.
type SandboxPolicy struct {
	ResourceLimits ResourceLimits
	Network        NetworkPolicy
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
