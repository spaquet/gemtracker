package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/spaquet/gemtracker/internal/gemfile"
)

// HealthCacheTTL is the time-to-live for cached health data
// Health metrics change on a "years" timescale, so 12 days is conservative
// while drastically reducing API calls on subsequent runs
const HealthCacheTTL = 12 * 24 * time.Hour

// HealthCacheEntry stores gem health data with a 12-day TTL
type HealthCacheEntry struct {
	Gems     map[string]*gemfile.GemHealth `json:"gems"`
	CachedAt time.Time                     `json:"cached_at"`
}

// ReadHealth reads health cache for a Gemfile.lock if it exists and is less than 12 days old
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

// WriteHealth writes health data to cache
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

// ClearHealth removes the health cache entry for a given Gemfile.lock
func ClearHealth(gemfileLockPath string) error {
	cachePath, err := getHealthCachePath(gemfileLockPath)
	if err != nil {
		return err
	}

	return os.Remove(cachePath)
}

// getHealthCachePath returns the health cache file path
func getHealthCachePath(gemfileLockPath string) (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	hash := hashPath(gemfileLockPath)
	return filepath.Join(cacheDir, hash+"_health.json"), nil
}
