package gemfile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spaquet/gemtracker/internal/logger"
)

const (
	OSVRequestTimeout = 30 * time.Second
)

var (
	OSVBatchEndpoint = "https://api.osv.dev/v1/query/batch"
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
	Severity  string `json:"severity"`
	Cvss      struct {
		Score float64 `json:"score"`
	} `json:"cvss,omitempty"`
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
			vuln := Vulnerability{
				GemName:     gem.Name,
				CVE:         osvVuln.ID,
				Description: osvVuln.Summary,
				Severity:    normalizeSeverity(osvVuln.Severity),
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

			// Set CVSS if available
			if osvVuln.Cvss.Score > 0 {
				vuln.CVSS = osvVuln.Cvss.Score
			}

			logger.Info("Processing CVE %s: %s [%s, CVSS: %.1f]", osvVuln.ID, osvVuln.Summary, vuln.Severity, vuln.CVSS)
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
