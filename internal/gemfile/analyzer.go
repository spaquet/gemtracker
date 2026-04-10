package gemfile

import (
	"fmt"
	"strings"
)

// GemStatus represents the current status and metadata of a gem, including its version,
// group assignments, and vulnerability/outdated status with additional information.
type GemStatus struct {
	// Name is the lowercase gem name
	Name string
	// Version is the currently installed version
	Version string
	// Groups lists the bundle groups this gem belongs to (e.g., "default", "development", "test")
	Groups []string
	// IsOutdated indicates whether a newer version is available
	IsOutdated bool
	// LatestVersion is the latest available version (only set if IsOutdated is true)
	LatestVersion string
	// IsVulnerable indicates whether known CVEs affect this gem version
	IsVulnerable bool
	// VulnerabilityInfo contains CVE ID and description (only set if IsVulnerable is true)
	VulnerabilityInfo string
	// VulnerabilityURL is the canonical OSV advisory URL (only set if IsVulnerable is true)
	VulnerabilityURL string
	// HomepageURL is the gem's homepage or source code repository URL
	HomepageURL string
	// Description is the gem description from rubygems.org
	Description string
	// Health contains gem maintenance status data (nil until fetched asynchronously)
	Health *GemHealth
	// OutdatedFailed is true if the outdated version check failed with an error
	OutdatedFailed bool
}

// AnalysisResult contains the results of analyzing a Gemfile.lock for vulnerabilities,
// outdated gems, and other quality metrics.
type AnalysisResult struct {
	// TotalGems is the total number of gems (first-level and transitive dependencies)
	TotalGems int
	// OutdatedGems is a list of gem names with available updates
	OutdatedGems []string
	// VulnerableGems is a list of gem names with known CVEs
	VulnerableGems []string
	// FirstLevelGems is a list of gem names directly required (in Gemfile/Gemfile.lock DEPENDENCIES)
	FirstLevelGems []string
	// AllGems is the complete list of parsed Gem objects
	AllGems []*Gem
	// GemStatuses is detailed status information for each gem (outdated, vulnerable, health, etc.)
	GemStatuses []*GemStatus
	// Summary is a brief one-line summary of the analysis results
	Summary string
	// Details is a detailed report of all gems and their status
	Details string
}

// Analyze performs a complete security and version analysis of a parsed Gemfile.
// It identifies first-level dependencies and prepares gem statuses for analysis.
// Vulnerability checking is done asynchronously via OSV.dev (not here) to avoid blocking
// and to use the authoritative live vulnerability database. Outdated version checking is
// also done separately by the UI.
func Analyze(gemfile *Gemfile) *AnalysisResult {
	allGems := gemfile.GetGemsAsList()
	outdatedList := []string{}
	vulnerableList := []string{} // Will be populated by OSV.dev async, not here
	firstLevelList := []string{}
	gemStatuses := make([]*GemStatus, 0, len(allGems))

	// Build a map of first-level gems for quick lookup
	firstLevelMap := make(map[string]bool)
	for _, name := range gemfile.FirstLevelGems {
		firstLevelMap[name] = true
	}

	// Check each gem for vulnerable and outdated status
	for _, gem := range allGems {
		status := &GemStatus{
			Name:    gem.Name,
			Version: gem.Version,
			Groups:  gem.Groups, // Copy group information
		}

		// Note: Vulnerability checking is deferred to OSV.dev async scan in the UI.
		// Do not use static vulnerability list - it gets out of sync with live data.
		// IsVulnerable will be set by the UI when CVE scan completes.

		// Track first-level gems (those in DEPENDENCIES section of Gemfile.lock, not transitive deps)
		// First, check if it was explicitly marked in DEPENDENCIES section
		// Fall back to checking for group information (for compatibility with Gemfile parsing)
		isFirstLevel := gem.IsFirstLevel || firstLevelMap[gem.Name] || len(gem.Groups) > 0
		if isFirstLevel {
			firstLevelList = append(firstLevelList, gem.Name)
		}

		gemStatuses = append(gemStatuses, status)
	}

	result := &AnalysisResult{
		TotalGems:      gemfile.GetGemCount(),
		OutdatedGems:   outdatedList,
		VulnerableGems: vulnerableList,
		FirstLevelGems: firstLevelList,
		AllGems:        allGems,
		GemStatuses:    gemStatuses,
	}

	// Generate summary
	result.Summary = generateSummary(result)

	// Generate detailed report
	result.Details = generateDetails(result)

	return result
}

// generateSummary creates a brief one-line summary of analysis results showing total gems,
// outdated count, and vulnerable count.
func generateSummary(result *AnalysisResult) string {
	summary := fmt.Sprintf(`Total Gems: %d  |  Outdated: %d  |  Vulnerable: %d

Status: ✓ Project analyzed`,
		result.TotalGems, len(result.OutdatedGems), len(result.VulnerableGems))

	return summary
}

// generateDetails creates a detailed multi-line report of all gems with status indicators:
// ✓ for healthy gems, ⚠ for outdated gems, and 🔒 for vulnerable gems.
func generateDetails(result *AnalysisResult) string {
	if len(result.GemStatuses) == 0 {
		return "No gems found in Gemfile.lock"
	}

	var sb strings.Builder

	for _, gemStatus := range result.GemStatuses {
		status := "✓"

		// Mark gems with issues
		if gemStatus.IsVulnerable {
			status = "🔒"
		} else if gemStatus.IsOutdated {
			status = "⚠"
		}

		sb.WriteString(fmt.Sprintf("%s %-30s v%s\n", status, gemStatus.Name, gemStatus.Version))
	}

	return sb.String()
}
