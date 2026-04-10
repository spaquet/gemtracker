package ui

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spaquet/gemtracker/internal/gemfile"
	"github.com/spaquet/gemtracker/internal/logger"
)

// ReportGenerator generates gem dependency reports in multiple formats (text, CSV, JSON)
// for non-interactive CI/CD integration and compliance use cases.
type ReportGenerator struct {
	// projectPath is the path to the Ruby project (directory or Gemfile.lock file)
	projectPath string
	// noCache indicates whether to skip cached analysis results
	noCache bool
	// verbose enables detailed logging
	verbose bool
}

// ReportData holds structured gem analysis data suitable for export in multiple formats.
type ReportData struct {
	// GeneratedAt is the timestamp when the report was generated
	GeneratedAt string
	// ProjectPath is the path to the analyzed Ruby project
	ProjectPath string
	// TotalGems is the count of all gems (first-level and transitive)
	TotalGems int
	// FirstLevelGems is the count of directly required gems
	FirstLevelGems int
	// TransitiveDependencies is the count of transitive gem dependencies
	TransitiveDependencies int
	// OutdatedGems lists gems with available updates
	OutdatedGems []*GemReport
	// VulnerableGems lists gems with known CVEs
	VulnerableGems []*GemReport
	// AllGems lists all gems in the project
	AllGems []*GemReport
	// Summary is a brief text summary of findings
	Summary string
	// OutdatedCount is the count of gems with updates available
	OutdatedCount int
	// VulnerableCount is the count of gems with known vulnerabilities
	VulnerableCount int
}

// GemReport represents a gem's details in the generated report, including status and metadata.
type GemReport struct {
	// Name is the gem name
	Name string
	// Version is the currently installed version
	Version string
	// Groups lists the bundle groups this gem belongs to
	Groups []string
	// IsFirstLevel indicates whether this is a directly required gem
	IsFirstLevel bool
	// IsOutdated indicates whether a newer version is available
	IsOutdated bool
	// LatestVersion is the latest available version (if IsOutdated is true)
	LatestVersion string
	// IsVulnerable indicates whether known CVEs affect this version
	IsVulnerable bool
	// VulnerabilityInfo contains CVE ID and description (if IsVulnerable is true)
	VulnerabilityInfo string
	// HomepageURL is the gem's homepage or source code URL
	HomepageURL string
	// Description is the gem description from rubygems.org
	Description string
	// ReverseDeps lists the names of gems that depend on this gem
	ReverseDeps []string
}

// NewReportGenerator creates a new ReportGenerator for the given project path.
func NewReportGenerator(projectPath string, noCache, verbose bool) *ReportGenerator {
	return &ReportGenerator{
		projectPath: projectPath,
		noCache:     noCache,
		verbose:     verbose,
	}
}

// resolveOutputPath checks if outputPath already exists and prompts the user for action.
// Returns the final path to write to and whether to proceed (false = user cancelled).
func resolveOutputPath(outputPath string) (string, bool, error) {
	if outputPath == "" {
		return "", true, nil // stdout — no conflict possible
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return outputPath, true, nil // file doesn't exist yet — proceed
	}

	ext := filepath.Ext(outputPath)

	for {
		fmt.Fprintf(os.Stderr, "\n⚠ Output file already exists: %s\n", outputPath)
		fmt.Fprintf(os.Stderr, "  [R] Replace existing file\n")
		fmt.Fprintf(os.Stderr, "  [C] Cancel\n")
		fmt.Fprintf(os.Stderr, "  [N] Enter a new filename\n")
		fmt.Fprintf(os.Stderr, "Your choice [R/C/N]: ")

		var choice string
		fmt.Fscan(os.Stdin, &choice)

		switch strings.ToUpper(strings.TrimSpace(choice)) {
		case "R":
			return outputPath, true, nil
		case "C":
			return "", false, nil
		case "N":
			fmt.Fprintf(os.Stderr, "New filename (without extension to keep %s): ", ext)
			var newName string
			fmt.Fscan(os.Stdin, &newName)
			newName = strings.TrimSpace(newName)
			if newName == "" {
				continue
			}
			// Add original extension if user didn't provide one
			if filepath.Ext(newName) == "" {
				newName += ext
			}
			outputPath = newName
			// Re-check if new name also exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "✓ Output will be written to: %s\n\n", outputPath)
				return outputPath, true, nil
			}
			// Loop again — new file also exists
			continue
		default:
			fmt.Fprintf(os.Stderr, "Please enter R, C, or N.\n")
		}
	}
}

// Generate analyzes the project and generates a report in the specified format (text, csv, or json).
// If outputPath is empty, writes to stdout. Returns an error if analysis or report writing fails.
func (rg *ReportGenerator) Generate(format, outputPath string) error {
	// Resolve output path — prompt user if file already exists
	resolvedPath, proceed, err := resolveOutputPath(outputPath)
	if err != nil {
		return err
	}
	if !proceed {
		fmt.Fprintf(os.Stderr, "Report generation cancelled.\n")
		return nil
	}
	outputPath = resolvedPath // use the resolved (possibly new) path
	// Parse the Gemfile
	printProgress("Parsing Gemfile.lock...")
	gf, err := gemfile.Parse(rg.projectPath)
	if err != nil {
		return fmt.Errorf("failed to parse gemfile: %w", err)
	}
	logger.Info("Parsing Gemfile...")
	printProgressDone("✓ Parsed %d gems", len(gf.GetGemsAsList()))

	// Analyze gems
	logger.Info("Analyzing gems...")
	analysis := gemfile.Analyze(gf)

	// Check for outdated gems
	printProgress("Checking for outdated gems... ")
	logger.Info("Checking for outdated gems...")
	outdatedChecker := gemfile.NewOutdatedChecker()

	// Build gem status map
	gemStatusMap := make(map[string]*gemfile.GemStatus)
	for _, status := range analysis.GemStatuses {
		gemStatusMap[status.Name] = status
	}

	// Check each gem for updates
	gems := gf.GetGemsAsList()
	total := len(gems)
	outdatedCount := 0
	for i, gem := range gems {
		status, ok := gemStatusMap[gem.Name]
		if !ok {
			continue
		}

		printProgress("Checking for outdated gems... (%d/%d) %s", i+1, total, gem.Name)

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
			outdatedCount++
		}
	}
	printProgressDone("✓ Checked %d gems for updates (%d outdated)", total, outdatedCount)

	// Check for vulnerabilities using OSV.dev
	printProgress("Scanning for vulnerabilities...")
	logger.Info("Scanning for vulnerabilities...")
	vulns, err := rg.scanVulnerabilities(analysis.AllGems)
	if err != nil {
		logger.Warn("Failed to scan vulnerabilities: %v", err)
		printProgressDone("⚠ Vulnerability scan failed, continuing without CVE data")
		// Continue with report generation, just without CVE data
	} else {
		// Merge vulnerability data into gem statuses
		rg.mergeVulnerabilitiesIntoGems(analysis.GemStatuses, vulns)
		printProgressDone("✓ Found %d vulnerabilities", len(vulns))
	}

	// Build report data
	printProgress("Building %s report...", strings.ToUpper(format))
	reportData := rg.buildReportData(analysis, gemStatusMap, gf)
	printProgressDone("✓ Report generated")

	// Generate report based on format
	var reportErr error
	switch strings.ToLower(format) {
	case "text":
		reportErr = rg.generateTextReport(reportData, outputPath)
	case "csv":
		reportErr = rg.generateCSVReport(reportData, outputPath)
	case "json":
		reportErr = rg.generateJSONReport(reportData, outputPath)
	default:
		return fmt.Errorf("unknown format: %s (supported: text, csv, json)", format)
	}

	if reportErr != nil {
		return reportErr
	}

	// Show final status
	if outputPath != "" {
		printProgressDone("✓ Report written to: %s", outputPath)
	}

	return nil
}

// buildReportData builds structured report data from analysis results
func (rg *ReportGenerator) buildReportData(analysis *gemfile.AnalysisResult, gemStatusMap map[string]*gemfile.GemStatus, gf *gemfile.Gemfile) *ReportData {
	// Build first-level gem map
	firstLevelMap := make(map[string]bool)
	for _, name := range gf.FirstLevelGems {
		firstLevelMap[name] = true
	}

	// Build reverse dependency map: gem name → list of gems that depend on it
	reverseDepMap := make(map[string][]string)
	for _, gem := range gf.Gems {
		for _, dep := range gem.Dependencies {
			reverseDepMap[dep] = append(reverseDepMap[dep], gem.Name)
		}
	}
	// Sort reverse deps for consistent output
	for _, deps := range reverseDepMap {
		sort.Strings(deps)
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
			ReverseDeps:       reverseDepMap[status.Name],
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

	// Calculate transitive dependencies
	transitiveDeps := len(allGems) - firstLevelCount

	// Build summary
	summary := fmt.Sprintf("Total gems: %d, Direct: %d, Transitive: %d, Outdated: %d, Vulnerable: %d",
		len(allGems), firstLevelCount, transitiveDeps, len(outdatedGems), len(vulnerableGems))

	// Extract project directory name from Gemfile path for display
	absPath, err := filepath.Abs(gf.Path)
	if err != nil {
		absPath = gf.Path
	}
	projectDir := filepath.Base(filepath.Dir(absPath))

	return &ReportData{
		GeneratedAt:            time.Now().Format(time.RFC3339),
		ProjectPath:            projectDir,
		TotalGems:              len(allGems),
		FirstLevelGems:         firstLevelCount,
		TransitiveDependencies: transitiveDeps,
		AllGems:                allGems,
		OutdatedGems:           outdatedGems,
		VulnerableGems:         vulnerableGems,
		Summary:                summary,
		OutdatedCount:          len(outdatedGems),
		VulnerableCount:        len(vulnerableGems),
	}
}

// groupGemsByGroup partitions gems into a map keyed by their bundle groups.
// Gems with no group are placed under "-".
// A gem may appear in multiple groups if it belongs to more than one.
func groupGemsByGroup(gems []*GemReport) map[string][]*GemReport {
	result := make(map[string][]*GemReport)
	for _, gem := range gems {
		if len(gem.Groups) == 0 {
			result["-"] = append(result["-"], gem)
		} else {
			for _, g := range gem.Groups {
				result[g] = append(result[g], gem)
			}
		}
	}
	return result
}

// sortedGroupKeys returns the group keys in canonical order:
// default, production, development, test, staging, then other groups alphabetically, then "-" last.
func sortedGroupKeys(gemsByGroup map[string][]*GemReport) []string {
	canonicalOrder := []string{"default", "production", "development", "test", "staging"}
	result := []string{}

	// First add canonical groups that have gems
	for _, group := range canonicalOrder {
		if len(gemsByGroup[group]) > 0 {
			result = append(result, group)
		}
	}

	// Then add other groups alphabetically (excluding "-")
	otherGroups := []string{}
	for group := range gemsByGroup {
		found := false
		for _, canonical := range canonicalOrder {
			if group == canonical {
				found = true
				break
			}
		}
		if !found && group != "-" {
			otherGroups = append(otherGroups, group)
		}
	}
	sort.Strings(otherGroups)
	result = append(result, otherGroups...)

	// Finally add "-" if it has gems
	if len(gemsByGroup["-"]) > 0 {
		result = append(result, "-")
	}

	return result
}

// writeGroupedGems writes gems grouped by bundle group, with direct/transitive markers and reverse dependencies.
// mode: "outdated" shows version updates (old → new); "all" shows current version with status markers.
func writeGroupedGems(output *strings.Builder, gems []*GemReport, mode string) {
	byGroup := groupGemsByGroup(gems)

	for _, group := range sortedGroupKeys(byGroup) {
		output.WriteString(fmt.Sprintf("Group: %s\n", group))
		groupGems := byGroup[group]

		// Sort: direct first, then transitive; alphabetical within each
		sort.Slice(groupGems, func(i, j int) bool {
			if groupGems[i].IsFirstLevel != groupGems[j].IsFirstLevel {
				return groupGems[i].IsFirstLevel
			}
			return groupGems[i].Name < groupGems[j].Name
		})

		for _, gem := range groupGems {
			marker := "[transitive]"
			if gem.IsFirstLevel {
				marker = "[direct]    "
			}

			var versionStr string
			if mode == "outdated" {
				versionStr = fmt.Sprintf("(%s → %s)", gem.Version, gem.LatestVersion)
			} else {
				versionStr = fmt.Sprintf("@%s", gem.Version)
				// Add status markers
				if gem.IsVulnerable {
					versionStr += " [VULNERABLE]"
				} else if gem.IsOutdated {
					versionStr += " [OUTDATED]"
				}
			}

			line := fmt.Sprintf("  %s %s %s", marker, gem.Name, versionStr)

			// Add reverse dependencies
			if len(gem.ReverseDeps) > 0 {
				line += fmt.Sprintf("  [used by: %s]", strings.Join(gem.ReverseDeps, ", "))
			}

			output.WriteString(line + "\n")
		}
		output.WriteString("\n")
	}
}

// generateTextReport generates a human-readable text report
func (rg *ReportGenerator) generateTextReport(data *ReportData, outputPath string) error {
	var output strings.Builder

	// Header
	output.WriteString("GEMTRACKER REPORT\n")
	output.WriteString("==================\n\n")

	output.WriteString(fmt.Sprintf("Generated: %s\n", data.GeneratedAt))
	output.WriteString(fmt.Sprintf("Project: %s\n\n", data.ProjectPath))

	// Gem statistics
	output.WriteString("Total Gems: " + fmt.Sprintf("%d\n", data.TotalGems))
	output.WriteString("  Direct Dependencies: " + fmt.Sprintf("%d\n", data.FirstLevelGems))
	output.WriteString("  Transitive Dependencies: " + fmt.Sprintf("%d\n", data.TransitiveDependencies))
	output.WriteString("\n")

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
		writeGroupedGems(&output, data.OutdatedGems, "outdated")
	}

	// All gems section
	output.WriteString("ALL GEMS\n")
	output.WriteString(strings.Repeat("-", 80) + "\n")
	writeGroupedGems(&output, data.AllGems, "all")

	// Write to file or stdout
	return rg.writeOutput(output.String(), outputPath)
}

// generateCSVReport generates a CSV report
func (rg *ReportGenerator) generateCSVReport(data *ReportData, outputPath string) error {
	var output strings.Builder

	// Add summary header comments
	output.WriteString(fmt.Sprintf("# Generated: %s\n", data.GeneratedAt))
	output.WriteString(fmt.Sprintf("# Project: %s\n", data.ProjectPath))
	output.WriteString(fmt.Sprintf("# Total Gems: %d\n", data.TotalGems))
	output.WriteString(fmt.Sprintf("# Direct Dependencies: %d\n", data.FirstLevelGems))
	output.WriteString(fmt.Sprintf("# Transitive Dependencies: %d\n", data.TransitiveDependencies))
	output.WriteString(fmt.Sprintf("# Outdated: %d, Vulnerable: %d\n#\n", data.OutdatedCount, data.VulnerableCount))

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

	return rg.writeOutput(output.String(), outputPath)
}

// generateJSONReport generates a JSON report
func (rg *ReportGenerator) generateJSONReport(data *ReportData, outputPath string) error {
	// Create a JSON-friendly structure
	jsonData := map[string]interface{}{
		"generated_at": data.GeneratedAt,
		"project_path": data.ProjectPath,
		"summary": map[string]interface{}{
			"total_gems":              data.TotalGems,
			"direct_dependencies":     data.FirstLevelGems,
			"transitive_dependencies": data.TransitiveDependencies,
			"outdated_count":          data.OutdatedCount,
			"vulnerable_count":        data.VulnerableCount,
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

	return nil
}

// Helper function to convert bool to string
func boolToString(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// printProgress writes a carriage-return-overwritten progress line to stderr.
// Always uses \r to overwrite the same line, so the terminal stays clean.
func printProgress(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\r"+format, args...)
}

// printProgressDone clears the progress line and prints a final status to stderr.
func printProgressDone(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\r%-80s\r", "") // clear line
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// scanVulnerabilities queries OSV.dev for vulnerabilities in the given gems
func (rg *ReportGenerator) scanVulnerabilities(gems []*gemfile.Gem) ([]*gemfile.Vulnerability, error) {
	if len(gems) == 0 {
		return []*gemfile.Vulnerability{}, nil
	}

	// Compute gems signature for cache key
	gemsSignature := gemfile.ComputeGemsSignature(gems)

	// Try to load from cache first
	logger.Info("Checking CVE cache...")
	cacheEntry, err := gemfile.LoadVulnerabilityCache(gemsSignature)
	if err == nil && cacheEntry != nil && gemfile.IsCacheValid(cacheEntry) {
		// Cache hit! Return cached data
		logger.Info("CVE cache hit: using cached vulnerabilities")
		printProgress("Using cached vulnerability data...")
		vulnPtrs := make([]*gemfile.Vulnerability, len(cacheEntry.Vulnerabilities))
		for i := range cacheEntry.Vulnerabilities {
			vulnPtrs[i] = &cacheEntry.Vulnerabilities[i]
		}
		return vulnPtrs, nil
	}

	// Cache miss, fetch from OSV.dev
	logger.Info("CVE cache miss, fetching from OSV.dev...")
	printProgress("Scanning for vulnerabilities... (querying OSV.dev)")
	osv := gemfile.NewOSVClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vulns, err := osv.QueryBatch(ctx, gems)
	if err != nil {
		logger.Warn("Failed to query OSV.dev: %v", err)
		return nil, err
	}

	// Save to cache
	cacheEntry = &gemfile.CacheEntry{
		GemsSignature:   gemsSignature,
		CachedAt:        time.Now(),
		ScannedAt:       time.Now(),
		TTLSeconds:      int(gemfile.VulnerabilityCacheTTL.Seconds()),
		GemCount:        len(gems),
		ScanStatus:      "success",
		Vulnerabilities: vulns,
	}

	if err := gemfile.SaveVulnerabilityCache(gemsSignature, cacheEntry); err != nil {
		logger.Warn("Failed to save CVE cache: %v", err)
	}

	// Convert to pointers for return
	vulnPtrs := make([]*gemfile.Vulnerability, len(vulns))
	for i := range vulns {
		vulnPtrs[i] = &vulns[i]
	}

	return vulnPtrs, nil
}

// mergeVulnerabilitiesIntoGems updates gem statuses with vulnerability information
func (rg *ReportGenerator) mergeVulnerabilitiesIntoGems(gemStatuses []*gemfile.GemStatus, vulnerabilities []*gemfile.Vulnerability) {
	// Build a map of vulnerabilities by gem name
	vulnByGem := make(map[string][]*gemfile.Vulnerability)
	for _, vuln := range vulnerabilities {
		vulnByGem[vuln.GemName] = append(vulnByGem[vuln.GemName], vuln)
	}

	// Update gem statuses with vulnerability info
	for _, gemStatus := range gemStatuses {
		if vulns, hasVulns := vulnByGem[gemStatus.Name]; hasVulns && len(vulns) > 0 {
			gemStatus.IsVulnerable = true
			// Use the first vulnerability for the summary info
			vuln := vulns[0]
			// Include severity in the vulnerability info, matching UI display
			info := fmt.Sprintf("%s [%s]: %s", vuln.CVE, vuln.Severity, vuln.Description)
			// Append CVSS score if available
			if vuln.CVSS > 0 {
				info += fmt.Sprintf(" (CVSS: %.1f)", vuln.CVSS)
			}
			gemStatus.VulnerabilityInfo = info
		} else {
			gemStatus.IsVulnerable = false
			gemStatus.VulnerabilityInfo = ""
		}
	}
}
