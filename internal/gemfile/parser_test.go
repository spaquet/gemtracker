package gemfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_ValidFile(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if gf == nil {
		t.Fatal("expected Gemfile, got nil")
	}

	// Check that gems were parsed
	if gf.GetGemCount() == 0 {
		t.Fatal("expected gems to be parsed, got 0")
	}

	// Verify specific gems exist
	expectedGems := []string{"rails", "devise", "actionpack", "concurrent-ruby"}
	for _, gemName := range expectedGems {
		if _, ok := gf.Gems[gemName]; !ok {
			t.Errorf("expected gem %q to be parsed", gemName)
		}
	}
}

func TestParse_RailsGem(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	rails := gf.Gems["rails"]
	if rails == nil {
		t.Fatal("rails gem not found")
	}

	if rails.Version != "7.0.0" {
		t.Errorf("expected rails version 7.0.0, got %s", rails.Version)
	}

	// Check dependencies
	if len(rails.Dependencies) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(rails.Dependencies))
	}

	expectedDeps := []string{"actionpack", "activesupport", "railties"}
	for _, dep := range expectedDeps {
		found := false
		for _, actualDep := range rails.Dependencies {
			if actualDep == dep {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected dependency %q not found", dep)
		}
	}
}

func TestParse_DeviseGem(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	devise := gf.Gems["devise"]
	if devise == nil {
		t.Fatal("devise gem not found")
	}

	if devise.Version != "4.8.0" {
		t.Errorf("expected devise version 4.8.0, got %s", devise.Version)
	}

	// Devise should have dependencies
	if len(devise.Dependencies) == 0 {
		t.Error("expected devise to have dependencies")
	}
}

func TestParse_NonexistentFile(t *testing.T) {
	path := "/nonexistent/path/Gemfile.lock"
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestParse_Directory(t *testing.T) {
	// Create a temporary directory with a Gemfile.lock
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "Gemfile.lock")

	content := `GEM
  remote: https://rubygems.org/
  specs:
    test-gem (1.0.0)

PLATFORMS
  ruby
`

	if err := os.WriteFile(lockPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse by directory path
	gf, err := Parse(dir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if gf.GetGemCount() != 1 {
		t.Errorf("expected 1 gem, got %d", gf.GetGemCount())
	}

	if _, ok := gf.Gems["test-gem"]; !ok {
		t.Error("expected test-gem to be parsed")
	}
}

func TestParse_SimpleFile(t *testing.T) {
	path := "../../testdata/Gemfile.lock.simple"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if gf.GetGemCount() != 2 {
		t.Errorf("expected 2 gems, got %d", gf.GetGemCount())
	}

	simpleGem := gf.Gems["simple-gem"]
	if simpleGem == nil {
		t.Fatal("simple-gem not found")
	}

	if simpleGem.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", simpleGem.Version)
	}

	// simple-gem should have no dependencies
	if len(simpleGem.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(simpleGem.Dependencies))
	}

	// another-gem should depend on simple-gem
	anotherGem := gf.Gems["another-gem"]
	if anotherGem == nil {
		t.Fatal("another-gem not found")
	}

	found := false
	for _, dep := range anotherGem.Dependencies {
		if dep == "simple-gem" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected another-gem to depend on simple-gem")
	}
}

func TestGetGemsAsList(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	gems := gf.GetGemsAsList()
	if len(gems) != gf.GetGemCount() {
		t.Errorf("expected %d gems, got %d", gf.GetGemCount(), len(gems))
	}

	// Verify all gems in map are in list
	for name := range gf.Gems {
		found := false
		for _, gem := range gems {
			if gem.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("gem %q not found in list", name)
		}
	}
}

func TestLoadGroupsFromGemfile(t *testing.T) {
	// Create a temporary directory with both Gemfile and Gemfile.lock
	dir := t.TempDir()

	lockContent := `GEM
  remote: https://rubygems.org/
  specs:
    rails (7.0.0)
    rspec (3.12.0)

PLATFORMS
  ruby
`

	gemfileContent := `source "https://rubygems.org"

gem "rails"

group :test do
  gem "rspec"
end
`

	lockPath := filepath.Join(dir, "Gemfile.lock")
	gemfilePath := filepath.Join(dir, "Gemfile")

	if err := os.WriteFile(lockPath, []byte(lockContent), 0644); err != nil {
		t.Fatalf("failed to write Gemfile.lock: %v", err)
	}

	if err := os.WriteFile(gemfilePath, []byte(gemfileContent), 0644); err != nil {
		t.Fatalf("failed to write Gemfile: %v", err)
	}

	gf, err := Parse(lockPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Load groups
	err = gf.LoadGroupsFromGemfile(dir)
	if err != nil {
		t.Fatalf("LoadGroupsFromGemfile failed: %v", err)
	}

	rails := gf.Gems["rails"]
	if rails == nil {
		t.Fatal("rails gem not found")
	}

	// Rails should be in "default" group
	found := false
	for _, group := range rails.Groups {
		if group == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected rails to be in default group")
	}

	rspec := gf.Gems["rspec"]
	if rspec == nil {
		t.Fatal("rspec gem not found")
	}

	// Rspec should be in "test" group
	found = false
	for _, group := range rspec.Groups {
		if group == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected rspec to be in test group")
	}
}

func TestLoadGroupsFromGemfile_NoGemfile(t *testing.T) {
	// Create a temporary directory without Gemfile
	dir := t.TempDir()

	lockContent := `GEM
  remote: https://rubygems.org/
  specs:
    rails (7.0.0)

PLATFORMS
  ruby
`

	lockPath := filepath.Join(dir, "Gemfile.lock")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0644); err != nil {
		t.Fatalf("failed to write Gemfile.lock: %v", err)
	}

	gf, err := Parse(lockPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// LoadGroupsFromGemfile should not fail even if Gemfile doesn't exist
	err = gf.LoadGroupsFromGemfile(dir)
	if err != nil {
		t.Fatalf("LoadGroupsFromGemfile should not fail: %v", err)
	}
}
