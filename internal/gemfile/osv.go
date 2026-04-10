package gemfile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spaquet/gemtracker/internal/logger"
	"golang.org/x/time/rate"
)

const (
	OSVRequestTimeout = 30 * time.Second
)

var (
	OSVBatchEndpoint      = "https://api.osv.dev/v1/querybatch"
	OSVVulnDetailEndpoint = "https://api.osv.dev/v1/vulns"
)

// OSVQueryRequest represents a single query in the batch request
type OSVQueryRequest struct {
	Package OSVPackage `json:"package"`
	Version string     `json:"version"`
}

// OSVPackage represents the package info in a query
type OSVPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

// OSVBatchRequest is the batch query request to OSV.dev
type OSVBatchRequest struct {
	Queries []OSVQueryRequest `json:"queries"`
}

// OSVBatchResponse is the response from OSV.dev batch endpoint
type OSVBatchResponse struct {
	Results []OSVResult `json:"results"`
}

// OSVResult is a single vulnerability result from OSV.dev
type OSVResult struct {
	Vulns []OSVVulnerability `json:"vulns"`
}

// OSVVulnerability represents a vulnerability from OSV.dev
type OSVVulnerability struct {
	ID               string                   `json:"id"`
	Summary          string                   `json:"summary"`
	Details          string                   `json:"details"`
	Published        string                   `json:"published"`
	Modified         string                   `json:"modified"`
	Severity         []map[string]interface{} `json:"severity"`          // Array of severity objects with type and score (CVSS string)
	DatabaseSpecific map[string]interface{}   `json:"database_specific"` // Contains severity for GitHub reviewed vulns
	References       []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"references"`
	Affected []struct {
		Package struct {
			Name      string `json:"name"`
			Ecosystem string `json:"ecosystem"`
		} `json:"package"`
		Ranges []struct {
			Type   string `json:"type"`
			Events []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			} `json:"events"`
		} `json:"ranges"`
		EcosystemSpecific map[string]interface{} `json:"ecosystem_specific"` // Contains severity for some ecosystems
	} `json:"affected"`
}

// OSVClient queries the OSV.dev API for vulnerability data
type OSVClient struct {
	httpClient *http.Client
}

// NewOSVClient creates a new OSV.dev client
func NewOSVClient() *OSVClient {
	return &OSVClient{
		httpClient: &http.Client{
			Timeout: OSVRequestTimeout,
		},
	}
}

// QueryBatch queries OSV.dev with a batch of gems
// Returns vulnerabilities found for gems that have them
// Filters out clean gems (those with no vulnerabilities)
func (c *OSVClient) QueryBatch(ctx context.Context, gems []*Gem) ([]Vulnerability, error) {
	if len(gems) == 0 {
		logger.Info("OSV batch query: no gems to scan")
		return []Vulnerability{}, nil
	}

	logger.Info("Starting OSV batch query for %d gems", len(gems))

	// Build batch request
	queries := make([]OSVQueryRequest, len(gems))
	for i, gem := range gems {
		queries[i] = OSVQueryRequest{
			Package: OSVPackage{
				Name:      gem.Name,
				Ecosystem: "RubyGems",
			},
			Version: gem.Version,
		}
	}

	reqBody := OSVBatchRequest{Queries: queries}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		logger.Warn("Failed to marshal OSV request: %v", err)
		return nil, fmt.Errorf("failed to marshal OSV request: %w", err)
	}

	// Make request
	logger.Info("Sending batch request to OSV.dev (endpoint: %s)", OSVBatchEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", OSVBatchEndpoint, bytes.NewReader(reqJSON))
	if err != nil {
		logger.Warn("Failed to create OSV request: %v", err)
		return nil, fmt.Errorf("failed to create OSV request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gemtracker/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Warn("OSV API request failed: %v", err)
		return nil, fmt.Errorf("OSV API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Warn("OSV API returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("OSV API returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.Info("OSV API response received (HTTP %d)", resp.StatusCode)

	// Parse response
	var batchResp OSVBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		logger.Warn("Failed to parse OSV response: %v", err)
		return nil, fmt.Errorf("failed to parse OSV response: %w", err)
	}

	logger.Info("Parsing OSV response: %d results", len(batchResp.Results))

	// Log sample vulnerability for debugging CVSS extraction
	if len(batchResp.Results) > 0 {
		for i, result := range batchResp.Results {
			if len(result.Vulns) > 0 {
				firstVuln := result.Vulns[0]
				logger.Info("[OSV Response Sample %d] CVE: %s, Has severity array: %v", i, firstVuln.ID, len(firstVuln.Severity) > 0)
				break
			}
		}
	}

	// Convert OSV vulnerabilities to our format, filtering clean gems
	vulns := c.parseOSVResponse(batchResp, gems)

	// Enrich vulnerabilities with detailed CVSS/Severity data
	// The batch endpoint doesn't include this, so we need individual requests
	logger.Info("Enriching %d vulnerabilities with detailed CVSS/Severity data...", len(vulns))

	// Convert to pointers for enrichment
	vulnPtrs := make([]*Vulnerability, len(vulns))
	for i := range vulns {
		vulnPtrs[i] = &vulns[i]
	}
	c.EnrichVulnerabilitiesWithDetails(ctx, vulnPtrs)

	logger.Info("OSV batch query complete: found %d vulnerabilities", len(vulns))
	return vulns, nil
}

// parseOSVResponse converts OSV.dev response to our Vulnerability structs
// Only returns vulnerabilities for gems that have them (filters out clean gems)
func (c *OSVClient) parseOSVResponse(resp OSVBatchResponse, gems []*Gem) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Build a map of gems for quick lookup
	gemMap := make(map[string]*Gem)
	for _, gem := range gems {
		gemMap[gem.Name] = gem
	}

	// Process each result
	for resultIdx, result := range resp.Results {
		if resultIdx >= len(gems) {
			break
		}

		gem := gems[resultIdx]

		if len(result.Vulns) > 0 {
			logger.Info("Found %d vulnerabilities for %s@%s", len(result.Vulns), gem.Name, gem.Version)
		}

		// Only add vulnerabilities for this gem if there are any
		for _, osvVuln := range result.Vulns {
			// Extract CVSS score and severity from OSV response
			cvssScore, severityStr := extractCVSSData(osvVuln)

			// Determine severity: use CVSS-based level if available, otherwise use OSV severity string
			severity := determineSeverityFromCVSS(cvssScore)
			if severity == "" {
				severity = normalizeSeverity(severityStr)
			}

			vuln := Vulnerability{
				GemName:     gem.Name,
				CVE:         osvVuln.ID,
				Description: osvVuln.Summary,
				Severity:    severity,
				CVSS:        cvssScore,
				OSVId:       osvVuln.ID,
				Source:      "osv.dev",
			}

			// Parse dates
			if osvVuln.Published != "" {
				if t, err := time.Parse(time.RFC3339, osvVuln.Published); err == nil {
					vuln.PublishedDate = t
				}
			}

			// Extract fixed version and affected ranges
			vuln.AffectedVersions = extractVersionRanges(&osvVuln)
			vuln.FixedVersion = extractFixedVersion(&osvVuln)

			// Add references
			for _, ref := range osvVuln.References {
				if ref.URL != "" {
					vuln.References = append(vuln.References, ref.URL)
				}
			}

			// Extract workarounds from details
			if osvVuln.Details != "" {
				vuln.Workarounds = extractWorkarounds(osvVuln.Details)
			}

			logger.Info("✓ CVE %s [%s] (CVSS: %.1f) - %s | Gem: %s@%s", osvVuln.ID, vuln.Severity, vuln.CVSS, osvVuln.Summary, gem.Name, gem.Version)
			vulnerabilities = append(vulnerabilities, vuln)
		}
	}

	return vulnerabilities
}

// normalizeSeverity ensures severity is in expected format
func normalizeSeverity(severity string) string {
	switch severity {
	case "CRITICAL", "HIGH", "MODERATE", "LOW":
		return severity
	default:
		if severity == "" {
			return "MODERATE" // Default if not specified
		}
		return severity
	}
}

// extractVersionRanges converts OSV event format to our VersionRange format
func extractVersionRanges(osVVuln *OSVVulnerability) []string {
	// For now, return a simple string representation
	// Full VersionRange struct can be used in future enhancements
	var ranges []string

	if len(osVVuln.Affected) > 0 {
		affected := osVVuln.Affected[0]
		if len(affected.Ranges) > 0 {
			rangeData := affected.Ranges[0]
			for _, event := range rangeData.Events {
				if event.Introduced != "" && event.Fixed != "" {
					ranges = append(ranges, fmt.Sprintf("%s < %s", event.Introduced, event.Fixed))
				} else if event.Introduced != "" {
					ranges = append(ranges, fmt.Sprintf(">= %s", event.Introduced))
				}
			}
		}
	}

	return ranges
}

// extractFixedVersion gets the fixed version from OSV response
func extractFixedVersion(osVVuln *OSVVulnerability) string {
	// Return the first fixed version found
	if len(osVVuln.Affected) > 0 {
		affected := osVVuln.Affected[0]
		if len(affected.Ranges) > 0 {
			rangeData := affected.Ranges[0]
			for _, event := range rangeData.Events {
				if event.Fixed != "" {
					return event.Fixed
				}
			}
		}
	}
	return ""
}

// determineSeverityFromCVSS maps CVSS score to severity level
// According to CVSS v3.1: None (0.0), Low (0.1-3.9), Moderate (4.0-6.9), High (7.0-8.9), Critical (9.0-10.0)
func determineSeverityFromCVSS(cvssScore float64) string {
	switch {
	case cvssScore >= 9.0:
		return "CRITICAL"
	case cvssScore >= 7.0:
		return "HIGH"
	case cvssScore >= 4.0:
		return "MODERATE"
	case cvssScore > 0:
		return "LOW"
	}
	return ""
}

// extractCVSSData extracts CVSS score and severity from OSV vulnerability response
// For GitHub-reviewed vulnerabilities (RubyGems), severity is in database_specific.severity
// CVSS is in the severity array as a CVSS string vector (e.g., "CVSS:3.1/AV:N/AC:L/...")
func extractCVSSData(osvVuln OSVVulnerability) (float64, string) {
	cvssScore := 0.0
	severity := ""

	// Primary source: database_specific.severity (GitHub reviewed vulnerabilities)
	if osvVuln.DatabaseSpecific != nil {
		if sevVal, ok := osvVuln.DatabaseSpecific["severity"]; ok {
			if sevStr, ok := sevVal.(string); ok {
				severity = sevStr
				logger.Info("CVE %s: Severity from database_specific = %s", osvVuln.ID, sevStr)
			}
		}
	}

	// Fallback: check affected[].ecosystem_specific.severity
	if severity == "" && len(osvVuln.Affected) > 0 {
		affected := osvVuln.Affected[0]
		if affected.EcosystemSpecific != nil {
			if sevVal, ok := affected.EcosystemSpecific["severity"]; ok {
				if sevStr, ok := sevVal.(string); ok {
					severity = sevStr
					logger.Info("CVE %s: Severity from affected[0].ecosystem_specific = %s", osvVuln.ID, sevStr)
				}
			}
		}
	}

	// Extract CVSS score from severity array (CVSS string vector)
	// Example: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"
	// We can't easily calculate the score from the vector, so we look for a separate score field
	if len(osvVuln.Severity) > 0 {
		for _, sevEntry := range osvVuln.Severity {
			// Check if this entry has a score field (different from the vector string)
			if score, ok := sevEntry["score"]; ok {
				switch s := score.(type) {
				case float64:
					cvssScore = s
					logger.Info("CVE %s: CVSS score from severity array = %.1f", osvVuln.ID, s)
					return cvssScore, severity
				case string:
					// Score might be a string representation
					if strings.Contains(s, "CVSS") {
						logger.Info("CVE %s: Found CVSS vector but no numeric score: %s", osvVuln.ID, s)
					}
				}
			}
		}
	}

	logger.Info("CVE %s: Final extracted - CVSS: %.1f, Severity: %s", osvVuln.ID, cvssScore, severity)
	return cvssScore, severity
}

// EnrichVulnerabilitiesWithDetails fetches detailed CVSS/Severity data and workarounds for vulnerabilities
// The batch endpoint doesn't include this data, so we query individual vulnerabilities
// Uses rate limiting to avoid overwhelming the OSV API (10 req/sec)
// Accepts pointers to allow modifying cached vulnerabilities
func (c *OSVClient) EnrichVulnerabilitiesWithDetails(ctx context.Context, vulns []*Vulnerability) {
	// Create a rate limiter: 10 requests per second
	limiter := rate.NewLimiter(rate.Limit(10), 1)

	logger.Info("Starting detailed vulnerability enrichment for %d vulnerabilities...", len(vulns))

	for i := range vulns {
		// Rate limit before making request
		if err := limiter.Wait(ctx); err != nil {
			logger.Warn("Rate limiter error: %v", err)
			break
		}

		// Fetch individual vulnerability details
		detailVuln, err := c.queryVulnerabilityDetails(ctx, vulns[i].OSVId)
		if err != nil {
			logger.Warn("Failed to fetch details for %s: %v", vulns[i].OSVId, err)
			continue
		}

		// Extract CVSS and severity from detailed response
		cvssScore, severityStr := extractCVSSData(*detailVuln)

		// Only update if we got better data (non-zero CVSS or non-empty severity)
		if cvssScore > 0 || severityStr != "" {
			vulns[i].CVSS = cvssScore
			severity := determineSeverityFromCVSS(cvssScore)
			if severity == "" {
				severity = normalizeSeverity(severityStr)
			}
			if severity != "" {
				vulns[i].Severity = severity
			}
			logger.Info("✓ Enriched %s: CVSS %.1f, Severity: %s", vulns[i].OSVId, cvssScore, vulns[i].Severity)
		}

		// Extract workarounds from detailed response (batch endpoint doesn't include Details)
		if detailVuln.Details != "" && vulns[i].Workarounds == "" {
			vulns[i].Workarounds = extractWorkarounds(detailVuln.Details)
			if vulns[i].Workarounds != "" {
				// Count lines in workarounds
				workaroundLineCount := len(strings.Split(vulns[i].Workarounds, "\n"))
				logger.Info("✓ Extracted workarounds for %s (%d lines)", vulns[i].OSVId, workaroundLineCount)
			} else {
				logger.Info("✗ No workarounds found in Details for %s (Details length: %d)", vulns[i].OSVId, len(detailVuln.Details))
			}
		} else {
			if detailVuln.Details == "" {
				logger.Info("✗ Details field empty for %s", vulns[i].OSVId)
			}
			if vulns[i].Workarounds != "" {
				logger.Info("✓ Workarounds already populated for %s", vulns[i].OSVId)
			}
		}
	}
	logger.Info("Vulnerability enrichment complete")
}

// queryVulnerabilityDetails fetches detailed information for a specific vulnerability
func (c *OSVClient) queryVulnerabilityDetails(ctx context.Context, vulnID string) (*OSVVulnerability, error) {
	url := fmt.Sprintf("%s/%s", OSVVulnDetailEndpoint, vulnID)
	logger.Info("Fetching vulnerability details: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "gemtracker/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vulnerability details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Warn("OSV detail endpoint returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("OSV detail endpoint returned status %d", resp.StatusCode)
	}

	// Read and log raw response for first few vulnerabilities
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var vuln OSVVulnerability
	if err := json.Unmarshal(body, &vuln); err != nil {
		return nil, fmt.Errorf("failed to parse vulnerability details: %w", err)
	}

	// Log what we got from the detail endpoint
	logger.Info("Detail response for %s: DatabaseSpecific=%v, Severity array len=%d", vuln.ID, vuln.DatabaseSpecific, len(vuln.Severity))

	return &vuln, nil
}

// extractWorkarounds extracts remediation guidance sections from OSV details text
// Looks for sections like: Workarounds, Mitigation, Mitigation/Remediation, etc.
// Preserves markdown formatting for rendering with glamour
func extractWorkarounds(details string) string {
	lines := strings.Split(details, "\n")

	// Find the start of any remediation section (Workarounds, Mitigation, etc.)
	var remediationLines []string
	inSection := false
	sectionHeaderIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of remediation section - check if header contains remediation keywords
		if !inSection && isRemediationHeader(trimmed) {
			inSection = true
			sectionHeaderIdx = i
			remediationLines = append(remediationLines, line)
			continue
		}

		// Stop if we hit another major section header (### or ##)
		if inSection && sectionHeaderIdx >= 0 && i > sectionHeaderIdx && isSectionHeader(trimmed) {
			break
		}

		if inSection {
			remediationLines = append(remediationLines, line)
		}
	}

	result := strings.TrimSpace(strings.Join(remediationLines, "\n"))
	return result
}

// isRemediationHeader checks if a line is a Markdown header for a remediation section
// Matches various keywords: Workarounds, Mitigation, Remediation, Solutions, etc.
func isRemediationHeader(line string) bool {
	// Remove Markdown header markers (###, ##, #)
	cleanLine := strings.TrimLeft(line, "#")
	cleanLine = strings.TrimSpace(cleanLine)
	lowerLine := strings.ToLower(cleanLine)

	// Check for common remediation keywords (case-insensitive)
	remediationKeywords := []string{
		"workaround",
		"workarounds",
		"mitigation",
		"remediation",
		"recommendation",
		"recommendations",
		"solution",
		"solutions",
		"fix",
		"fixes",
		"patch",
		"patches",
		"upgrade",
		"mitigation/remediation", // Combined keyword some CVEs use
	}

	for _, keyword := range remediationKeywords {
		if strings.Contains(lowerLine, keyword) {
			return true
		}
	}

	return false
}

// isSectionHeader checks if a line is a Markdown section header
func isSectionHeader(line string) bool {
	if !strings.HasPrefix(line, "#") {
		return false
	}
	// Only major headers (##, ###, ####) mark section boundaries
	return strings.HasPrefix(line, "##")
}
