package gemfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Gem struct {
	Name         string
	Version      string
	Dependencies []string
	Groups       []string // e.g., "default", "development", "test", "production"
}

type Gemfile struct {
	Path string
	Gems map[string]*Gem
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
		path = filepath.Join(path, "Gemfile.lock")
	}

	// Read the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Gemfile.lock: %w", err)
	}
	defer file.Close()

	gf := &Gemfile{
		Path: path,
		Gems: make(map[string]*Gem),
	}

	scanner := bufio.NewScanner(file)
	inGemSection := false

	gemLineRegex := regexp.MustCompile(`(?i)^\s{4}([a-z0-9_-]+)\s+\(([^)]+)\)`)
	dependencyRegex := regexp.MustCompile(`(?i)^\s{6}([a-z0-9_-]+)`)

	var currentGem *Gem

	for scanner.Scan() {
		line := scanner.Text()

		// Look for GEM section
		if strings.HasPrefix(line, "GEM") {
			inGemSection = true
			continue
		}

		// Stop at PLATFORMS or other sections
		if inGemSection && strings.HasPrefix(line, "PLATFORMS") {
			break
		}

		// Skip non-gem lines
		if !inGemSection || strings.TrimSpace(line) == "" || strings.HasPrefix(line, "  remote:") || strings.HasPrefix(line, "  specs:") {
			continue
		}

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

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

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
		gemfilePath = filepath.Join(gemfilePath, "Gemfile")
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
