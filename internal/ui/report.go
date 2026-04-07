package ui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spaquet/gemtracker/internal/gemfile"
	"github.com/spaquet/gemtracker/internal/logger"
)

// ReportGenerator generates reports in various formats
type ReportGenerator struct {
	projectPath string
	noCache     bool
	verbose     bool
}

// ReportData holds the structured data for export
type ReportData struct {
	GeneratedAt      string
	ProjectPath      string
	TotalGems        int
	FirstLevelGems   int
	OutdatedGems     []*GemReport
	VulnerableGems   []*GemReport
	AllGems          []*GemReport
	Summary          string
	OutdatedCount    int
	VulnerableCount  int
}

// GemReport represents a gem in the report
type GemReport struct {
	Name              string
	Version           string
	Groups            []string
	IsFirstLevel      bool
	IsOutdated        bool
	LatestVersion     string
	IsVulnerable      bool
	VulnerabilityInfo string
	HomepageURL       string
	Description       string
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(projectPath string, noCache, verbose bool) *ReportGenerator {
	return &ReportGenerator{
		projectPath: projectPath,
		noCache:     noCache,
		verbose:     verbose,
	}
}

// Generate generates and writes a report in the specified format
func (rg *ReportGenerator) Generate(format, outputPath string) error {
	// Parse the Gemfile
	logger.Info("Parsing Gemfile...")
	gf, err := gemfile.Parse(rg.projectPath)
	if err != nil {
		return fmt.Errorf("failed to parse gemfile: %w", err)
	}

	// Analyze gems
	logger.Info("Analyzing gems...")
	analysis := gemfile.Analyze(gf)

	// Check for outdated gems
	logger.Info("Checking for outdated gems...")
	outdatedChecker := gemfile.NewOutdatedChecker()

	// Build gem status map
	gemStatusMap := make(map[string]*gemfile.GemStatus)
	for _, status := range analysis.GemStatuses {
		gemStatusMap[status.Name] = status
	}

	// Check each gem for updates
	for _, gem := range gf.GetGemsAsList() {
		status, ok := gemStatusMap[gem.Name]
		if !ok {
			continue
		}

		isOutdated, latestVersion, err := outdatedChecker.IsOutdated(gem.Name, gem.Version)
		if err != nil {
			logger.Warn("Failed to check if %s is outdated: %v", gem.Name, err)
			status.OutdatedFailed = true
			continue
		}

		if isOutdated && latestVersion != "" {
			status.IsOutdated = true
			status.LatestVersion = latestVersion
			status.HomepageURL = outdatedChecker.GetHomepage(gem.Name)
			status.Description = outdatedChecker.GetDescription(gem.Name)
		}
	}

	// Check for vulnerabilities (already done in Analyze)
	// but we need to get the vulnerability details

	// Build report data
	reportData := rg.buildReportData(analysis, gemStatusMap, gf)

	// Generate report based on format
	switch strings.ToLower(format) {
	case "text":
		return rg.generateTextReport(reportData, outputPath)
	case "csv":
		return rg.generateCSVReport(reportData, outputPath)
	case "json":
		return rg.generateJSONReport(reportData, outputPath)
	default:
		return fmt.Errorf("unknown format: %s (supported: text, csv, json)", format)
	}
}

// buildReportData builds structured report data from analysis results
func (rg *ReportGenerator) buildReportData(analysis *gemfile.AnalysisResult, gemStatusMap map[string]*gemfile.GemStatus, gf *gemfile.Gemfile) *ReportData {
	// Build first-level gem map
	firstLevelMap := make(map[string]bool)
	for _, name := range gf.FirstLevelGems {
		firstLevelMap[name] = true
	}

	// Convert gem statuses to reports
	allGems := make([]*GemReport, 0)
	outdatedGems := make([]*GemReport, 0)
	vulnerableGems := make([]*GemReport, 0)

	for _, status := range analysis.GemStatuses {
		report := &GemReport{
			Name:              status.Name,
			Version:           status.Version,
			Groups:            status.Groups,
			IsFirstLevel:      firstLevelMap[status.Name],
			IsOutdated:        status.IsOutdated,
			LatestVersion:     status.LatestVersion,
			IsVulnerable:      status.IsVulnerable,
			VulnerabilityInfo: status.VulnerabilityInfo,
			HomepageURL:       status.HomepageURL,
			Description:       status.Description,
		}

		allGems = append(allGems, report)

		if status.IsOutdated {
			outdatedGems = append(outdatedGems, report)
		}
		if status.IsVulnerable {
			vulnerableGems = append(vulnerableGems, report)
		}
	}

	// Sort gems by name
	sort.Slice(allGems, func(i, j int) bool { return allGems[i].Name < allGems[j].Name })
	sort.Slice(outdatedGems, func(i, j int) bool { return outdatedGems[i].Name < outdatedGems[j].Name })
	sort.Slice(vulnerableGems, func(i, j int) bool { return vulnerableGems[i].Name < vulnerableGems[j].Name })

	// Count first-level gems
	firstLevelCount := 0
	for _, gem := range allGems {
		if gem.IsFirstLevel {
			firstLevelCount++
		}
	}

	// Build summary
	summary := fmt.Sprintf("Total gems: %d, First-level: %d, Outdated: %d, Vulnerable: %d",
		len(allGems), firstLevelCount, len(outdatedGems), len(vulnerableGems))

	return &ReportData{
		GeneratedAt:     time.Now().Format(time.RFC3339),
		ProjectPath:     rg.projectPath,
		TotalGems:       len(allGems),
		FirstLevelGems:  firstLevelCount,
		AllGems:         allGems,
		OutdatedGems:    outdatedGems,
		VulnerableGems:  vulnerableGems,
		Summary:         summary,
		OutdatedCount:   len(outdatedGems),
		VulnerableCount: len(vulnerableGems),
	}
}

// generateTextReport generates a human-readable text report
func (rg *ReportGenerator) generateTextReport(data *ReportData, outputPath string) error {
	var output strings.Builder

	// Header
	output.WriteString("GEMTRACKER REPORT\n")
	output.WriteString("==================\n\n")

	output.WriteString(fmt.Sprintf("Generated: %s\n", data.GeneratedAt))
	output.WriteString(fmt.Sprintf("Project: %s\n", data.ProjectPath))
	output.WriteString(fmt.Sprintf("Summary: %s\n\n", data.Summary))

	// Vulnerable gems section
	if data.VulnerableCount > 0 {
		output.WriteString("VULNERABLE GEMS\n")
		output.WriteString(strings.Repeat("-", 80) + "\n")
		for _, gem := range data.VulnerableGems {
			output.WriteString(fmt.Sprintf("  • %s (%s)\n", gem.Name, gem.Version))
			output.WriteString(fmt.Sprintf("    Issue: %s\n", gem.VulnerabilityInfo))
		}
		output.WriteString("\n")
	}

	// Outdated gems section
	if data.OutdatedCount > 0 {
		output.WriteString("OUTDATED GEMS (Updates Available)\n")
		output.WriteString(strings.Repeat("-", 80) + "\n")
		for _, gem := range data.OutdatedGems {
			output.WriteString(fmt.Sprintf("  • %s (%s → %s)\n", gem.Name, gem.Version, gem.LatestVersion))
			if gem.IsFirstLevel {
				output.WriteString("    [Direct dependency]\n")
			}
		}
		output.WriteString("\n")
	}

	// All gems section
	output.WriteString("ALL GEMS\n")
	output.WriteString(strings.Repeat("-", 80) + "\n")
	for _, gem := range data.AllGems {
		marker := ""
		if gem.IsVulnerable {
			marker = " [VULNERABLE]"
		} else if gem.IsOutdated {
			marker = " [OUTDATED]"
		}

		groups := ""
		if len(gem.Groups) > 0 {
			groups = fmt.Sprintf(" [%s]", strings.Join(gem.Groups, ", "))
		}

		depType := ""
		if gem.IsFirstLevel {
			depType = " *"
		}

		output.WriteString(fmt.Sprintf("  %s@%s%s%s%s\n", gem.Name, gem.Version, depType, groups, marker))
	}

	// Write to file or stdout
	return rg.writeOutput(output.String(), outputPath)
}

// generateCSVReport generates a CSV report
func (rg *ReportGenerator) generateCSVReport(data *ReportData, outputPath string) error {
	var output strings.Builder

	writer := csv.NewWriter(&output)
	defer writer.Flush()

	// Header row
	headers := []string{
		"Name",
		"Version",
		"Groups",
		"Direct Dependency",
		"Outdated",
		"Latest Version",
		"Vulnerable",
		"Vulnerability Info",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Data rows
	for _, gem := range data.AllGems {
		record := []string{
			gem.Name,
			gem.Version,
			strings.Join(gem.Groups, ";"),
			boolToString(gem.IsFirstLevel),
			boolToString(gem.IsOutdated),
			gem.LatestVersion,
			boolToString(gem.IsVulnerable),
			gem.VulnerabilityInfo,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("CSV writer error: %w", err)
	}

	// Add summary as comments at the end
	summary := fmt.Sprintf("\n# Generated: %s\n# Project: %s\n# %s\n",
		data.GeneratedAt, data.ProjectPath, data.Summary)
	output.WriteString(summary)

	return rg.writeOutput(output.String(), outputPath)
}

// generateJSONReport generates a JSON report
func (rg *ReportGenerator) generateJSONReport(data *ReportData, outputPath string) error {
	// Create a JSON-friendly structure
	jsonData := map[string]interface{}{
		"generated_at": data.GeneratedAt,
		"project_path": data.ProjectPath,
		"summary": map[string]interface{}{
			"total_gems":        data.TotalGems,
			"first_level_gems":  data.FirstLevelGems,
			"outdated_count":    data.OutdatedCount,
			"vulnerable_count":  data.VulnerableCount,
		},
		"gems": data.AllGems,
	}

	// Marshal to JSON with nice formatting
	jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return rg.writeOutput(string(jsonBytes), outputPath)
}

// writeOutput writes the report to a file or stdout
func (rg *ReportGenerator) writeOutput(content string, outputPath string) error {
	if outputPath == "" {
		// Write to stdout
		fmt.Print(content)
		return nil
	}

	// Write to file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Report written to: %s\n", outputPath)
	return nil
}

// Helper function to convert bool to string
func boolToString(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
