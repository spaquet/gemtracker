package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/spaquet/gemtracker/internal/gemfile"
)

// CacheEntry represents a cached analysis result with metadata
type CacheEntry struct {
	Result            *gemfile.AnalysisResult `json:"result"`
	GemfileLockMtime  int64                   `json:"gemfile_lock_mtime"`
	CachedAt          time.Time               `json:"cached_at"`
	RubyVersion       string                  `json:"ruby_version"`
	BundleVersion     string                  `json:"bundle_version"`
	FrameworkDetected string                  `json:"framework_detected"`
	RailsVersion      string                  `json:"rails_version"`
}

// GetCacheDir returns the cache directory path
func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(homeDir, ".cache", "gemtracker")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	return cacheDir, nil
}

// GetCachePath returns the cache file path for a given Gemfile.lock
func GetCachePath(gemfileLockPath string) (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	// Use hash of the path as cache filename
	hash := hashPath(gemfileLockPath)
	return filepath.Join(cacheDir, hash+".json"), nil
}

// Read reads cached analysis result if it exists and is valid
func Read(gemfileLockPath string) (*CacheEntry, error) {
	cachePath, err := GetCachePath(gemfileLockPath)
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

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Check if Gemfile.lock has been modified
	info, err := os.Stat(gemfileLockPath)
	if err != nil {
		return nil, err
	}

	if info.ModTime().Unix() != entry.GemfileLockMtime {
		// File has been modified, cache is invalid
		return nil, os.ErrNotExist
	}

	return &entry, nil
}

// Write writes analysis result to cache
func Write(gemfileLockPath string, entry *CacheEntry) error {
	// Update mtime and cached time
	info, err := os.Stat(gemfileLockPath)
	if err != nil {
		return err
	}

	entry.GemfileLockMtime = info.ModTime().Unix()
	entry.CachedAt = time.Now()

	cachePath, err := GetCachePath(gemfileLockPath)
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

// Clear removes the cache entry for a given Gemfile.lock
func Clear(gemfileLockPath string) error {
	cachePath, err := GetCachePath(gemfileLockPath)
	if err != nil {
		return err
	}

	return os.Remove(cachePath)
}

// hashPath creates a hash of the file path for use in cache filenames
func hashPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	// Use SHA256 for deterministic, readable filenames
	hash := sha256.Sum256([]byte(abs))
	hashStr := hex.EncodeToString(hash[:])[:8] // Use first 8 hex chars for brevity

	return filepath.Base(abs) + "_" + hashStr
}
