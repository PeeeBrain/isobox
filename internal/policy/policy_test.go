package policy_test

import (
	"testing"

	"isobox/internal/policy"
)

func TestDefaultResourceLimitsAreZeroValued(t *testing.T) {
	defaults := policy.DefaultResourceLimits()
	if defaults != (policy.ResourceLimits{}) {
		t.Fatalf("default resource limits = %+v, want zero value", defaults)
	}
}

func TestResolveResourceLimitsAppliesDefaults(t *testing.T) {
	resolved := policy.ResolveResourceLimits(policy.ResourceLimits{})
	want := policy.DefaultResourceLimits()
	if resolved != want {
		t.Fatalf("resolved resource limits = %+v, want %+v", resolved, want)
	}
}

func TestResolveResourceLimitsPreservesExplicitValues(t *testing.T) {
	requested := policy.ResourceLimits{
		MaxDurationSeconds: 120,
		MaxOutputBytes:     4096,
		MaxCPUCores:        2,
		MaxMemoryBytes:     1024 * 1024 * 1024,
		MaxProcesses:       64,
		MaxDiskBytes:       1024 * 1024,
		MaxFileDescriptors: 256,
	}

	resolved := policy.ResolveResourceLimits(requested)

	if resolved != requested {
		t.Fatalf("resolved resource limits = %+v, want %+v", resolved, requested)
	}
}

func TestResolveResourceLimitsMergesExplicitValuesWithDefaults(t *testing.T) {
	requested := policy.ResourceLimits{
		MaxDurationSeconds: 60,
		MaxMemoryBytes:     512 * 1024 * 1024,
	}

	resolved := policy.ResolveResourceLimits(requested)

	if resolved.MaxDurationSeconds != 60 {
		t.Fatalf("max_duration_seconds = %d, want 60", resolved.MaxDurationSeconds)
	}
	if resolved.MaxMemoryBytes != 512*1024*1024 {
		t.Fatalf("max_memory_bytes = %d, want %d", resolved.MaxMemoryBytes, 512*1024*1024)
	}
	if resolved.MaxOutputBytes != 0 {
		t.Fatalf("max_output_bytes = %d, want 0", resolved.MaxOutputBytes)
	}
	if resolved.MaxCPUCores != 0 {
		t.Fatalf("max_cpu_cores = %d, want 0", resolved.MaxCPUCores)
	}
	if resolved.MaxProcesses != 0 {
		t.Fatalf("max_processes = %d, want 0", resolved.MaxProcesses)
	}
	if resolved.MaxDiskBytes != 0 {
		t.Fatalf("max_disk_bytes = %d, want 0", resolved.MaxDiskBytes)
	}
	if resolved.MaxFileDescriptors != 0 {
		t.Fatalf("max_file_descriptors = %d, want 0", resolved.MaxFileDescriptors)
	}
}

func TestResourceEnforcementLimitationStrings(t *testing.T) {
	re := policy.ResourceEnforcement{
		RuntimeBackend: "host-process",
		Limits: []policy.ResourceLimitEnforcement{
			{
				Name:   "time",
				Status: policy.NotEnforced,
				Detail: "host backend does not enforce time limits",
			},
		},
	}

	limitations := re.LimitationStrings()
	if len(limitations) != 1 {
		t.Fatalf("limitations = %d, want 1", len(limitations))
	}
	if limitations[0] != "host-process: resource limit 'time' is not_enforced; host backend does not enforce time limits" {
		t.Fatalf("limitation = %q", limitations[0])
	}
}
