// Package doctorenv builds the read-only global Doctor Checks that depend
// on host state (PATH, executable presence). It exists separately from the
// doctor package so the doctor package can stay free of host-aware
// dependencies and so the global checks can be tested with an injected
// PathLookup instead of the real host's PATH.
//
// All checks produced by this package are global: they inspect host
// conditions, not project conditions, and they never call the network or
// perform side effects. The package is the single place to look when
// confirming that `isobox doctor` does not perform online checks.
package doctorenv

import "isobox/internal/doctor"

// PathLookup abstracts the host-level lookups the global Doctor Checks
// need. The interface is small so tests can simulate PATH conditions
// without depending on the host machine's actual dependency state.
type PathLookup interface {
	// LookPath returns the absolute path of the named binary as resolved
	// by the host PATH, or an error if the binary is not present.
	LookPath(name string) (string, error)
	// IsoboxEntries returns the active isobox executable resolved by the
	// host PATH plus any additional isobox executables that appear later
	// on PATH. When the binary is not on PATH at all, the error is
	// non-nil and the strings are empty.
	IsoboxEntries() (active string, duplicates []string, err error)
}

// CheckInputs is the static input every global check may consume. New
// fields are added in follow-up slices; today's checks only need the
// version metadata and the PathLookup.
type CheckInputs struct {
	Version string
	Commit  string
	Lookup  PathLookup
}

// GlobalChecks returns the bundle of global Doctor Checks built from the
// provided inputs. The order is stable so the rendered report is
// predictable: version first, then the host-tooling checks in ID order.
func GlobalChecks(in CheckInputs) []doctor.Check {
	checks := []doctor.Check{
		doctor.OK("version", "isobox version", formatVersion(in.Version, in.Commit)),
	}
	checks = append(checks, CheckGitOnPath(in.Lookup))
	checks = append(checks, CheckBwrapOnPath(in.Lookup))
	checks = append(checks, CheckIsoboxOnPath(in.Lookup))
	if dup := CheckIsoboxDuplicates(in.Lookup); dup != nil {
		checks = append(checks, *dup)
	}
	return checks
}

// formatVersion returns the rendered version line used by the version
// check. The exact shape is the same as the previous skeleton so existing
// reports stay readable; "dev" is treated as a normal version string and
// never implies an update warning.
func formatVersion(version, commit string) string {
	if version == "" {
		version = "unknown"
	}
	if commit == "" {
		return version
	}
	return version + " (commit " + commit + ")"
}

// CheckGitOnPath reports whether `git` is on the host PATH. Missing git
// is an error: no isobox workflow can run without it.
func CheckGitOnPath(lookup PathLookup) doctor.Check {
	path, err := lookup.LookPath("git")
	if err == nil && path != "" {
		return doctor.OK("git-on-path", "git is on PATH", path)
	}
	return doctor.Error(
		"git-on-path",
		"git is not on PATH",
		"no isobox workflow can run; every isobox command needs git to locate or operate on a repository",
		"install git for your platform and ensure it is reachable on PATH",
	)
}

// CheckBwrapOnPath reports whether `bwrap` (bubblewrap) is on the host
// PATH. Missing bwrap is a warning: isobox itself still runs, but the
// Tool-Call Sandbox workflow (isobox tool) cannot create the filesystem
// containment boundary.
func CheckBwrapOnPath(lookup PathLookup) doctor.Check {
	path, err := lookup.LookPath("bwrap")
	if err == nil && path != "" {
		return doctor.OK("bwrap-on-path", "bubblewrap (bwrap) is on PATH", path)
	}
	return doctor.Warning(
		"bwrap-on-path",
		"bubblewrap (bwrap) is not on PATH",
		"`isobox tool` cannot create a Tool-Call Sandbox; the project-level doctor check may also fail readiness",
		"install bubblewrap (bwrap) for your platform and ensure it is reachable on PATH",
	)
}

// CheckIsoboxOnPath reports whether the `isobox` binary the user
// actually invokes is reachable on PATH. Missing isobox is a warning:
// the binary still runs from its real install location, but shell
// invocation is ambiguous.
func CheckIsoboxOnPath(lookup PathLookup) doctor.Check {
	path, err := lookup.LookPath("isobox")
	if err == nil && path != "" {
		return doctor.OK("isobox-on-path", "isobox is on PATH", path)
	}
	return doctor.Warning(
		"isobox-on-path",
		"isobox is not on PATH",
		"the running isobox binary is not the one the shell resolves first; `isobox update` may not be able to find the active Update Target",
		"add the directory containing the isobox binary to PATH so the shell can resolve it",
	)
}

// CheckIsoboxDuplicates returns a warning Doctor Check when more than
// one isobox binary appears on the host PATH, or nil when exactly one
// entry (or none) is present. The message is the active binary, and the
// consequence lists the duplicates so the user can decide which to keep.
//
// A nil return is preferred over a zero-severity check so the report
// remains a single concise line per finding category.
func CheckIsoboxDuplicates(lookup PathLookup) *doctor.Check {
	active, duplicates, err := lookup.IsoboxEntries()
	if err != nil || active == "" {
		return nil
	}
	if len(duplicates) == 0 {
		return nil
	}

	consequence := "additional isobox binaries on PATH: " + joinPaths(duplicates)
	fix := "remove the duplicate binaries or reorder PATH so only one isobox entry is reachable"
	return &doctor.Check{
		ID:          "isobox-duplicates",
		Severity:    doctor.SeverityWarning,
		Title:       "multiple isobox binaries on PATH",
		Message:     active,
		Consequence: consequence,
		Fix:         fix,
	}
}

// joinPaths concatenates a slice of paths with a " | " separator. The
// separator is chosen so duplicate lists remain readable on a single
// report line.
func joinPaths(in []string) string {
	if len(in) == 0 {
		return ""
	}
	out := in[0]
	for _, p := range in[1:] {
		out += " | " + p
	}
	return out
}
