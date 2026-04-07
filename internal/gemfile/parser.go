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

// detectSection checks if a line starts a new section and returns the section name and whether to skip to next line
func detectSection(line string) (string, bool) {
	switch {
	case strings.HasPrefix(line, "GIT"):
		return "GIT", true
	case strings.HasPrefix(line, "GEM"):
		return "GEM", true
	case strings.HasPrefix(line, "PLATFORMS"):
		return "PLATFORMS", true
	case strings.HasPrefix(line, "DEPENDENCIES"):
		return "DEPENDENCIES", true
	case strings.HasPrefix(line, "BUNDLED WITH"):
		return "BUNDLED", true
	}
	return "", false
}

// shouldSkipLine checks if a line should be skipped during parsing
func shouldSkipLine(line string, inSection string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	// Skip metadata lines
	if strings.HasPrefix(line, "  remote:") || strings.HasPrefix(line, "  specs:") ||
		strings.HasPrefix(line, "  revision:") || strings.HasPrefix(line, "  branch:") ||
		strings.HasPrefix(line, "  tag:") {
		return true
	}
	// Skip PLATFORMS section content
	if inSection == "PLATFORMS" && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, " ")) {
		return true
	}
	return false
}

// parseGemOrGitLine handles parsing gem lines in GIT/GEM sections
func parseGemOrGitLine(line string, gf *Gemfile, currentGem *Gem, gemLineRegex, depRegex *regexp.Regexp) *Gem {
	// Parse gem lines (4-space indent)
	matches := gemLineRegex.FindStringSubmatch(line)
	if len(matches) > 0 {
		name := strings.ToLower(matches[1])
		version := matches[2]
		gem := &Gem{
			Name:         name,
			Version:      version,
			Dependencies: []string{},
			Groups:       []string{},
			IsFirstLevel: false,
		}
		gf.Gems[name] = gem
		return gem
	}

	// Parse dependency lines (6-space indent)
	if currentGem != nil {
		depMatches := depRegex.FindStringSubmatch(line)
		if len(depMatches) > 0 {
			depName := strings.ToLower(depMatches[1])
			currentGem.Dependencies = append(currentGem.Dependencies, depName)
		}
	}
	return currentGem
}

// parseDependenciesLine handles parsing gem names in DEPENDENCIES section
func parseDependenciesLine(line string, gf *Gemfile, depItemRegex *regexp.Regexp) {
	matches := depItemRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return
	}

	gemName := strings.ToLower(matches[1])
	gemName = strings.TrimSuffix(gemName, "!")
	gf.FirstLevelGems = append(gf.FirstLevelGems, gemName)

	if gem, ok := gf.Gems[gemName]; ok {
		gem.IsFirstLevel = true
	}
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
	inSection := ""

	gemLineRegex := regexp.MustCompile(`(?i)^\s{4}([a-z0-9_-]+)\s+\(([^)]+)\)`)
	dependencyRegex := regexp.MustCompile(`(?i)^\s{6}([a-z0-9_-]+)`)
	dependencyItemRegex := regexp.MustCompile(`(?i)^\s{2}([a-z0-9_-]+)`)

	var currentGem *Gem

	for scanner.Scan() {
		line := scanner.Text()

		// Check for section headers
		if newSection, isSectionHeader := detectSection(line); isSectionHeader {
			if newSection == "BUNDLED" {
				break
			}
			inSection = newSection
			continue
		}

		if shouldSkipLine(line, inSection) {
			continue
		}

		// Parse GIT and GEM sections
		if inSection == "GIT" || inSection == "GEM" {
			currentGem = parseGemOrGitLine(line, gf, currentGem, gemLineRegex, dependencyRegex)
		}

		// Parse DEPENDENCIES section
		if inSection == "DEPENDENCIES" {
			parseDependenciesLine(line, gf, dependencyItemRegex)
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

// resolvePath expands ~ and resolves directory to lock file path
func resolvePath(path string, findFile func(string) string) string {
	// Expand ~ if needed
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if it's a directory
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	if info.IsDir() {
		resolved := findFile(path)
		if resolved == "" {
			return ""
		}
		return resolved
	}

	return path
}

// addGroupsToGem adds groups to a gem, avoiding duplicates
func addGroupsToGem(gem *Gem, groups []string) {
	for _, group := range groups {
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

// LoadGroupsFromGemfile parses the Gemfile to extract group information
func (g *Gemfile) LoadGroupsFromGemfile(gemfilePath string) error {
	gemfilePath = resolvePath(gemfilePath, FindGemfile)
	if gemfilePath == "" {
		return nil
	}

	// Try to read the Gemfile
	file, err := os.Open(gemfilePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	gemRegex := regexp.MustCompile(`^\s*gem\s+["']([a-z0-9_-]+)["']`)
	groupRegex := regexp.MustCompile(`^\s*group\s+:([a-z_]+)\s+do`)
	groupEndRegex := regexp.MustCompile(`^\s*end\s*$`)

	currentGroups := []string{"default"}
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
				addGroupsToGem(gem, currentGroups)
			}
		}
	}

	return nil
}

// ExtractRubyVersion extracts the Ruby version from Gemfile.lock
func ExtractRubyVersion(path string) string {
	path = resolvePath(path, FindLockFile)
	if path == "" {
		return "Unknown"
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
			version = strings.Trim(version, "\"'")
			return version
		}
	}

	return "Unknown"
}

// ExtractBundleVersion extracts the Bundle version from Gemfile.lock
func ExtractBundleVersion(path string) string {
	path = resolvePath(path, FindLockFile)
	if path == "" {
		return "Unknown"
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
