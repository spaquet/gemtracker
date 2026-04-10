package gemfile

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DetectRubyManager extracts the Ruby version manager name from gem env gemdir output.
// Examples:
//
//	/Users/user/.frum/versions/3.4.4/lib/ruby/gems/3.4.0 → "frum"
//	/Users/user/.rbenv/versions/3.4.4/lib/ruby/gems/3.4.0 → "rbenv"
//	/Users/user/.rvm/gems/ruby-3.4.4 → "rvm"
//	/usr/lib/ruby/gems/3.4.0 → "system"
func DetectRubyManager(gemDirPath string) string {
	// Check for known manager directories in the path
	if strings.Contains(gemDirPath, "/.frum/") {
		return "frum"
	}
	if strings.Contains(gemDirPath, "/.rbenv/") {
		return "rbenv"
	}
	if strings.Contains(gemDirPath, "/.rvm/") {
		return "rvm"
	}
	if strings.Contains(gemDirPath, "/.asdf/") {
		return "asdf"
	}
	if strings.Contains(gemDirPath, "/.rubies/") {
		return "chruby"
	}
	return "system"
}

// GetGemDirPath executes `gem env gemdir` and returns the gem directory path.
func GetGemDirPath() (string, error) {
	cmd := exec.Command("gem", "env", "gemdir")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get gem directory: %w", err)
	}

	gemDir := strings.TrimSpace(out.String())
	if gemDir == "" {
		return "", fmt.Errorf("gem env gemdir returned empty path")
	}

	return gemDir, nil
}

// GetGemSize calculates the total size of a gem directory in bytes.
// Returns 0 if the gem is not found, error if the calculation fails.
func GetGemSize(gemName string, gemDirPath string) (int64, error) {
	// Construct the gem directory path (gems are stored in gemDirPath/gems/)
	gemPath := filepath.Join(gemDirPath, "gems", gemName+"-*")

	// Use glob to find gems with platform suffixes (e.g., gem-1.0-x86_64-linux)
	matches, err := filepath.Glob(gemPath)
	if err != nil {
		return 0, fmt.Errorf("failed to glob gem path: %w", err)
	}

	if len(matches) == 0 {
		// Gem not found, return 0 (not an error)
		return 0, nil
	}

	// If multiple versions found, sum them all
	var totalSize int64
	for _, match := range matches {
		size, err := dirSize(match)
		if err != nil {
			// Log but continue with other gems
			continue
		}
		totalSize += size
	}

	return totalSize, nil
}

// dirSize recursively calculates the total size of a directory in bytes.
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// GetGemInfo executes `gem info <gemName>` and returns the sanitized output.
// Uses a timeout to prevent hanging if the gem command is slow or unresponsive.
func GetGemInfo(gemName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gem", "info", gemName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("gem info command timed out (3s)")
	}

	output := out.String()

	// Sanitize output: remove ANSI codes and clean up whitespace
	output = sanitizeGemOutput(output)

	if err != nil {
		// gem info returns non-zero if gem not found, but output is still useful
		return output, fmt.Errorf("gem info command failed: %w", err)
	}

	return output, nil
}

// sanitizeGemOutput removes ANSI escape codes and cleans up output for safe display
func sanitizeGemOutput(s string) string {
	// Remove ANSI escape sequences (colors, formatting)
	s = removeANSICodes(s)

	// Convert to valid UTF-8, replacing invalid sequences
	s = replaceInvalidUTF8(s)

	// Limit total length to prevent huge outputs from crashing rendering
	maxLen := 5000
	if len(s) > maxLen {
		s = s[:maxLen] + "\n... (output truncated)"
	}

	return s
}

// removeANSICodes removes ANSI escape sequences from a string
func removeANSICodes(s string) string {
	// Basic ANSI escape sequence removal
	result := ""
	inEscape := false
	for _, ch := range s {
		if ch == '\x1b' {
			inEscape = true
		} else if inEscape {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEscape = false
			}
		} else {
			result += string(ch)
		}
	}
	return result
}

// replaceInvalidUTF8 replaces invalid UTF-8 sequences with '?'
func replaceInvalidUTF8(s string) string {
	result := ""
	for _, ch := range s {
		if ch == '\ufffd' { // Unicode replacement character
			result += "?"
		} else {
			result += string(ch)
		}
	}
	return result
}

// CalculateProjectSize calculates the total size of all project gems and returns
// a map of gem name to size in bytes.
func CalculateProjectSize(gems []*Gem, gemDirPath string) (int64, map[string]int64, error) {
	sizes := make(map[string]int64)
	var totalSize int64

	for _, gem := range gems {
		size, err := GetGemSize(gem.Name, gemDirPath)
		if err != nil {
			// Log but continue with other gems
			continue
		}
		sizes[gem.Name] = size
		totalSize += size
	}

	return totalSize, sizes, nil
}

// FormatBytes converts bytes to human-readable format (KB, MB, GB).
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
