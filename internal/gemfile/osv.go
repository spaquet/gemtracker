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
)

const (
	OSVRequestTimeout = 30 * time.Second
)

var (
	OSVBatchEndpoint = "https://api.osv.dev/v1/querybatch"
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
	ID        string `json:"id"`
	Summary   string `json:"summary"`
	Details   string `json:"details"`
	Published string `json:"published"`
	Modified  string `json:"modified"`
	Severity  interface{} `json:"severity"` // Can be string or object with nested fields
	Cvss      interface{} `json:"cvss"`     // Can be object or have multiple formats
	References []struct {
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
				logger.Info("[OSV Response Sample %d] CVE: %s, Severity field: %v, CVSS field: %v", i, firstVuln.ID, firstVuln.Severity, firstVuln.Cvss)
				break
			}
		}
	}

	// Convert OSV vulnerabilities to our format, filtering clean gems
	vulns := c.parseOSVResponse(batchResp, gems)
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
	case "CRITICAL", "HIGH", "MEDIUM", "LOW":
		return severity
	default:
		if severity == "" {
			return "MEDIUM" // Default if not specified
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
// According to CVSS v3.1: None (0.0), Low (0.1-3.9), Medium (4.0-6.9), High (7.0-8.9), Critical (9.0-10.0)
func determineSeverityFromCVSS(cvssScore float64) string {
	switch {
	case cvssScore >= 9.0:
		return "CRITICAL"
	case cvssScore >= 7.0:
		return "HIGH"
	case cvssScore >= 4.0:
		return "MEDIUM"
	case cvssScore > 0:
		return "LOW"
	}
	return ""
}

// extractCVSSData extracts CVSS score and severity from OSV vulnerability response
// Handles multiple possible response formats from OSV.dev
func extractCVSSData(osvVuln OSVVulnerability) (float64, string) {
	cvssScore := 0.0
	severity := ""

	// Try to extract severity string
	if osvVuln.Severity != nil {
		switch v := osvVuln.Severity.(type) {
		case string:
			severity = v
			logger.Info("CVE %s: Severity string = %s", osvVuln.ID, v)
		case map[string]interface{}:
			// Could be {value: "HIGH"} format
			if val, ok := v["value"]; ok {
				if str, ok := val.(string); ok {
					severity = str
					logger.Info("CVE %s: Severity from object = %s", osvVuln.ID, str)
				}
			}
		}
	}

	// Try to extract CVSS score from cvss field
	if osvVuln.Cvss != nil {
		switch cvss := osvVuln.Cvss.(type) {
		case map[string]interface{}:
			// Try v3.score first (most common)
			if v3, ok := cvss["v3"]; ok {
				if v3Map, ok := v3.(map[string]interface{}); ok {
					if score, ok := v3Map["score"]; ok {
						if floatScore, ok := score.(float64); ok {
							cvssScore = floatScore
							logger.Info("CVE %s: CVSS v3 score = %.1f", osvVuln.ID, floatScore)
							return cvssScore, severity
						}
					}
				}
			}

			// Try generic score field
			if score, ok := cvss["score"]; ok {
				if floatScore, ok := score.(float64); ok {
					cvssScore = floatScore
					logger.Info("CVE %s: CVSS score = %.1f", osvVuln.ID, floatScore)
					return cvssScore, severity
				}
			}

			// Try v2 score as fallback
			if v2, ok := cvss["v2"]; ok {
				if v2Map, ok := v2.(map[string]interface{}); ok {
					if score, ok := v2Map["score"]; ok {
						if floatScore, ok := score.(float64); ok {
							cvssScore = floatScore
							logger.Info("CVE %s: CVSS v2 score = %.1f", osvVuln.ID, floatScore)
							return cvssScore, severity
						}
					}
				}
			}

			// Log if no score was found
			logger.Info("CVE %s: No CVSS score found in response. Available fields: %v", osvVuln.ID, cvss)
		}
	}

	logger.Info("CVE %s: Final extracted - CVSS: %.1f, Severity: %s", osvVuln.ID, cvssScore, severity)
	return cvssScore, severity
}

// extractWorkarounds extracts the "Workarounds" section from OSV details text
func extractWorkarounds(details string) string {
	lines := strings.Split(details, "\n")

	// Find the start of the Workarounds section
	var workaroundLines []string
	inWorkarounds := false

	for _, line := range lines {
		// Start of workarounds section
		if strings.EqualFold(strings.TrimSpace(line), "Workarounds") {
			inWorkarounds = true
			continue
		}

		// Stop if we hit another major section (indicated by a line with just capital letters followed by newline)
		if inWorkarounds && strings.TrimSpace(line) != "" {
			trimmed := strings.TrimSpace(line)
			// Check if it looks like a new section header (all caps, 3-15 characters)
			if len(trimmed) > 3 && len(trimmed) < 15 && strings.ToUpper(trimmed) == trimmed {
				// But continue if it looks like part of content (has punctuation, numbers, or mixed case)
				if !strings.ContainsAny(trimmed, ".,:-()0123456789") && !strings.ContainsAny(trimmed, "abcdefghijklmnopqrstuvwxyz") {
					break
				}
			}
		}

		if inWorkarounds {
			workaroundLines = append(workaroundLines, line)
		}
	}

	result := strings.TrimSpace(strings.Join(workaroundLines, "\n"))
	return result
}
