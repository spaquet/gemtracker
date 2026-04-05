package gemfile

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RubygemeInfo represents gem data from rubygems.org API
type RubygemeInfo struct {
	Version            string `json:"version"`
	VersionCreatedAt   string `json:"version_created_at"`
	HomepageURI        string `json:"homepage_uri"`
	SourceCodeURI      string `json:"source_code_uri"`
	Info               string `json:"info"`
}

// OutdatedChecker checks if gems are outdated by querying rubygems.org
type OutdatedChecker struct {
	client              *http.Client
	mu                  sync.Mutex        // protects all maps below
	cache               map[string]string // gem name -> latest version
	homepages           map[string]string // gem name -> homepage URL
	descriptions        map[string]string // gem name -> description
	sourceCodeURIs      map[string]string // gem name -> source code URI
	versionCreatedAts   map[string]string // gem name -> version created at
}

// NewOutdatedChecker creates a new checker with HTTP client
func NewOutdatedChecker() *OutdatedChecker {
	return &OutdatedChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:             make(map[string]string),
		homepages:         make(map[string]string),
		descriptions:      make(map[string]string),
		sourceCodeURIs:    make(map[string]string),
		versionCreatedAts: make(map[string]string),
	}
}

// IsOutdated checks if a gem version is outdated and returns the latest version and any error
func (oc *OutdatedChecker) IsOutdated(gemName, currentVersion string) (bool, string, error) {
	// Get latest version from cache or API
	latestVersion, err := oc.getLatestVersion(gemName)
	if err != nil {
		// Return error instead of silently failing
		return false, "", err
	}

	// Compare versions: if current is different from latest, it's outdated
	isOutdated := currentVersion != latestVersion && isVersionLess(currentVersion, latestVersion)
	return isOutdated, latestVersion, nil
}

// getLatestVersion fetches the latest version of a gem from rubygems.org
func (oc *OutdatedChecker) getLatestVersion(gemName string) (string, error) {
	// Check cache first (with lock)
	oc.mu.Lock()
	if cached, ok := oc.cache[gemName]; ok {
		oc.mu.Unlock()
		return cached, nil
	}
	oc.mu.Unlock()

	// Query rubygems.org API (without lock - don't hold lock during HTTP request)
	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", gemName)
	resp, err := oc.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch gem info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return "", fmt.Errorf("rate limited (429) by rubygems.org")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gem not found on rubygems.org (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var info RubygemeInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the result (with lock)
	oc.mu.Lock()
	oc.cache[gemName] = info.Version

	// Cache homepage URL with fallback chain
	homepage := info.HomepageURI
	if homepage == "" {
		homepage = info.SourceCodeURI
	}
	if homepage == "" {
		homepage = fmt.Sprintf("https://rubygems.org/gems/%s", gemName)
	}
	oc.homepages[gemName] = homepage

	// Cache description
	oc.descriptions[gemName] = info.Info

	// Cache source code URI and version created at for health checking
	oc.sourceCodeURIs[gemName] = info.SourceCodeURI
	oc.versionCreatedAts[gemName] = info.VersionCreatedAt
	oc.mu.Unlock()

	return info.Version, nil
}

// GetHomepage returns the homepage URL for a gem, using cached data or fetching if needed
func (oc *OutdatedChecker) GetHomepage(gemName string) string {
	// If we have it cached, return it
	oc.mu.Lock()
	if url, ok := oc.homepages[gemName]; ok {
		oc.mu.Unlock()
		return url
	}
	oc.mu.Unlock()

	// Fetch it (this will populate the cache as a side effect)
	oc.getLatestVersion(gemName)

	// Return cached value or fallback
	oc.mu.Lock()
	if url, ok := oc.homepages[gemName]; ok {
		oc.mu.Unlock()
		return url
	}
	oc.mu.Unlock()

	// Ultimate fallback
	return fmt.Sprintf("https://rubygems.org/gems/%s", gemName)
}

// GetDescription returns the description for a gem, using cached data or fetching if needed
func (oc *OutdatedChecker) GetDescription(gemName string) string {
	// If we have it cached, return it
	oc.mu.Lock()
	if desc, ok := oc.descriptions[gemName]; ok {
		oc.mu.Unlock()
		return desc
	}
	oc.mu.Unlock()

	// Fetch it (this will populate the cache as a side effect)
	oc.getLatestVersion(gemName)

	// Return cached value or empty string
	oc.mu.Lock()
	if desc, ok := oc.descriptions[gemName]; ok {
		oc.mu.Unlock()
		return desc
	}
	oc.mu.Unlock()

	return ""
}

// GetSourceCodeURI returns the source code URI for a gem, using cached data or fetching if needed
func (oc *OutdatedChecker) GetSourceCodeURI(gemName string) string {
	// If we have it cached, return it
	oc.mu.Lock()
	if uri, ok := oc.sourceCodeURIs[gemName]; ok {
		oc.mu.Unlock()
		return uri
	}
	oc.mu.Unlock()

	// Fetch it (this will populate the cache as a side effect)
	oc.getLatestVersion(gemName)

	// Return cached value or empty string
	oc.mu.Lock()
	if uri, ok := oc.sourceCodeURIs[gemName]; ok {
		oc.mu.Unlock()
		return uri
	}
	oc.mu.Unlock()

	return ""
}

// GetVersionCreatedAt returns the version created at timestamp for a gem, using cached data or fetching if needed
func (oc *OutdatedChecker) GetVersionCreatedAt(gemName string) string {
	// If we have it cached, return it
	oc.mu.Lock()
	if ts, ok := oc.versionCreatedAts[gemName]; ok {
		oc.mu.Unlock()
		return ts
	}
	oc.mu.Unlock()

	// Fetch it (this will populate the cache as a side effect)
	oc.getLatestVersion(gemName)

	// Return cached value or empty string
	oc.mu.Lock()
	if ts, ok := oc.versionCreatedAts[gemName]; ok {
		oc.mu.Unlock()
		return ts
	}
	oc.mu.Unlock()

	return ""
}

// isVersionLess compares two semantic versions
// Returns true if v1 < v2
// Handles pre-release versions (1.0.0-alpha < 1.0.0)
func isVersionLess(v1, v2 string) bool {
	// Normalize versions: remove leading 'v' if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Strip build metadata (everything after '+')
	v1 = strings.Split(v1, "+")[0]
	v2 = strings.Split(v2, "+")[0]

	// Split on pre-release indicator
	v1Parts := strings.Split(v1, "-")
	v2Parts := strings.Split(v2, "-")

	v1Base := v1Parts[0]
	v2Base := v2Parts[0]

	// Compare base versions (major.minor.patch...)
	v1Nums := strings.Split(v1Base, ".")
	v2Nums := strings.Split(v2Base, ".")

	// Compare numeric parts
	maxLen := len(v1Nums)
	if len(v2Nums) > maxLen {
		maxLen = len(v2Nums)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int

		if i < len(v1Nums) {
			fmt.Sscanf(v1Nums[i], "%d", &num1)
		}
		if i < len(v2Nums) {
			fmt.Sscanf(v2Nums[i], "%d", &num2)
		}

		if num1 < num2 {
			return true
		}
		if num1 > num2 {
			return false
		}
	}

	// Base versions are equal; if v1 has pre-release and v2 doesn't, v1 is less
	// e.g., 1.0.0-alpha < 1.0.0
	hasV1Prerelease := len(v1Parts) > 1
	hasV2Prerelease := len(v2Parts) > 1

	if hasV1Prerelease && !hasV2Prerelease {
		return true
	}
	if !hasV1Prerelease && hasV2Prerelease {
		return false
	}

	return false
}
