package policy_test

import (
	"strings"
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

func TestDefaultCredentialPolicyDeniesCredentialExposure(t *testing.T) {
	defaulted := policy.DefaultCredentialPolicy()
	if defaulted.Default != policy.CredentialDefaultDeny {
		t.Fatalf("default credential policy default = %q, want %q", defaulted.Default, policy.CredentialDefaultDeny)
	}
}

func TestResolveCredentialPolicyAppliesDenyByDefault(t *testing.T) {
	resolved := policy.ResolveCredentialPolicy(policy.CredentialPolicy{})
	if resolved.Default != policy.CredentialDefaultDeny {
		t.Fatalf("resolved credential default = %q, want %q", resolved.Default, policy.CredentialDefaultDeny)
	}
}

func TestCredentialEnforcementLimitationStrings(t *testing.T) {
	ce := policy.CredentialEnforcement{
		RuntimeBackend: "bubblewrap",
		Rules: []policy.CredentialEnforcementRule{
			{
				Aspect: "default_deny",
				Status: policy.Enforced,
				Detail: "no credential material was exposed",
			},
		},
	}

	limitations := ce.LimitationStrings()
	if len(limitations) != 1 {
		t.Fatalf("limitations = %d, want 1", len(limitations))
	}
	if limitations[0] != "bubblewrap: credential policy 'default_deny' is enforced; no credential material was exposed" {
		t.Fatalf("limitation = %q", limitations[0])
	}
}

func TestValidateReuseInputKindAcceptsSupportedKinds(t *testing.T) {
	kinds := []string{
		string(policy.ReuseInputHostBinary),
		string(policy.ReuseInputPath),
		string(policy.ReuseInputEnvVar),
		string(policy.ReuseInputCredentialRef),
		string(policy.ReuseInputLocalIntegration),
	}
	for _, kind := range kinds {
		if err := policy.ValidateReuseInputKind(kind); err != nil {
			t.Fatalf("validate reuse input kind %q: %v", kind, err)
		}
	}
}

func TestValidateReuseInputKindRejectsUnknownKind(t *testing.T) {
	if err := policy.ValidateReuseInputKind("home_directory"); err == nil {
		t.Fatal("validate reuse input kind accepted broad implicit kind home_directory")
	}
	if err := policy.ValidateReuseInputKind(""); err == nil {
		t.Fatal("validate reuse input kind accepted empty kind")
	}
}

func TestResolveReuseInputsEmptyRecordsNoInheritance(t *testing.T) {
	resolved, err := policy.ResolveReuseInputs(nil)
	if err != nil {
		t.Fatalf("resolve nil reuse inputs failed: %v", err)
	}
	if len(resolved) != 0 {
		t.Fatalf("resolved nil reuse inputs = %+v, want empty (no silent host inheritance)", resolved)
	}

	resolved, err = policy.ResolveReuseInputs([]policy.ReuseInput{})
	if err != nil {
		t.Fatalf("resolve empty reuse inputs failed: %v", err)
	}
	if len(resolved) != 0 {
		t.Fatalf("resolved empty reuse inputs = %+v, want empty", resolved)
	}
}

func TestResolveReuseInputsPreservesExplicitDeclarations(t *testing.T) {
	requested := []policy.ReuseInput{
		{Kind: policy.ReuseInputHostBinary, Value: "/usr/local/bin/codex"},
		{Kind: policy.ReuseInputPath, Value: "/home/user/.codex"},
		{Kind: policy.ReuseInputEnvVar, Value: "ANTHROPIC_API_KEY"},
		{Kind: policy.ReuseInputCredentialRef, Value: "keychain://anthropic"},
		{Kind: policy.ReuseInputLocalIntegration, Value: "filesystem-mcp"},
	}

	resolved, err := policy.ResolveReuseInputs(requested)
	if err != nil {
		t.Fatalf("resolve reuse inputs failed: %v", err)
	}
	if len(resolved) != len(requested) {
		t.Fatalf("resolved reuse inputs = %d, want %d", len(resolved), len(requested))
	}
	for i, input := range requested {
		if resolved[i] != input {
			t.Fatalf("resolved[%d] = %+v, want %+v", i, resolved[i], input)
		}
	}
}

func TestResolveReuseInputsRejectsUnknownKind(t *testing.T) {
	_, err := policy.ResolveReuseInputs([]policy.ReuseInput{
		{Kind: policy.ReuseInputKind("home_directory"), Value: "/home/user"},
	})
	if err == nil {
		t.Fatal("resolve reuse inputs accepted an unsupported broad kind")
	}
}

func TestResolveReuseInputsRejectsEmptyValue(t *testing.T) {
	_, err := policy.ResolveReuseInputs([]policy.ReuseInput{
		{Kind: policy.ReuseInputHostBinary, Value: ""},
	})
	if err == nil {
		t.Fatal("resolve reuse inputs accepted an empty value")
	}
}

func TestReuseInputsLimitationMentionsExposureAndLoweredAssurance(t *testing.T) {
	inputs := []policy.ReuseInput{
		{Kind: policy.ReuseInputHostBinary, Value: "/usr/local/bin/codex"},
		{Kind: policy.ReuseInputPath, Value: "/home/user/.codex"},
	}
	limitation := policy.ReuseInputsLimitation(inputs)
	if !strings.Contains(limitation, "host-agent-reuse") {
		t.Fatalf("limitation does not mention host-agent-reuse: %q", limitation)
	}
	if !strings.Contains(limitation, "2 explicit Reuse Input") {
		t.Fatalf("limitation does not mention reuse input count: %q", limitation)
	}
	if !strings.Contains(limitation, "lowers isolation assurance") {
		t.Fatalf("limitation does not document lowered assurance: %q", limitation)
	}
}
