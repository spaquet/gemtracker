package gemfile

import (
	"os"
	"strings"
	"testing"
)

func TestAnalyze_BasicMetrics(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := Analyze(gf)

	if result == nil {
		t.Fatal("expected AnalysisResult, got nil")
	}

	// Check total gems count
	if result.TotalGems == 0 {
		t.Error("expected gems in analysis result")
	}

	if result.TotalGems != gf.GetGemCount() {
		t.Errorf("expected %d total gems, got %d", gf.GetGemCount(), result.TotalGems)
	}

	if len(result.AllGems) != result.TotalGems {
		t.Errorf("expected AllGems to match TotalGems count")
	}
}

func TestAnalyze_FirstLevelGems(t *testing.T) {
	// Create a test gemfile with group information
	gf := &Gemfile{
		Path: "test",
		Gems: map[string]*Gem{
			"rails": {
				Name:    "rails",
				Version: "7.0.0",
				Groups:  []string{"default"},
			},
			"rspec": {
				Name:    "rspec",
				Version: "3.12.0",
				Groups:  []string{"test"},
			},
			"rake": {
				Name:    "rake",
				Version: "13.0.6",
				Groups:  []string{}, // Transitive dependency
			},
		},
	}

	result := Analyze(gf)

	// First-level gems are those with groups
	if len(result.FirstLevelGems) != 2 {
		t.Errorf("expected 2 first-level gems, got %d", len(result.FirstLevelGems))
	}

	// Check that rails and rspec are in first-level
	foundRails := false
	foundRspec := false

	for _, gem := range result.FirstLevelGems {
		if gem == "rails" {
			foundRails = true
		}
		if gem == "rspec" {
			foundRspec = true
		}
	}

	if !foundRails {
		t.Error("expected rails in first-level gems")
	}

	if !foundRspec {
		t.Error("expected rspec in first-level gems")
	}
}

func TestAnalyze_GemStatuses(t *testing.T) {
	gf := &Gemfile{
		Path: "test",
		Gems: map[string]*Gem{
			"rails": {
				Name:    "rails",
				Version: "7.0.0",
				Groups:  []string{"default"},
			},
		},
	}

	result := Analyze(gf)

	if len(result.GemStatuses) != 1 {
		t.Errorf("expected 1 gem status, got %d", len(result.GemStatuses))
	}

	status := result.GemStatuses[0]
	if status.Name != "rails" {
		t.Errorf("expected gem name 'rails', got %q", status.Name)
	}

	if status.Version != "7.0.0" {
		t.Errorf("expected version 7.0.0, got %q", status.Version)
	}

	if len(status.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(status.Groups))
	}
}

func TestAnalyze_Summary(t *testing.T) {
	gf := &Gemfile{
		Path: "test",
		Gems: map[string]*Gem{
			"gem1": {Name: "gem1", Version: "1.0.0"},
			"gem2": {Name: "gem2", Version: "2.0.0"},
		},
	}

	result := Analyze(gf)

	if result.Summary == "" {
		t.Error("expected summary to be generated")
	}

	if !strings.Contains(result.Summary, "Total Gems: 2") {
		t.Errorf("expected summary to contain gem count, got %q", result.Summary)
	}
}

func TestAnalyze_Details(t *testing.T) {
	gf := &Gemfile{
		Path: "test",
		Gems: map[string]*Gem{
			"rails": {
				Name:    "rails",
				Version: "7.0.0",
			},
		},
	}

	result := Analyze(gf)

	if result.Details == "" {
		t.Error("expected details to be generated")
	}

	if !strings.Contains(result.Details, "rails") {
		t.Errorf("expected details to contain gem name")
	}

	if !strings.Contains(result.Details, "7.0.0") {
		t.Errorf("expected details to contain version")
	}
}

func TestAnalyze_RealGemfile(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := Analyze(gf)

	// Should have analyzed gems
	if len(result.GemStatuses) == 0 {
		t.Error("expected gem statuses")
	}

	// Each gem status should have name and version
	for _, status := range result.GemStatuses {
		if status.Name == "" {
			t.Error("expected gem status to have name")
		}

		if status.Version == "" {
			t.Error("expected gem status to have version")
		}
	}
}

func TestGenerateSummary(t *testing.T) {
	result := &AnalysisResult{
		TotalGems:      10,
		OutdatedGems:   []string{"gem1", "gem2"},
		VulnerableGems: []string{"gem3"},
	}

	summary := generateSummary(result)

	if !strings.Contains(summary, "Total Gems: 10") {
		t.Error("expected total gems in summary")
	}

	if !strings.Contains(summary, "Outdated: 2") {
		t.Error("expected outdated count in summary")
	}

	if !strings.Contains(summary, "Vulnerable: 1") {
		t.Error("expected vulnerable count in summary")
	}
}

func TestGenerateDetails(t *testing.T) {
	result := &AnalysisResult{
		GemStatuses: []*GemStatus{
			{
				Name:    "rails",
				Version: "7.0.0",
			},
			{
				Name:         "devise",
				Version:      "4.7.0",
				IsVulnerable: true,
			},
		},
	}

	details := generateDetails(result)

	if !strings.Contains(details, "rails") {
		t.Error("expected gem name in details")
	}

	if !strings.Contains(details, "7.0.0") {
		t.Error("expected version in details")
	}

	// Vulnerable gem should have vulnerability symbol
	if !strings.Contains(details, "🔒") {
		t.Error("expected vulnerability symbol in details")
	}
}

func TestGenerateDetails_NoGems(t *testing.T) {
	result := &AnalysisResult{
		GemStatuses: []*GemStatus{},
	}

	details := generateDetails(result)

	if !strings.Contains(details, "No gems") {
		t.Errorf("expected 'No gems' message, got %q", details)
	}
}

func TestGemStatus_DefaultValues(t *testing.T) {
	gf := &Gemfile{
		Path: "test",
		Gems: map[string]*Gem{
			"simple": {
				Name:    "simple",
				Version: "1.0.0",
				Groups:  []string{},
			},
		},
	}

	result := Analyze(gf)

	status := result.GemStatuses[0]

	// By default, a gem should not be marked as outdated or vulnerable
	// (unless detected by the checker)
	if status.Name == "" {
		t.Error("expected name to be set")
	}

	if status.Version == "" {
		t.Error("expected version to be set")
	}
}

func TestAnalyze_MultipleGroups(t *testing.T) {
	gf := &Gemfile{
		Path: "test",
		Gems: map[string]*Gem{
			"multi-gem": {
				Name:    "multi-gem",
				Version: "1.0.0",
				Groups:  []string{"default", "test", "development"},
			},
		},
	}

	result := Analyze(gf)

	status := result.GemStatuses[0]

	if len(status.Groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(status.Groups))
	}
}

func TestAnalyze_GroupsPreserved(t *testing.T) {
	gf := &Gemfile{
		Path: "test",
		Gems: map[string]*Gem{
			"test-gem": {
				Name:    "test-gem",
				Version: "1.0.0",
				Groups:  []string{"test"},
			},
		},
	}

	result := Analyze(gf)

	status := result.GemStatuses[0]

	if len(status.Groups) == 0 {
		t.Error("expected groups to be preserved")
	}

	found := false
	for _, group := range status.Groups {
		if group == "test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'test' group in status")
	}
}

func TestAnalyze_ListopiaGemfile(t *testing.T) {
	// Test parsing and analyzing the listopia Gemfile.lock
	// Skip if file doesn't exist (common in CI/different machines)
	listopiaPath := "/Users/spaquet/Sites/listopia/Gemfile.lock"
	if _, err := os.Stat(listopiaPath); err != nil {
		t.Skipf("Skipping listopia test - file not found at %s", listopiaPath)
	}

	gf, err := Parse(listopiaPath)
	if err != nil {
		t.Fatalf("Failed to parse listopia Gemfile.lock: %v", err)
	}

	// Check that key gems are present
	checkGems := []string{
		"acts-as-taggable-on",
		"rails",
		"actioncable",
		"actionmailbox",
		"action_text-trix",
		"actiontext",
		"addressable",
	}

	for _, gemName := range checkGems {
		if _, ok := gf.Gems[gemName]; !ok {
			t.Errorf("Expected gem %q to be parsed from listopia Gemfile.lock", gemName)
		}
	}

	// Verify acts-as-taggable-on is marked as first-level (from DEPENDENCIES section with !)
	if atsGem, ok := gf.Gems["acts-as-taggable-on"]; ok {
		if !atsGem.IsFirstLevel {
			t.Error("Expected acts-as-taggable-on to be marked as first-level (from GIT+DEPENDENCIES)")
		}
	}

	// Verify rails is marked as first-level
	if railsGem, ok := gf.Gems["rails"]; ok {
		if !railsGem.IsFirstLevel {
			t.Error("Expected rails to be marked as first-level")
		}
	}

	// Analyze the gemfile
	result := Analyze(gf)

	// Rails should definitely be in first-level gems
	foundRails := false
	for _, name := range result.FirstLevelGems {
		if name == "rails" {
			foundRails = true
			break
		}
	}
	if !foundRails {
		t.Error("Expected rails in first-level gems from Analyze result")
	}

	t.Logf("Successfully parsed %d gems from listopia Gemfile.lock", gf.GetGemCount())
	t.Logf("First-level gems: %d", len(result.FirstLevelGems))
}
