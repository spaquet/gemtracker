package gemfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spaquet/gemtracker/internal/logger"
)

type Gem struct {
	Name         string
	Version      string
	Dependencies []string
	Groups       []string // e.g., "default", "development", "test", "production"
	IsFirstLevel bool     // true if this gem is in DEPENDENCIES section (directly required)
}

type Gemfile struct {
	Path           string
	Gems           map[string]*Gem
	FirstLevelGems []string // Names of gems listed in DEPENDENCIES section
}

// FindLockFile searches for a lock file in the given directory.
// It probes in priority order: gems.locked, Gemfile.lock
// Returns the filename if found, empty string otherwise.
func FindLockFile(dir string) string {
	candidates := []string{"gems.locked", "Gemfile.lock"}
	for _, filename := range candidates {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// FindGemfile searches for a Gemfile in the given directory.
// It probes in priority order: gems.rb, Gemfile
// Returns the filename if found, empty string otherwise.
func FindGemfile(dir string) string {
	candidates := []string{"gems.rb", "Gemfile"}
	for _, filename := range candidates {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func Parse(path string) (*Gemfile, error) {
	// Expand ~ if needed
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if it's a directory, if so look for Gemfile.lock
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	if info.IsDir() {
		lockFile := FindLockFile(path)
		if lockFile == "" {
			logger.Warn("No lock file found (gems.locked or Gemfile.lock) in %s", path)
			return nil, fmt.Errorf("no lock file found (gems.locked or Gemfile.lock) in %s", path)
		}
		path = lockFile
		logger.Info("Using lock file: %s", filepath.Base(lockFile))
	}

	// Read the file
	file, err := os.Open(path)
	if err != nil {
		logger.Error("Failed to open lock file %s: %v", path, err)
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	defer file.Close()

	gf := &Gemfile{
		Path:           path,
		Gems:           make(map[string]*Gem),
		FirstLevelGems: []string{},
	}

	scanner := bufio.NewScanner(file)
	inSection := "" // track current section: "GIT", "GEM", "DEPENDENCIES", etc.

	gemLineRegex := regexp.MustCompile(`(?i)^\s{4}([a-z0-9_-]+)\s+\(([^)]+)\)`)
	dependencyRegex := regexp.MustCompile(`(?i)^\s{6}([a-z0-9_-]+)`)
	dependencyItemRegex := regexp.MustCompile(`(?i)^\s{2}([a-z0-9_-]+)`)

	var currentGem *Gem

	for scanner.Scan() {
		line := scanner.Text()

		// Check for section headers
		if strings.HasPrefix(line, "GIT") {
			inSection = "GIT"
			continue
		} else if strings.HasPrefix(line, "GEM") {
			inSection = "GEM"
			continue
		} else if strings.HasPrefix(line, "PLATFORMS") {
			inSection = "PLATFORMS"
			continue
		} else if strings.HasPrefix(line, "DEPENDENCIES") {
			inSection = "DEPENDENCIES"
			continue
		} else if strings.HasPrefix(line, "BUNDLED WITH") {
			inSection = "BUNDLED"
			break
		}

		// Skip empty lines and section metadata
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "  remote:") || strings.HasPrefix(line, "  specs:") || strings.HasPrefix(line, "  revision:") || strings.HasPrefix(line, "  branch:") || strings.HasPrefix(line, "  tag:") {
			continue
		}

		// Skip PLATFORMS section content - just skip lines that are indented (platform names)
		if inSection == "PLATFORMS" && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, " ")) {
			continue
		}

		// Parse GIT and GEM sections
		if inSection == "GIT" || inSection == "GEM" {
			// Parse gem lines (4-space indent)
			matches := gemLineRegex.FindStringSubmatch(line)
			if len(matches) > 0 {
				name := strings.ToLower(matches[1])
				version := matches[2]

				currentGem = &Gem{
					Name:         name,
					Version:      version,
					Dependencies: []string{},
					Groups:       []string{},
					IsFirstLevel: false,
				}
				gf.Gems[name] = currentGem
				continue
			}

			// Parse dependency lines (6-space indent)
			if currentGem != nil {
				depMatches := dependencyRegex.FindStringSubmatch(line)
				if len(depMatches) > 0 {
					depName := strings.ToLower(depMatches[1])
					currentGem.Dependencies = append(currentGem.Dependencies, depName)
				}
			}
		}

		// Parse DEPENDENCIES section (2-space indent for gem names)
		if inSection == "DEPENDENCIES" {
			matches := dependencyItemRegex.FindStringSubmatch(line)
			if len(matches) > 0 {
				gemName := strings.ToLower(matches[1])
				// Remove trailing '!' if it's a git dependency in DEPENDENCIES
				gemName = strings.TrimSuffix(gemName, "!")

				gf.FirstLevelGems = append(gf.FirstLevelGems, gemName)

				// Mark the gem as first-level if it exists
				if gem, ok := gf.Gems[gemName]; ok {
					gem.IsFirstLevel = true
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading lock file: %v", err)
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	logger.Info("Successfully parsed lock file: %d total gems, %d first-level", len(gf.Gems), len(gf.FirstLevelGems))
	return gf, nil
}

func (g *Gemfile) GetGemCount() int {
	return len(g.Gems)
}

func (g *Gemfile) GetGemsAsList() []*Gem {
	gems := make([]*Gem, 0, len(g.Gems))
	for _, gem := range g.Gems {
		gems = append(gems, gem)
	}
	return gems
}

// LoadGroupsFromGemfile parses the Gemfile to extract group information
// It updates the gems map with group information
func (g *Gemfile) LoadGroupsFromGemfile(gemfilePath string) error {
	// Expand ~ if needed
	if strings.HasPrefix(gemfilePath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		gemfilePath = filepath.Join(home, gemfilePath[1:])
	}

	// Check if it's a directory, if so look for Gemfile
	info, err := os.Stat(gemfilePath)
	if err != nil {
		// Gemfile might not exist, which is okay - just return
		return nil
	}

	if info.IsDir() {
		gemfilePath = FindGemfile(gemfilePath)
		if gemfilePath == "" {
			// No Gemfile found, which is okay - just return
			return nil
		}
	}

	// Try to read the Gemfile
	file, err := os.Open(gemfilePath)
	if err != nil {
		// Gemfile doesn't exist, which is okay
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	gemRegex := regexp.MustCompile(`^\s*gem\s+["']([a-z0-9_-]+)["']`)
	groupRegex := regexp.MustCompile(`^\s*group\s+:([a-z_]+)\s+do`)
	groupEndRegex := regexp.MustCompile(`^\s*end\s*$`)

	currentGroups := []string{"default"} // Gems outside groups are in "default"
	inGroup := false
	groupStack := []string{}

	for scanner.Scan() {
		line := scanner.Text()

		// Check for group start
		matches := groupRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			groupName := matches[1]
			groupStack = append(groupStack, groupName)
			currentGroups = []string{groupName}
			inGroup = true
			continue
		}

		// Check for group end
		if inGroup && groupEndRegex.MatchString(line) {
			if len(groupStack) > 0 {
				groupStack = groupStack[:len(groupStack)-1]
			}
			if len(groupStack) == 0 {
				currentGroups = []string{"default"}
				inGroup = false
			}
			continue
		}

		// Check for gem declaration
		gemMatches := gemRegex.FindStringSubmatch(line)
		if len(gemMatches) > 0 {
			gemName := gemMatches[1]
			if gem, ok := g.Gems[gemName]; ok {
				// Add groups to this gem (avoid duplicates)
				for _, group := range currentGroups {
					found := false
					for _, existingGroup := range gem.Groups {
						if existingGroup == group {
							found = true
							break
						}
					}
					if !found {
						gem.Groups = append(gem.Groups, group)
					}
				}
			}
		}
	}

	return nil
}

// ExtractRubyVersion extracts the Ruby version from Gemfile.lock
func ExtractRubyVersion(path string) string {
	// Expand ~ if needed
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "Unknown"
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if it's a directory, if so look for Gemfile.lock
	info, err := os.Stat(path)
	if err != nil {
		return "Unknown"
	}

	if info.IsDir() {
		lockFile := FindLockFile(path)
		if lockFile == "" {
			return "Unknown"
		}
		path = lockFile
	}

	file, err := os.Open(path)
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
			// Remove quotes if present
			version = strings.Trim(version, "\"'")
			return version
		}
	}

	return "Unknown"
}

// ExtractBundleVersion extracts the Bundle version from Gemfile.lock
func ExtractBundleVersion(path string) string {
	// Expand ~ if needed
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "Unknown"
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if it's a directory, if so look for Gemfile.lock
	info, err := os.Stat(path)
	if err != nil {
		return "Unknown"
	}

	if info.IsDir() {
		lockFile := FindLockFile(path)
		if lockFile == "" {
			return "Unknown"
		}
		path = lockFile
	}

	file, err := os.Open(path)
	if err != nil {
		return "Unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	prevLine := ""

	for scanner.Scan() {
		line := scanner.Text()

		// Check if previous line was "BUNDLED WITH"
		if strings.Contains(strings.ToUpper(prevLine), "BUNDLED WITH") {
			version := strings.TrimSpace(line)
			return version
		}

		prevLine = line
	}

	return "Unknown"
}

// DetectFramework detects the primary framework (Rails, Sinatra, etc.) from installed gems
func DetectFramework(gf *Gemfile) (framework string, version string) {
	frameworkNames := []string{"rails", "sinatra", "hanami", "roda", "cuba", "grape"}

	for _, name := range frameworkNames {
		if gem, ok := gf.Gems[name]; ok {
			return name, gem.Version
		}
	}

	return "", ""
}
