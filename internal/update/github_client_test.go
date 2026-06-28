package update_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"isobox/internal/update"
)

func TestGitHubReleaseClientDecodesReleaseList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/repos/owner/repo/releases") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{"tag_name":"v0.1.0","prerelease":false,"draft":false,"published_at":"2025-01-01T00:00:00Z"},
			{"tag_name":"v0.1.1-rc.1","prerelease":true,"draft":false,"published_at":"2025-02-01T00:00:00Z"}
		]`))
	}))
	defer server.Close()

	client := &update.GitHubReleaseClient{
		Repo:    "owner/repo",
		BaseURL: server.URL,
	}
	releases, err := client.ListReleases()
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("got %d releases, want 2", len(releases))
	}
	if releases[0].TagName != "v0.1.0" || releases[1].Prerelease != true {
		t.Errorf("decoded releases = %+v, want the JSON shape", releases)
	}
}

func TestGitHubReleaseClientSurfacesErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	client := &update.GitHubReleaseClient{
		Repo:    "owner/repo",
		BaseURL: server.URL,
	}
	if _, err := client.ListReleases(); err == nil {
		t.Fatal("ListReleases did not surface a non-2xx response as an error")
	}
}

func TestGitHubReleaseClientSurfacesNetworkError(t *testing.T) {
	client := &update.GitHubReleaseClient{
		Repo:    "owner/repo",
		BaseURL: "http://127.0.0.1:1",
	}
	if _, err := client.ListReleases(); err == nil {
		t.Fatal("ListReleases did not surface a network error")
	}
}
