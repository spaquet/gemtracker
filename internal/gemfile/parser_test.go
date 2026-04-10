package gemfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_ValidFile(t *testing.T) {
	path := "../../testdata/projects/minimal-example/Gemfile.lock"
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
	path := "../../testdata/projects/minimal-example/Gemfile.lock"
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
	path := "../../testdata/projects/minimal-example/Gemfile.lock"
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
	path := "../../testdata/projects/simple-deps/Gemfile.lock"
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
	path := "../../testdata/projects/minimal-example/Gemfile.lock"
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

func TestParse_FirstLevelGems(t *testing.T) {
	// Test that DEPENDENCIES section is properly parsed to identify first-level gems
	dir := t.TempDir()

	lockContent := `GEM
  remote: https://rubygems.org/
  specs:
    rails (7.0.0)
      actionpack (= 7.0.0)
      activesupport (= 7.0.0)
    actionpack (7.0.0)
    activesupport (7.0.0)
    devise (4.8.0)
      rails (>= 5.0)

PLATFORMS
  ruby

DEPENDENCIES
  rails
  devise

BUNDLED WITH
   2.3.0
`

	lockPath := filepath.Join(dir, "Gemfile.lock")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0644); err != nil {
		t.Fatalf("failed to write Gemfile.lock: %v", err)
	}

	gf, err := Parse(lockPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that DEPENDENCIES were parsed
	if len(gf.FirstLevelGems) != 2 {
		t.Errorf("expected 2 first-level gems, got %d", len(gf.FirstLevelGems))
	}

	// Check that specific gems are marked as first-level
	rails := gf.Gems["rails"]
	if rails == nil {
		t.Fatal("rails gem not found")
	}
	if !rails.IsFirstLevel {
		t.Error("expected rails to be marked as first-level")
	}

	devise := gf.Gems["devise"]
	if devise == nil {
		t.Fatal("devise gem not found")
	}
	if !devise.IsFirstLevel {
		t.Error("expected devise to be marked as first-level")
	}

	// Check that transitive deps are NOT marked as first-level
	actionpack := gf.Gems["actionpack"]
	if actionpack == nil {
		t.Fatal("actionpack gem not found")
	}
	if actionpack.IsFirstLevel {
		t.Error("expected actionpack to NOT be marked as first-level")
	}
}

func TestParse_GitSection(t *testing.T) {
	// Test that GIT section is properly parsed
	dir := t.TempDir()

	lockContent := `GIT
  remote: https://github.com/mbleigh/acts-as-taggable-on.git
  revision: 1df5ac334c7f6321ac6b967fb014f834b3aa1e09
  branch: master
  specs:
    acts-as-taggable-on (13.0.0)
      activerecord (>= 7.1, < 8.2)

GEM
  remote: https://rubygems.org/
  specs:
    activerecord (8.1.0)
    activesupport (8.1.0)

PLATFORMS
  ruby

DEPENDENCIES
  acts-as-taggable-on!

BUNDLED WITH
   2.7.1
`

	lockPath := filepath.Join(dir, "Gemfile.lock")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0644); err != nil {
		t.Fatalf("failed to write Gemfile.lock: %v", err)
	}

	gf, err := Parse(lockPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that GIT section was parsed
	actsTags := gf.Gems["acts-as-taggable-on"]
	if actsTags == nil {
		t.Fatal("acts-as-taggable-on gem not found - GIT section was not parsed")
	}

	if actsTags.Version != "13.0.0" {
		t.Errorf("expected version 13.0.0, got %s", actsTags.Version)
	}

	// Check that it's marked as first-level
	if !actsTags.IsFirstLevel {
		t.Error("expected acts-as-taggable-on to be marked as first-level")
	}
}

func TestFindLockFile_PreferGemsLocked(t *testing.T) {
	dir := t.TempDir()

	// Create both gems.locked and Gemfile.lock
	gemsLocked := filepath.Join(dir, "gems.locked")
	gemfileLock := filepath.Join(dir, "Gemfile.lock")

	if err := os.WriteFile(gemsLocked, []byte("# gems.locked"), 0644); err != nil {
		t.Fatalf("failed to create gems.locked: %v", err)
	}
	if err := os.WriteFile(gemfileLock, []byte("# Gemfile.lock"), 0644); err != nil {
		t.Fatalf("failed to create Gemfile.lock: %v", err)
	}

	result := FindLockFile(dir)
	if result != gemsLocked {
		t.Errorf("expected FindLockFile to prefer gems.locked, got %s", result)
	}
}

func TestFindLockFile_FallbackToGemfileLock(t *testing.T) {
	dir := t.TempDir()

	// Create only Gemfile.lock
	gemfileLock := filepath.Join(dir, "Gemfile.lock")
	if err := os.WriteFile(gemfileLock, []byte("# Gemfile.lock"), 0644); err != nil {
		t.Fatalf("failed to create Gemfile.lock: %v", err)
	}

	result := FindLockFile(dir)
	if result != gemfileLock {
		t.Errorf("expected FindLockFile to return Gemfile.lock, got %s", result)
	}
}

func TestFindLockFile_NotFound(t *testing.T) {
	dir := t.TempDir()

	// No lock files exist
	result := FindLockFile(dir)
	if result != "" {
		t.Errorf("expected FindLockFile to return empty string, got %s", result)
	}
}

func TestFindLockFile_OnlyGemsLocked(t *testing.T) {
	dir := t.TempDir()

	// Create only gems.locked
	gemsLocked := filepath.Join(dir, "gems.locked")
	if err := os.WriteFile(gemsLocked, []byte("# gems.locked"), 0644); err != nil {
		t.Fatalf("failed to create gems.locked: %v", err)
	}

	result := FindLockFile(dir)
	if result != gemsLocked {
		t.Errorf("expected FindLockFile to return gems.locked, got %s", result)
	}
}

func TestFindGemfile_PreferGemsRb(t *testing.T) {
	dir := t.TempDir()

	// Create both gems.rb and Gemfile
	gemsRb := filepath.Join(dir, "gems.rb")
	gemfile := filepath.Join(dir, "Gemfile")

	if err := os.WriteFile(gemsRb, []byte("# gems.rb"), 0644); err != nil {
		t.Fatalf("failed to create gems.rb: %v", err)
	}
	if err := os.WriteFile(gemfile, []byte("# Gemfile"), 0644); err != nil {
		t.Fatalf("failed to create Gemfile: %v", err)
	}

	result := FindGemfile(dir)
	if result != gemsRb {
		t.Errorf("expected FindGemfile to prefer gems.rb, got %s", result)
	}
}

func TestFindGemfile_FallbackToGemfile(t *testing.T) {
	dir := t.TempDir()

	// Create only Gemfile
	gemfile := filepath.Join(dir, "Gemfile")
	if err := os.WriteFile(gemfile, []byte("# Gemfile"), 0644); err != nil {
		t.Fatalf("failed to create Gemfile: %v", err)
	}

	result := FindGemfile(dir)
	if result != gemfile {
		t.Errorf("expected FindGemfile to return Gemfile, got %s", result)
	}
}

func TestFindGemfile_NotFound(t *testing.T) {
	dir := t.TempDir()

	// No Gemfile exists
	result := FindGemfile(dir)
	if result != "" {
		t.Errorf("expected FindGemfile to return empty string, got %s", result)
	}
}

func TestFindGemfile_OnlyGemsRb(t *testing.T) {
	dir := t.TempDir()

	// Create only gems.rb
	gemsRb := filepath.Join(dir, "gems.rb")
	if err := os.WriteFile(gemsRb, []byte("# gems.rb"), 0644); err != nil {
		t.Fatalf("failed to create gems.rb: %v", err)
	}

	result := FindGemfile(dir)
	if result != gemsRb {
		t.Errorf("expected FindGemfile to return gems.rb, got %s", result)
	}
}

func TestParse_InsecureGitSources(t *testing.T) {
	dir := t.TempDir()

	// Create a Gemfile.lock with insecure git sources
	lockContent := `GIT
  remote: http://insecure.example.com/repo.git
  revision: abc123
  branch: master
  specs:
    insecure-gem (1.0.0)

GIT
  remote: git://another-insecure.example.com/repo.git
  revision: def456
  branch: main
  specs:
    git-gem (2.0.0)

GEM
  remote: https://rubygems.org/
  specs:
    rails (7.0.0)

DEPENDENCIES
  insecure-gem
  git-gem
  rails
`

	lockFile := filepath.Join(dir, "Gemfile.lock")
	if err := os.WriteFile(lockFile, []byte(lockContent), 0644); err != nil {
		t.Fatalf("failed to create Gemfile.lock: %v", err)
	}

	gf, err := Parse(lockFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test insecure-gem (http://)
	insecureGem := gf.Gems["insecure-gem"]
	if insecureGem == nil {
		t.Fatal("insecure-gem not found")
	}
	if !insecureGem.InsecureSource {
		t.Errorf("expected insecure-gem to have InsecureSource=true, got false")
	}
	if insecureGem.Source != "http://insecure.example.com/repo.git" {
		t.Errorf("expected source 'http://insecure.example.com/repo.git', got %s", insecureGem.Source)
	}

	// Test git-gem (git://)
	gitGem := gf.Gems["git-gem"]
	if gitGem == nil {
		t.Fatal("git-gem not found")
	}
	if !gitGem.InsecureSource {
		t.Errorf("expected git-gem to have InsecureSource=true, got false")
	}
	if gitGem.Source != "git://another-insecure.example.com/repo.git" {
		t.Errorf("expected source 'git://another-insecure.example.com/repo.git', got %s", gitGem.Source)
	}

	// Test rails (https://) - should NOT be insecure
	rails := gf.Gems["rails"]
	if rails == nil {
		t.Fatal("rails not found")
	}
	if rails.InsecureSource {
		t.Errorf("expected rails to have InsecureSource=false, got true")
	}
	if rails.Source != "https://rubygems.org/" {
		t.Errorf("expected source 'https://rubygems.org/', got %s", rails.Source)
	}

	// Test GetInsecureSourceGems
	insecureGems := gf.GetInsecureSourceGems()
	if len(insecureGems) != 2 {
		t.Errorf("expected 2 insecure gems, got %d", len(insecureGems))
	}
}
