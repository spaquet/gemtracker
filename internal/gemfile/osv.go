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
	resp, err := c.sendBatchRequest(ctx, gems)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Warn("OSV API returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("OSV API returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.Info("OSV API response received (HTTP %d)", resp.StatusCode)
	var batchResp OSVBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		logger.Warn("Failed to parse OSV response: %v", err)
		return nil, fmt.Errorf("failed to parse OSV response: %w", err)
	}
	logger.Info("Parsing OSV response: %d results", len(batchResp.Results))
	logOSVSample(batchResp.Results)
	vulns := c.parseOSVResponse(batchResp, gems)
	c.enrichBatchVulns(ctx, vulns)
	logger.Info("OSV batch query complete: found %d vulnerabilities", len(vulns))
	return vulns, nil
}

func (c *OSVClient) sendBatchRequest(ctx context.Context, gems []*Gem) (*http.Response, error) {
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
	return resp, nil
}

func logOSVSample(results []OSVResult) {
	for i, result := range results {
		if len(result.Vulns) > 0 {
			firstVuln := result.Vulns[0]
			logger.Info("[OSV Response Sample %d] CVE: %s, Has severity array: %v", i, firstVuln.ID, len(firstVuln.Severity) > 0)
			return
		}
	}
}

func (c *OSVClient) enrichBatchVulns(ctx context.Context, vulns []Vulnerability) {
	logger.Info("Enriching %d vulnerabilities with detailed CVSS/Severity data...", len(vulns))
	vulnPtrs := make([]*Vulnerability, len(vulns))
	for i := range vulns {
		vulnPtrs[i] = &vulns[i]
	}
	c.EnrichVulnerabilitiesWithDetails(ctx, vulnPtrs)
}

// parseOSVResponse converts OSV.dev response to our Vulnerability structs
// Only returns vulnerabilities for gems that have them (filters out clean gems)
func (c *OSVClient) parseOSVResponse(resp OSVBatchResponse, gems []*Gem) []Vulnerability {
	var vulnerabilities []Vulnerability

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
			vuln := vulnerabilityFromOSV(gem, &osvVuln)
			logger.Info("✓ CVE %s [%s] (CVSS: %.1f) - %s | Gem: %s@%s", osvVuln.ID, vuln.Severity, vuln.CVSS, osvVuln.Summary, gem.Name, gem.Version)
			vulnerabilities = append(vulnerabilities, vuln)
		}
	}

	return vulnerabilities
}

func vulnerabilityFromOSV(gem *Gem, osvVuln *OSVVulnerability) Vulnerability {
	cvssScore, severityStr := extractCVSSData(*osvVuln)
	severity := determineSeverityFromCVSS(cvssScore)
	if severity == "" {
		severity = normalizeSeverity(severityStr)
	}
	vuln := Vulnerability{
		GemName: gem.Name, CVE: osvVuln.ID, Description: osvVuln.Summary,
		Severity: severity, CVSS: cvssScore, OSVId: osvVuln.ID, Source: "osv.dev",
		AffectedVersions: extractVersionRanges(osvVuln), FixedVersion: extractFixedVersion(osvVuln),
	}
	if osvVuln.Published != "" {
		if t, err := time.Parse(time.RFC3339, osvVuln.Published); err == nil {
			vuln.PublishedDate = t
		}
	}
	for _, ref := range osvVuln.References {
		if ref.URL != "" {
			vuln.References = append(vuln.References, ref.URL)
		}
	}
	if osvVuln.Details != "" {
		vuln.Workarounds = extractWorkarounds(osvVuln.Details)
	}
	return vuln
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
	severity := severityField(osvVuln.DatabaseSpecific)
	if severity != "" {
		logger.Info("CVE %s: Severity from database_specific = %s", osvVuln.ID, severity)
	}

	if severity == "" && len(osvVuln.Affected) > 0 {
		severity = severityField(osvVuln.Affected[0].EcosystemSpecific)
		if severity != "" {
			logger.Info("CVE %s: Severity from affected[0].ecosystem_specific = %s", osvVuln.ID, severity)
		}
	}

	// A vector string is not a numeric score.
	for _, entry := range osvVuln.Severity {
		switch score := entry["score"].(type) {
		case float64:
			logger.Info("CVE %s: CVSS score from severity array = %.1f", osvVuln.ID, score)
			return score, severity
		case string:
			if strings.Contains(score, "CVSS") {
				logger.Info("CVE %s: Found CVSS vector but no numeric score: %s", osvVuln.ID, score)
			}
		}
	}

	logger.Info("CVE %s: Final extracted - CVSS: %.1f, Severity: %s", osvVuln.ID, 0.0, severity)
	return 0, severity
}

func severityField(fields map[string]interface{}) string {
	severity, _ := fields["severity"].(string)
	return severity
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
		applyVulnerabilityDetails(vulns[i], detailVuln)
	}
	logger.Info("Vulnerability enrichment complete")
}

func applyVulnerabilityDetails(vuln *Vulnerability, detailVuln *OSVVulnerability) {
	cvssScore, severityStr := extractCVSSData(*detailVuln)

	if cvssScore > 0 || severityStr != "" {
		vuln.CVSS = cvssScore
		severity := determineSeverityFromCVSS(cvssScore)
		if severity == "" {
			severity = normalizeSeverity(severityStr)
		}
		if severity != "" {
			vuln.Severity = severity
		}
		logger.Info("✓ Enriched %s: CVSS %.1f, Severity: %s", vuln.OSVId, cvssScore, vuln.Severity)
	}

	if detailVuln.Details != "" && vuln.Workarounds == "" {
		vuln.Workarounds = extractWorkarounds(detailVuln.Details)
		if vuln.Workarounds != "" {
			workaroundLineCount := len(strings.Split(vuln.Workarounds, "\n"))
			logger.Info("✓ Extracted workarounds for %s (%d lines)", vuln.OSVId, workaroundLineCount)
		} else {
			logger.Info("✗ No workarounds found in Details for %s (Details length: %d)", vuln.OSVId, len(detailVuln.Details))
		}
	} else {
		if detailVuln.Details == "" {
			logger.Info("✗ Details field empty for %s", vuln.OSVId)
		}
		if vuln.Workarounds != "" {
			logger.Info("✓ Workarounds already populated for %s", vuln.OSVId)
		}
	}
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
