package cache

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spaquet/gemtracker/internal/gemfile"
)

func TestGetCacheDir(t *testing.T) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() failed: %v", err)
	}

	if cacheDir == "" {
		t.Fatal("GetCacheDir() returned empty string")
	}

	// Check that directory was created
	info, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("Cache directory does not exist: %v", err)
	}

	if !info.IsDir() {
		t.Fatal("Cache path is not a directory")
	}

	// Verify it contains ".cache/gemtracker"
	if !strings.HasSuffix(cacheDir, filepath.Join(".cache", "gemtracker")) {
		t.Fatalf("Cache directory doesn't have expected path suffix: %s", cacheDir)
	}
}

func TestHashPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantBase string // Expected base filename
	}{
		{
			name:     "Simple filename",
			path:     "Gemfile.lock",
			wantBase: "Gemfile.lock",
		},
		{
			name:     "Path with directory",
			path:     "/path/to/project/Gemfile.lock",
			wantBase: "Gemfile.lock",
		},
		{
			name:     "Relative path",
			path:     "subdir/Gemfile.lock",
			wantBase: "Gemfile.lock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashPath(tt.path)

			// Check that base filename is preserved
			if !strings.HasPrefix(hash, tt.wantBase) {
				t.Errorf("hashPath(%q) doesn't start with %q, got %q", tt.path, tt.wantBase, hash)
			}

			// Check that hash is in expected format: "filename_8hexchars"
			parts := filepath.SplitList(hash)
			lastPart := parts[len(parts)-1]
			if len(lastPart) < len(tt.wantBase)+9 { // filename + "_" + 8 chars
				t.Errorf("hashPath(%q) returned too short result: %q", tt.path, hash)
			}
		})
	}

	// Test determinism: same input should produce same hash
	path := "/path/to/some/Gemfile.lock"
	hash1 := hashPath(path)
	hash2 := hashPath(path)
	if hash1 != hash2 {
		t.Errorf("hashPath is not deterministic: %s != %s", hash1, hash2)
	}

	// Test uniqueness: different paths should produce different hashes
	hash3 := hashPath("/different/path/Gemfile.lock")
	if hash1 == hash3 {
		t.Errorf("hashPath produced same hash for different paths")
	}
}

func TestGetCachePath(t *testing.T) {
	gemfileLockPath := "Gemfile.lock"

	cachePath, err := GetCachePath(gemfileLockPath)
	if err != nil {
		t.Fatalf("GetCachePath() failed: %v", err)
	}

	if cachePath == "" {
		t.Fatal("GetCachePath() returned empty string")
	}

	// Check that it's in the cache directory
	cacheDir, _ := GetCacheDir()
	if !strings.HasPrefix(cachePath, cacheDir) {
		t.Errorf("Cache path not in cache directory: %s", cachePath)
	}

	// Check that it ends with .json
	if !strings.HasSuffix(cachePath, ".json") {
		t.Errorf("Cache path doesn't end with .json: %s", cachePath)
	}
}

func TestReadWriteClear(t *testing.T) {
	// Create a temporary Gemfile.lock for testing
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Create a test analysis result
	entry := &CacheEntry{
		Result: &gemfile.AnalysisResult{
			FirstLevelGems: []string{"rails", "bundler"},
			TotalGems:      10,
		},
		RubyVersion:   "3.2.0",
		BundleVersion: "2.4.0",
	}

	// Test Write
	err = Write(gemfileLockPath, entry)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Test Read
	readEntry, err := Read(gemfileLockPath)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if readEntry == nil {
		t.Fatal("Read() returned nil entry")
	}

	// Verify cached data
	if readEntry.RubyVersion != entry.RubyVersion {
		t.Errorf("RubyVersion mismatch: %s != %s", readEntry.RubyVersion, entry.RubyVersion)
	}

	if readEntry.BundleVersion != entry.BundleVersion {
		t.Errorf("BundleVersion mismatch: %s != %s", readEntry.BundleVersion, entry.BundleVersion)
	}

	if readEntry.Result == nil {
		t.Fatal("Cached result is nil")
	}

	if len(readEntry.Result.FirstLevelGems) != 2 {
		t.Fatalf("Expected 2 gems, got %d", len(readEntry.Result.FirstLevelGems))
	}

	if readEntry.Result.FirstLevelGems[0] != "rails" {
		t.Errorf("Gem name mismatch: %s", readEntry.Result.FirstLevelGems[0])
	}

	// Test Clear
	err = Clear(gemfileLockPath)
	if err != nil {
		t.Fatalf("Clear() failed: %v", err)
	}

	// Verify cache is cleared
	readEntry, err = Read(gemfileLockPath)
	if err == nil {
		t.Fatal("Read() should fail after Clear()")
	}

	if readEntry != nil {
		t.Fatal("Read() should return nil after Clear()")
	}
}

func TestReadInvalidCache(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Try to read non-existent cache
	entry, err := Read(gemfileLockPath)
	if err == nil {
		t.Fatal("Read() should fail for non-existent cache")
	}

	if entry != nil {
		t.Fatal("Read() should return nil for non-existent cache")
	}
}

func TestReadStaleCache(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock
	err := os.WriteFile(gemfileLockPath, []byte("original"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Write cache
	entry := &CacheEntry{
		Result: &gemfile.AnalysisResult{},
	}
	err = Write(gemfileLockPath, entry)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Modify the Gemfile.lock file (change mtime)
	time.Sleep(1100 * time.Millisecond) // Wait > 1 second to ensure mtime changes
	err = os.WriteFile(gemfileLockPath, []byte("modified"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify Gemfile.lock: %v", err)
	}

	// Try to read cache - should fail because file was modified
	readEntry, err := Read(gemfileLockPath)
	if err == nil {
		t.Fatal("Read() should fail for modified Gemfile.lock")
	}

	if readEntry != nil {
		t.Fatal("Read() should return nil for modified Gemfile.lock")
	}
}

func TestHashPathEdgeCases(t *testing.T) {
	// Test with absolute path that should work
	abs := "/absolute/path/Gemfile.lock"
	hash1 := hashPath(abs)
	if len(hash1) == 0 {
		t.Fatal("hashPath returned empty string for absolute path")
	}

	// Verify hash format: "Gemfile.lock_" + 8 hex chars
	if len(hash1) < 21 { // len("Gemfile.lock_") + 8
		t.Errorf("hashPath returned string too short: %s", hash1)
	}

	// Verify it has the underscore separator
	if !strings.HasPrefix(hash1, "Gemfile.lock_") {
		t.Errorf("hashPath doesn't have expected prefix: %s", hash1)
	}

	// Extract and verify the hash part is valid hex
	parts := filepath.SplitList(hash1)
	lastPart := parts[len(parts)-1]
	hashPart := lastPart[len("Gemfile.lock_"):]
	if len(hashPart) != 8 {
		t.Errorf("Hash part has wrong length: %d", len(hashPart))
	}

	_, err := hex.DecodeString(hashPart)
	if err != nil {
		t.Errorf("Hash part is not valid hex: %s", hashPart)
	}
}

func TestCacheEntryMtime(t *testing.T) {
	tmpDir := t.TempDir()
	gemfileLockPath := filepath.Join(tmpDir, "Gemfile.lock")

	// Create a test Gemfile.lock with specific content
	originalContent := []byte("test content")
	err := os.WriteFile(gemfileLockPath, originalContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test Gemfile.lock: %v", err)
	}

	// Get the original mtime
	info, _ := os.Stat(gemfileLockPath)
	originalMtime := info.ModTime().Unix()

	// Write cache
	entry := &CacheEntry{
		Result: &gemfile.AnalysisResult{},
	}
	err = Write(gemfileLockPath, entry)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Read cache and verify mtime is recorded
	readEntry, err := Read(gemfileLockPath)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if readEntry.GemfileLockMtime != originalMtime {
		t.Errorf("Mtime mismatch: %d != %d", readEntry.GemfileLockMtime, originalMtime)
	}
}
