// Package gemfile provides parsing and analysis of Ruby Gemfile.lock files.
//
// It handles:
//   - Parsing Gemfile.lock (and gems.locked) files to extract gem dependencies
//   - Extracting group information from Gemfile (and gems.rb) files
//   - Building dependency trees (forward and reverse dependencies)
//   - Analyzing gem health, outdated versions, and vulnerabilities
//   - Generating reports in multiple formats (text, CSV, JSON)
package gemfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/spaquet/gemtracker/internal/logger"
)

// Gem represents a Ruby gem with its version, dependencies, and group assignments.
type Gem struct {
	// Name is the lowercase gem name
	Name string
	// Version is the installed version string (may include platform suffixes like "x86_64-linux")
	Version string
	// Dependencies is a list of gem names that this gem depends on
	Dependencies []string
	// Groups lists the bundle groups this gem belongs to (e.g., "default", "development", "test", "production")
	Groups []string
	// IsFirstLevel is true if this gem is in the DEPENDENCIES section (directly required)
	IsFirstLevel bool
	// Source indicates where the gem is sourced from (e.g., "rubygems.org", a git URL)
	Source string
	// InsecureSource is true if the gem is sourced from an insecure protocol (http://, git://)
	InsecureSource bool
	// Constraint is the version constraint from Gemfile/gems.rb/gemspec (e.g., "~> 7.2", ">= 1.0")
	Constraint string
	// GitHubSource stores the GitHub source from Gemfile (e.g., "owner/repo" for github: option)
	GitHubSource string
	// GitHubRef stores the git ref from Gemfile (commit SHA, branch, tag)
	GitHubRef string
}

// Gemfile represents the parsed contents of a Gemfile.lock file.
type Gemfile struct {
	// Path is the absolute path to the Gemfile.lock file
	Path string
	// Gems is a map of all gems (by lowercase name) found in the lock file
	Gems map[string]*Gem
	// FirstLevelGems is a list of gem names that are directly required (in DEPENDENCIES section)
	FirstLevelGems []string
}

// FindLockFile searches for a Ruby lock file in the given directory.
// It probes in priority order: gems.locked, Gemfile.lock.
// Returns the absolute path to the lock file if found, empty string otherwise.
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

// FindGemfile searches for a Ruby Gemfile in the given directory.
// It probes in priority order: gems.rb, Gemfile.
// Returns the absolute path to the Gemfile if found, empty string otherwise.
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

// parseState tracks the current parsing state while processing a Gemfile.lock.
type parseState struct {
	inSection     string
	currentSource string
	currentGem    *Gem
}

// resolveLockFilePath resolves a path to a lock file, handling tilde expansion and directory lookup.
func resolveLockFilePath(path string) (string, error) {
	// Expand ~ if needed
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if it's a directory, if so look for Gemfile.lock
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %w", err)
	}

	if info.IsDir() {
		lockFile := FindLockFile(path)
		if lockFile == "" {
			logger.Warn("No lock file found (gems.locked or Gemfile.lock) in %s", path)
			return "", fmt.Errorf("no lock file found (gems.locked or Gemfile.lock) in %s", path)
		}
		logger.Info("Using lock file: %s", filepath.Base(lockFile))
		return lockFile, nil
	}

	return path, nil
}

// processParserLine processes a single line from the Gemfile.lock file.
// Returns true if parsing should break, false otherwise.
func processParserLine(line string, gf *Gemfile, state *parseState, gemLineRegex, dependencyRegex, dependencyItemRegex, remoteRegex *regexp.Regexp) bool {
	// Check for section headers
	if newSection, isSectionHeader := detectSection(line); isSectionHeader {
		if newSection == "BUNDLED" {
			return true
		}
		state.inSection = newSection
		if newSection == "GEM" {
			state.currentSource = "https://rubygems.org/"
		} else if newSection == "PATH" {
			// PATH section source will be set by the remote line, default to "."
			state.currentSource = "."
		}
		return false
	}

	if shouldSkipLine(line, state.inSection) {
		return false
	}

	// Parse remote lines in GIT and PATH sections
	if state.inSection == "GIT" || state.inSection == "PATH" {
		remoteMatches := remoteRegex.FindStringSubmatch(line)
		if len(remoteMatches) > 0 {
			state.currentSource = strings.TrimSpace(remoteMatches[1])
			return false
		}
	}

	// Parse GIT, PATH, and GEM sections (all contain gem specifications)
	if state.inSection == "GIT" || state.inSection == "PATH" || state.inSection == "GEM" {
		state.currentGem = parseGemOrGitLine(line, gf, state.currentGem, gemLineRegex, dependencyRegex)
		if state.currentGem != nil && state.currentGem.Source == "" {
			state.currentGem.Source = state.currentSource
			state.currentGem.InsecureSource = isInsecureSource(state.currentSource)
		}
	}

	// Parse DEPENDENCIES section
	if state.inSection == "DEPENDENCIES" {
		parseDependenciesLine(line, gf, dependencyItemRegex)
	}

	return false
}

// detectSection checks if a line is a Gemfile.lock section header and returns the section name
// and a boolean indicating whether it's a section header.
func detectSection(line string) (string, bool) {
	switch {
	case strings.HasPrefix(line, "PATH"):
		return "PATH", true
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

// isInsecureSource checks if a source URL uses an insecure protocol.
// Returns true for http://, git://, git+http:// protocols.
func isInsecureSource(source string) bool {
	source = strings.TrimSpace(source)
	return strings.HasPrefix(source, "http://") ||
		strings.HasPrefix(source, "git://") ||
		strings.HasPrefix(source, "git+http://")
}

// shouldSkipLine determines if a line should be skipped during Gemfile.lock parsing.
// It skips blank lines and metadata lines (remote, specs, revision, branch, tag).
// Note: remote lines in GIT sections are NOT skipped so we can extract source information.
func shouldSkipLine(line string, inSection string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	// Skip specs, revision, branch, tag lines
	if strings.HasPrefix(line, "  specs:") ||
		strings.HasPrefix(line, "  revision:") ||
		strings.HasPrefix(line, "  branch:") ||
		strings.HasPrefix(line, "  tag:") {
		return true
	}
	// In GIT section, don't skip remote line (we need it)
	// In GEM section, skip remote line (it's always rubygems.org)
	if strings.HasPrefix(line, "  remote:") && inSection != "GIT" {
		return true
	}
	// Skip PLATFORMS section content
	if inSection == "PLATFORMS" && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, " ")) {
		return true
	}
	return false
}

// getCurrentPlatform returns the platform identifier for the current system.
// This matches the format used in Gemfile.lock (e.g., "arm64-darwin", "x86_64-linux")
func getCurrentPlatform() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	if goos == "darwin" {
		if goarch == "arm64" {
			return "arm64-darwin"
		}
		return "x86_64-darwin"
	}
	if goos == "linux" {
		if goarch == "arm64" {
			return "aarch64-linux"
		}
		return "x86_64-linux"
	}
	return goos + "-" + goarch
}

// matchesPlatform checks if a version string's platform suffix matches the given platform.
// Handles versions like "1.6.3-arm64-darwin", "1.6.3-x86_64-linux-musl", "1.6.3" (generic)
func matchesPlatform(version, platform string) bool {
	parts := strings.Split(version, "-")
	if len(parts) <= 1 {
		return false // Generic version, no platform suffix
	}
	// Check if version starts with the platform prefix
	return strings.HasPrefix(version, platform)
}

// parseGemOrGitLine parses gem spec lines (4-space indent) and dependency lines (6-space indent)
// from GIT/GEM sections of the Gemfile.lock. Returns the current or newly created Gem.
// When multiple platform-specific versions exist for a gem, it selects the one matching the current system.
func parseGemOrGitLine(line string, gf *Gemfile, currentGem *Gem, gemLineRegex, depRegex *regexp.Regexp) *Gem {
	// Parse gem lines (4-space indent)
	matches := gemLineRegex.FindStringSubmatch(line)
	if len(matches) > 0 {
		name := strings.ToLower(matches[1])
		version := matches[2]

		// Check if we already have a version for this gem and if it's the correct platform
		existingGem, exists := gf.Gems[name]
		shouldReplace := !exists

		if exists {
			// Replace if current platform matches better, or if we don't have the current platform yet
			currentPlatform := getCurrentPlatform()
			existingIsCurrentPlatform := matchesPlatform(version, currentPlatform)
			existingIsCurrentPlatform = existingIsCurrentPlatform || matchesPlatform(existingGem.Version, currentPlatform)

			// Prefer current platform version; if existing is generic and new is specific to current platform, replace
			// Also replace if existing doesn't match current platform but new does
			if strings.HasPrefix(version, currentPlatform) ||
				(!strings.HasPrefix(existingGem.Version, currentPlatform) && strings.HasPrefix(version, currentPlatform)) {
				shouldReplace = true
			}
		}

		if shouldReplace {
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
		return existingGem
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

// parseDependenciesLine parses gem names from the DEPENDENCIES section of Gemfile.lock.
// Marks the gem as first-level (directly required) if it exists in the gems map.
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

// Parse parses a Gemfile.lock file (or gems.locked) and returns the parsed Gemfile structure.
// It accepts either a file path or directory path; if a directory is provided, it searches for
// a lock file in that directory. Expands ~/ in paths. Returns an error if the file cannot be
// found, opened, or parsed.
func Parse(path string) (*Gemfile, error) {
	resolvedPath, err := resolveLockFilePath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		logger.Error("Failed to open lock file %s: %v", resolvedPath, err)
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	defer file.Close()

	gf := &Gemfile{
		Path:           resolvedPath,
		Gems:           make(map[string]*Gem),
		FirstLevelGems: []string{},
	}

	scanner := bufio.NewScanner(file)
	state := &parseState{
		inSection:     "",
		currentSource: "https://rubygems.org/",
		currentGem:    nil,
	}

	gemLineRegex := regexp.MustCompile(`(?i)^\s{4}([a-z0-9_-]+)\s+\(([^)]+)\)`)
	dependencyRegex := regexp.MustCompile(`(?i)^\s{6}([a-z0-9_-]+)`)
	dependencyItemRegex := regexp.MustCompile(`(?i)^\s{2}([a-z0-9_-]+)`)
	remoteRegex := regexp.MustCompile(`^\s{2}remote:\s+(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		if shouldBreak := processParserLine(line, gf, state, gemLineRegex, dependencyRegex, dependencyItemRegex, remoteRegex); shouldBreak {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading lock file: %v", err)
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	logger.Info("Successfully parsed lock file: %d total gems, %d first-level", len(gf.Gems), len(gf.FirstLevelGems))
	return gf, nil
}

// GetGemCount returns the total number of gems in the parsed Gemfile.
func (g *Gemfile) GetGemCount() int {
	return len(g.Gems)
}

// GetGemsAsList returns all gems in the Gemfile as a slice.
func (g *Gemfile) GetGemsAsList() []*Gem {
	gems := make([]*Gem, 0, len(g.Gems))
	for _, gem := range g.Gems {
		gems = append(gems, gem)
	}
	return gems
}

// resolvePath expands ~/ in paths and resolves a directory to a lock/Gemfile path
// using the provided findFile function. Returns empty string if the path is invalid.
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

// addGroupsToGem adds one or more groups to a gem, avoiding duplicate group assignments.
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

// LoadGroupsFromGemfile parses the Gemfile (or gems.rb) to extract group assignments for gems.
// It processes group blocks (e.g., "group :development do") and assigns those groups to gems defined within.
// Returns an error if the Gemfile cannot be read; returns nil if the Gemfile is not found (graceful degradation).
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

// GetInsecureSourceGems returns all gems that are sourced from insecure protocols (http://, git://)
func (g *Gemfile) GetInsecureSourceGems() []*Gem {
	var insecureGems []*Gem
	for _, gem := range g.Gems {
		if gem.InsecureSource {
			insecureGems = append(insecureGems, gem)
		}
	}
	return insecureGems
}

// LoadConstraintsFromGemfile parses the Gemfile (or gems.rb) to extract version constraints for gems.
// It processes gem declarations with version constraints (e.g., "gem 'rails', '~> 7.2'").
// Handles multiple constraints: gem 'pg', '>= 1.1', '< 2.0' -> '>= 1.1, < 2.0'
// Returns an error if the Gemfile cannot be read; returns nil if the Gemfile is not found (graceful degradation).
func (g *Gemfile) LoadConstraintsFromGemfile(gemfilePath string) error {
	gemfilePath = resolvePath(gemfilePath, FindGemfile)
	if gemfilePath == "" {
		logger.Info("Gemfile not found, skipping constraint loading")
		return nil
	}

	logger.Info("Loading constraints from Gemfile: %s", gemfilePath)

	file, err := os.Open(gemfilePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Matches: gem 'name' or gem 'name', ...
	gemLineRegex := regexp.MustCompile(`^\s*gem\s+["']([a-z0-9_-]+)["'](.*)$`)
	// Extracts all quoted strings from the remainder of the line
	quotedRegex := regexp.MustCompile(`["']([^"']+)["']`)

	for scanner.Scan() {
		line := scanner.Text()

		matches := gemLineRegex.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		gemName := matches[1]
		remainder := matches[2]

		// Extract all quoted strings after gem name (these are constraints and options)
		quotedMatches := quotedRegex.FindAllStringSubmatch(remainder, -1)
		var constraints []string

		for _, qm := range quotedMatches {
			if len(qm) > 1 {
				quoted := qm[1]
				// Skip option keys like "git", "path", "platforms", etc.
				// Version constraints start with operators: ~>, >=, >, <=, <, =
				if isVersionConstraint(quoted) {
					constraints = append(constraints, quoted)
				}
			}
		}

		constraint := strings.Join(constraints, ", ")

		// Store constraint only for first-level gems (those in DEPENDENCIES section of lock file)
		if gem, ok := g.Gems[gemName]; ok && gem.IsFirstLevel {
			gem.Constraint = constraint
			if constraint != "" {
				logger.Info("Loaded constraint for %s: %s", gemName, constraint)
			}
		}
	}

	return nil
}

// LoadConstraintsFromGemspec parses .gemspec files to extract version constraints for gem dependencies.
// It looks for add_runtime_dependency and add_development_dependency declarations.
// Handles multiple constraints: spec.add_runtime_dependency "pg", ">= 1.1", "< 2.0" -> ">= 1.1, < 2.0"
// Returns an error if the gemspec cannot be read; returns nil if not found (graceful degradation).
func (g *Gemfile) LoadConstraintsFromGemspec(gemspecPath string) error {
	// Try to find gemspec file in same directory as lock file
	if gemspecPath == "" {
		lockDir := filepath.Dir(g.Path)
		entries, err := os.ReadDir(lockDir)
		if err != nil {
			return nil
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gemspec") {
				gemspecPath = filepath.Join(lockDir, entry.Name())
				break
			}
		}
	}

	if gemspecPath == "" {
		return nil
	}

	file, err := os.Open(gemspecPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Matches: add_runtime_dependency "name", ... or add_development_dependency "name", ...
	gemspecDepRegex := regexp.MustCompile(`^\s*(?:add_runtime_dependency|add_development_dependency)\s+["']([a-z0-9_-]+)["'](.*)$`)
	// Extracts all quoted strings
	quotedRegex := regexp.MustCompile(`["']([^"']+)["']`)

	for scanner.Scan() {
		line := scanner.Text()

		matches := gemspecDepRegex.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		gemName := matches[1]
		remainder := matches[2]

		// Extract all quoted strings after gem name (these are constraints)
		quotedMatches := quotedRegex.FindAllStringSubmatch(remainder, -1)
		var constraints []string

		for _, qm := range quotedMatches {
			if len(qm) > 1 {
				quoted := qm[1]
				if isVersionConstraint(quoted) {
					constraints = append(constraints, quoted)
				}
			}
		}

		constraint := strings.Join(constraints, ", ")

		// For gemspec, add constraint only if not already set by Gemfile (Gemfile takes precedence)
		if gem, ok := g.Gems[gemName]; ok && gem.Constraint == "" {
			gem.Constraint = constraint
		}
	}

	return nil
}

// isVersionConstraint checks if a string is a version constraint (starts with operator).
// Returns true for: ~>, >=, >, <=, <, =
func isVersionConstraint(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Check for version constraint operators
	return strings.HasPrefix(s, "~>") ||
		strings.HasPrefix(s, ">=") ||
		strings.HasPrefix(s, ">") ||
		strings.HasPrefix(s, "<=") ||
		strings.HasPrefix(s, "<") ||
		strings.HasPrefix(s, "=") ||
		// Also match plain version numbers (e.g., "1.2.3")
		(s[0] >= '0' && s[0] <= '9')
}

// LoadGitHubSourcesFromGemfile parses the Gemfile to extract GitHub sources specified via the github: option.
// This allows gemtracker to display the correct reference URL when users use a fork/custom repo.
// Example: gem "gemname", github: "owner/repo", ref: "abc123"
func (g *Gemfile) LoadGitHubSourcesFromGemfile(gemfilePath string) error {
	gemfilePath = resolvePath(gemfilePath, FindGemfile)
	if gemfilePath == "" {
		return nil
	}

	logger.Info("Loading GitHub sources from Gemfile: %s", gemfilePath)

	file, err := os.Open(gemfilePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	githubLineRegex := regexp.MustCompile(`^\s*gem\s+["']([a-z0-9_-]+)["'].*github:\s+["']([^"']+)["']`)
	refLineRegex := regexp.MustCompile(`ref:\s+["']([^"']+)["']`)

	for scanner.Scan() {
		line := scanner.Text()

		matches := githubLineRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		gemName := matches[1]
		githubSource := matches[2]

		refMatch := refLineRegex.FindStringSubmatch(line)
		var ref string
		if len(refMatch) > 1 {
			ref = refMatch[1]
		}

		if gem, ok := g.Gems[gemName]; ok {
			gem.GitHubSource = githubSource
			gem.GitHubRef = ref
			logger.Info("Loaded GitHub source for %s: %s (ref: %s)", gemName, githubSource, ref)
		}
	}

	return nil
}
