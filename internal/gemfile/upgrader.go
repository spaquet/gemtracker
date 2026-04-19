package gemfile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type UpgradeResult struct {
	GemName string
	Success bool
	Error   string
}

var cacheDir string

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		cacheDir = ".cache/gemtracker"
		return
	}
	cacheDir = filepath.Join(homeDir, ".cache", "gemtracker")
}

func getCacheDir() string {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		os.MkdirAll(cacheDir, 0755)
	}
	return cacheDir
}

func UpgradeGems(gemNames []string, projectPath string) ([]UpgradeResult, error) {
	var results []UpgradeResult

	if len(gemNames) == 0 {
		return results, nil
	}

	errorLogPath := filepath.Join(getCacheDir(), "upgrade_errors.log")

	for _, gemName := range gemNames {
		result := UpgradeResult{GemName: gemName}

		cmd := exec.Command("bundle", "update", gemName)
		cmd.Dir = projectPath
		output, err := cmd.CombinedOutput()

		if err != nil {
			result.Success = false
			result.Error = string(output)
			writeErrorLog(errorLogPath, gemName, err.Error(), string(output))
		} else {
			result.Success = true
		}
		results = append(results, result)
	}

	return results, nil
}

func writeErrorLog(logPath, gemName, errMsg, output string) {
	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("[%s] %s: %s\n%s\n\n", timestamp, gemName, errMsg, output)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(logEntry)
}

func GetUpgradeErrorLogPath() string {
	return filepath.Join(getCacheDir(), "upgrade_errors.log")
}

func ClearUpgradeErrorLog() error {
	logPath := GetUpgradeErrorLogPath()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(logPath)
}

func ParseUpgradeErrors() []string {
	logPath := GetUpgradeErrorLogPath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		return nil
	}

	var errors []string
	entries := strings.Split(string(content), "\n\n")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		errors = append(errors, entry)
	}

	return errors
}
