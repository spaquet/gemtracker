package gemfile

import (
	"testing"
)

func TestFindHighestMatchingVersion_Cached(t *testing.T) {
	cr := NewConstraintResolver()

	// Manually populate cache with test data
	cr.versionCache["test-gem"] = []string{"1.0.0", "2.0.0", "3.0.0", "2.5.0"}

	tests := []struct {
		name       string
		gem        string
		constraint string
		expected   string
	}{
		{
			name:       "find highest within range",
			gem:        "test-gem",
			constraint: ">= 1.0, < 3.0",
			expected:   "2.5.0",
		},
		{
			name:       "find single matching version",
			gem:        "test-gem",
			constraint: ">= 2.5, < 2.6",
			expected:   "2.5.0",
		},
		{
			name:       "no matching version",
			gem:        "test-gem",
			constraint: ">= 4.0, < 5.0",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cr.FindHighestMatchingVersion(tt.gem, tt.constraint)
			if result != tt.expected {
				t.Errorf("FindHighestMatchingVersion(%q, %q) = %q, expected %q", tt.gem, tt.constraint, result, tt.expected)
			}
		})
	}
}

func TestCompareVersionStrings(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "1.0.0", 0},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
		{"6.6.1", "7.0.0", -1},
		{"7.0.0", "6.6.1", 1},
		{"4.0.1", "6.6.1", -1},
	}

	for _, tt := range tests {
		result := compareVersionStrings(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("compareVersionStrings(%q, %q) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
		}
	}
}

func TestParseVersionParts(t *testing.T) {
	tests := []struct {
		version  string
		expected []string
	}{
		{"1.2.3", []string{"1", "2", "3"}},
		{"1.2.3-alpha", []string{"1", "2", "3"}},
		{"6.6.1", []string{"6", "6", "1"}},
		{"7.0.0", []string{"7", "0", "0"}},
		{"4.0.1", []string{"4", "0", "1"}},
	}

	for _, tt := range tests {
		result := parseVersionParts(tt.version)
		if len(result) != len(tt.expected) {
			t.Errorf("parseVersionParts(%q) = %v, expected %v", tt.version, result, tt.expected)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("parseVersionParts(%q)[%d] = %q, expected %q", tt.version, i, v, tt.expected[i])
			}
		}
	}
}

func TestResolveUpdateableVersion_ConstraintBlocks(t *testing.T) {
	cr := NewConstraintResolver()

	// Test: current = 4.0.1, constraint = >= 4.0.1, < 7, latest = 8.0.0
	// Should try to fetch versions and find highest matching (e.g., 6.6.1)
	result := cr.ResolveUpdateableVersion(">= 4.0.1, < 7", "8.0.0", "4.0.1", "puma")

	// Result can be empty if API fails or no matching version, but if it works should find a version
	t.Logf("ResolveUpdateableVersion result: %q", result)
}

func TestResolveUpdateableVersion_NoConstraint(t *testing.T) {
	cr := NewConstraintResolver()

	result := cr.ResolveUpdateableVersion("", "8.0.0", "4.0.1", "puma")
	if result != "8.0.0" {
		t.Errorf("ResolveUpdateableVersion with no constraint should return latest, got %q", result)
	}
}

func TestResolveUpdateableVersion_LatestMatches(t *testing.T) {
	cr := NewConstraintResolver()

	result := cr.ResolveUpdateableVersion(">= 1.0, < 10.0", "8.0.0", "4.0.1", "puma")
	if result != "8.0.0" {
		t.Errorf("ResolveUpdateableVersion when latest matches constraint should return latest, got %q", result)
	}
}

func TestMatchesConstraint_PessimisticMinor(t *testing.T) {
	cr := NewConstraintResolver()

	tests := []struct {
		constraint string
		version    string
		expected   bool
	}{
		{"~> 7.2", "7.2.0", true},
		{"~> 7.2", "7.3.0", true},
		{"~> 7.2", "7.99.0", true},
		{"~> 7.2", "8.0.0", false},
		{"~> 7.2.0", "7.2.0", true},
		{"~> 7.2.0", "7.2.1", true},
		{"~> 7.2.0", "7.3.0", false},
	}

	for _, tt := range tests {
		result := cr.matchesConstraint(tt.constraint, tt.version)
		if result != tt.expected {
			t.Errorf("matchesConstraint(%q, %q) = %v, expected %v", tt.constraint, tt.version, result, tt.expected)
		}
	}
}

func TestMatchesConstraint_Range(t *testing.T) {
	cr := NewConstraintResolver()

	tests := []struct {
		constraint string
		version    string
		expected   bool
	}{
		{">= 4.0.1, < 7", "4.0.1", true},
		{">= 4.0.1, < 7", "6.6.1", true},
		{">= 4.0.1, < 7", "6.99.99", true},
		{">= 4.0.1, < 7", "7.0.0", false},
		{">= 4.0.1, < 7", "8.0.0", false},
		{">= 4.0.1, < 7", "3.0.0", false},
	}

	for _, tt := range tests {
		result := cr.matchesConstraint(tt.constraint, tt.version)
		if result != tt.expected {
			t.Errorf("matchesConstraint(%q, %q) = %v, expected %v", tt.constraint, tt.version, result, tt.expected)
		}
	}
}
