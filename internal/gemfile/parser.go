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
	Name        string
	Version     string
	Dependencies []string
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

	gemLineRegex := regexp.MustCompile(`^\s{4}([a-z0-9_-]+)\s+\(([^)]+)\)`)

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

		// Parse gem lines
		matches := gemLineRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			name := matches[1]
			version := matches[2]

			gf.Gems[name] = &Gem{
				Name:        name,
				Version:     version,
				Dependencies: []string{},
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
