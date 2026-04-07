package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spaquet/gemtracker/internal/gemfile"
)

func TestReadWriteHealthCache(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Create test health data
	entry := &HealthCacheEntry{
		Gems: map[string]*gemfile.GemHealth{
			"rails": {
				Score:           gemfile.HealthHealthy,
				LastRelease:     time.Now(),
				MaintainerCount: 5,
				Stars:           50000,
			},
			"bundler": {
				Score:           gemfile.HealthWarning,
				LastRelease:     time.Now().AddDate(-2, 0, 0),
				MaintainerCount: 1,
				Stars:           15000,
			},
		},
	}

	// Test WriteHealth
	err = WriteHealth(gemfileLockPath, entry)
	if err != nil {
		t.Fatalf("WriteHealth() failed: %v", err)
	}

	// Test ReadHealth
	readEntry, err := ReadHealth(gemfileLockPath)
	if err != nil {
		t.Fatalf("ReadHealth() failed: %v", err)
	}

	if readEntry == nil {
		t.Fatal("ReadHealth() returned nil entry")
	}

	// Verify data
	if len(readEntry.Gems) != 2 {
		t.Fatalf("Expected 2 gems, got %d", len(readEntry.Gems))
	}

	if rails, ok := readEntry.Gems["rails"]; !ok {
		t.Fatal("rails gem not found in cache")
	} else if rails.Score != gemfile.HealthHealthy {
		t.Errorf("rails health mismatch: %v", rails.Score)
	} else if rails.MaintainerCount != 5 {
		t.Errorf("rails maintainer count mismatch: %d", rails.MaintainerCount)
	}

	if bundler, ok := readEntry.Gems["bundler"]; !ok {
		t.Fatal("bundler gem not found in cache")
	} else if bundler.Score != gemfile.HealthWarning {
		t.Errorf("bundler health mismatch: %v", bundler.Score)
	}
}

func TestReadHealthNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Try to read non-existent health cache
	entry, err := ReadHealth(gemfileLockPath)
	if err == nil {
		t.Fatal("ReadHealth() should fail for non-existent cache")
	}

	if entry != nil {
		t.Fatal("ReadHealth() should return nil for non-existent cache")
	}
}

func TestHealthCacheTTL(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Create test health data
	entry := &HealthCacheEntry{
		Gems: map[string]*gemfile.GemHealth{
			"rails": {
				Score: gemfile.HealthHealthy,
			},
		},
		CachedAt: time.Now().Add(-13 * 24 * time.Hour), // Older than TTL
	}

	// Write cache directly to set old CachedAt time
	cachePath, err := getHealthCachePath(gemfileLockPath)
	if err != nil {
		t.Fatalf("getHealthCachePath() failed: %v", err)
	}

	// Ensure cache dir exists
	cacheDir := filepath.Dir(cachePath)
	os.MkdirAll(cacheDir, 0755)

	// Write directly without using WriteHealth (which updates CachedAt)
	data, _ := json.MarshalIndent(entry, "", "  ")
	err = os.WriteFile(cachePath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test cache: %v", err)
	}

	// Try to read - should fail because cache is expired
	readEntry, err := ReadHealth(gemfileLockPath)
	if err == nil {
		t.Fatal("ReadHealth() should fail for expired cache")
	}

	if readEntry != nil {
		t.Fatal("ReadHealth() should return nil for expired cache")
	}
}

func TestClearHealth(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Write health cache
	entry := &HealthCacheEntry{
		Gems: map[string]*gemfile.GemHealth{
			"rails": {
				Score: gemfile.HealthHealthy,
			},
		},
	}

	err = WriteHealth(gemfileLockPath, entry)
	if err != nil {
		t.Fatalf("WriteHealth() failed: %v", err)
	}

	// Verify cache exists
	readEntry, err := ReadHealth(gemfileLockPath)
	if err != nil {
		t.Fatal("ReadHealth() failed after WriteHealth()")
	}

	if readEntry == nil {
		t.Fatal("ReadHealth() should return valid entry")
	}

	// Clear cache
	err = ClearHealth(gemfileLockPath)
	if err != nil {
		t.Fatalf("ClearHealth() failed: %v", err)
	}

	// Verify cache is cleared
	readEntry, err = ReadHealth(gemfileLockPath)
	if err == nil {
		t.Fatal("ReadHealth() should fail after ClearHealth()")
	}

	if readEntry != nil {
		t.Fatal("ReadHealth() should return nil after ClearHealth()")
	}
}

func TestGetHealthCachePath(t *testing.T) {
	gemfileLockPath := "Gemfile.lock"

	cachePath, err := getHealthCachePath(gemfileLockPath)
	if err != nil {
		t.Fatalf("getHealthCachePath() failed: %v", err)
	}

	if cachePath == "" {
		t.Fatal("getHealthCachePath() returned empty string")
	}

	// Check that it ends with _health.json
	if !strings.HasSuffix(cachePath, "_health.json") {
		t.Errorf("Health cache path doesn't end with _health.json: %s", cachePath)
	}

	// Check that it's in the cache directory
	cacheDir, _ := GetCacheDir()
	if !strings.HasPrefix(cachePath, cacheDir) {
		t.Errorf("Health cache path not in cache directory: %s", cachePath)
	}
}

func TestHealthCacheEmptyGems(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Create empty health data
	entry := &HealthCacheEntry{
		Gems: make(map[string]*gemfile.GemHealth),
	}

	// Write and read
	err = WriteHealth(gemfileLockPath, entry)
	if err != nil {
		t.Fatalf("WriteHealth() failed: %v", err)
	}

	readEntry, err := ReadHealth(gemfileLockPath)
	if err != nil {
		t.Fatalf("ReadHealth() failed: %v", err)
	}

	if readEntry == nil {
		t.Fatal("ReadHealth() returned nil entry")
	}

	if len(readEntry.Gems) != 0 {
		t.Errorf("Expected empty gems map, got %d", len(readEntry.Gems))
	}
}

func TestHealthCacheMultipleGems(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Create health data with multiple gems
	gems := map[string]*gemfile.GemHealth{
		"rails":      {Score: gemfile.HealthHealthy},
		"bundler":    {Score: gemfile.HealthWarning},
		"rspec":      {Score: gemfile.HealthCritical},
		"puma":       {Score: gemfile.HealthHealthy},
		"postgresql": {Score: gemfile.HealthHealthy},
	}

	entry := &HealthCacheEntry{
		Gems: gems,
	}

	err = WriteHealth(gemfileLockPath, entry)
	if err != nil {
		t.Fatalf("WriteHealth() failed: %v", err)
	}

	readEntry, err := ReadHealth(gemfileLockPath)
	if err != nil {
		t.Fatalf("ReadHealth() failed: %v", err)
	}

	if len(readEntry.Gems) != 5 {
		t.Fatalf("Expected 5 gems, got %d", len(readEntry.Gems))
	}

	// Verify all gems are present
	expectedGems := []string{"rails", "bundler", "rspec", "puma", "postgresql"}
	for _, gemName := range expectedGems {
		if _, ok := readEntry.Gems[gemName]; !ok {
			t.Errorf("Gem %s not found in cache", gemName)
		}
	}
}
