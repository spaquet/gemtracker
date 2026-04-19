package gemfile

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ConstraintResolver resolves version constraints and finds compatible versions.
type ConstraintResolver struct {
	versionCache map[string][]string // Cache for fetched versions: gemName -> []version
}

// NewConstraintResolver creates a new ConstraintResolver.
func NewConstraintResolver() *ConstraintResolver {
	return &ConstraintResolver{
		versionCache: make(map[string][]string),
	}
}

// ResolveUpdateableVersion finds the highest version that matches the constraint.
// If no constraint is provided, returns the latest version.
// If the latest version matches the constraint, returns it; otherwise queries rubygems for the highest matching version.
func (cr *ConstraintResolver) ResolveUpdateableVersion(constraint, latestVersion, currentVersion, gemName string) string {
	if constraint == "" {
		return latestVersion
	}

	// Parse the constraint and check if latestVersion matches it
	if cr.matchesConstraint(constraint, latestVersion) {
		return latestVersion
	}

	// If latest doesn't match constraint, find highest version that does match
	return cr.FindHighestMatchingVersion(gemName, constraint)
}

// matchesConstraint checks if a version matches a constraint.
// Supports: ~> (pessimistic), >= (greater than or equal), > (greater than),
// <= (less than or equal), < (less than), = (equal), no operator (equal)
func (cr *ConstraintResolver) matchesConstraint(constraint, version string) bool {
	constraint = strings.TrimSpace(constraint)
	version = cr.normalizeVersion(version)

	// Handle multiple constraints separated by commas (e.g., ">= 1.0, < 2.0")
	if strings.Contains(constraint, ",") {
		parts := strings.Split(constraint, ",")
		for _, part := range parts {
			if !cr.matchesSingleConstraint(strings.TrimSpace(part), version) {
				return false
			}
		}
		return true
	}

	return cr.matchesSingleConstraint(constraint, version)
}

// matchesSingleConstraint checks if a version matches a single constraint expression.
func (cr *ConstraintResolver) matchesSingleConstraint(constraint, version string) bool {
	constraint = strings.TrimSpace(constraint)

	// Pessimistic version constraint: ~> X.Y allows X.Y.z but not X.(Y+1)
	if strings.HasPrefix(constraint, "~>") {
		return cr.matchesPessimisticVersion(constraint[2:], version)
	}

	// Comparison operators
	if strings.HasPrefix(constraint, ">=") {
		return cr.compareVersions(version, constraint[2:]) >= 0
	}
	if strings.HasPrefix(constraint, "<=") {
		return cr.compareVersions(version, constraint[2:]) <= 0
	}
	if strings.HasPrefix(constraint, ">") {
		return cr.compareVersions(version, constraint[1:]) > 0
	}
	if strings.HasPrefix(constraint, "<") {
		return cr.compareVersions(version, constraint[1:]) < 0
	}
	if strings.HasPrefix(constraint, "=") {
		return cr.compareVersions(version, constraint[1:]) == 0
	}

	// No operator means exact match
	return cr.compareVersions(version, constraint) == 0
}

// matchesPessimisticVersion handles ~> constraints.
// ~> X.Y.z allows X.Y.z and X.Y.(z+1) but not X.(Y+1)
// ~> X.Y allows X.Y and X.(Y+1) but not (X+1)
func (cr *ConstraintResolver) matchesPessimisticVersion(constraintStr, version string) bool {
	constraintStr = strings.TrimSpace(constraintStr)
	constraintParts := cr.ParseVersion(constraintStr)
	versionParts := cr.ParseVersion(version)

	if len(constraintParts) == 0 || len(versionParts) == 0 {
		return false
	}

	// For ~> X.Y.Z, allow X.Y.* but not X.(Y+1)
	if len(constraintParts) >= 3 {
		// Major must match exactly
		if versionParts[0] != constraintParts[0] {
			return false
		}
		// Minor must match exactly
		if len(versionParts) < 2 || len(constraintParts) < 2 || versionParts[1] != constraintParts[1] {
			return false
		}
		// Patch can be >= constraint's patch
		if len(versionParts) >= 3 && len(constraintParts) >= 3 {
			return versionParts[2] >= constraintParts[2]
		}
		return true
	}

	// For ~> X.Y, allow X.* but not (X+1)
	if len(constraintParts) >= 2 {
		// Major must match exactly
		if versionParts[0] != constraintParts[0] {
			return false
		}
		// Minor can be >= constraint's minor
		if len(versionParts) >= 2 {
			return versionParts[1] >= constraintParts[1]
		}
		return true
	}

	// Single version number constraint
	return versionParts[0] >= constraintParts[0]
}

// ParseVersion parses a semantic version string into integer parts.
// Handles pre-release versions by ignoring the pre-release suffix.
// E.g., "1.2.3" -> [1, 2, 3], "1.2.3-alpha.1" -> [1, 2, 3]
func (cr *ConstraintResolver) ParseVersion(versionStr string) []int {
	versionStr = strings.TrimSpace(versionStr)
	versionStr = strings.Trim(versionStr, "\"'")

	// Remove pre-release and build metadata
	if idx := strings.IndexAny(versionStr, "-+"); idx != -1 {
		versionStr = versionStr[:idx]
	}

	parts := strings.Split(versionStr, ".")
	var result []int

	for _, part := range parts {
		// Convert each part to int, stop on first non-numeric part
		if num, err := strconv.Atoi(part); err == nil {
			result = append(result, num)
		} else {
			break
		}
	}

	return result
}

// compareVersions compares two semantic version strings.
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func (cr *ConstraintResolver) compareVersions(v1, v2 string) int {
	parts1 := cr.ParseVersion(v1)
	parts2 := cr.ParseVersion(v2)

	// Pad shorter version with zeros
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		num1 := 0
		if i < len(parts1) {
			num1 = parts1[i]
		}

		num2 := 0
		if i < len(parts2) {
			num2 = parts2[i]
		}

		if num1 < num2 {
			return -1
		}
		if num1 > num2 {
			return 1
		}
	}

	return 0
}

// normalizeVersion removes platform suffixes from version strings.
// E.g., "1.2.3-x86_64-linux" -> "1.2.3"
func (cr *ConstraintResolver) normalizeVersion(version string) string {
	// Remove platform suffix (e.g., "-x86_64-linux", "-arm64-darwin")
	platformSuffixes := []string{"-x86_64-linux", "-aarch64-linux", "-x86_64-linux-musl", "-arm64-darwin", "-x86_64-darwin"}
	for _, suffix := range platformSuffixes {
		if strings.HasSuffix(version, suffix) {
			return strings.TrimSuffix(version, suffix)
		}
	}
	return version
}

// FindHighestMatchingVersion queries rubygems.org for available versions and returns the highest that matches the constraint.
// Returns empty string if no matching version is found or if the API fails.
func (cr *ConstraintResolver) FindHighestMatchingVersion(gemName, constraint string) string {
	// Check cache first
	versions, cached := cr.versionCache[gemName]
	if !cached {
		// Not in cache, fetch from API
		var err error
		versions, err = cr.fetchAvailableVersions(gemName)
		if err != nil || len(versions) == 0 {
			return ""
		}
		// Cache the result
		cr.versionCache[gemName] = versions
	}

	// Sort versions in descending order to find highest match first
	sortedVersions := sortVersionStringsDescending(versions)

	for _, v := range sortedVersions {
		if cr.matchesConstraint(constraint, v) {
			return v
		}
	}

	return ""
}

// fetchAvailableVersions queries rubygems.org API for all available versions of a gem.
// Returns a slice of version strings or an error.
func (cr *ConstraintResolver) fetchAvailableVersions(gemName string) ([]string, error) {
	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", gemName)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rubygems API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var gem struct {
		Versions []string `json:"versions"`
	}

	if err := json.Unmarshal(body, &gem); err != nil {
		return nil, err
	}

	return gem.Versions, nil
}

// sortVersionStringsDescending sorts versions in descending order (highest first).
func sortVersionStringsDescending(versions []string) []string {
	result := make([]string, len(versions))
	copy(result, versions)

	// Simple bubble sort for version comparison
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if compareVersionStrings(result[i], result[j]) < 0 {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// compareVersionStrings compares two version strings.
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if v1 == v2
func compareVersionStrings(v1, v2 string) int {
	parts1 := parseVersionParts(v1)
	parts2 := parseVersionParts(v2)

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		n1, _ := strconv.Atoi(parts1[i])
		n2, _ := strconv.Atoi(parts2[i])

		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	if len(parts1) > len(parts2) {
		return 1
	}
	if len(parts1) < len(parts2) {
		return -1
	}

	return 0
}

// parseVersionParts extracts numeric parts from a version string.
// E.g., "1.2.3" -> ["1", "2", "3"]
func parseVersionParts(version string) []string {
	var parts []string
	var current string

	for _, ch := range version {
		if (ch >= '0' && ch <= '9') || ch == '.' {
			current += string(ch)
		} else {
			if current != "" && current != "." {
				parts = append(parts, strings.Split(strings.Trim(current, "."), ".")...)
			}
			current = ""
		}
	}

	if current != "" && current != "." {
		parts = append(parts, strings.Split(strings.Trim(current, "."), ".")...)
	}

	// Filter out empty strings
	var filtered []string
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}

	return filtered
}
