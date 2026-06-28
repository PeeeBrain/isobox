package update_test

import (
	"errors"
	"testing"
	"time"

	"isobox/internal/update"
)

// fakeClient is an in-memory ReleaseClient used to exercise the update
// flow without calling GitHub. The Releases field is returned in the
// order produced by the fake; SelectLatestStable must not depend on
// the order, only on the per-release fields.
type fakeClient struct {
	Releases []update.Release
	Err      error
	Calls    int
}

func (f *fakeClient) ListReleases() ([]update.Release, error) {
	f.Calls++
	return f.Releases, f.Err
}

func TestSelectLatestStablePicksHighestStable(t *testing.T) {
	releases := []update.Release{
		{TagName: "v0.1.0", PublishedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{TagName: "v0.1.2", PublishedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{TagName: "v0.1.1", PublishedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}
	client := &fakeClient{Releases: releases}

	latest, err := update.SelectLatestStable(client)
	if err != nil {
		t.Fatalf("SelectLatestStable: %v", err)
	}
	if latest.TagName != "v0.1.2" {
		t.Errorf("latest = %q, want v0.1.2", latest.TagName)
	}
	if client.Calls != 1 {
		t.Errorf("client was called %d times, want 1", client.Calls)
	}
}

func TestSelectLatestStableSkipsPrereleasesAndDrafts(t *testing.T) {
	releases := []update.Release{
		{TagName: "v0.1.0", PublishedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{TagName: "v0.2.0-rc.1", Prerelease: true, PublishedAt: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)},
		{TagName: "v0.2.0-draft", Draft: true, PublishedAt: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)},
		{TagName: "v0.1.5", PublishedAt: time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	client := &fakeClient{Releases: releases}

	latest, err := update.SelectLatestStable(client)
	if err != nil {
		t.Fatalf("SelectLatestStable: %v", err)
	}
	if latest.TagName != "v0.1.5" {
		t.Errorf("latest = %q, want v0.1.5 (must skip prerelease and draft)", latest.TagName)
	}
}

func TestSelectLatestStableErrorsWhenNoStableReleases(t *testing.T) {
	releases := []update.Release{
		{TagName: "v0.1.0-rc.1", Prerelease: true},
		{TagName: "v0.2.0-draft", Draft: true},
	}
	client := &fakeClient{Releases: releases}

	_, err := update.SelectLatestStable(client)
	if err == nil {
		t.Fatal("SelectLatestStable returned no error for an all-draft/prerelease list")
	}
}

func TestSelectLatestStablePropagatesClientError(t *testing.T) {
	client := &fakeClient{Err: errors.New("network down")}

	_, err := update.SelectLatestStable(client)
	if err == nil {
		t.Fatal("SelectLatestStable returned no error when the client failed")
	}
}

func TestCompareVersionsReportsUpToDateAndOutdated(t *testing.T) {
	cases := []struct {
		name       string
		current    string
		latest     string
		wantStatus update.Status
	}{
		{"equal versions", "v0.1.2", "v0.1.2", update.StatusUpToDate},
		{"current behind", "v0.1.1", "v0.1.2", update.StatusBehind},
		{"current ahead (pre-release line)", "v0.2.0", "v0.1.5", update.StatusAhead},
		{"missing leading v", "0.1.2", "0.1.2", update.StatusUpToDate},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, err := update.CompareVersions(c.current, c.latest)
			if err != nil {
				t.Fatalf("CompareVersions(%q, %q): %v", c.current, c.latest, err)
			}
			if status != c.wantStatus {
				t.Errorf("CompareVersions(%q, %q) = %q, want %q", c.current, c.latest, status, c.wantStatus)
			}
		})
	}
}

func TestRefuseDevVersionRejectsDevCurrent(t *testing.T) {
	err := update.RefuseDevVersion("dev")
	if err == nil {
		t.Fatal("RefuseDevVersion did not reject the dev version")
	}
}

func TestRefuseDevVersionAllowsTaggedCurrent(t *testing.T) {
	if err := update.RefuseDevVersion("v0.1.1"); err != nil {
		t.Errorf("RefuseDevVersion rejected a tagged version: %v", err)
	}
}
