package gemfile

import (
	"strings"
)

// Vulnerability represents a known vulnerability
type Vulnerability struct {
	GemName          string
	AffectedVersions []string // e.g., "< 6.1.4", ">= 6.0.0, < 6.0.5"
	Description      string
	CVE              string
}

// VulnerabilityChecker checks if gems have known vulnerabilities
type VulnerabilityChecker struct {
	vulnerabilities []Vulnerability
}

// NewVulnerabilityChecker creates a new checker with known vulnerabilities
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

// HasVulnerability checks if a gem has known vulnerabilities
// Returns (hasVulnerability, cveID, description)
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

// versionIsAffected checks if a version matches the affected version spec
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

// matchesSpec checks if a version matches a spec like "< 6.1.4" or ">= 6.0.0"
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
