// Package update implements the read-only update check and updater for
// isobox. The package is intentionally small: it models GitHub Release
// metadata, provides a stable-only selector, and compares semver-style
// version strings so the CLI can report current-vs-latest status
// without performing any side effects.
//
// The package never writes to the host. Replacement, backup, and
// rollback belong in a separate slice; the current package is limited
// to metadata selection, version comparison, and the eligibility
// checks the update command must enforce before touching the Update
// Target.
package update

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Release is the subset of GitHub Release metadata the update flow
// needs. Prerelease and Draft are explicit fields so the stable-only
// selector can filter them without inspecting tag-name conventions.
// The JSON tags match the GitHub Releases API wire format so the
// production HTTP client and the test execReleaseClient can share a
// single decoder path.
type Release struct {
	TagName     string    `json:"tag_name"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset is a downloadable file attached to a GitHub Release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// ReleaseClient is the dependency boundary between the update package
// and any network or filesystem source of release metadata. Tests use
// an in-memory fake; production code uses an HTTP client wrapper
// around the GitHub Releases API.
type ReleaseClient interface {
	ListReleases() ([]Release, error)
}

// Status describes how the running version relates to the latest
// stable release.
type Status int

const (
	// StatusUpToDate means the running version matches the latest
	// stable release.
	StatusUpToDate Status = iota
	// StatusBehind means a newer stable release is available.
	StatusBehind
	// StatusAhead means the running version is newer than the latest
	// stable release (typical for development builds ahead of a
	// release line, or for early access installs).
	StatusAhead
)

// String returns the canonical lower-case name of the status. The
// representation is stable because update command output and tests
// assert against it.
func (s Status) String() string {
	switch s {
	case StatusUpToDate:
		return "up-to-date"
	case StatusBehind:
		return "behind"
	case StatusAhead:
		return "ahead"
	default:
		return "unknown"
	}
}

// SelectLatestStable returns the most recent stable Release published
// by the client. Drafts and prereleases are skipped because the
// product policy treats the first stable line as the only update
// surface in this milestone.
//
// The function does not assume a particular input order. It sorts the
// stable subset by PublishedAt descending and returns the first entry.
func SelectLatestStable(client ReleaseClient) (Release, error) {
	releases, err := client.ListReleases()
	if err != nil {
		return Release{}, fmt.Errorf("list releases: %w", err)
	}
	var stable []Release
	for _, r := range releases {
		if r.Prerelease || r.Draft {
			continue
		}
		stable = append(stable, r)
	}
	if len(stable) == 0 {
		return Release{}, errors.New("no stable releases available")
	}
	sort.SliceStable(stable, func(i, j int) bool {
		return stable[i].PublishedAt.After(stable[j].PublishedAt)
	})
	return stable[0], nil
}

// CompareVersions reports the relationship between the running
// version and a candidate latest version. Both inputs may carry a
// leading "v"; the comparison strips it before parsing.
func CompareVersions(current, latest string) (Status, error) {
	c, err := parseSemver(current)
	if err != nil {
		return 0, fmt.Errorf("parse current version %q: %w", current, err)
	}
	l, err := parseSemver(latest)
	if err != nil {
		return 0, fmt.Errorf("parse latest version %q: %w", latest, err)
	}
	switch {
	case equal(c, l):
		return StatusUpToDate, nil
	case lessThan(c, l):
		return StatusBehind, nil
	default:
		return StatusAhead, nil
	}
}

// RefuseDevVersion returns an error when the current version is "dev",
// because self-update is not safe for development builds. Tagged
// versions pass through unchanged.
func RefuseDevVersion(current string) error {
	if current == "dev" {
		return errors.New("refusing to update a dev build; install a tagged release before running `isobox update`")
	}
	return nil
}

// parseSemver parses a version string of the form vX.Y.Z (or X.Y.Z)
// into a comparable triple. The parser is intentionally minimal:
// pre-release and build metadata are ignored, and any non-numeric
// component after the major segment is rejected so the caller does not
// silently treat "v0.1.0-rc.1" as "v0.1.0".
func parseSemver(in string) ([3]int, error) {
	s := strings.TrimSpace(in)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return [3]int{}, errors.New("empty version")
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("expected MAJOR.MINOR.PATCH, got %q", in)
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return [3]int{}, fmt.Errorf("invalid version component %q in %q", p, in)
		}
		out[i] = n
	}
	return out, nil
}

func equal(a, b [3]int) bool { return a == b }

func lessThan(a, b [3]int) bool {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}
