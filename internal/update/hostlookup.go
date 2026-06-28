package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HostLookup is the production PathLookup used by the `isobox update`
// command. It is structurally identical to doctorenv.HostLookup but
// lives in the update package to keep the cross-package interface
// trivial; a future shared package could collapse them.
type HostLookup struct{}

func NewHostLookup() *HostLookup { return &HostLookup{} }

func (h *HostLookup) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

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

// NotFoundError mirrors doctorenv.NotFoundError so callers in the
// update package can distinguish "no isobox at all" from a generic
// error. Keeping the type local to each package avoids a
// cross-package dependency for a single sentinel.
type NotFoundError struct{ Name string }

func (e *NotFoundError) Error() string { return "isobox not found on PATH" }
