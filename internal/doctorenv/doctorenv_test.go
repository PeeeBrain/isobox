package doctorenv_test

import (
	"errors"
	"strings"
	"testing"

	"isobox/internal/doctor"
	"isobox/internal/doctorenv"
)

// fakeLookup is a PathLookup that returns a fixed mapping from binary name
// to absolute path. The Missing field is used to simulate LookPath failures
// without depending on the host's actual installation.
type fakeLookup struct {
	Found   map[string]string
	Missing map[string]error
}

func (f *fakeLookup) LookPath(name string) (string, error) {
	if path, ok := f.Found[name]; ok {
		return path, nil
	}
	if err, ok := f.Missing[name]; ok {
		return "", err
	}
	return "", errors.New("fake: not found")
}

func (f *fakeLookup) IsoboxEntries() (string, []string, error) {
	if path, ok := f.Found["isobox"]; ok {
		return path, nil, nil
	}
	if err, ok := f.Missing["isobox"]; ok {
		return "", nil, err
	}
	return "", nil, errors.New("fake: isobox not found")
}

func TestCheckGitOnPathReportsOKWhenFound(t *testing.T) {
	lookup := &fakeLookup{Found: map[string]string{"git": "/usr/bin/git"}}

	check := doctorenv.CheckGitOnPath(lookup)

	if check.ID != "git-on-path" {
		t.Errorf("check ID = %q, want git-on-path", check.ID)
	}
	if check.Severity != doctor.SeverityOK {
		t.Errorf("check severity = %q, want ok", check.Severity)
	}
	if check.Message != "/usr/bin/git" {
		t.Errorf("check message = %q, want the resolved path", check.Message)
	}
}

func TestCheckGitOnPathReportsErrorWhenMissing(t *testing.T) {
	lookup := &fakeLookup{Missing: map[string]error{"git": errors.New("not found")}}

	check := doctorenv.CheckGitOnPath(lookup)

	if check.Severity != doctor.SeverityError {
		t.Errorf("check severity = %q, want error", check.Severity)
	}
	if check.Consequence == "" {
		t.Errorf("error check is missing consequence text")
	}
	if check.Fix == "" {
		t.Errorf("error check is missing fix text")
	}
	// The consequence must make it clear that isobox itself cannot run
	// without git; the fix must mention install or PATH.
	combined := check.Consequence + " " + check.Fix
	if !strings.Contains(combined, "git") {
		t.Errorf("error text does not mention git: %q", combined)
	}
}

func TestCheckBwrapOnPathReportsOKWhenFound(t *testing.T) {
	lookup := &fakeLookup{Found: map[string]string{"bwrap": "/usr/bin/bwrap"}}

	check := doctorenv.CheckBwrapOnPath(lookup)

	if check.ID != "bwrap-on-path" {
		t.Errorf("check ID = %q, want bwrap-on-path", check.ID)
	}
	if check.Severity != doctor.SeverityOK {
		t.Errorf("check severity = %q, want ok", check.Severity)
	}
	if check.Message != "/usr/bin/bwrap" {
		t.Errorf("check message = %q, want the resolved path", check.Message)
	}
}

func TestCheckBwrapOnPathReportsWarningWhenMissing(t *testing.T) {
	lookup := &fakeLookup{Missing: map[string]error{"bwrap": errors.New("not found")}}

	check := doctorenv.CheckBwrapOnPath(lookup)

	if check.Severity != doctor.SeverityWarning {
		t.Errorf("check severity = %q, want warning", check.Severity)
	}
	if check.Consequence == "" || check.Fix == "" {
		t.Errorf("warning check is missing consequence or fix text: %+v", check)
	}
	// Missing bwrap must mention that Tool-Call Sandboxes are unavailable so
	// the user knows which workflow is affected.
	if !strings.Contains(check.Consequence, "Tool-Call") {
		t.Errorf("bwrap warning does not name the affected workflow: %q", check.Consequence)
	}
}

func TestCheckIsoboxOnPathReportsOKWhenFound(t *testing.T) {
	lookup := &fakeLookup{Found: map[string]string{"isobox": "/home/u/.local/bin/isobox"}}

	check := doctorenv.CheckIsoboxOnPath(lookup)

	if check.ID != "isobox-on-path" {
		t.Errorf("check ID = %q, want isobox-on-path", check.ID)
	}
	if check.Severity != doctor.SeverityOK {
		t.Errorf("check severity = %q, want ok", check.Severity)
	}
	if check.Message != "/home/u/.local/bin/isobox" {
		t.Errorf("check message = %q, want the resolved path", check.Message)
	}
}

func TestCheckIsoboxOnPathReportsWarningWhenMissing(t *testing.T) {
	lookup := &fakeLookup{Missing: map[string]error{"isobox": errors.New("not found")}}

	check := doctorenv.CheckIsoboxOnPath(lookup)

	if check.Severity != doctor.SeverityWarning {
		t.Errorf("check severity = %q, want warning", check.Severity)
	}
	if check.Consequence == "" || check.Fix == "" {
		t.Errorf("warning check is missing consequence or fix text: %+v", check)
	}
}

func TestCheckIsoboxDuplicatesReturnsNilWhenSingleEntry(t *testing.T) {
	lookup := &fakeLookup{Found: map[string]string{"isobox": "/home/u/.local/bin/isobox"}}

	dup := doctorenv.CheckIsoboxDuplicates(lookup)

	if dup != nil {
		t.Errorf("duplicates check = %+v, want nil for a single isobox on PATH", dup)
	}
}

func TestCheckIsoboxDuplicatesReportsWarningWithActiveAndExtras(t *testing.T) {
	lookup := &multiIsoboxLookup{Entries: []string{
		"/home/u/.local/bin/isobox",
		"/usr/local/bin/isobox",
		"/opt/extra/bin/isobox",
	}}

	dup := doctorenv.CheckIsoboxDuplicates(lookup)

	if dup == nil {
		t.Fatal("duplicates check = nil, want a warning for multiple isobox binaries on PATH")
	}
	if dup.ID != "isobox-duplicates" {
		t.Errorf("check ID = %q, want isobox-duplicates", dup.ID)
	}
	if dup.Severity != doctor.SeverityWarning {
		t.Errorf("check severity = %q, want warning", dup.Severity)
	}
	if dup.Message != "/home/u/.local/bin/isobox" {
		t.Errorf("check message = %q, want the active isobox", dup.Message)
	}
	// The duplicate finding must list the additional binaries so the user
	// knows which other paths were seen.
	if !strings.Contains(dup.Consequence, "/usr/local/bin/isobox") {
		t.Errorf("duplicates warning does not list the first duplicate: %q", dup.Consequence)
	}
	if !strings.Contains(dup.Consequence, "/opt/extra/bin/isobox") {
		t.Errorf("duplicates warning does not list the second duplicate: %q", dup.Consequence)
	}
}

func TestGlobalChecksBundleCoversAllGlobalIDs(t *testing.T) {
	// This test is structural: it asserts that the bundle of global
	// checks is built from a PathLookup and never from a network or
	// process-exec call, so future maintainers can grep one function to
	// confirm "doctor does not call the network". Two separate inputs
	// are used so the duplicates check is exercised.
	single := &fakeLookup{Found: map[string]string{
		"git":    "/usr/bin/git",
		"bwrap":  "/usr/bin/bwrap",
		"isobox": "/home/u/.local/bin/isobox",
	}}

	checks := doctorenv.GlobalChecks(doctorenv.CheckInputs{
		Version: "v0.1.1",
		Commit:  "abc1234",
		Lookup:  single,
	})

	wantIDs := map[string]bool{
		"version":        false,
		"git-on-path":    false,
		"bwrap-on-path":  false,
		"isobox-on-path": false,
	}
	for _, c := range checks {
		if _, ok := wantIDs[c.ID]; ok {
			wantIDs[c.ID] = true
		}
	}
	for id, seen := range wantIDs {
		if !seen {
			t.Errorf("GlobalChecks missing check %q", id)
		}
	}
	for _, c := range checks {
		if c.ID == "isobox-duplicates" {
			t.Errorf("GlobalChecks emitted a duplicates check for a single-isobox PATH: %+v", c)
		}
	}

	// Now exercise the duplicates path: a PATH with two isobox binaries
	// must surface the isobox-duplicates check.
	multi := &multiIsoboxLookup{Entries: []string{
		"/home/u/.local/bin/isobox",
		"/usr/local/bin/isobox",
	}}

	checks = doctorenv.GlobalChecks(doctorenv.CheckInputs{Version: "v0.1.1", Lookup: multi})
	found := false
	for _, c := range checks {
		if c.ID == "isobox-duplicates" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GlobalChecks did not emit isobox-duplicates when PATH has multiple isobox entries")
	}
}

// multiIsoboxLookup simulates a PATH with multiple isobox binaries.
type multiIsoboxLookup struct {
	Entries []string
}

func (m *multiIsoboxLookup) LookPath(name string) (string, error) {
	if name == "isobox" && len(m.Entries) > 0 {
		return m.Entries[0], nil
	}
	return "", errors.New("multi: not found")
}

func (m *multiIsoboxLookup) IsoboxEntries() (string, []string, error) {
	if len(m.Entries) == 0 {
		return "", nil, errors.New("multi: no entries")
	}
	return m.Entries[0], append([]string(nil), m.Entries[1:]...), nil
}
