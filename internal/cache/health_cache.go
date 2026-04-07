// Health cache handling is included in the cache package.
// See the cache package documentation for overall caching strategy.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/spaquet/gemtracker/internal/gemfile"
)

// HealthCacheTTL is the time-to-live for cached health data (12 days).
// Health metrics (last release, maintainer count, activity) change on a yearly timescale,
// so this conservative value drastically reduces API calls while remaining accurate.
const HealthCacheTTL = 12 * 24 * time.Hour

// HealthCacheEntry stores cached gem health metrics for a Gemfile.lock with a 12-day time-to-live.
// It maps gem names to their cached health data including maintenance status and metrics.
type HealthCacheEntry struct {
	Gems     map[string]*gemfile.GemHealth `json:"gems"`
	CachedAt time.Time                     `json:"cached_at"`
}

// ReadHealth reads and returns the cached gem health data for a Gemfile.lock if it exists
// and is still valid (less than HealthCacheTTL old). Returns nil if the cache file doesn't exist
// or has expired. Returns an error if the cache file cannot be read or parsed.
func ReadHealth(gemfileLockPath string) (*HealthCacheEntry, error) {
	cachePath, err := getHealthCachePath(gemfileLockPath)
	if err != nil {
		return nil, err
	}

	// Check if cache file exists
	_, err = os.Stat(cachePath)
	if err != nil {
		return nil, err
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var entry HealthCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Check if cache is older than HealthCacheTTL
	if time.Since(entry.CachedAt) > HealthCacheTTL {
		return nil, os.ErrNotExist
	}

	return &entry, nil
}

// WriteHealth writes gem health data to cache with the current timestamp.
// Returns an error if the cache file cannot be written.
func WriteHealth(gemfileLockPath string, entry *HealthCacheEntry) error {
	entry.CachedAt = time.Now()

	cachePath, err := getHealthCachePath(gemfileLockPath)
	if err != nil {
		return err
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	// Write to cache file
	return os.WriteFile(cachePath, data, 0644)
}

// ClearHealth removes the health cache entry for a given Gemfile.lock.
// Returns an error if the cache file cannot be deleted.
func ClearHealth(gemfileLockPath string) error {
	cachePath, err := getHealthCachePath(gemfileLockPath)
	if err != nil {
		return err
	}

	return os.Remove(cachePath)
}

// getHealthCachePath returns the health cache file path for a given Gemfile.lock.
// The filename is based on the Gemfile.lock path hash with a "_health.json" suffix.
func getHealthCachePath(gemfileLockPath string) (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	hash := hashPath(gemfileLockPath)
	return filepath.Join(cacheDir, hash+"_health.json"), nil
}
