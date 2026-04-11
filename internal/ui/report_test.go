package ui

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spaquet/gemtracker/internal/gemfile"
)

// TestNewReportGenerator tests the ReportGenerator creation
func TestNewReportGenerator(t *testing.T) {
	rg := NewReportGenerator("/test/path", false, true)
	if rg.projectPath != "/test/path" {
		t.Errorf("expected projectPath to be /test/path, got %s", rg.projectPath)
	}
	if rg.noCache != false {
		t.Errorf("expected noCache to be false, got %v", rg.noCache)
	}
	if rg.verbose != true {
		t.Errorf("expected verbose to be true, got %v", rg.verbose)
	}
}

// TestBuildReportData tests the report data construction
func TestBuildReportData(t *testing.T) {
	rg := NewReportGenerator("/test", false, false)

	// Create mock analysis result
	analysis := &gemfile.AnalysisResult{
		TotalGems: 5,
		GemStatuses: []*gemfile.GemStatus{
			{
				Name:              "rails",
				Version:           "7.0.0",
				Groups:            []string{"default"},
				IsOutdated:        true,
				LatestVersion:     "8.0.0",
				IsVulnerable:      false,
				VulnerabilityInfo: "",
			},
			{
				Name:              "devise",
				Version:           "4.8.0",
				Groups:            []string{"default"},
				IsOutdated:        false,
				LatestVersion:     "",
				IsVulnerable:      true,
				VulnerabilityInfo: "CVE-2021-41113 [HIGH]: Auth bypass (CVSS: 7.5)",
			},
			{
				Name:              "rake",
				Version:           "13.0.6",
				Groups:            []string{"development"},
				IsOutdated:        false,
				LatestVersion:     "",
				IsVulnerable:      false,
				VulnerabilityInfo: "",
			},
		},
	}

	// Create mock Gemfile with first-level gems
	gf := &gemfile.Gemfile{
		FirstLevelGems: []string{"rails", "devise"},
		Gems:           make(map[string]*gemfile.Gem),
	}

	gemStatusMap := make(map[string]*gemfile.GemStatus)
	for _, status := range analysis.GemStatuses {
		gemStatusMap[status.Name] = status
	}

	reportData := rg.buildReportData(analysis, gemStatusMap, gf)

	// Verify summary stats
	if reportData.TotalGems != 3 {
		t.Errorf("expected TotalGems to be 3, got %d", reportData.TotalGems)
	}
	if reportData.FirstLevelGems != 2 {
		t.Errorf("expected FirstLevelGems to be 2, got %d", reportData.FirstLevelGems)
	}
	if reportData.OutdatedCount != 1 {
		t.Errorf("expected OutdatedCount to be 1, got %d", reportData.OutdatedCount)
	}
	if reportData.VulnerableCount != 1 {
		t.Errorf("expected VulnerableCount to be 1, got %d", reportData.VulnerableCount)
	}

	// Verify gems are properly categorized
	if len(reportData.OutdatedGems) != 1 {
		t.Errorf("expected 1 outdated gem, got %d", len(reportData.OutdatedGems))
	}
	if reportData.OutdatedGems[0].Name != "rails" {
		t.Errorf("expected outdated gem to be 'rails', got %s", reportData.OutdatedGems[0].Name)
	}

	if len(reportData.VulnerableGems) != 1 {
		t.Errorf("expected 1 vulnerable gem, got %d", len(reportData.VulnerableGems))
	}
	if reportData.VulnerableGems[0].Name != "devise" {
		t.Errorf("expected vulnerable gem to be 'devise', got %s", reportData.VulnerableGems[0].Name)
	}
}

// TestGenerateTextReport tests text report generation
func TestGenerateTextReport(t *testing.T) {
	rg := NewReportGenerator("/test", false, false)

	reportData := &ReportData{
		GeneratedAt:     "2026-04-06T10:00:00Z",
		ProjectPath:     "/test/project",
		TotalGems:       3,
		FirstLevelGems:  2,
		OutdatedCount:   1,
		VulnerableCount: 1,
		AllGems: []*GemReport{
			{Name: "devise", Version: "4.8.0", IsFirstLevel: true, IsVulnerable: true, VulnerabilityInfo: "CVE-2021-41113 [HIGH]: Auth bypass (CVSS: 7.5)"},
			{Name: "rails", Version: "7.0.0", IsFirstLevel: true, IsOutdated: true, LatestVersion: "8.0.0"},
			{Name: "rake", Version: "13.0.6", IsFirstLevel: false},
		},
		OutdatedGems: []*GemReport{
			{Name: "rails", Version: "7.0.0", LatestVersion: "8.0.0"},
		},
		VulnerableGems: []*GemReport{
			{Name: "devise", Version: "4.8.0", VulnerabilityInfo: "CVE-2021-41113 [HIGH]: Auth bypass (CVSS: 7.5)"},
		},
		Summary: "Test summary",
	}

	err := rg.generateTextReport(reportData, "")
	if err != nil {
		t.Errorf("generateTextReport returned unexpected error: %v", err)
	}
}

// TestGenerateCSVReport tests CSV report generation
func TestGenerateCSVReport(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-report-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	rg := NewReportGenerator("/test", false, false)

	reportData := &ReportData{
		GeneratedAt:     "2026-04-06T10:00:00Z",
		ProjectPath:     "/test/project",
		TotalGems:       2,
		FirstLevelGems:  1,
		OutdatedCount:   1,
		VulnerableCount: 0,
		AllGems: []*GemReport{
			{
				Name:          "rails",
				Version:       "7.0.0",
				IsFirstLevel:  true,
				IsOutdated:    true,
				LatestVersion: "8.0.0",
				Groups:        []string{"default"},
			},
			{
				Name:         "rake",
				Version:      "13.0.6",
				IsFirstLevel: false,
				Groups:       []string{"development"},
			},
		},
		OutdatedGems:   []*GemReport{},
		VulnerableGems: []*GemReport{},
		Summary:        "Test summary",
	}

	err = rg.generateCSVReport(reportData, tmpFile.Name())
	if err != nil {
		t.Errorf("generateCSVReport returned unexpected error: %v", err)
	}

	// Verify CSV file was created and has correct content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read CSV file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		t.Errorf("expected at least 2 lines in CSV (header + data), got %d", len(lines))
	}

	// Check header row (skip comment lines at the top)
	headerIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "Name,") || strings.Contains(line, "Name") && !strings.HasPrefix(line, "#") {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		t.Errorf("CSV header not found in output")
	}
	if headerIdx >= 0 && (!strings.Contains(lines[headerIdx], "Name") || !strings.Contains(lines[headerIdx], "Version")) {
		t.Errorf("CSV header doesn't contain expected columns")
	}

	// Verify data rows contain gem names
	csvContent := string(content)
	if !strings.Contains(csvContent, "rails") {
		t.Errorf("CSV content doesn't contain 'rails' gem")
	}
	if !strings.Contains(csvContent, "rake") {
		t.Errorf("CSV content doesn't contain 'rake' gem")
	}
}

// TestGenerateJSONReport tests JSON report generation
func TestGenerateJSONReport(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-report-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	rg := NewReportGenerator("/test", false, false)

	reportData := &ReportData{
		GeneratedAt:     "2026-04-06T10:00:00Z",
		ProjectPath:     "/test/project",
		TotalGems:       2,
		FirstLevelGems:  1,
		OutdatedCount:   1,
		VulnerableCount: 0,
		AllGems: []*GemReport{
			{
				Name:          "rails",
				Version:       "7.0.0",
				IsFirstLevel:  true,
				IsOutdated:    true,
				LatestVersion: "8.0.0",
			},
		},
		OutdatedGems:   []*GemReport{},
		VulnerableGems: []*GemReport{},
		Summary:        "Test summary",
	}

	err = rg.generateJSONReport(reportData, tmpFile.Name())
	if err != nil {
		t.Errorf("generateJSONReport returned unexpected error: %v", err)
	}

	// Verify JSON file was created and contains valid JSON
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(content, &jsonData)
	if err != nil {
		t.Errorf("JSON file contains invalid JSON: %v", err)
	}

	// Verify structure
	if _, ok := jsonData["generated_at"]; !ok {
		t.Errorf("JSON missing 'generated_at' field")
	}
	if _, ok := jsonData["gems"]; !ok {
		t.Errorf("JSON missing 'gems' field")
	}
	if _, ok := jsonData["summary"]; !ok {
		t.Errorf("JSON missing 'summary' field")
	}

	// Verify summary data
	summary, ok := jsonData["summary"].(map[string]interface{})
	if !ok {
		t.Errorf("summary field is not a map")
	}
	if total, ok := summary["total_gems"].(float64); !ok || total != 2 {
		t.Errorf("expected total_gems to be 2, got %v", summary["total_gems"])
	}
}

// TestWriteOutputToFile tests writing output to a file
func TestWriteOutputToFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-output-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	rg := NewReportGenerator("/test", false, false)
	testContent := "Test output content\nLine 2\nLine 3"

	err = rg.writeOutput(testContent, tmpFile.Name())
	if err != nil {
		t.Errorf("writeOutput returned unexpected error: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("file content mismatch.\nexpected: %s\ngot: %s", testContent, string(content))
	}
}

// TestGenerateWithInvalidFormat tests error handling for invalid format
func TestGenerateWithInvalidFormat(t *testing.T) {
	if _, err := os.Stat("../../testdata/projects/minimal-example/Gemfile.lock"); os.IsNotExist(err) {
		t.Fatalf("testdata/projects/minimal-example/Gemfile.lock not found (required for test)")
	}

	rg := NewReportGenerator("../../testdata/projects/minimal-example", false, false)
	err := rg.Generate("invalid", "")
	if err == nil {
		t.Errorf("expected error for invalid format, got nil")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("expected 'unknown format' error, got: %v", err)
	}
}

// TestGenerateWithNonexistentPath tests error handling for invalid paths
func TestGenerateWithNonexistentPath(t *testing.T) {
	rg := NewReportGenerator("/nonexistent/path/that/does/not/exist", false, false)
	err := rg.Generate("text", "")
	if err == nil {
		t.Errorf("expected error for nonexistent path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse dependencies") {
		t.Errorf("expected parsing error, got: %v", err)
	}
}

// TestBoolToString tests the boolean to string conversion
func TestBoolToString(t *testing.T) {
	tests := []struct {
		input    bool
		expected string
	}{
		{true, "yes"},
		{false, "no"},
	}

	for _, test := range tests {
		result := boolToString(test.input)
		if result != test.expected {
			t.Errorf("boolToString(%v) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

// TestGenerateWithRealGemfile tests report generation with actual test data
func TestGenerateWithRealGemfile(t *testing.T) {
	if _, err := os.Stat("../../testdata/projects/minimal-example/Gemfile.lock"); os.IsNotExist(err) {
		t.Fatalf("testdata/projects/minimal-example/Gemfile.lock not found (required for integration test)")
	}

	rg := NewReportGenerator("../../testdata/projects/minimal-example", false, false)

	// Test all three formats
	formats := []string{"text", "csv", "json"}
	for _, format := range formats {
		tmpFile, err := os.CreateTemp("", "test-gemtracker-*."+format)
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		err = rg.Generate(format, tmpFile.Name())
		if err != nil {
			t.Errorf("Generate failed for format %s: %v", format, err)
			continue
		}

		// Verify file exists and has content
		info, err := os.Stat(tmpFile.Name())
		if err != nil {
			t.Errorf("failed to stat output file for format %s: %v", format, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("output file for format %s is empty", format)
		}

		// Format-specific validation
		content, _ := os.ReadFile(tmpFile.Name())
		switch format {
		case "json":
			var jsonData interface{}
			if err := json.Unmarshal(content, &jsonData); err != nil {
				t.Errorf("invalid JSON output for format json: %v", err)
			}
		case "csv":
			reader := csv.NewReader(strings.NewReader(string(content)))
			reader.Comment = '#' // Skip comment lines at the top
			_, err := reader.ReadAll()
			if err != nil {
				t.Errorf("invalid CSV output for format csv: %v", err)
			}
		case "text":
			text := string(content)
			if !strings.Contains(text, "GEMTRACKER REPORT") {
				t.Errorf("text report missing expected header")
			}
		}
	}
}

// BenchmarkGenerateTextReport benchmarks text report generation
func BenchmarkGenerateTextReport(b *testing.B) {
	rg := NewReportGenerator("/test", false, false)

	reportData := createLargeReportData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rg.generateTextReport(reportData, "")
	}
}

// BenchmarkGenerateCSVReport benchmarks CSV report generation
func BenchmarkGenerateCSVReport(b *testing.B) {
	rg := NewReportGenerator("/test", false, false)
	tmpFile, _ := os.CreateTemp("", "bench-*.csv")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	reportData := createLargeReportData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rg.generateCSVReport(reportData, tmpFile.Name())
	}
}

// BenchmarkGenerateJSONReport benchmarks JSON report generation
func BenchmarkGenerateJSONReport(b *testing.B) {
	rg := NewReportGenerator("/test", false, false)
	tmpFile, _ := os.CreateTemp("", "bench-*.json")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	reportData := createLargeReportData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rg.generateJSONReport(reportData, tmpFile.Name())
	}
}

// Helper function to create large report data for benchmarks
func createLargeReportData() *ReportData {
	gems := make([]*GemReport, 100)
	for i := 0; i < 100; i++ {
		gems[i] = &GemReport{
			Name:          "gem-" + string(rune(i)),
			Version:       "1.0.0",
			IsFirstLevel:  i%2 == 0,
			IsOutdated:    i%3 == 0,
			LatestVersion: "2.0.0",
			IsVulnerable:  i%5 == 0,
			Groups:        []string{"default"},
		}
	}

	outdated := make([]*GemReport, 0)
	vulnerable := make([]*GemReport, 0)
	for _, g := range gems {
		if g.IsOutdated {
			outdated = append(outdated, g)
		}
		if g.IsVulnerable {
			vulnerable = append(vulnerable, g)
		}
	}

	return &ReportData{
		GeneratedAt:     "2026-04-06T10:00:00Z",
		ProjectPath:     "/test/project",
		TotalGems:       100,
		FirstLevelGems:  50,
		AllGems:         gems,
		OutdatedGems:    outdated,
		VulnerableGems:  vulnerable,
		OutdatedCount:   len(outdated),
		VulnerableCount: len(vulnerable),
		Summary:         "Benchmark test data",
	}
}
