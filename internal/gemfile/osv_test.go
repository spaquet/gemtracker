package gemfile

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewOSVClient(t *testing.T) {
	client := NewOSVClient()

	if client == nil {
		t.Fatal("expected OSVClient, got nil")
	}

	if client.httpClient == nil {
		t.Fatal("expected httpClient to be initialized")
	}
}

func TestOSVClient_QueryBatch_EmptyGems(t *testing.T) {
	client := NewOSVClient()
	ctx := context.Background()

	vulns, err := client.QueryBatch(ctx, []*Gem{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(vulns) != 0 {
		t.Errorf("expected empty vulnerabilities, got %d", len(vulns))
	}
}

func TestOSVClient_QueryBatch_Success(t *testing.T) {
	// Mock OSV.dev response
	mockResponse := `{
		"results": [
			{
				"vulns": [
					{
						"id": "CVE-2021-22942",
						"summary": "SQL injection in Rails",
						"severity": "HIGH",
						"published": "2021-06-01T00:00:00Z",
						"cvss": {
							"score": 7.5
						},
						"references": [
							{
								"type": "WEB",
								"url": "https://rails-security.org"
							}
						],
						"affected": [
							{
								"package": {
									"name": "rails",
									"ecosystem": "RubyGems"
								},
								"ranges": [
									{
										"type": "SEMVER",
										"events": [
											{
												"introduced": "6.0.0",
												"fixed": "6.1.4"
											}
										]
									}
								]
							}
						]
					}
				]
			},
			{
				"vulns": []
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/query/batch" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, mockResponse)
	}))
	defer server.Close()

	// Override endpoint for test
	client := NewOSVClient()
	originalEndpoint := OSVBatchEndpoint
	OSVBatchEndpoint = server.URL + "/v1/query/batch"
	defer func() { OSVBatchEndpoint = originalEndpoint }()

	gems := []*Gem{
		{Name: "rails", Version: "6.0.0"},
		{Name: "devise", Version: "4.8.0"},
	}

	ctx := context.Background()
	vulns, err := client.QueryBatch(ctx, gems)

	if err != nil {
		t.Fatalf("QueryBatch failed: %v", err)
	}

	if len(vulns) != 1 {
		t.Errorf("expected 1 vulnerability, got %d", len(vulns))
	}

	vuln := vulns[0]
	if vuln.GemName != "rails" {
		t.Errorf("expected gem rails, got %s", vuln.GemName)
	}

	if vuln.CVE != "CVE-2021-22942" {
		t.Errorf("expected CVE-2021-22942, got %s", vuln.CVE)
	}

	if vuln.Severity != "HIGH" {
		t.Errorf("expected severity HIGH, got %s", vuln.Severity)
	}

	if vuln.CVSS != 7.5 {
		t.Errorf("expected CVSS 7.5, got %f", vuln.CVSS)
	}

	if vuln.Source != "osv.dev" {
		t.Errorf("expected source osv.dev, got %s", vuln.Source)
	}

	if len(vuln.References) != 1 {
		t.Errorf("expected 1 reference, got %d", len(vuln.References))
	}

	if !strings.Contains(vuln.References[0], "rails-security.org") {
		t.Errorf("expected reference to contain rails-security.org, got %s", vuln.References[0])
	}
}

func TestOSVClient_QueryBatch_NoVulnerabilities(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"vulns": []
			},
			{
				"vulns": []
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, mockResponse)
	}))
	defer server.Close()

	client := NewOSVClient()
	originalEndpoint := OSVBatchEndpoint
	OSVBatchEndpoint = server.URL + "/v1/query/batch"
	defer func() { OSVBatchEndpoint = originalEndpoint }()

	gems := []*Gem{
		{Name: "safe-gem", Version: "1.0.0"},
		{Name: "another-safe", Version: "2.0.0"},
	}

	ctx := context.Background()
	vulns, err := client.QueryBatch(ctx, gems)

	if err != nil {
		t.Fatalf("QueryBatch failed: %v", err)
	}

	if len(vulns) != 0 {
		t.Errorf("expected 0 vulnerabilities, got %d", len(vulns))
	}
}

func TestOSVClient_QueryBatch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "Internal server error")
	}))
	defer server.Close()

	client := NewOSVClient()
	originalEndpoint := OSVBatchEndpoint
	OSVBatchEndpoint = server.URL + "/v1/query/batch"
	defer func() { OSVBatchEndpoint = originalEndpoint }()

	gems := []*Gem{{Name: "rails", Version: "6.0.0"}}
	ctx := context.Background()
	_, err := client.QueryBatch(ctx, gems)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected status 500 error, got %v", err)
	}
}

func TestOSVClient_QueryBatch_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "{invalid json}")
	}))
	defer server.Close()

	client := NewOSVClient()
	originalEndpoint := OSVBatchEndpoint
	OSVBatchEndpoint = server.URL + "/v1/query/batch"
	defer func() { OSVBatchEndpoint = originalEndpoint }()

	gems := []*Gem{{Name: "rails", Version: "6.0.0"}}
	ctx := context.Background()
	_, err := client.QueryBatch(ctx, gems)

	if err == nil {
		t.Fatal("expected error parsing malformed JSON")
	}

	if !strings.Contains(err.Error(), "parse OSV response") {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestNormalizeSeverity_ValidValues(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"CRITICAL", "CRITICAL"},
		{"HIGH", "HIGH"},
		{"MEDIUM", "MEDIUM"},
		{"LOW", "LOW"},
		{"", "MEDIUM"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		got := normalizeSeverity(tt.input)
		if got != tt.expect {
			t.Errorf("normalizeSeverity(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestExtractVersionRanges(t *testing.T) {
	// Test via JSON unmarshaling to properly construct the structs
	tests := []struct {
		name     string
		jsonData string
		expected []string
	}{
		{
			name: "introduced and fixed",
			jsonData: `{
				"affected": [
					{
						"ranges": [
							{
								"type": "SEMVER",
								"events": [
									{"introduced": "6.0.0", "fixed": "6.1.4"}
								]
							}
						]
					}
				]
			}`,
			expected: []string{"6.0.0 < 6.1.4"},
		},
		{
			name: "only introduced",
			jsonData: `{
				"affected": [
					{
						"ranges": [
							{
								"type": "SEMVER",
								"events": [
									{"introduced": "1.0.0"}
								]
							}
						]
					}
				]
			}`,
			expected: []string{">= 1.0.0"},
		},
		{
			name:     "empty affected",
			jsonData: `{}`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		var vuln OSVVulnerability
		err := json.Unmarshal([]byte(tt.jsonData), &vuln)
		if err != nil {
			t.Fatalf("%s: failed to unmarshal: %v", tt.name, err)
		}

		got := extractVersionRanges(&vuln)
		if len(got) != len(tt.expected) {
			t.Errorf("%s: got %d ranges, expected %d", tt.name, len(got), len(tt.expected))
			continue
		}

		for i, v := range got {
			if v != tt.expected[i] {
				t.Errorf("%s: got %q, expected %q", tt.name, v, tt.expected[i])
			}
		}
	}
}

func TestExtractFixedVersion(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected string
	}{
		{
			name: "has fixed version",
			jsonData: `{
				"affected": [
					{
						"ranges": [
							{
								"type": "SEMVER",
								"events": [
									{"introduced": "1.0.0", "fixed": "1.2.3"}
								]
							}
						]
					}
				]
			}`,
			expected: "1.2.3",
		},
		{
			name:     "no fixed version",
			jsonData: `{}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		var vuln OSVVulnerability
		err := json.Unmarshal([]byte(tt.jsonData), &vuln)
		if err != nil {
			t.Fatalf("%s: failed to unmarshal: %v", tt.name, err)
		}

		got := extractFixedVersion(&vuln)
		if got != tt.expected {
			t.Errorf("%s: got %q, expected %q", tt.name, got, tt.expected)
		}
	}
}

func TestOSVClient_QueryBatch_MultipleVulnerabilities(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"vulns": [
					{
						"id": "CVE-2021-22942",
						"summary": "SQL injection in Rails",
						"severity": "HIGH",
						"published": "2021-06-01T00:00:00Z",
						"affected": [
							{
								"ranges": [
									{
										"events": [
											{
												"introduced": "6.0.0",
												"fixed": "6.1.4"
											}
										]
									}
								]
							}
						]
					},
					{
						"id": "CVE-2021-22880",
						"summary": "Another Rails vulnerability",
						"severity": "CRITICAL",
						"published": "2021-05-01T00:00:00Z",
						"affected": [
							{
								"ranges": [
									{
										"events": [
											{
												"introduced": "0.0.1",
												"fixed": "6.1.2"
											}
										]
									}
								]
							}
						]
					}
				]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, mockResponse)
	}))
	defer server.Close()

	client := NewOSVClient()
	originalEndpoint := OSVBatchEndpoint
	OSVBatchEndpoint = server.URL + "/v1/query/batch"
	defer func() { OSVBatchEndpoint = originalEndpoint }()

	gems := []*Gem{{Name: "rails", Version: "6.0.0"}}
	ctx := context.Background()
	vulns, err := client.QueryBatch(ctx, gems)

	if err != nil {
		t.Fatalf("QueryBatch failed: %v", err)
	}

	if len(vulns) != 2 {
		t.Errorf("expected 2 vulnerabilities, got %d", len(vulns))
	}

	if vulns[0].Severity != "HIGH" {
		t.Errorf("expected HIGH severity, got %s", vulns[0].Severity)
	}

	if vulns[1].Severity != "CRITICAL" {
		t.Errorf("expected CRITICAL severity, got %s", vulns[1].Severity)
	}
}

func TestOSVClient_QueryBatch_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		select {
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	client := NewOSVClient()
	originalEndpoint := OSVBatchEndpoint
	OSVBatchEndpoint = server.URL + "/v1/query/batch"
	defer func() { OSVBatchEndpoint = originalEndpoint }()

	gems := []*Gem{{Name: "rails", Version: "6.0.0"}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.QueryBatch(ctx, gems)

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
