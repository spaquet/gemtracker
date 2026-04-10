package gemfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/spaquet/gemtracker/internal/logger"
)

// CVECommentDecision represents the user's decision on a CVE
type CVECommentDecision string

const (
	DecisionAcknowledged CVECommentDecision = "acknowledged"
	DecisionIgnored      CVECommentDecision = "ignored"
)

// CVEComment represents a user's comment and decision on a CVE advisory
type CVEComment struct {
	Decision   CVECommentDecision `json:"decision"`
	Comment    string             `json:"comment"`
	GemName    string             `json:"gem_name"`
	GemVersion string             `json:"gem_version"`  // installed version at save time
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// CVEComments holds all comment entries for a project
type CVEComments struct {
	Version int                    `json:"version"`
	Entries map[string]*CVEComment `json:"entries"` // key = CVE ID (vuln.CVE or vuln.OSVId)
}

const (
	CommentFileName = ".gemtracker_comments.json"
	CommentVersion  = 1
)

// LoadCVEComments loads CVE comments from the project directory
func LoadCVEComments(projectDir string) (*CVEComments, error) {
	filePath := filepath.Join(projectDir, CommentFileName)

	// If file doesn't exist, return empty comments
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Info("CVE comments file not found at %s, returning empty comments", filePath)
		return &CVEComments{
			Version: CommentVersion,
			Entries: make(map[string]*CVEComment),
		}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Warn("Failed to read CVE comments file: %v", err)
		return &CVEComments{
			Version: CommentVersion,
			Entries: make(map[string]*CVEComment),
		}, nil
	}

	var comments CVEComments
	err = json.Unmarshal(data, &comments)
	if err != nil {
		logger.Warn("Failed to parse CVE comments file: %v", err)
		return &CVEComments{
			Version: CommentVersion,
			Entries: make(map[string]*CVEComment),
		}, nil
	}

	// Ensure map is initialized
	if comments.Entries == nil {
		comments.Entries = make(map[string]*CVEComment)
	}

	logger.Info("Loaded %d CVE comments from %s", len(comments.Entries), filePath)
	return &comments, nil
}

// SaveCVEComments saves CVE comments to the project directory
func SaveCVEComments(projectDir string, comments *CVEComments) error {
	filePath := filepath.Join(projectDir, CommentFileName)

	if comments == nil {
		comments = &CVEComments{
			Version: CommentVersion,
			Entries: make(map[string]*CVEComment),
		}
	}

	// Ensure version is set
	comments.Version = CommentVersion

	data, err := json.MarshalIndent(comments, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal CVE comments: %v", err)
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		logger.Error("Failed to write CVE comments file: %v", err)
		return err
	}

	logger.Info("Saved %d CVE comments to %s", len(comments.Entries), filePath)
	return nil
}

// GetCVECommentKey returns the appropriate key for a vulnerability (prefer CVE, fall back to OSVId)
func GetCVECommentKey(vuln *Vulnerability) string {
	if vuln.CVE != "" {
		return vuln.CVE
	}
	return vuln.OSVId
}
