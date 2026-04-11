package gemfile

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/spaquet/gemtracker/internal/logger"
)

// RubygemeInfo represents gem metadata from the rubygems.org API.
type RubygemeInfo struct {
	// Version is the latest available version of the gem
	Version string `json:"version"`
	// VersionCreatedAt is the timestamp when the latest version was released
	VersionCreatedAt string `json:"version_created_at"`
	// HomepageURI is the gem's official homepage URL
	HomepageURI string `json:"homepage_uri"`
	// SourceCodeURI is the source code repository URL
	SourceCodeURI string `json:"source_code_uri"`
	// Info is the gem's description
	Info string `json:"info"`
}

// RubygemesDependency represents a dependency returned by the RubyGems dependencies API.
type RubygemesDependency struct {
	// Name is the name of the dependency gem
	Name string `json:"name"`
	// Requirements describes the version constraint(s) for this dependency
	Requirements interface{} `json:"requirements"`
}

// DependencyType distinguishes between runtime and development dependencies.
type DependencyType int

const (
	// RuntimeDependency is a production runtime dependency
	RuntimeDependency DependencyType = iota
	// DevelopmentDependency is a development-only dependency
	DevelopmentDependency
)

// OutdatedChecker checks if gems have newer versions available and fetches gem metadata
// from the rubygems.org API. It caches all results to minimize API calls.
type OutdatedChecker struct {
	// client is the HTTP client used for API requests
	client *http.Client
	// mu protects all maps below
	mu sync.Mutex
	// cache maps gem names to their latest available versions
	cache map[string]string
	// homepages maps gem names to their homepage URLs
	homepages map[string]string
	// descriptions maps gem names to their gem descriptions
	descriptions map[string]string
	// sourceCodeURIs maps gem names to source code repository URLs
	sourceCodeURIs map[string]string
	// versionCreatedAts maps gem names to the release timestamp of the latest version
	versionCreatedAts map[string]string
	// dependencies maps gem names to their dependency lists (for gemspec enrichment)
	dependencies map[string][]string
}

// NewOutdatedChecker creates a new OutdatedChecker with a 10-second HTTP timeout
// and empty caches for gem metadata.
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
		dependencies:      make(map[string][]string),
	}
}

// IsOutdated checks if a gem has a newer version available. It handles platform suffixes
// and pre-release versions correctly (e.g., "1.6.3-x86_64-linux" is compared as "1.6.3").
// Returns (isOutdated, latestVersion, error).
// If a gem is not found on rubygems.org (404), returns (false, "", nil) - not an error.
func (oc *OutdatedChecker) IsOutdated(gemName, currentVersion string) (bool, string, error) {
	// Get latest version from cache or API
	latestVersion, err := oc.getLatestVersion(gemName)
	if err != nil {
		// Return error instead of silently failing
		return false, "", err
	}

	// If gem not found on rubygems.org (latestVersion is empty), it's not outdated
	// This is normal for local gems, git sources, or removed gems
	if latestVersion == "" {
		return false, "", nil
	}

	// Normalize both versions by stripping platform suffixes before comparison
	// This handles native gem versions like "1.6.3-x86_64-linux" vs "1.6.3"
	cleanCurrent := stripPlatformSuffix(currentVersion)
	cleanLatest := stripPlatformSuffix(latestVersion)

	// Compare versions: if current is different from latest, it's outdated
	isOutdated := cleanCurrent != cleanLatest && isVersionLess(cleanCurrent, cleanLatest)
	return isOutdated, latestVersion, nil
}

// getLatestVersion fetches the latest version and metadata for a gem from the rubygems.org API.
// It caches all results, so subsequent calls for the same gem are instant. Returns an error
// if the gem is not found or if the API request fails.
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

	// Handle 404 gracefully - gem not found on rubygems.org
	// This is normal for local gems, git source gems, or removed gems
	if resp.StatusCode == http.StatusNotFound {
		return "", nil // Return empty string, not an error
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

// GetHomepage returns the homepage URL for a gem from cache or fetches it if not cached.
// Returns a fallback URL to rubygems.org if no homepage is available.
func (oc *OutdatedChecker) GetHomepage(gemName string) string {
	// If we have it cached, return it
	oc.mu.Lock()
	if url, ok := oc.homepages[gemName]; ok {
		oc.mu.Unlock()
		return url
	}
	oc.mu.Unlock()

	// Fetch it (this will populate the cache as a side effect)
	if _, err := oc.getLatestVersion(gemName); err != nil {
		logger.Warn("Failed to fetch homepage for gem %q: %v", gemName, err)
	}

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

// GetDescription returns the gem's description from cache or fetches it if not cached.
// Returns an empty string if no description is available.
func (oc *OutdatedChecker) GetDescription(gemName string) string {
	// If we have it cached, return it
	oc.mu.Lock()
	if desc, ok := oc.descriptions[gemName]; ok {
		oc.mu.Unlock()
		return desc
	}
	oc.mu.Unlock()

	// Fetch it (this will populate the cache as a side effect)
	if _, err := oc.getLatestVersion(gemName); err != nil {
		logger.Warn("Failed to fetch description for gem %q: %v", gemName, err)
	}

	// Return cached value or empty string
	oc.mu.Lock()
	if desc, ok := oc.descriptions[gemName]; ok {
		oc.mu.Unlock()
		return desc
	}
	oc.mu.Unlock()

	return ""
}

// GetSourceCodeURI returns the source code repository URL for a gem from cache or fetches it if not cached.
// Returns an empty string if no source code URI is available.
func (oc *OutdatedChecker) GetSourceCodeURI(gemName string) string {
	// If we have it cached, return it
	oc.mu.Lock()
	if uri, ok := oc.sourceCodeURIs[gemName]; ok {
		oc.mu.Unlock()
		return uri
	}
	oc.mu.Unlock()

	// Fetch it (this will populate the cache as a side effect)
	if _, err := oc.getLatestVersion(gemName); err != nil {
		logger.Warn("Failed to fetch source code URI for gem %q: %v", gemName, err)
	}

	// Return cached value or empty string
	oc.mu.Lock()
	if uri, ok := oc.sourceCodeURIs[gemName]; ok {
		oc.mu.Unlock()
		return uri
	}
	oc.mu.Unlock()

	return ""
}

// GetVersionCreatedAt returns the release timestamp of the latest version from cache or fetches it if not cached.
// Returns an empty string if the timestamp is not available.
func (oc *OutdatedChecker) GetVersionCreatedAt(gemName string) string {
	// If we have it cached, return it
	oc.mu.Lock()
	if ts, ok := oc.versionCreatedAts[gemName]; ok {
		oc.mu.Unlock()
		return ts
	}
	oc.mu.Unlock()

	// Fetch it (this will populate the cache as a side effect)
	if _, err := oc.getLatestVersion(gemName); err != nil {
		logger.Warn("Failed to fetch version created at for gem %q: %v", gemName, err)
	}

	// Return cached value or empty string
	oc.mu.Lock()
	if ts, ok := oc.versionCreatedAts[gemName]; ok {
		oc.mu.Unlock()
		return ts
	}
	oc.mu.Unlock()

	return ""
}

// EnrichGemspecDependencies fetches runtime and development dependencies from RubyGems for each
// gem in the parsed gemspec file. This resolves the full dependency tree that was not available
// in the gemspec file itself. Dependencies are fetched in parallel with rate limiting to avoid
// overwhelming the RubyGems API. Returns the number of gems successfully enriched.
func (oc *OutdatedChecker) EnrichGemspecDependencies(gf *Gemfile) int {
	if gf == nil || len(gf.Gems) == 0 {
		return 0
	}

	enrichedCount := 0

	// Fetch dependencies for each gem
	for gemName := range gf.Gems {
		deps, err := oc.fetchGemDependenciesFromAPI(gemName)
		if err != nil {
			logger.Info("Skipping dependency enrichment for %q (not found on RubyGems)", gemName)
			continue
		}

		if len(deps) > 0 {
			gf.Gems[gemName].Dependencies = deps
			enrichedCount++
		}
	}

	return enrichedCount
}

// fetchGemDependenciesFromAPI fetches the list of dependency gem names for a given gem
// from the RubyGems dependencies API. Returns cached results if available.
// Returns an empty list if the gem is not found or an error occurs.
func (oc *OutdatedChecker) fetchGemDependenciesFromAPI(gemName string) ([]string, error) {
	// Check cache first
	oc.mu.Lock()
	if cached, ok := oc.dependencies[gemName]; ok {
		oc.mu.Unlock()
		return cached, nil
	}
	oc.mu.Unlock()

	// Query the RubyGems dependencies API
	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s/dependencies", gemName)
	resp, err := oc.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dependencies: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting and not found
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429) by rubygems.org")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Gem not found, return empty list
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var deps []RubygemesDependency
	if err := json.Unmarshal(body, &deps); err != nil {
		return nil, fmt.Errorf("failed to parse dependencies: %w", err)
	}

	// Extract just the dependency names
	depNames := make([]string, 0, len(deps))
	for _, dep := range deps {
		if dep.Name != "" {
			depNames = append(depNames, strings.ToLower(dep.Name))
		}
	}

	// Cache the result
	oc.mu.Lock()
	oc.dependencies[gemName] = depNames
	oc.mu.Unlock()

	return depNames, nil
}

// stripPlatformSuffix removes platform/architecture suffixes from version strings while preserving
// pre-release identifiers (alpha, beta, rc, dev, etc.). This allows correct version comparison
// for native gem versions like "1.6.3-x86_64-linux" which should compare as "1.6.3".
// Examples: "1.6.3-x86_64-linux" -> "1.6.3", "1.0.0-beta.1" -> "1.0.0-beta.1"
func stripPlatformSuffix(version string) string {
	parts := strings.Split(version, "-")
	if len(parts) <= 1 {
		return version
	}

	// Known pre-release keywords that should be kept
	preReleaseKeywords := map[string]bool{
		"alpha": true, "a": true,
		"beta": true, "b": true,
		"rc": true, "release-candidate": true,
		"pre": true, "preview": true,
		"dev": true, "development": true,
		"snapshot": true,
	}

	// Check the part after the first dash
	suffix := strings.ToLower(parts[1])

	// Check if suffix contains any pre-release keywords
	isPreRelease := false
	for keyword := range preReleaseKeywords {
		if strings.Contains(suffix, keyword) {
			isPreRelease = true
			break
		}
	}

	// If it's a pre-release, keep it; otherwise discard it (platform suffix)
	if isPreRelease {
		return version // Keep the whole version including pre-release
	}

	return parts[0] // Return just the base version, discard platform suffix
}

// isVersionLess compares two semantic versions and returns true if v1 < v2.
// It handles pre-release versions (1.0.0-alpha < 1.0.0), platform suffixes (1.6.3-x86_64-linux),
// build metadata (version+build), and leading 'v' prefixes. Only compares major.minor.patch
// (first 3 numeric parts).
func isVersionLess(v1, v2 string) bool {
	// Normalize versions: remove leading 'v' if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Strip build metadata (everything after '+')
	v1 = strings.Split(v1, "+")[0]
	v2 = strings.Split(v2, "+")[0]

	// Strip platform suffixes (x86_64-linux, arm64-darwin, etc.)
	v1 = stripPlatformSuffix(v1)
	v2 = stripPlatformSuffix(v2)

	// Split on pre-release indicator
	v1Parts := strings.Split(v1, "-")
	v2Parts := strings.Split(v2, "-")

	v1Base := v1Parts[0]
	v2Base := v2Parts[0]

	// Compare base versions (major.minor.patch...)
	v1Nums := strings.Split(v1Base, ".")
	v2Nums := strings.Split(v2Base, ".")

	// Compare numeric parts (major.minor.patch only)
	maxLen := 3 // Only compare first 3 parts
	if len(v1Nums) < maxLen {
		maxLen = len(v1Nums)
	}
	if len(v2Nums) < maxLen {
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

	return false
}
