package gemfile

import (
	"testing"
)

func TestNewVulnerabilityChecker(t *testing.T) {
	vc := NewVulnerabilityChecker()

	if vc == nil {
		t.Fatal("expected VulnerabilityChecker, got nil")
	}

	if len(vc.vulnerabilities) == 0 {
		t.Error("expected vulnerabilities to be initialized")
	}
}

func TestHasVulnerability_RailsVuln(t *testing.T) {
	vc := NewVulnerabilityChecker()

	// Rails 6.0.0 should be vulnerable (< 6.1.4)
	hasVuln, cveID, desc := vc.HasVulnerability("rails", "6.0.0")

	if !hasVuln {
		t.Error("expected rails 6.0.0 to be vulnerable")
	}

	if cveID == "" {
		t.Error("expected CVE ID to be set")
	}

	if desc == "" {
		t.Error("expected description to be set")
	}
}

func TestHasVulnerability_RailsNotVuln(t *testing.T) {
	vc := NewVulnerabilityChecker()

	// Rails 7.0.0 should not be vulnerable (>= 6.1.4 and >= 7.0.0)
	hasVuln, _, _ := vc.HasVulnerability("rails", "7.0.0")

	if hasVuln {
		t.Error("expected rails 7.0.0 to not be vulnerable")
	}
}

func TestHasVulnerability_DeviseVuln(t *testing.T) {
	vc := NewVulnerabilityChecker()

	// Devise 4.7.0 should be vulnerable (< 4.8.0)
	hasVuln, cveID, _ := vc.HasVulnerability("devise", "4.7.0")

	if !hasVuln {
		t.Error("expected devise 4.7.0 to be vulnerable")
	}

	if cveID != "CVE-2021-41113" {
		t.Errorf("expected CVE-2021-41113, got %q", cveID)
	}
}

func TestHasVulnerability_UnknownGem(t *testing.T) {
	vc := NewVulnerabilityChecker()

	hasVuln, cveID, desc := vc.HasVulnerability("unknown-gem", "1.0.0")

	if hasVuln {
		t.Error("expected unknown gem to not be vulnerable")
	}

	if cveID != "" {
		t.Errorf("expected empty CVE ID, got %q", cveID)
	}

	if desc != "" {
		t.Errorf("expected empty description, got %q", desc)
	}
}

func TestHasVulnerability_CaseInsensitive(t *testing.T) {
	vc := NewVulnerabilityChecker()

	// Gem names should be case insensitive
	hasVuln, _, _ := vc.HasVulnerability("RAILS", "6.0.0")

	if !hasVuln {
		t.Error("expected uppercase RAILS to match rails vulnerability")
	}
}

func TestMatchesSpec_LessThan(t *testing.T) {
	vc := NewVulnerabilityChecker()

	tests := []struct {
		version string
		spec    string
		want    bool
	}{
		{"6.0.0", "< 6.1.4", true},
		{"6.1.4", "< 6.1.4", false},
		{"6.2.0", "< 6.1.4", false},
	}

	for _, tt := range tests {
		got := vc.matchesSpec(tt.version, tt.spec)
		if got != tt.want {
			t.Errorf("matchesSpec(%q, %q) = %v, want %v", tt.version, tt.spec, got, tt.want)
		}
	}
}

func TestMatchesSpec_GreaterThanOrEqual(t *testing.T) {
	vc := NewVulnerabilityChecker()

	tests := []struct {
		version string
		spec    string
		want    bool
	}{
		{"6.0.1", ">= 6.0.0", true},
		{"6.1.0", ">= 6.0.0", true},
		{"5.9.9", ">= 6.0.0", false},
	}

	for _, tt := range tests {
		got := vc.matchesSpec(tt.version, tt.spec)
		if got != tt.want {
			t.Errorf("matchesSpec(%q, %q) = %v, want %v", tt.version, tt.spec, got, tt.want)
		}
	}
}

func TestMatchesSpec_GreaterThan(t *testing.T) {
	vc := NewVulnerabilityChecker()

	tests := []struct {
		version string
		spec    string
		want    bool
	}{
		{"6.0.1", "> 6.0.0", true},
		{"6.0.0", "> 6.0.0", false},
		{"5.9.9", "> 6.0.0", false},
	}

	for _, tt := range tests {
		got := vc.matchesSpec(tt.version, tt.spec)
		if got != tt.want {
			t.Errorf("matchesSpec(%q, %q) = %v, want %v", tt.version, tt.spec, got, tt.want)
		}
	}
}

func TestMatchesSpec_LessThanOrEqual(t *testing.T) {
	vc := NewVulnerabilityChecker()

	tests := []struct {
		version string
		spec    string
		want    bool
	}{
		{"6.1.4", "<= 6.1.4", true},
		{"6.1.3", "<= 6.1.4", true},
		{"6.1.5", "<= 6.1.4", false},
	}

	for _, tt := range tests {
		got := vc.matchesSpec(tt.version, tt.spec)
		if got != tt.want {
			t.Errorf("matchesSpec(%q, %q) = %v, want %v", tt.version, tt.spec, got, tt.want)
		}
	}
}

func TestMatchesSpec_ExactMatch(t *testing.T) {
	vc := NewVulnerabilityChecker()

	tests := []struct {
		version string
		spec    string
		want    bool
	}{
		{"6.1.4", "6.1.4", true},
		{"6.1.3", "6.1.4", false},
	}

	for _, tt := range tests {
		got := vc.matchesSpec(tt.version, tt.spec)
		if got != tt.want {
			t.Errorf("matchesSpec(%q, %q) = %v, want %v", tt.version, tt.spec, got, tt.want)
		}
	}
}

func TestVersionIsAffected_MultipleSpecs(t *testing.T) {
	vc := NewVulnerabilityChecker()

	// Rack has multiple affected version ranges
	affectedSpecs := []string{"< 2.1.4", ">= 2.2.0, < 2.2.3"}

	// 2.1.3 should be affected (< 2.1.4)
	if !vc.versionIsAffected("2.1.3", affectedSpecs) {
		t.Error("expected 2.1.3 to be affected")
	}

	// 2.2.1 should be affected (>= 2.2.0, < 2.2.3)
	// Note: this checks if any spec matches
	affected := vc.versionIsAffected("2.2.1", affectedSpecs)
	// The current implementation checks each spec individually, so this will match the second spec
	if !affected {
		t.Error("expected 2.2.1 to be affected")
	}

	// 2.1.4 should not be affected
	if vc.versionIsAffected("2.1.4", affectedSpecs) {
		t.Error("expected 2.1.4 to not be affected")
	}

	// 2.2.0 is the boundary - ">= 2.2.0, < 2.2.3" should match it
	affected = vc.versionIsAffected("2.2.0", affectedSpecs)
	if !affected {
		t.Error("expected 2.2.0 to be affected (>= 2.2.0)")
	}
}

func TestHasVulnerability_RackVuln(t *testing.T) {
	vc := NewVulnerabilityChecker()

	// Rack 2.2.2 should be vulnerable
	hasVuln, cveID, _ := vc.HasVulnerability("rack", "2.2.2")

	if !hasVuln {
		t.Error("expected rack 2.2.2 to be vulnerable")
	}

	if cveID != "CVE-2022-24834" {
		t.Errorf("expected CVE-2022-24834, got %q", cveID)
	}
}

func TestHasVulnerability_ActionpackVuln(t *testing.T) {
	vc := NewVulnerabilityChecker()

	// Actionpack 6.1.4 should not be vulnerable (< 6.1.5)
	hasVuln, _, _ := vc.HasVulnerability("actionpack", "6.1.4")

	if !hasVuln {
		t.Error("expected actionpack 6.1.4 to be vulnerable")
	}

	// Actionpack 6.1.5 should not be vulnerable
	hasVuln, _, _ = vc.HasVulnerability("actionpack", "6.1.5")

	if hasVuln {
		t.Error("expected actionpack 6.1.5 to not be vulnerable")
	}
}
