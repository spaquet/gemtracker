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

// ParseGemspec parses a Ruby .gemspec file to extract gem dependencies declared via
// add_runtime_dependency, add_development_dependency, and add_dependency directives.
// It accepts either a file path or a directory path; if a directory is provided, it searches for
// the first .gemspec file in that directory. Version constraints from unresolved gemspec declarations
// are extracted but cannot be compared against actual installed versions without Gemfile.lock.
// Returns a Gemfile structure with all gems marked as first-level dependencies.
func ParseGemspec(path string) (*Gemfile, error) {
	gemspecPath, err := resolveGemspecPath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(gemspecPath)
	if err != nil {
		logger.Error("Failed to open gemspec file %s: %v", gemspecPath, err)
		return nil, fmt.Errorf("failed to open gemspec file: %w", err)
	}
	defer file.Close()

	logger.Info("Parsing gemspec file: %s", gemspecPath)

	gf := &Gemfile{
		Path:           gemspecPath,
		Gems:           make(map[string]*Gem),
		FirstLevelGems: []string{},
	}

	scanner := bufio.NewScanner(file)

	// Regex patterns for dependency declarations
	runtimeDepRegex := regexp.MustCompile(`(?:spec\.|s\.)?add_(?:runtime_)?dependency\s+['"]([^'"]+)['"]\s*(?:,\s*['"]([^'"]+)['"])?`)
	developmentDepRegex := regexp.MustCompile(`(?:spec\.|s\.)?add_development_dependency\s+['"]([^'"]+)['"]\s*(?:,\s*['"]([^'"]+)['"])?`)

	for scanner.Scan() {
		line := scanner.Text()
		processGemspecLine(line, gf, runtimeDepRegex, developmentDepRegex)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading gemspec file %s: %v", gemspecPath, err)
		return nil, fmt.Errorf("error reading gemspec file: %w", err)
	}

	logger.Info("Successfully parsed gemspec: %d gems found (%d runtime, %d development)",
		len(gf.FirstLevelGems),
		countGemsByGroup(gf, ""),
		countGemsByGroup(gf, "development"))

	return gf, nil
}

// resolveGemspecPath resolves a path to a gemspec file, handling tilde expansion and directory lookup.
func resolveGemspecPath(path string) (string, error) {
	// Expand ~ if needed
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if it's a directory, if so look for *.gemspec
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %w", err)
	}

	if info.IsDir() {
		return findGemspecInDir(path)
	}

	return path, nil
}

// findGemspecInDir searches for a .gemspec file in the given directory.
func findGemspecInDir(path string) (string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".gemspec") {
			return filepath.Join(path, file.Name()), nil
		}
	}

	return "", fmt.Errorf("no .gemspec file found in %s", path)
}

// processGemspecLine processes a single line from a gemspec file and adds dependencies to the gemfile.
func processGemspecLine(line string, gf *Gemfile, runtimeDepRegex, developmentDepRegex *regexp.Regexp) {
	// Skip comments and empty lines
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return
	}

	// Check for development dependency
	if matches := developmentDepRegex.FindStringSubmatch(line); len(matches) > 0 {
		addGemFromMatch(gf, matches, []string{"development"})
		return
	}

	// Check for runtime dependency (including plain add_dependency)
	if matches := runtimeDepRegex.FindStringSubmatch(line); len(matches) > 0 {
		// Only add if not already added as development dependency
		gemName := strings.ToLower(matches[1])
		if _, exists := gf.Gems[gemName]; !exists {
			addGemFromMatch(gf, matches, []string{"production"})
		}
	}
}

// addGemFromMatch creates a gem from a regex match and adds it to the gemfile.
func addGemFromMatch(gf *Gemfile, matches []string, groups []string) {
	gemName := strings.ToLower(matches[1])
	version := ""
	if len(matches) > 2 && matches[2] != "" {
		version = matches[2]
	}

	gem := &Gem{
		Name:         gemName,
		Version:      version,
		Dependencies: []string{},
		Groups:       groups,
		IsFirstLevel: true,
	}

	gf.Gems[gemName] = gem
	gf.FirstLevelGems = append(gf.FirstLevelGems, gemName)
}

// countGemsByGroup returns the count of gems in a specific group.
// If group is empty string, counts gems that are NOT in the "development" group (i.e., runtime gems).
func countGemsByGroup(gf *Gemfile, group string) int {
	count := 0
	for _, gem := range gf.Gems {
		if group == "" {
			// Count gems without "development" group
			hasDev := false
			for _, g := range gem.Groups {
				if g == "development" {
					hasDev = true
					break
				}
			}
			if !hasDev {
				count++
			}
		} else {
			// Count gems with specific group
			for _, g := range gem.Groups {
				if g == group {
					count++
					break
				}
			}
		}
	}
	return count
}
