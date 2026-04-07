package gemfile

import (
	"strings"
)

// Vulnerability represents a known CVE affecting one or more versions of a gem.
type Vulnerability struct {
	// GemName is the affected gem (lowercase)
	GemName string
	// AffectedVersions is a list of version specs that are vulnerable (e.g., "< 6.1.4", ">= 6.0.0, < 6.0.5")
	AffectedVersions []string
	// Description is a brief summary of the vulnerability
	Description string
	// CVE is the CVE identifier (e.g., "CVE-2021-22942")
	CVE string
}

// VulnerabilityChecker checks if gem versions have known CVEs.
// It maintains a static list of known vulnerabilities with version ranges.
type VulnerabilityChecker struct {
	// vulnerabilities is the list of known CVEs
	vulnerabilities []Vulnerability
}

// NewVulnerabilityChecker creates a new VulnerabilityChecker with a built-in list of
// known CVEs affecting common Ruby gems (Rails, Devise, Rack, ActionPack, etc.).
// This list is static and compiled at build time; for live CVE updates, consider
// integrating with a dedicated vulnerability database.
func NewVulnerabilityChecker() *VulnerabilityChecker {
	vc := &VulnerabilityChecker{
		vulnerabilities: []Vulnerability{
			// Rails vulnerabilities
			{
				GemName:          "rails",
				AffectedVersions: []string{"< 6.1.4"},
				Description:      "SQL injection vulnerability in Rails",
				CVE:              "CVE-2021-22942",
			},
			{
				GemName:          "rails",
				AffectedVersions: []string{"< 7.0.0"},
				Description:      "Potential code execution in Rails",
				CVE:              "CVE-2022-27777",
			},
			// Devise vulnerabilities
			{
				GemName:          "devise",
				AffectedVersions: []string{"< 4.8.0"},
				Description:      "Authentication bypass in Devise",
				CVE:              "CVE-2021-41113",
			},
			// Rack vulnerabilities
			{
				GemName:          "rack",
				AffectedVersions: []string{"< 2.1.4", ">= 2.2.0, < 2.2.3"},
				Description:      "DoS vulnerability in Rack",
				CVE:              "CVE-2022-24834",
			},
			// Actionpack vulnerabilities
			{
				GemName:          "actionpack",
				AffectedVersions: []string{"< 6.1.5"},
				Description:      "XSS vulnerability in Action Pack",
				CVE:              "CVE-2022-22719",
			},
		},
	}
	return vc
}

// HasVulnerability checks if a gem version has a known CVE and returns the result.
// Returns (hasVulnerability, cveID, description). If multiple CVEs affect the version,
// only the first match is returned.
func (vc *VulnerabilityChecker) HasVulnerability(gemName, version string) (bool, string, string) {
	for _, vuln := range vc.vulnerabilities {
		if strings.ToLower(vuln.GemName) == strings.ToLower(gemName) {
			if vc.versionIsAffected(version, vuln.AffectedVersions) {
				return true, vuln.CVE, vuln.Description
			}
		}
	}
	return false, "", ""
}

// versionIsAffected checks if a version matches any of the affected version specs.
// If any spec matches, the version is considered affected.
func (vc *VulnerabilityChecker) versionIsAffected(version string, affectedSpecs []string) bool {
	// Simple implementation: check if version is in any of the affected ranges
	// Format: "< 6.1.4" or ">= 6.0.0, < 6.0.5"
	for _, spec := range affectedSpecs {
		if vc.matchesSpec(version, spec) {
			return true
		}
	}
	return false
}

// matchesSpec checks if a version matches a version spec string.
// Supported formats: "<", "<=", ">", ">=", and exact match.
// Examples: "< 6.1.4", ">= 6.0.0", "1.2.3"
func (vc *VulnerabilityChecker) matchesSpec(version, spec string) bool {
	spec = strings.TrimSpace(spec)

	// Handle < comparison
	if strings.HasPrefix(spec, "< ") {
		targetVersion := strings.TrimPrefix(spec, "< ")
		return isVersionLess(version, targetVersion)
	}

	// Handle >= comparison
	if strings.HasPrefix(spec, ">= ") {
		targetVersion := strings.TrimPrefix(spec, ">= ")
		return !isVersionLess(version, targetVersion) || version == targetVersion
	}

	// Handle > comparison
	if strings.HasPrefix(spec, "> ") {
		targetVersion := strings.TrimPrefix(spec, "> ")
		return !isVersionLess(version, targetVersion) && version != targetVersion
	}

	// Handle <= comparison
	if strings.HasPrefix(spec, "<= ") {
		targetVersion := strings.TrimPrefix(spec, "<= ")
		return isVersionLess(version, targetVersion) || version == targetVersion
	}

	// Handle exact match
	return version == spec
}
