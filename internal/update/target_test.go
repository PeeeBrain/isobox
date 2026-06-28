package update_test

import (
	"errors"
	"strings"
	"testing"

	"isobox/internal/update"
)

// fixedLookup is a PathLookup that returns the configured active
// binary plus a fixed set of duplicates. It is used to exercise
// target resolution and managed-path eligibility without depending on
// the host filesystem.
type fixedLookup struct {
	Active     string
	Duplicates []string
	Err        error
}

func (f *fixedLookup) LookPath(name string) (string, error) {
	if f.Err != nil {
		return "", f.Err
	}
	return f.Active, nil
}

func (f *fixedLookup) IsoboxEntries() (string, []string, error) {
	if f.Err != nil {
		return "", nil, f.Err
	}
	return f.Active, append([]string(nil), f.Duplicates...), nil
}

func TestResolveUpdateTargetReturnsActiveEntry(t *testing.T) {
	lookup := &fixedLookup{Active: "/home/u/.local/bin/isobox"}

	target, err := update.ResolveUpdateTarget(lookup)
	if err != nil {
		t.Fatalf("ResolveUpdateTarget: %v", err)
	}
	if target.Path != "/home/u/.local/bin/isobox" {
		t.Errorf("target = %q, want /home/u/.local/bin/isobox", target.Path)
	}
	if target.IsManaged {
		t.Errorf("target.IsManaged = true, want false for a user-local bin directory")
	}
	if len(target.Duplicates) != 0 {
		t.Errorf("target.Duplicates = %v, want empty", target.Duplicates)
	}
}

func TestResolveUpdateTargetListsDuplicates(t *testing.T) {
	lookup := &fixedLookup{
		Active:     "/home/u/.local/bin/isobox",
		Duplicates: []string{"/usr/local/bin/isobox"},
	}

	target, err := update.ResolveUpdateTarget(lookup)
	if err != nil {
		t.Fatalf("ResolveUpdateTarget: %v", err)
	}
	if target.Path != "/home/u/.local/bin/isobox" {
		t.Errorf("target = %q, want active path", target.Path)
	}
	if len(target.Duplicates) != 1 || target.Duplicates[0] != "/usr/local/bin/isobox" {
		t.Errorf("target.Duplicates = %v, want [/usr/local/bin/isobox]", target.Duplicates)
	}
}

func TestResolveUpdateTargetErrorsWhenNoIsoboxOnPath(t *testing.T) {
	lookup := &fixedLookup{Err: errors.New("isobox not found")}

	_, err := update.ResolveUpdateTarget(lookup)
	if err == nil {
		t.Fatal("ResolveUpdateTarget did not error when isobox is not on PATH")
	}
}

func TestIsManagedPathDetectsPackageManagerLocations(t *testing.T) {
	cases := []struct {
		path    string
		managed bool
	}{
		{"/home/u/.local/bin/isobox", false},
		{"/usr/local/bin/isobox", false},
		{"/usr/bin/isobox", true},
		{"/opt/homebrew/bin/isobox", true},
		{"/snap/bin/isobox", true},
		{"/var/lib/dpkg/info/isobox", true},
		{"/var/lib/rpm/isobox", true},
		{"/var/lib/pacman/local/isobox", true},
		{"/nix/store/abc-isobox/bin/isobox", true},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			if got := update.IsManagedPath(c.path); got != c.managed {
				t.Errorf("IsManagedPath(%q) = %v, want %v", c.path, got, c.managed)
			}
		})
	}
}

func TestCheckManagedTargetRejectsClearlyPackageManagedPaths(t *testing.T) {
	cases := []string{
		"/usr/bin/isobox",
		"/opt/homebrew/bin/isobox",
		"/snap/bin/isobox",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			err := update.CheckManagedTarget(path)
			if err == nil {
				t.Errorf("CheckManagedTarget(%q) = nil, want refusal", path)
			}
			if err != nil && !strings.Contains(err.Error(), "package manager") && !strings.Contains(err.Error(), "system") {
				t.Errorf("CheckManagedTarget(%q) error does not mention package manager: %v", path, err)
			}
		})
	}
}

func TestCheckManagedTargetAllowsWritableManualTargets(t *testing.T) {
	cases := []string{
		"/home/u/.local/bin/isobox",
		"/usr/local/bin/isobox",
		"/opt/isobox/bin/isobox",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			if err := update.CheckManagedTarget(path); err != nil {
				t.Errorf("CheckManagedTarget(%q) = %v, want nil", path, err)
			}
		})
	}
}
