package ui

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spaquet/gemtracker/internal/cache"
	"github.com/spaquet/gemtracker/internal/gemfile"
)

// Spinner frames for loading animation (8-frame braille sequence)
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧"}

// ============================================================================
// View Modes
// ============================================================================

type ViewMode int

const (
	ViewLoading ViewMode = iota
	ViewGemList
	ViewGemDetail
	ViewSearch
	ViewCVE
	ViewProjectInfo
	ViewFilterMenu
	ViewSelectPath
	ViewError
)

// ============================================================================
// Messages
// ============================================================================

type AnalysisCompleteMsg struct {
	Result *gemfile.AnalysisResult
	Error  error
}

type DependencyCompleteMsg struct {
	Result *gemfile.DependencyResult
	Error  error
}

type SpinnerTickMsg struct{}

type VersionCheckMsg struct {
	LatestVersion string
	HasUpdate     bool
}

type ProgressMsg struct {
	Stage      string // "parsing", "checking-updates", "scanning-cves", "complete"
	Percentage int    // 0-100
	Message    string // Status message
}

type StageUpdateMsg struct {
	Stage          string                 // "parsing", "checking-updates", "scanning-cves"
	CurrentCount   int                    // Current gems processed
	TotalCount     int                    // Total gems to process
	Percentage     int                    // 0-100
	Result         *gemfile.AnalysisResult // Accumulated results so far
	OutdatedGems   []*gemfile.GemStatus   // Updated gems with version info
	VulnerableGems []*gemfile.GemStatus   // Updated with CVE info
}

// ============================================================================
// Model
// ============================================================================

type Model struct {
	// Window dimensions
	Width  int
	Height int

	// Current view and navigation
	CurrentView ViewMode
	ActiveTab   ViewMode // Persists across ViewLoading/ViewGemDetail

	// Data
	AnalysisResult   *gemfile.AnalysisResult
	DependencyResult *gemfile.DependencyResult

	// Gem List screen state
	FirstLevelGems      []*gemfile.GemStatus
	GemListCursor       int
	GemListOffset       int
	UnfilteredGems      []*gemfile.GemStatus // All first-level gems (for filter operations)
	SelectedGroups      map[string]bool       // Groups to filter by (if empty, show all)
	ShowOnlyUpgradable  bool                  // Filter to show only gems with updates
	AvailableGroups     []string              // All unique groups found in gems

	// Filter Menu screen state
	FilterMenuCursor int // Position in the filter menu (0 = upgradable, 1+ = groups)

	// Gem Detail screen state
	SelectedGem             *gemfile.GemStatus
	DetailSection           int // 0 = forward deps, 1 = reverse deps
	DetailForwardOffset     int
	DetailReverseOffset     int
	DetailTreeCursor        int                     // Selected line in current tree panel
	DetailForwardLines      []string                // Gem names at each line in forward tree
	DetailReverseLines      []string                // Gem names at each line in reverse tree
	DetailCurrentlyViewing  *gemfile.GemStatus      // The gem currently being viewed in detail (may differ from SelectedGem)
	DetailCurrentReverseDep *gemfile.DependencyInfo // Current gem's reverse dependencies

	// Search screen state
	SearchInput   textinput.Model
	SearchQuery   string
	SearchResults []*gemfile.GemStatus
	SearchCursor  int
	SearchOffset  int

	// CVE screen state
	VulnerableGems []*gemfile.GemStatus
	CVECursor      int
	CVEOffset      int

	// Project Info screen state
	RubyVersion      string
	RailsVersion     string
	BundleVersion    string
	OtherFramework   string            // For non-Rails projects
	TotalGems        int
	FirstLevelCount  int
	TransitiveDeps   int
	FrameworkDetected string           // The name of the framework detected

	// Path selection modal
	PathInput textinput.Model

	// Loading state
	Loading              bool
	LoadingMessage       string
	AnimationFrame       int
	AnalysisStage        string // "parsing", "checking-updates", "scanning-cves"
	AnalysisPercentage   int    // 0-100
	AnalysisCurrentCount int    // Current item in stage
	AnalysisTotalCount   int    // Total items for stage

	// Error state
	ErrorMessage string

	// Project state
	ProjectPath     string
	GemfileLockPath string

	// App metadata
	Version             string
	Commit              string
	Date                string
	NewVersionAvailable string // empty = no update, otherwise holds latest version tag
	Quitting            bool
	NoCache             bool   // Skip cache and force fresh analysis
}

// ============================================================================
// Initialization
// ============================================================================

func NewModel(version, commit, date, projectPath string, noCache bool) *Model {
	m := &Model{
		Version:        version,
		Commit:         commit,
		Date:           date,
		CurrentView:    ViewGemList,
		ActiveTab:      ViewGemList,
		SearchInput:    textinput.New(),
		PathInput:      textinput.New(),
		SelectedGroups: make(map[string]bool),
		NoCache:        noCache,
	}

	// Configure search input
	m.SearchInput.Placeholder = "Search gems..."
	m.SearchInput.PlaceholderStyle = textinput.NewModel().PlaceholderStyle
	m.SearchInput.PromptStyle = textinput.NewModel().PromptStyle
	m.SearchInput.TextStyle = textinput.NewModel().TextStyle
	m.SearchInput.Cursor.Style = textinput.NewModel().Cursor.Style

	// Configure path input
	m.PathInput.Placeholder = "/path/to/project"
	m.PathInput.PlaceholderStyle = textinput.NewModel().PlaceholderStyle
	m.PathInput.PromptStyle = textinput.NewModel().PromptStyle
	m.PathInput.TextStyle = textinput.NewModel().TextStyle
	m.PathInput.Cursor.Style = textinput.NewModel().Cursor.Style

	// Load the provided project path
	m.loadProject(projectPath)

	return m
}

func (m *Model) Init() tea.Cmd {
	// Auto-start analysis if Gemfile.lock exists
	if _, err := os.Stat(m.GemfileLockPath); err == nil {
		// File exists, start analysis
		m.CurrentView = ViewLoading
		m.ActiveTab = ViewGemList
		m.Loading = true
		m.LoadingMessage = "Parsing Gemfile.lock..."
		m.AnalysisStage = "parsing"
		m.AnalysisPercentage = 0
		m.AnimationFrame = 0

		return tea.Batch(
			// Progress ticker - increments percentage while analysis runs
			tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
				return ProgressTickMsg{}
			}),
			performAnalysis(m.GemfileLockPath, m.NoCache),
			checkLatestVersion(m.Version),
		)
	}

	// File doesn't exist, show path selection
	m.CurrentView = ViewSelectPath
	m.PathInput.Focus()
	// Still check for updates in background
	return checkLatestVersion(m.Version)
}

// ============================================================================
// Project Loading
// ============================================================================

func (m *Model) loadProject(path string) {
	// Expand ~ to home directory
	expandedPath := path
	if len(path) > 0 && path[0] == '~' {
		home := os.Getenv("HOME")
		expandedPath = home + path[1:]
	}

	// Convert to absolute path for reliable handling
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		absPath = expandedPath
	}

	// Check if path is a file (Gemfile.lock) or directory
	fileInfo, err := os.Stat(absPath)
	if err == nil && !fileInfo.IsDir() {
		// It's a file - assume it's Gemfile.lock
		m.GemfileLockPath = absPath
		m.ProjectPath = filepath.Dir(absPath)
		return
	}

	// It's a directory (or doesn't exist yet)
	m.ProjectPath = absPath
	m.GemfileLockPath = filepath.Join(m.ProjectPath, "Gemfile.lock")
}

// ============================================================================
// Async Tasks
// ============================================================================

type ProgressTickMsg struct{}

func performAnalysis(gemfilePath string, noCache bool) tea.Cmd {
	return func() tea.Msg {
		// Try to load from cache first (unless --no-cache flag is set)
		if !noCache {
			cacheEntry, cacheErr := cache.Read(gemfilePath)
			if cacheErr == nil && cacheEntry != nil && cacheEntry.Result != nil {
				// Cache hit! Return cached result
				return AnalysisCompleteMsg{
					Result: cacheEntry.Result,
					Error:  nil,
				}
			}
		}

		// Cache miss or invalid, do full analysis
		gf, err := gemfile.Parse(gemfilePath)
		if err != nil {
			return AnalysisCompleteMsg{
				Result: nil,
				Error:  err,
			}
		}

		// Load group information from Gemfile
		dir := filepath.Dir(gemfilePath)
		gf.LoadGroupsFromGemfile(dir)

		result := gemfile.Analyze(gf)
		return AnalysisCompleteMsg{
			Result: result,
			Error:  nil,
		}
	}
}

// performAnalysisWithProgress does analysis with progress reporting
// Emits ProgressMsg messages to show stages, then AnalysisCompleteMsg with results
func performAnalysisWithProgress(gemfilePath string) tea.Cmd {
	return func() tea.Msg {
		// Try to load from cache first
		cacheEntry, cacheErr := cache.Read(gemfilePath)
		if cacheErr == nil && cacheEntry != nil && cacheEntry.Result != nil {
			// Cache hit! Return complete analysis immediately
			return AnalysisCompleteMsg{
				Result: cacheEntry.Result,
				Error:  nil,
			}
		}

		// Stage 1: Parse Gemfile.lock (0-40%)
		gf, err := gemfile.Parse(gemfilePath)
		if err != nil {
			return AnalysisCompleteMsg{
				Result: nil,
				Error:  err,
			}
		}

		// Load group information from Gemfile
		dir := filepath.Dir(gemfilePath)
		gf.LoadGroupsFromGemfile(dir)

		// Stage 2: Analyze gems (40-70%)
		result := gemfile.Analyze(gf)

		// Stage 3: Return complete results (100%)
		return AnalysisCompleteMsg{
			Result: result,
			Error:  nil,
		}
	}
}

// performAnalysisWithProgressStages returns a batch of commands that emit progress
// This chains multiple progress updates through the message system
func performAnalysisWithProgressStages(gemfilePath string) tea.Cmd {
	return tea.Batch(
		// Emit initial parsing message
		func() tea.Msg {
			return ProgressMsg{
				Stage:      "parsing",
				Percentage: 10,
				Message:    "Parsing Gemfile.lock...",
			}
		},
		// Do the actual analysis after a small delay
		func() tea.Msg {
			time.Sleep(100 * time.Millisecond)

			// Try to load from cache first
			cacheEntry, cacheErr := cache.Read(gemfilePath)
			if cacheErr == nil && cacheEntry != nil && cacheEntry.Result != nil {
				// Cache hit! Return complete analysis
				return AnalysisCompleteMsg{
					Result: cacheEntry.Result,
					Error:  nil,
				}
			}

			// Do full analysis
			gf, err := gemfile.Parse(gemfilePath)
			if err != nil {
				return AnalysisCompleteMsg{
					Result: nil,
					Error:  err,
				}
			}

			// Load group information from Gemfile
			dir := filepath.Dir(gemfilePath)
			gf.LoadGroupsFromGemfile(dir)

			// Analyze gems
			result := gemfile.Analyze(gf)

			return AnalysisCompleteMsg{
				Result: result,
				Error:  nil,
			}
		},
	)
}

func performDependencyAnalysis(gemfilePath string, gemName string) tea.Cmd {
	return func() tea.Msg {
		gf, err := gemfile.Parse(gemfilePath)
		if err != nil {
			return DependencyCompleteMsg{Error: err}
		}

		// Load group information from Gemfile
		dir := filepath.Dir(gemfilePath)
		gf.LoadGroupsFromGemfile(dir)

		result := gemfile.AnalyzeDependencies(gf, gemName)
		return DependencyCompleteMsg{Result: result}
	}
}

func checkLatestVersion(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		// Create HTTP client with timeout
		client := &http.Client{Timeout: 5 * time.Second}

		// Fetch latest release from GitHub
		resp, err := client.Get("https://api.github.com/repos/spaquet/gemtracker/releases/latest")
		if err != nil {
			// Silently fail - don't disrupt user experience
			return VersionCheckMsg{HasUpdate: false}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return VersionCheckMsg{HasUpdate: false}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return VersionCheckMsg{HasUpdate: false}
		}

		// Parse JSON response
		var release struct {
			TagName string `json:"tag_name"`
		}
		if err := json.Unmarshal(body, &release); err != nil {
			return VersionCheckMsg{HasUpdate: false}
		}

		// Simple version comparison: strip "v" prefix and compare as strings
		latestVersion := strings.TrimPrefix(release.TagName, "v")
		currentVer := strings.TrimPrefix(currentVersion, "v")

		// If current is "dev" or empty, don't suggest upgrade
		if currentVer == "dev" || currentVer == "" {
			return VersionCheckMsg{HasUpdate: false}
		}

		// Basic comparison: if latest > current (string comparison)
		// In a real app, use semver, but for now string comparison works for standard versions
		hasUpdate := latestVersion > currentVer
		if hasUpdate {
			return VersionCheckMsg{LatestVersion: release.TagName, HasUpdate: true}
		}

		return VersionCheckMsg{HasUpdate: false}
	}
}

// ============================================================================
// Filter Methods
// ============================================================================

// extractAvailableGroups extracts unique groups from a list of gems
func (m *Model) extractAvailableGroups(gems []*gemfile.GemStatus) []string {
	groupSet := make(map[string]bool)
	for _, gem := range gems {
		for _, g := range gem.Groups {
			groupSet[g] = true
		}
	}

	var groups []string
	for g := range groupSet {
		groups = append(groups, g)
	}

	// Sort for consistent display
	sort.Strings(groups)
	return groups
}

// applyFilters applies the current filter state to FirstLevelGems
func (m *Model) applyFilters() {
	if m.UnfilteredGems == nil || len(m.UnfilteredGems) == 0 {
		return
	}

	m.FirstLevelGems = make([]*gemfile.GemStatus, 0)

	for _, gem := range m.UnfilteredGems {
		// Check upgradable filter
		if m.ShowOnlyUpgradable && !gem.IsOutdated {
			continue
		}

		// Check group filter - if no groups selected, show all
		if len(m.SelectedGroups) > 0 {
			gemHasSelectedGroup := false
			for _, gemGroup := range gem.Groups {
				if m.SelectedGroups[gemGroup] {
					gemHasSelectedGroup = true
					break
				}
			}
			if !gemHasSelectedGroup {
				continue
			}
		}

		m.FirstLevelGems = append(m.FirstLevelGems, gem)
	}

	// Reset cursor if out of bounds
	if m.GemListCursor >= len(m.FirstLevelGems) {
		m.GemListCursor = 0
	}
	m.GemListOffset = 0
}

// toggleGroupFilter toggles a group in the filter
func (m *Model) toggleGroupFilter(group string) {
	if m.SelectedGroups[group] {
		delete(m.SelectedGroups, group)
	} else {
		m.SelectedGroups[group] = true
	}
	m.applyFilters()
}

// toggleUpgradableFilter toggles the upgradable-only filter
func (m *Model) toggleUpgradableFilter() {
	m.ShowOnlyUpgradable = !m.ShowOnlyUpgradable
	m.applyFilters()
}

// clearFilters clears all applied filters
func (m *Model) clearFilters() {
	m.SelectedGroups = make(map[string]bool)
	m.ShowOnlyUpgradable = false
	m.applyFilters()
}

// hasActiveFilters returns true if any filters are applied
func (m *Model) hasActiveFilters() bool {
	return len(m.SelectedGroups) > 0 || m.ShowOnlyUpgradable
}
