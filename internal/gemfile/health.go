package gemfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/spaquet/gemtracker/internal/logger"
)

// HealthScore represents the maintenance health tier of a gem.
// Health is determined by last release date, maintainer count, activity, and repository status.
type HealthScore int

const (
	// HealthUnknown indicates health data could not be fetched (rate limited, network error, etc.)
	HealthUnknown HealthScore = iota
	// HealthHealthy indicates active gem with regular releases and multiple maintainers (🟢)
	HealthHealthy
	// HealthWarning indicates stale gem with no recent activity or single maintainer (🟡)
	HealthWarning
	// HealthCritical indicates inactive gem, archived, or disabled repository (🔴)
	HealthCritical
)

func (hs HealthScore) String() string {
	switch hs {
	case HealthHealthy:
		return "HEALTHY"
	case HealthWarning:
		return "WARNING"
	case HealthCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// GemHealth contains maintenance and activity metrics for a gem from RubyGems and GitHub.
type GemHealth struct {
	// Score is the computed health tier (Healthy, Warning, Critical, Unknown)
	Score HealthScore `json:"score"`
	// LastRelease is the timestamp of the latest gem release from RubyGems
	LastRelease time.Time `json:"last_release"`
	// GitHubPushedAt is the timestamp of the last commit pushed to the repository
	GitHubPushedAt time.Time `json:"github_pushed_at"`
	// Stars is the number of GitHub stars on the repository
	Stars int `json:"stars"`
	// OpenIssues is the number of open issues on the repository
	OpenIssues int `json:"open_issues"`
	// Archived indicates whether the GitHub repository is archived
	Archived bool `json:"archived"`
	// Disabled indicates whether the GitHub repository is disabled
	Disabled bool `json:"disabled"`
	// MaintainerCount is the number of maintainers listed on RubyGems
	MaintainerCount int `json:"maintainer_count"`
	// RateLimited indicates if GitHub API rate limit was exceeded (partial data)
	RateLimited bool `json:"rate_limited"`
	// FetchedAt is the timestamp when this health data was fetched
	FetchedAt time.Time `json:"fetched_at"`
}

// RepoOwnerPair represents a gem and its GitHub repository for batch fetching.
// Used for efficient GraphQL batch queries to GitHub.
type RepoOwnerPair struct {
	// GemName is the gem name
	GemName string
	// Owner is the GitHub repository owner
	Owner string
	// Repo is the GitHub repository name
	Repo string
}

// HealthChecker fetches and caches gem health data from the RubyGems and GitHub APIs.
// It supports batch GitHub queries via GraphQL and graceful handling of rate limiting.
type HealthChecker struct {
	// client is the HTTP client for API requests
	client *http.Client
	// githubCache maps "owner/repo" to GitHub repo data for fast lookups
	githubCache map[string]*githubRepo
	// mu protects githubCache from concurrent access
	mu sync.Mutex
}

// NewHealthChecker creates a new HealthChecker with a 10-second HTTP timeout and empty caches.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		githubCache: make(map[string]*githubRepo),
	}
}

// rubygems owner response struct
type rubygemsOwner struct {
	Handle string `json:"handle"`
	Role   string `json:"role"`
}

// github repo response struct (REST API)
type githubRepo struct {
	StargazersCount int       `json:"stargazers_count"`
	OpenIssuesCount int       `json:"open_issues_count"`
	PushedAt        time.Time `json:"pushed_at"`
	Archived        bool      `json:"archived"`
	Disabled        bool      `json:"disabled"`
}

// githubGraphQLRepo is the GraphQL response structure for a single repo
type githubGraphQLRepo struct {
	PushedAt       string `json:"pushedAt"`
	StargazerCount int    `json:"stargazerCount"`
	IsArchived     bool   `json:"isArchived"`
	IsDisabled     bool   `json:"isDisabled"`
	OpenIssues     struct {
		TotalCount int `json:"totalCount"`
	} `json:"openIssues"`
}

// githubGraphQLResponse is the top-level GraphQL response with aliases
type githubGraphQLResponse struct {
	Data   map[string]*githubGraphQLRepo `json:"data"`
	Errors []map[string]interface{}      `json:"errors,omitempty"`
}

// FetchHealth fetches and computes health data for a gem from RubyGems and GitHub APIs.
// It uses cached GitHub data from FetchGitHubBatch when available, then falls back to individual REST calls.
// Returns (*GemHealth, error). If GitHub rate limited, returns partial data with RateLimited=true.
func (hc *HealthChecker) FetchHealth(gemName, sourceCodeURI, homepageURI, versionCreatedAtStr, ownersURL string) (*GemHealth, error) {
	health := &GemHealth{
		FetchedAt: time.Now(),
	}

	// Parse version created at (rubygems returns with fractional seconds, use RFC3339Nano)
	if versionCreatedAtStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, versionCreatedAtStr); err != nil {
			logger.Warn("Failed to parse version created at for gem %q: %v", gemName, err)
		} else {
			health.LastRelease = t
		}
	}

	// Fetch maintainer count from RubyGems
	if ownersURL != "" {
		owners, err := hc.fetchRubyGemsOwners(ownersURL)
		if err != nil {
			logger.Warn("Failed to fetch gem owners for %q: %v", gemName, err)
		} else {
			health.MaintainerCount = owners
		}
	}

	// Fetch GitHub stats if source URI provided, fallback to homepage URI
	// First check cache (populated by FetchGitHubBatch), then fall back to REST API if available
	githubURI := sourceCodeURI
	if githubURI == "" {
		githubURI = homepageURI
	}
	if githubURI != "" {
		owner, repo, ok := ExtractGitHubOwnerRepo(githubURI)
		if ok {
			// Check GraphQL batch cache first
			hc.mu.Lock()
			key := strings.ToLower(owner + "/" + repo)
			if githubHealth, cached := hc.githubCache[key]; cached {
				hc.mu.Unlock()
				health.GitHubPushedAt = githubHealth.PushedAt
				health.Stars = githubHealth.StargazersCount
				health.OpenIssues = githubHealth.OpenIssuesCount
				health.Archived = githubHealth.Archived
				health.Disabled = githubHealth.Disabled
				health.Score = ComputeHealthScore(health)
				return health, nil
			}
			hc.mu.Unlock()

			// If no cache hit and we have a GITHUB_TOKEN, try REST API (fallback)
			if os.Getenv("GITHUB_TOKEN") != "" {
				githubHealth, rateLimited := hc.fetchGitHubRepo(owner, repo)
				if rateLimited {
					health.RateLimited = true
				} else if githubHealth != nil {
					health.GitHubPushedAt = githubHealth.PushedAt
					health.Stars = githubHealth.StargazersCount
					health.OpenIssues = githubHealth.OpenIssuesCount
					health.Archived = githubHealth.Archived
					health.Disabled = githubHealth.Disabled
				}
			}
			// If no GITHUB_TOKEN, we just skip GitHub data (no error)
		}
	}

	health.Score = ComputeHealthScore(health)
	return health, nil
}

// FetchGitHubBatch fetches GitHub data for multiple repositories in batched GraphQL requests.
// It caches all results for use by FetchHealth. If GITHUB_TOKEN is not set, it silently
// returns without fetching (GitHub data is optional). Returns an error only if the API request fails.
func (hc *HealthChecker) FetchGitHubBatch(pairs []RepoOwnerPair) error {
	// If no token, skip GitHub entirely
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil
	}

	if len(pairs) == 0 {
		return nil
	}

	// Batch requests in groups of 50 (GraphQL API limit)
	batchSize := 50
	for i := 0; i < len(pairs); i += batchSize {
		end := i + batchSize
		if end > len(pairs) {
			end = len(pairs)
		}

		batch := pairs[i:end]
		if err := hc.fetchGitHubBatchGroup(batch, token); err != nil {
			// Log but don't fail completely - partial data is still useful
			fmt.Printf("Warning: GitHub batch fetch failed: %v\n", err)
		}
	}

	return nil
}

// fetchGitHubBatchGroup fetches a single batch (up to 50 repos) via GraphQL
func (hc *HealthChecker) fetchGitHubBatchGroup(pairs []RepoOwnerPair, token string) error {
	// Build GraphQL query with aliases (r0, r1, ...)
	var queryBuilder strings.Builder
	queryBuilder.WriteString("query {")
	for i, pair := range pairs {
		alias := fmt.Sprintf("r%d", i)
		queryBuilder.WriteString(fmt.Sprintf(
			`%s: repository(owner: "%s", name: "%s") {
				pushedAt
				stargazerCount
				isArchived
				isDisabled
				openIssues: issues(states: OPEN) { totalCount }
			}`,
			alias, pair.Owner, pair.Repo,
		))
	}
	queryBuilder.WriteString("}")

	query := queryBuilder.String()

	// Execute GraphQL request
	reqBody := map[string]string{"query": query}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle rate limiting gracefully (don't error, just skip)
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("github rate limited")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github returned status %d", resp.StatusCode)
	}

	var result githubGraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Check for GraphQL errors
	if len(result.Errors) > 0 {
		// Log but don't fail - some repos may be private or deleted
		fmt.Printf("Warning: GraphQL errors in batch fetch: %v\n", result.Errors)
	}

	// Store results in cache (keyed by owner/repo)
	hc.mu.Lock()
	defer hc.mu.Unlock()

	for alias, repoData := range result.Data {
		if repoData == nil {
			continue
		}

		// Find the corresponding pair by alias index
		idx := 0
		fmt.Sscanf(alias, "r%d", &idx)
		if idx >= len(pairs) {
			continue
		}

		pair := pairs[idx]
		key := strings.ToLower(pair.Owner + "/" + pair.Repo)

		// Convert GraphQL response to githubRepo struct
		ghRepo := &githubRepo{
			StargazersCount: repoData.StargazerCount,
			OpenIssuesCount: repoData.OpenIssues.TotalCount,
			Archived:        repoData.IsArchived,
			Disabled:        repoData.IsDisabled,
		}

		// Parse pushed_at timestamp
		if repoData.PushedAt != "" {
			if t, err := time.Parse(time.RFC3339, repoData.PushedAt); err == nil {
				ghRepo.PushedAt = t
			}
		}

		hc.githubCache[key] = ghRepo
	}

	return nil
}

// fetchRubyGemsOwners returns the count of gem owners
func (hc *HealthChecker) fetchRubyGemsOwners(url string) (int, error) {
	resp, err := hc.client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to fetch owners: %d", resp.StatusCode)
	}

	var owners []rubygemsOwner
	if err := json.NewDecoder(resp.Body).Decode(&owners); err != nil {
		return 0, err
	}

	return len(owners), nil
}

// fetchGitHubRepo returns GitHub stats or nil if rate limited
// Second return value indicates if GitHub rate limited
func (hc *HealthChecker) fetchGitHubRepo(owner, repo string) (*githubRepo, bool) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false
	}

	// Add GitHub token if available for higher rate limits (5000/hr vs 60/hr)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, true
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	var ghRepo githubRepo
	if err := json.NewDecoder(resp.Body).Decode(&ghRepo); err != nil {
		return nil, false
	}

	return &ghRepo, false
}

// ComputeHealthScore computes a health tier from gem health metrics.
// Tiers: CRITICAL (archived/disabled/3+ yrs inactive), WARNING (1-3 yrs inactive or single maintainer),
// HEALTHY (active within 1 year with 2+ maintainers), UNKNOWN (rate limited or no data).
func ComputeHealthScore(h *GemHealth) HealthScore {
	if h.RateLimited {
		return HealthUnknown
	}

	if h.Archived || h.Disabled {
		return HealthCritical
	}

	// Use the more recent of LastRelease and GitHubPushedAt
	lastActivity := h.LastRelease
	if h.GitHubPushedAt.After(lastActivity) {
		lastActivity = h.GitHubPushedAt
	}

	// If we have no activity data (zero time), we couldn't assess health
	// Return Unknown instead of assuming it's critical
	if lastActivity.IsZero() {
		return HealthUnknown
	}

	now := time.Now()
	threeYearsAgo := now.AddDate(-3, 0, 0)
	oneYearAgo := now.AddDate(-1, 0, 0)

	if lastActivity.Before(threeYearsAgo) {
		return HealthCritical
	}

	if lastActivity.Before(oneYearAgo) {
		return HealthWarning
	}

	if h.MaintainerCount == 1 {
		return HealthWarning
	}

	return HealthHealthy
}

// ExtractGitHubOwnerRepo extracts the GitHub owner and repository name from a URI.
// It handles HTTPS, HTTP, and SSH-style URIs with optional .git suffix.
// Returns (owner, repo, true) if extraction succeeds, ("", "", false) otherwise.
func ExtractGitHubOwnerRepo(uri string) (owner, repo string, ok bool) {
	// Regex: github.com/owner/repo or github.com:owner/repo
	re := regexp.MustCompile(`github\.com[:/]([^/]+)/([^/.\s]+)`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) < 3 {
		return "", "", false
	}

	owner = matches[1]
	repo = matches[2]
	// Strip .git suffix if present
	if len(repo) > 4 && repo[len(repo)-4:] == ".git" {
		repo = repo[:len(repo)-4]
	}
	return owner, repo, true
}
