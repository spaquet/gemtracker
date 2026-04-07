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

// ParseGemspec parses a Ruby .gemspec file to extract gem dependencies.
// It extracts add_runtime_dependency, add_development_dependency, and add_dependency declarations.
// Returns a Gemfile structure with all gems as first-level dependencies.
func ParseGemspec(path string) (*Gemfile, error) {
	// Expand ~ if needed
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if it's a directory, if so look for *.gemspec
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	var gemspecPath string
	if info.IsDir() {
		// Look for any .gemspec file in the directory
		files, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".gemspec") {
				gemspecPath = filepath.Join(path, file.Name())
				break
			}
		}

		if gemspecPath == "" {
			return nil, fmt.Errorf("no .gemspec file found in %s", path)
		}
	} else {
		gemspecPath = path
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
	// Matches: add_runtime_dependency 'gem_name', '>= 1.0' or add_development_dependency, etc.
	// Also matches spec.add_runtime_dependency and s.add_runtime_dependency patterns
	runtimeDepRegex := regexp.MustCompile(`(?:spec\.|s\.)?add_(?:runtime_)?dependency\s+['"]([^'"]+)['"]\s*(?:,\s*['"]([^'"]+)['"])?`)
	developmentDepRegex := regexp.MustCompile(`(?:spec\.|s\.)?add_development_dependency\s+['"]([^'"]+)['"]\s*(?:,\s*['"]([^'"]+)['"])?`)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for development dependency
		matches := developmentDepRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			gemName := strings.ToLower(matches[1])
			version := ""
			if len(matches) > 2 && matches[2] != "" {
				version = matches[2]
			}

			gem := &Gem{
				Name:         gemName,
				Version:      version,
				Dependencies: []string{},
				Groups:       []string{"development"},
				IsFirstLevel: true,
			}

			gf.Gems[gemName] = gem
			gf.FirstLevelGems = append(gf.FirstLevelGems, gemName)
			continue
		}

		// Check for runtime dependency (including plain add_dependency)
		matches = runtimeDepRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			gemName := strings.ToLower(matches[1])
			version := ""
			if len(matches) > 2 && matches[2] != "" {
				version = matches[2]
			}

			// Only add if not already added as development dependency
			if _, exists := gf.Gems[gemName]; !exists {
				gem := &Gem{
					Name:         gemName,
					Version:      version,
					Dependencies: []string{},
					Groups:       []string{"production"},
					IsFirstLevel: true,
				}

				gf.Gems[gemName] = gem
				gf.FirstLevelGems = append(gf.FirstLevelGems, gemName)
			}
		}
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

// countGemsByGroup returns the count of gems in a specific group
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
