package gemfile

import (
	"time"
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
	// Severity is the vulnerability severity level (CRITICAL, HIGH, MEDIUM, LOW)
	Severity string
	// CVSS is the CVSS score (0-10)
	CVSS float64
	// FixedVersion is the first version that fixes the vulnerability
	FixedVersion string
	// PublishedDate is when the vulnerability was published
	PublishedDate time.Time
	// References are links to additional information about the vulnerability
	References []string
	// OSVId is the OSV identifier (e.g., GHSA-xxxx or CVE-2021-xxxx)
	OSVId string
	// Source indicates where the vulnerability data came from (e.g., "osv.dev", "static")
	Source string
}
