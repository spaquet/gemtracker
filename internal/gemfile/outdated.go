package gemfile

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RubygemeInfo represents gem data from rubygems.org API
type RubygemeInfo struct {
	Version       string `json:"version"`
	HomepageURI   string `json:"homepage_uri"`
	SourceCodeURI string `json:"source_code_uri"`
	Info          string `json:"info"`
}

// OutdatedChecker checks if gems are outdated by querying rubygems.org
type OutdatedChecker struct {
	client       *http.Client
	cache        map[string]string // gem name -> latest version
	homepages    map[string]string // gem name -> homepage URL
	descriptions map[string]string // gem name -> description
}

// NewOutdatedChecker creates a new checker with HTTP client
func NewOutdatedChecker() *OutdatedChecker {
	return &OutdatedChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:        make(map[string]string),
		homepages:    make(map[string]string),
		descriptions: make(map[string]string),
	}
}

// IsOutdated checks if a gem version is outdated and returns the latest version
func (oc *OutdatedChecker) IsOutdated(gemName, currentVersion string) (bool, string) {
	// Get latest version from cache or API
	latestVersion, err := oc.getLatestVersion(gemName)
	if err != nil {
		// If we can't check, assume it's not outdated
		return false, ""
	}

	// Compare versions: if current is different from latest, it's outdated
	isOutdated := currentVersion != latestVersion && isVersionLess(currentVersion, latestVersion)
	return isOutdated, latestVersion
}

// getLatestVersion fetches the latest version of a gem from rubygems.org
func (oc *OutdatedChecker) getLatestVersion(gemName string) (string, error) {
	// Check cache first
	if cached, ok := oc.cache[gemName]; ok {
		return cached, nil
	}

	// Query rubygems.org API
	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", gemName)
	resp, err := oc.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch gem info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gem not found on rubygems.org")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var info RubygemeInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the result
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

	return info.Version, nil
}

// GetHomepage returns the homepage URL for a gem, using cached data or fetching if needed
func (oc *OutdatedChecker) GetHomepage(gemName string) string {
	// If we have it cached, return it
	if url, ok := oc.homepages[gemName]; ok {
		return url
	}

	// Fetch it (this will populate the cache as a side effect)
	oc.getLatestVersion(gemName)

	// Return cached value or fallback
	if url, ok := oc.homepages[gemName]; ok {
		return url
	}

	// Ultimate fallback
	return fmt.Sprintf("https://rubygems.org/gems/%s", gemName)
}

// GetDescription returns the description for a gem, using cached data or fetching if needed
func (oc *OutdatedChecker) GetDescription(gemName string) string {
	// If we have it cached, return it
	if desc, ok := oc.descriptions[gemName]; ok {
		return desc
	}

	// Fetch it (this will populate the cache as a side effect)
	oc.getLatestVersion(gemName)

	// Return cached value or empty string
	if desc, ok := oc.descriptions[gemName]; ok {
		return desc
	}

	return ""
}

// isVersionLess compares two semantic versions
// Returns true if v1 < v2 (simplified comparison)
func isVersionLess(v1, v2 string) bool {
	// Split versions into parts
	parts1 := strings.Split(strings.Split(v1, "-")[0], ".")
	parts2 := strings.Split(strings.Split(v2, "-")[0], ".")

	// Compare major, minor, patch
	for i := 0; i < 3 && i < len(parts1) && i < len(parts2); i++ {
		var num1, num2 int
		fmt.Sscanf(parts1[i], "%d", &num1)
		fmt.Sscanf(parts2[i], "%d", &num2)

		if num1 < num2 {
			return true
		}
		if num1 > num2 {
			return false
		}
	}

	return false
}
