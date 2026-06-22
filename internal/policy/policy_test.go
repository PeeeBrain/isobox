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

func TestDefaultNetworkPolicyIsDenyByDefault(t *testing.T) {
	defaulted := policy.DefaultNetworkPolicy()
	if defaulted.Default != policy.NetworkDefaultDeny {
		t.Fatalf("default network policy default = %q, want %q", defaulted.Default, policy.NetworkDefaultDeny)
	}
	if len(defaulted.Allow) != 0 {
		t.Fatalf("default network policy allow = %d rules, want 0", len(defaulted.Allow))
	}
}

func TestResolveNetworkPolicyAppliesDenyByDefault(t *testing.T) {
	resolved := policy.ResolveNetworkPolicy(policy.NetworkPolicy{})
	want := policy.DefaultNetworkPolicy()
	if resolved.Default != want.Default {
		t.Fatalf("resolved default = %q, want %q", resolved.Default, want.Default)
	}
	if len(resolved.Allow) != 0 {
		t.Fatalf("resolved allow = %d rules, want 0", len(resolved.Allow))
	}
}

func TestResolveNetworkPolicyPreservesExplicitDefault(t *testing.T) {
	resolved := policy.ResolveNetworkPolicy(policy.NetworkPolicy{Default: "allow"})
	if resolved.Default != "allow" {
		t.Fatalf("resolved default = %q, want allow", resolved.Default)
	}
}

func TestResolveNetworkPolicyPreservesAllowRules(t *testing.T) {
	rules := []policy.NetworkAllowRule{
		{Origin: "github.com"},
		{Origin: "api.example.com", PathPrefix: "/v1", Method: "GET"},
	}
	resolved := policy.ResolveNetworkPolicy(policy.NetworkPolicy{Allow: rules})

	if resolved.Default != policy.NetworkDefaultDeny {
		t.Fatalf("resolved default = %q, want %q", resolved.Default, policy.NetworkDefaultDeny)
	}
	if len(resolved.Allow) != len(rules) {
		t.Fatalf("resolved allow = %d rules, want %d", len(resolved.Allow), len(rules))
	}
	for i, rule := range rules {
		if resolved.Allow[i] != rule {
			t.Fatalf("resolved allow[%d] = %+v, want %+v", i, resolved.Allow[i], rule)
		}
	}
}

func TestNetworkAllowRuleShapeCapturesOriginPathPrefixAndMethod(t *testing.T) {
	rule := policy.NetworkAllowRule{Origin: "github.com", PathPrefix: "/api", Method: "POST"}
	if rule.Origin != "github.com" {
		t.Fatalf("origin = %q, want github.com", rule.Origin)
	}
	if rule.PathPrefix != "/api" {
		t.Fatalf("path_prefix = %q, want /api", rule.PathPrefix)
	}
	if rule.Method != "POST" {
		t.Fatalf("method = %q, want POST", rule.Method)
	}
}

func TestNetworkEnforcementLimitationStrings(t *testing.T) {
	ne := policy.NetworkEnforcement{
		RuntimeBackend: "host-process",
		Rules: []policy.NetworkEnforcementRule{
			{
				Aspect: "default_deny",
				Status: policy.NotEnforced,
				Detail: "the host backend does not enforce the deny-by-default network policy",
			},
		},
	}

	limitations := ne.LimitationStrings()
	if len(limitations) != 1 {
		t.Fatalf("limitations = %d, want 1", len(limitations))
	}
	if limitations[0] != "host-process: network policy 'default_deny' is not_enforced; the host backend does not enforce the deny-by-default network policy" {
		t.Fatalf("limitation = %q", limitations[0])
	}
}
