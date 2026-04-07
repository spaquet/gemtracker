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
	if rail, ok := gf.Gems["rails"]; !ok {
		t.Error("Expected 'rails' gem to be parsed")
	} else {
		if rail.Version != ">= 6.0" {
			t.Errorf("Expected rails version '>= 6.0', got '%s'", rail.Version)
		}
		if !rail.IsFirstLevel {
			t.Error("Expected rails to be first-level")
		}
		// Rails should not be in development group
		if len(rail.Groups) > 0 {
			t.Errorf("Expected rails to have no groups, got %v", rail.Groups)
		}
	}

	// Check development dependency
	if rspec, ok := gf.Gems["rspec"]; !ok {
		t.Error("Expected 'rspec' gem to be parsed")
	} else {
		if rspec.Version != "~> 3.0" {
			t.Errorf("Expected rspec version '~> 3.0', got '%s'", rspec.Version)
		}
		if !rspec.IsFirstLevel {
			t.Error("Expected rspec to be first-level")
		}
		// Check if in development group
		hasDev := false
		for _, g := range rspec.Groups {
			if g == "development" {
				hasDev = true
				break
			}
		}
		if !hasDev {
			t.Error("Expected rspec to be in development group")
		}
	}

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
