package gemfile

import (
	"fmt"
	"strings"
)

// GemStatus represents the status information for a gem
type GemStatus struct {
	Name              string
	Version           string
	Groups            []string // e.g., "default", "development", "test"
	IsOutdated        bool
	LatestVersion     string // Latest available version
	IsVulnerable      bool
	VulnerabilityInfo string     // Detailed vulnerability info
	HomepageURL       string     // Homepage or source code URL
	Description       string     // Gem description from rubygems.org
	Health            *GemHealth // Gem health data (nil until fetched)
	OutdatedFailed    bool       // true if outdated check failed with an error
}

type AnalysisResult struct {
	TotalGems      int
	OutdatedGems   []string
	VulnerableGems []string
	FirstLevelGems []string // Names of directly installed gems (from Gemfile, not transitive)
	AllGems        []*Gem
	GemStatuses    []*GemStatus
	Summary        string
	Details        string
}

func Analyze(gemfile *Gemfile) *AnalysisResult {
	vulnChecker := NewVulnerabilityChecker()

	allGems := gemfile.GetGemsAsList()
	outdatedList := []string{}
	vulnerableList := []string{}
	firstLevelList := []string{}
	gemStatuses := make([]*GemStatus, 0, len(allGems))

	// Build a map of first-level gems for quick lookup
	firstLevelMap := make(map[string]bool)
	for _, name := range gemfile.FirstLevelGems {
		firstLevelMap[name] = true
	}

	// Check each gem for outdated and vulnerable status
	for _, gem := range allGems {
		status := &GemStatus{
			Name:    gem.Name,
			Version: gem.Version,
			Groups:  gem.Groups, // Copy group information
		}

		// Check if vulnerable
		hasVuln, cveID, vulnDesc := vulnChecker.HasVulnerability(gem.Name, gem.Version)
		if hasVuln {
			status.IsVulnerable = true
			status.VulnerabilityInfo = fmt.Sprintf("%s: %s", cveID, vulnDesc)
			vulnerableList = append(vulnerableList, gem.Name)
		}

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

func generateSummary(result *AnalysisResult) string {
	summary := fmt.Sprintf(`Total Gems: %d  |  Outdated: %d  |  Vulnerable: %d

Status: ✓ Project analyzed`,
		result.TotalGems, len(result.OutdatedGems), len(result.VulnerableGems))

	return summary
}

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
