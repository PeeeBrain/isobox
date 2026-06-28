package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// GitHubRepo is the repository the GitHubReleaseClient queries for
// release metadata. The default repository is the upstream isobox
// project; tests can substitute a different value via NewGitHubReleaseClient.
const GitHubRepo = "PeeeBrain/isobox"

// GitHubReleaseClient fetches release metadata from the GitHub
// Releases API. The client is read-only: it only calls
// GET /repos/{owner}/{repo}/releases and decodes the JSON array of
// release objects into the local Release shape. Network failures
// surface as the error from ListReleases; the client does not retry.
type GitHubReleaseClient struct {
	// Repo is the GitHub owner/name pair (e.g. "PeeeBrain/isobox").
	Repo string
	// BaseURL is the GitHub API base URL. Tests can point it at a
	// local httptest server; production code uses the default
	// https://api.github.com.
	BaseURL string
	// HTTPClient is the underlying HTTP client. When nil, a default
	// client with a 30s timeout is used.
	HTTPClient *http.Client
}

// NewGitHubReleaseClient returns a GitHubReleaseClient for the
// default isobox repository. The returned client is safe for
// concurrent use.
func NewGitHubReleaseClient() *GitHubReleaseClient {
	return &GitHubReleaseClient{
		Repo:    GitHubRepo,
		BaseURL: "https://api.github.com",
	}
}

type githubReleasePayload struct {
	TagName     string    `json:"tag_name"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// ListReleases fetches the full release list for the configured
// repository and decodes it into the local Release shape. The full
// list is requested because GitHub's default ordering is by creation
// date and the stable-only selector sorts by PublishedAt regardless.
func (c *GitHubReleaseClient) ListReleases() ([]Release, error) {
	repo := c.Repo
	if repo == "" {
		repo = GitHubRepo
	}
	base := c.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	endpoint := fmt.Sprintf("%s/repos/%s/releases", base, repo)
	if _, err := url.Parse(endpoint); err != nil {
		return nil, fmt.Errorf("parse GitHub API URL: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github releases API returned status %d", resp.StatusCode)
	}
	var payload []githubReleasePayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode releases: %w", err)
	}
	releases := make([]Release, 0, len(payload))
	for _, r := range payload {
		releases = append(releases, Release{
			TagName:     r.TagName,
			Prerelease:  r.Prerelease,
			Draft:       r.Draft,
			PublishedAt: r.PublishedAt,
			Assets:      r.Assets,
		})
	}
	return releases, nil
}
