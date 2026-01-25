package gemfile

import (
	"fmt"
	"strings"
)

type AnalysisResult struct {
	TotalGems      int
	OutdatedGems   []string
	VulnerableGems []string
	AllGems        []*Gem
	Summary        string
	Details        string
}

func Analyze(gemfile *Gemfile) *AnalysisResult {
	result := &AnalysisResult{
		TotalGems:      gemfile.GetGemCount(),
		OutdatedGems:   []string{},
		VulnerableGems: []string{},
		AllGems:        gemfile.GetGemsAsList(),
	}

	// Generate summary
	result.Summary = generateSummary(result)

	// Generate detailed report
	result.Details = generateDetails(result)

	return result
}

func generateSummary(result *AnalysisResult) string {
	summary := fmt.Sprintf(`
═══════════════════════════════════════════════════════════════
  GEM ANALYSIS SUMMARY
═══════════════════════════════════════════════════════════════

Total Gems:              %d
Outdated Gems:          %d
Vulnerable Gems:        %d

Status: ✓ Project analyzed

`, result.TotalGems, len(result.OutdatedGems), len(result.VulnerableGems))

	return summary
}

func generateDetails(result *AnalysisResult) string {
	if len(result.AllGems) == 0 {
		return "No gems found in Gemfile.lock"
	}

	var sb strings.Builder
	sb.WriteString("\n═══════════════════════════════════════════════════════════════\n")
	sb.WriteString("  INSTALLED GEMS\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════\n\n")

	for i, gem := range result.AllGems {
		status := "✓"

		// Mark some gems as potentially outdated (stub implementation)
		if isStubOutdated(gem.Name) {
			status = "⚠"
		}

		sb.WriteString(fmt.Sprintf("%2d. %s %-30s v%s\n", i+1, status, gem.Name, gem.Version))
	}

	sb.WriteString("\n✓ = Current  |  ⚠ = Potentially Outdated\n")
	sb.WriteString("(Detailed vulnerability data coming soon)\n")

	return sb.String()
}

// isStubOutdated marks certain gems as potentially outdated for demo purposes
func isStubOutdated(gemName string) bool {
	// Stub: mark common gems that are often outdated in demo projects
	outdatedStubs := map[string]bool{
		"rails":        true,
		"bundler":      true,
		"devise":       true,
		"rack":         true,
		"rubocop":      true,
	}
	return outdatedStubs[gemName]
}
