package update

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathLookup is the dependency boundary between the update package and
// the host filesystem. It mirrors the doctorenv.PathLookup shape so
// the two packages can share the same HostLookup implementation
// without sharing the type. Keeping the type local to each package
// avoids an unnecessary cross-package dependency for what is a
// trivial 2-method interface.
type PathLookup interface {
	LookPath(name string) (string, error)
	IsoboxEntries() (active string, duplicates []string, err error)
}

// Target describes the resolved Update Target plus the optional
// duplicate binaries the user should know about. The IsManaged flag
// records whether the path looks package-manager-managed; that flag
// is read by the eligibility check rather than computed lazily.
type Target struct {
	// Path is the absolute path of the first isobox executable
	// resolved on the host PATH.
	Path string
	// Duplicates are additional isobox executables found later on
	// PATH, in PATH order. The slice may be empty.
	Duplicates []string
	// IsManaged reports whether the path is inside a clearly
	// package-manager- or system-managed location. The flag is set
	// during ResolveUpdateTarget so the caller does not have to
	// re-evaluate the path.
	IsManaged bool
}

// managedPathPrefixes enumerates the host directories that the
// updater treats as clearly package-manager- or system-managed. The
// list is intentionally narrow: directories such as /home/<u>/.local
// and /usr/local/bin remain writable manual-style targets even when
// they share a prefix with a managed path.
var managedPathPrefixes = []string{
	"/usr/bin/",
	"/opt/homebrew/",
	"/snap/",
	"/var/lib/dpkg/",
	"/var/lib/rpm/",
	"/var/lib/pacman/",
	"/nix/store/",
}

// ResolveUpdateTarget returns the active Update Target resolved from
// the first isobox executable on the host PATH. The returned Target
// always reports the active path; the duplicates slice is populated
// when additional isobox binaries are present.
//
// The function does not perform any eligibility check; callers use
// CheckManagedTarget (or read Target.IsManaged) to decide whether the
// target is safe to replace.
func ResolveUpdateTarget(lookup PathLookup) (Target, error) {
	active, duplicates, err := lookup.IsoboxEntries()
	if err != nil {
		return Target{}, fmt.Errorf("resolve isobox on PATH: %w", err)
	}
	if active == "" {
		return Target{}, fmt.Errorf("isobox is not on PATH; cannot select an Update Target")
	}
	return Target{
		Path:       active,
		Duplicates: duplicates,
		IsManaged:  IsManagedPath(active),
	}, nil
}

// IsManagedPath reports whether the given path lives under a
// directory the updater treats as clearly package-manager- or
// system-managed. The check is a prefix comparison against a small,
// auditable allowlist of directories. Paths under /usr/local/bin and
// user-local bin directories are NOT considered managed because
// users typically write to those locations manually.
func IsManagedPath(path string) bool {
	cleaned := filepath.Clean(path)
	if !strings.HasSuffix(cleaned, string(filepath.Separator)) {
		// Treat the path as a directory tree root for prefix
		// comparison: /usr/bin matches /usr/bin but not /usr/binx.
		cleaned = cleaned + string(filepath.Separator)
	}
	for _, prefix := range managedPathPrefixes {
		if strings.HasPrefix(cleaned, prefix) {
			return true
		}
	}
	return false
}

// CheckManagedTarget returns an error explaining the refusal when
// the given target path is clearly package-manager- or system-
// managed. The error text names the directory and tells the user to
// use the package manager instead of `isobox update`.
func CheckManagedTarget(path string) error {
	if !IsManagedPath(path) {
		return nil
	}
	return fmt.Errorf("refusing to update %s: this path is inside a package-manager- or system-managed location; reinstall isobox through your package manager or move it to a writable manual-style directory (e.g. %s or %s) before running `isobox update`", path, "${HOME}/.local/bin", "/usr/local/bin")
}
