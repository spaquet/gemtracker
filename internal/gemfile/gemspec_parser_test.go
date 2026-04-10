package gemfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGemspec_BasicParsing(t *testing.T) {
	// Create a temporary gemspec file
	tmpDir := t.TempDir()
	gemspecPath := filepath.Join(tmpDir, "test.gemspec")

	gemspecContent := `Gem::Specification.new do |spec|
  spec.name          = "test-gem"
  spec.version       = "1.0.0"

  spec.add_runtime_dependency "rails", ">= 6.0"
  spec.add_runtime_dependency "pg", ">= 1.1"
  spec.add_development_dependency "rspec", "~> 3.0"
end
`

	if err := os.WriteFile(gemspecPath, []byte(gemspecContent), 0644); err != nil {
		t.Fatalf("Failed to write test gemspec: %v", err)
	}

	gf, err := ParseGemspec(gemspecPath)
	if err != nil {
		t.Fatalf("Failed to parse gemspec: %v", err)
	}

	// Verify we got the expected gems
	if len(gf.Gems) != 3 {
		t.Errorf("Expected 3 gems, got %d", len(gf.Gems))
	}

	// Check runtime dependencies
	assertGemExists(t, gf, "rails", ">= 6.0", true, "production")
	assertGemExists(t, gf, "pg", ">= 1.1", true, "production")
	assertGemExists(t, gf, "rspec", "~> 3.0", true, "development")

	// Verify FirstLevelGems is populated
	if len(gf.FirstLevelGems) != 3 {
		t.Errorf("Expected 3 first-level gems, got %d", len(gf.FirstLevelGems))
	}
}

func TestParseGemspec_WithSpecPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	gemspecPath := filepath.Join(tmpDir, "test.gemspec")

	// Test with spec. prefix (common pattern)
	gemspecContent := `Gem::Specification.new do |spec|
  spec.add_runtime_dependency "sinatra", ">= 2.0"
end
`

	if err := os.WriteFile(gemspecPath, []byte(gemspecContent), 0644); err != nil {
		t.Fatalf("Failed to write test gemspec: %v", err)
	}

	gf, err := ParseGemspec(gemspecPath)
	if err != nil {
		t.Fatalf("Failed to parse gemspec: %v", err)
	}

	if _, ok := gf.Gems["sinatra"]; !ok {
		t.Error("Expected 'sinatra' gem to be parsed from spec. prefix")
	}
}

func TestParseGemspec_WithSPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	gemspecPath := filepath.Join(tmpDir, "test.gemspec")

	// Test with s. prefix (most common pattern in real gemspecs)
	gemspecContent := `Gem::Specification.new do |s|
  s.name = "example-gem"
  s.version = "1.0.0"
  s.add_dependency "rails", "~> 7.0"
  s.add_dependency "sqlite3", "~> 1.4"
  s.add_development_dependency "rspec", "~> 3.0"
end
`

	if err := os.WriteFile(gemspecPath, []byte(gemspecContent), 0644); err != nil {
		t.Fatalf("Failed to write test gemspec: %v", err)
	}

	gf, err := ParseGemspec(gemspecPath)
	if err != nil {
		t.Fatalf("Failed to parse gemspec: %v", err)
	}

	// Should parse all 3 gems
	if len(gf.Gems) != 3 {
		t.Errorf("Expected 3 gems from s. prefix, got %d", len(gf.Gems))
	}

	// Check dependencies
	assertGemExists(t, gf, "rails", "~> 7.0", true, "production")
	assertGemExists(t, gf, "sqlite3", "~> 1.4", true, "production")
	assertGemExists(t, gf, "rspec", "~> 3.0", true, "development")
}

func TestParseGemspec_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	gemspecPath := filepath.Join(tmpDir, "example.gemspec")

	gemspecContent := "Gem::Specification.new do |spec|\nend\n"
	if err := os.WriteFile(gemspecPath, []byte(gemspecContent), 0644); err != nil {
		t.Fatalf("Failed to write test gemspec: %v", err)
	}

	// Pass directory, not file
	gf, err := ParseGemspec(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse gemspec from directory: %v", err)
	}

	if gf == nil {
		t.Fatal("Expected non-nil result")
	}
}

// assertGemExists checks if a gem exists with the expected properties.
func assertGemExists(t *testing.T, gf *Gemfile, name, version string, isFirstLevel bool, expectedGroup string) {
	t.Helper()

	gem, ok := gf.Gems[name]
	if !ok {
		t.Errorf("Expected '%s' gem to be parsed", name)
		return
	}

	if gem.Version != version {
		t.Errorf("Expected %s version '%s', got '%s'", name, version, gem.Version)
	}

	if gem.IsFirstLevel != isFirstLevel {
		t.Errorf("Expected %s IsFirstLevel to be %v, got %v", name, isFirstLevel, gem.IsFirstLevel)
	}

	if !hasGroup(gem, expectedGroup) {
		t.Errorf("Expected %s to be in %s group, got %v", name, expectedGroup, gem.Groups)
	}
}

// hasGroup checks if a gem has a specific group.
func hasGroup(gem *Gem, group string) bool {
	for _, g := range gem.Groups {
		if g == group {
			return true
		}
	}
	return false
}
