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
