package gemfile

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// RubyVersionInfo holds the detected Ruby version and how it was detected.
type RubyVersionInfo struct {
	Version string // The detected Ruby version (e.g., "3.2.0") or "-" if not found
	Source  string // How the version was detected (e.g., "from .ruby-version file")
}

// DetectRubyVersion detects the Ruby version using multiple strategies in priority order:
// 1. Check for .ruby-version or .ruby file in project root
// 2. Run `ruby --version` command in the project directory
// 3. For Rails projects, check Dockerfile for Ruby version
// 4. Fall back to extracting from Gemfile.lock
// 5. Default to "-" if all methods fail
func DetectRubyVersion(projectPath string) *RubyVersionInfo {
	// Try .ruby-version or .ruby file first (most reliable)
	if version, ok := readRubyVersionFile(projectPath); ok {
		return &RubyVersionInfo{
			Version: version,
			Source:  "from .ruby-version file",
		}
	}

	// Try running `ruby --version` in project directory
	if version, ok := getRubyVersionFromCommand(projectPath); ok {
		return &RubyVersionInfo{
			Version: version,
			Source:  "from ruby --version",
		}
	}

	// Try Dockerfile for Rails projects
	if version, ok := readRubyVersionFromDockerfile(projectPath); ok {
		return &RubyVersionInfo{
			Version: version,
			Source:  "from Dockerfile",
		}
	}

	// Fall back to Gemfile.lock
	if version := extractFromGemfileLock(projectPath); version != "Unknown" {
		return &RubyVersionInfo{
			Version: version,
			Source:  "from Gemfile.lock",
		}
	}

	// Default to "-" if nothing found
	return &RubyVersionInfo{
		Version: "-",
		Source:  "(not identified)",
	}
}

// readRubyVersionFile attempts to read Ruby version from .ruby-version or .ruby file.
// Returns version and true if found, empty string and false otherwise.
func readRubyVersionFile(projectPath string) (string, bool) {
	candidates := []string{".ruby-version", ".ruby"}
	for _, filename := range candidates {
		filepath := filepath.Join(projectPath, filename)
		content, err := os.ReadFile(filepath)
		if err == nil {
			version := strings.TrimSpace(string(content))
			if version != "" {
				return version, true
			}
		}
	}
	return "", false
}

// getRubyVersionFromCommand runs `ruby --version` in the project directory
// and extracts the version number.
// Returns version and true if successful, empty string and false otherwise.
func getRubyVersionFromCommand(projectPath string) (string, bool) {
	cmd := exec.Command("ruby", "--version")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return "", false
	}

	// Parse output like "ruby 3.2.0 (2022-12-25 revision abc123) [x86_64-darwin21]"
	versionRegex := regexp.MustCompile(`ruby\s+(\d+\.\d+\.\d+)`)
	matches := versionRegex.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		version := matches[1]
		return version, true
	}

	return "", false
}

// readRubyVersionFromDockerfile attempts to extract Ruby version from Dockerfile.
// Looks for lines like:
//
//	FROM ruby:3.2.0
//	FROM ruby:3.2.0-alpine
//	FROM ruby:3.2
//
// Returns version and true if found, empty string and false otherwise.
func readRubyVersionFromDockerfile(projectPath string) (string, bool) {
	dockerfilePath := filepath.Join(projectPath, "Dockerfile")
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return "", false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	// Match: FROM ruby:X.Y.Z or FROM ruby:X.Y or FROM ruby:latest
	fromRegex := regexp.MustCompile(`(?i)^FROM\s+ruby:(\d+\.\d+(?:\.\d+)?)`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := fromRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := matches[1]
			return version, true
		}
	}

	return "", false
}

// extractFromGemfileLock extracts Ruby version from Gemfile.lock's ruby specification.
// This is the least reliable method since Gemfile.lock may not always have this info.
func extractFromGemfileLock(projectPath string) string {
	lockFile := FindLockFile(projectPath)
	if lockFile == "" {
		return "Unknown"
	}

	file, err := os.Open(lockFile)
	if err != nil {
		return "Unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	rubyVersionRegex := regexp.MustCompile(`(?i)^\s*ruby\s+(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := rubyVersionRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			version := strings.TrimSpace(matches[1])
			version = strings.Trim(version, "\"'")
			return version
		}
	}

	return "Unknown"
}

// String returns a formatted display string for RubyVersionInfo.
// Format: "3.2.0 (from .ruby-version file)" or "- (not identified)"
func (r *RubyVersionInfo) String() string {
	if r.Source != "" && r.Source != "(not identified)" {
		return fmt.Sprintf("%s (%s)", r.Version, r.Source)
	}
	return r.Version
}
