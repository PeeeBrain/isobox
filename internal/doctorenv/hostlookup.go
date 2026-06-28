package doctorenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HostLookup is a PathLookup that resolves binaries against the
// current process PATH. It is the production implementation used by
// the `isobox doctor` command; tests use a fake lookup instead.
//
// HostLookup walks the PATH directories itself for IsoboxEntries so the
// command can detect duplicates that exec.LookPath would have hidden by
// returning only the first match.
type HostLookup struct{}

// NewHostLookup returns a PathLookup that consults the current process
// environment. The lookup is read-only and safe to share across calls.
func NewHostLookup() *HostLookup { return &HostLookup{} }

// LookPath returns the first absolute path for the given binary on the
// process PATH, mirroring exec.LookPath behavior.
func (h *HostLookup) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// IsoboxEntries walks PATH and returns every executable named `isobox`
// that it encounters, in PATH order. The first entry is the active
// binary; the remaining entries are duplicates. The function is the
// only place in the doctor pipeline that inspects multiple PATH
// matches, so the duplicates logic stays auditable.
func (h *HostLookup) IsoboxEntries() (string, []string, error) {
	pathDirs := filepath.SplitList(os.Getenv("PATH"))
	var entries []string
	seen := make(map[string]bool)
	for _, dir := range pathDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "isobox")
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		// m.Mode()&0o111 != 0 reports any execute bit; treat the
		// candidate as an executable when at least one execute bit is
		// set. This is best-effort: a binary with no execute bit on
		// disk will not be returned, matching exec.LookPath behavior.
		if info.Mode().Perm()&0o111 == 0 {
			continue
		}
		abs := candidate
		if a, err := filepath.Abs(candidate); err == nil {
			abs = a
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		entries = append(entries, abs)
	}
	if len(entries) == 0 {
		return "", nil, &NotFoundError{Name: "isobox"}
	}
	if len(entries) == 1 {
		return entries[0], nil, nil
	}
	return entries[0], entries[1:], nil
}

// NotFoundError is returned by IsoboxEntries when no isobox binary
// appears anywhere on PATH. It exists so callers can distinguish
// "no binary at all" from a generic error.
type NotFoundError struct{ Name string }

func (e *NotFoundError) Error() string { return "isobox not found on PATH" }
