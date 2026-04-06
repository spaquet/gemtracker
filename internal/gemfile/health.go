package gemfile

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/spaquet/gemtracker/internal/logger"
)

// HealthScore represents the health tier of a gem
type HealthScore int

const (
	HealthUnknown HealthScore = iota
	HealthHealthy
	HealthWarning
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

// GemHealth contains health indicators for a gem
type GemHealth struct {
	Score           HealthScore `json:"score"`
	LastRelease     time.Time   `json:"last_release"`     // from rubygems version_created_at
	GitHubPushedAt  time.Time   `json:"github_pushed_at"` // from github pushed_at
	Stars           int         `json:"stars"`
	OpenIssues      int         `json:"open_issues"`
	Archived        bool        `json:"archived"`
	Disabled        bool        `json:"disabled"`
	MaintainerCount int         `json:"maintainer_count"`
	RateLimited     bool        `json:"rate_limited"` // GitHub rate limit hit, data partial
	FetchedAt       time.Time   `json:"fetched_at"`
}

// HealthChecker fetches health data from RubyGems and GitHub APIs
type HealthChecker struct {
	client *http.Client
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// rubygems owner response struct
type rubygemsOwner struct {
	Handle string `json:"handle"`
	Role   string `json:"role"`
}

// github repo response struct
type githubRepo struct {
	StargazersCount int       `json:"stargazers_count"`
	OpenIssuesCount int       `json:"open_issues_count"`
	PushedAt        time.Time `json:"pushed_at"`
	Archived        bool      `json:"archived"`
	Disabled        bool      `json:"disabled"`
}

// FetchHealth fetches health data for a gem from RubyGems and GitHub
// Returns (*GemHealth, error). If GitHub rate limited, returns partial data with RateLimited=true
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
	githubURI := sourceCodeURI
	if githubURI == "" {
		githubURI = homepageURI
	}
	if githubURI != "" {
		owner, repo, ok := ExtractGitHubOwnerRepo(githubURI)
		if ok {
			githubHealth, rateLimited := hc.fetchGitHubRepo(owner, repo)
			if rateLimited {
				health.RateLimited = true
			} else if githubHealth != nil {
				health.GitHubPushedAt = githubHealth.PushedAt
				health.Stars = githubHealth.StargazersCount
				health.OpenIssues = githubHealth.OpenIssuesCount
				health.Archived = githubHealth.Archived
				health.Disabled = githubHealth.Disabled
			} else if !rateLimited {
				// GitHub fetch failed (non-rate-limit error)
				logger.Warn("Failed to fetch GitHub data for gem %q (owner=%s, repo=%s)", gemName, owner, repo)
			}
		} else {
			// Could not extract GitHub owner/repo
			logger.Warn("Could not extract GitHub owner/repo from URI for gem %q: %s", gemName, githubURI)
		}
	}

	health.Score = ComputeHealthScore(health)
	return health, nil
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

// ComputeHealthScore computes the health score based on available data
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

// ExtractGitHubOwnerRepo extracts GitHub owner and repo from source URIs
// Handles: https://github.com/owner/repo, https://github.com/owner/repo.git, http://github.com/owner/repo, etc.
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
