// Package ui provides the interactive Terminal User Interface (TUI) for gemtracker using BubbleTea.
//
// The TUI displays gem dependency analysis across multiple views (gem list, details, search, vulnerabilities,
// project info) with real-time background updates for gem health, outdated versions, and health scoring.
// It handles user keyboard input, manages application state, and coordinates async data fetching.
package ui

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/spaquet/gemtracker/internal/logger"
)

// Spinner frames for loading animation (8-frame braille sequence)
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧"}

// ============================================================================
// Framework Gems
// ============================================================================

// frameworkGems maps gem names to their framework families
// Used for categorizing upgradeable gems
var frameworkGems = map[string]string{
	// Rails
	"actioncable":   "rails",
	"actionmailer":  "rails",
	"actionpack":    "rails",
	"actiontext":    "rails",
	"actionview":    "rails",
	"activejob":     "rails",
	"activemodel":   "rails",
	"activerecord":  "rails",
	"activestorage": "rails",
	"activesupport": "rails",
	"railties":      "rails",
	"activeconfig":  "rails",
	// Sinatra
	"sinatra-contrib": "sinatra",
	"rack-protection": "sinatra",
	// Hanami
	"hanami-controller": "hanami",
	"hanami-view":       "hanami",
	"hanami-router":     "hanami",
}

// ============================================================================
// View Modes
// ============================================================================

// ViewMode represents the current screen being displayed in the TUI.
type ViewMode int

const (
	// ViewLoading displays the loading/progress screen while analyzing the project
	ViewLoading ViewMode = iota
	// ViewGemList displays all first-level gems (directly required dependencies)
	ViewGemList
	// ViewGemDetail displays forward and reverse dependencies for a selected gem
	ViewGemDetail
	// ViewSearch displays search results for a gem query
	ViewSearch
	// ViewUpgradeable displays gems with available updates, organized by type
	ViewUpgradeable
	// ViewCVE displays vulnerable gems with CVE information
	ViewCVE
	// ViewSanity displays gem sizes and project health metrics
	ViewSanity
	// ViewProjectInfo displays project metadata (Ruby version, framework, gem counts, etc.)
	ViewProjectInfo
	// ViewFilterMenu displays options to filter gems by group or upgradability
	ViewFilterMenu
	// ViewCVEFilterMenu displays options to filter CVEs by severity or direct/transitive
	ViewCVEFilterMenu
	// ViewCVEInfo displays detailed information for a selected CVE
	ViewCVEInfo
	// ViewSelectPath displays an input prompt to select a project directory
	ViewSelectPath
	// ViewError displays an error message
	ViewError
)

// ============================================================================
// Messages
// ============================================================================

type AnalysisCompleteMsg struct {
	Result          *gemfile.AnalysisResult
	Error           error
	OutdatedChecker *gemfile.OutdatedChecker
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
	Stage          string                  // "parsing", "checking-updates", "scanning-cves"
	CurrentCount   int                     // Current gems processed
	TotalCount     int                     // Total gems to process
	Percentage     int                     // 0-100
	Result         *gemfile.AnalysisResult // Accumulated results so far
	OutdatedGems   []*gemfile.GemStatus    // Updated gems with version info
	VulnerableGems []*gemfile.GemStatus    // Updated with CVE info
}

type HealthItemMsg struct {
	GemName string
	Health  *gemfile.GemHealth
	Error   error
}

type HealthCompleteMsg struct{}

type HealthRateLimitedMsg struct {
	StoppedAt string // gem name where rate limiting occurred
}

type GitHubBatchCompleteMsg struct {
	Error error
}

type OutdatedItemMsg struct {
	GemName       string
	IsOutdated    bool
	LatestVersion string
	HomepageURL   string
	Description   string
	Error         error
}

type OutdatedCompleteMsg struct{}

type CVEScanStartedMsg struct{}

type CVEProgressMsg struct {
	GemsProcessed int
	TotalGems     int
}

type CVECompleteMsg struct {
	Vulnerabilities []*gemfile.Vulnerability
	Error           error
}

type CVELoadFromCacheMsg struct {
	Vulnerabilities []*gemfile.Vulnerability
	CacheAge        time.Duration
	CacheTTL        time.Duration
}

type SanityDataMsg struct {
	GemDirPath       string
	RubyManager      string
	ProjectTotalSize int64
	GemSizes         map[string]int64
	Error            error
}

type GemInfoMsg struct {
	GemName string
	Output  string
	Error   error
}

// ============================================================================
// Model
// ============================================================================

// Model is the central BubbleTea model that manages all TUI state, including the current screen,
// gem data, navigation, filtering, async operations (health checks, outdated version checks),
// and error states. It implements the tea.Model interface and coordinates all UI updates.
type Model struct {
	// Window dimensions (updated on resize)
	Width  int
	Height int

	// CurrentView is the screen currently being displayed
	CurrentView ViewMode
	// ActiveTab persists the tab (ViewGemList, ViewUpgradeable, ViewCVE, ViewProjectInfo)
	// across navigation away and back, restoring state when returning
	ActiveTab ViewMode

	// Data
	AnalysisResult   *gemfile.AnalysisResult
	DependencyResult *gemfile.DependencyResult

	// Gem List screen state
	FirstLevelGems     []*gemfile.GemStatus
	GemListCursor      int
	GemListOffset      int
	UnfilteredGems     []*gemfile.GemStatus // All first-level gems (for filter operations)
	SelectedGroups     map[string]bool      // Groups to filter by (if empty, show all)
	ShowOnlyUpgradable bool                 // Filter to show only gems with updates
	AvailableGroups    []string             // All unique groups found in gems

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

	// Upgradeable screen state
	UpgradeableGems           []*gemfile.GemStatus
	UpgradeableFrameworkGems  []*gemfile.GemStatus
	UpgradeableTransitiveDeps []*gemfile.GemStatus // Transitive dependency gems that can be upgraded
	UpgradeableCursor         int
	UpgradeableOffset         int

	// CVE screen state
	VulnerableGems        []*gemfile.GemStatus
	CVECursor             int
	CVEOffset             int
	CVEInfoScroll         int                      // Scroll offset for CVE info modal content
	CVEVulnerabilities    []*gemfile.Vulnerability // Actual vulnerability data from OSV.dev
	UnfilteredCVEs        []*gemfile.Vulnerability // All CVEs before filtering
	CVESelectedSeverities map[string]bool          // "CRITICAL","HIGH","MODERATE","LOW" → true/false
	CVEShowOnlyDirect     bool                     // Filter to show only direct dependency CVEs
	CVEFilterMenuCursor   int                      // Position in the CVE filter menu
	LastGemsSignature     string                   // SHA256 of last scanned gems
	CVERefreshInProgress  bool                     // Is a CVE refresh happening in background?
	CVELastScanTime       time.Time                // When was CVE data last scanned?
	CVECacheLoadedAt      time.Time                // When was cache loaded?
	CVECacheTTL           time.Duration            // Default: 1 hour
	CVELastError          string                   // Last error message if scan failed

	// Sanity screen state
	GemDirPath            string           // Result of `gem env gemdir`
	RubyManager           string           // Detected Ruby version manager
	ProjectTotalSizeBytes int64            // Total size of all project gems
	GemSizes              map[string]int64 // Gem name → size in bytes
	SanityCursor          int              // Selection position in gem list
	SanityOffset          int              // Scroll offset for pagination
	ShowingGemInfo        bool             // Toggle for modal visibility
	CurrentGemInfoOutput  string           // Output from `gem info` command
	SanityLoading         bool             // Is size calculation in progress?

	// Project Info screen state
	RubyVersion        string
	RailsVersion       string
	BundleVersion      string
	OtherFramework     string // For non-Rails projects
	TotalGems          int
	FirstLevelCount    int
	TransitiveDeps     int
	FrameworkDetected  string         // The name of the framework detected
	InsecureSourceGems []*gemfile.Gem // Gems sourced from insecure protocols (http://, git://)

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

	// Health loading state
	HealthLoading     bool
	HealthRateLimited bool
	HealthLoadedCount int
	HealthTotalCount  int
	HealthPending     []*gemfile.GemStatus // Queue for sequential fetching
	HealthChecker     *gemfile.HealthChecker
	OutdatedChecker   *gemfile.OutdatedChecker // Reused for health data extraction

	// Outdated checking state
	OutdatedLoading     bool
	OutdatedPending     []*gemfile.GemStatus // Queue for sequential fetching
	OutdatedErrorCount  int
	OutdatedRateLimited bool

	// Error state
	ErrorMessage string

	// Project state
	ProjectPath     string
	GemfileLockPath string
	GemfileSource   string // "Gemfile.lock", "gems.locked", ".gemspec", etc.

	// App metadata
	Version             string
	Commit              string
	Date                string
	NewVersionAvailable string // empty = no update, otherwise holds latest version tag
	Quitting            bool
	NoCache             bool // Skip cache and force fresh analysis
	Verbose             bool // Enable verbose logging
}

// ============================================================================
// Initialization
// ============================================================================

// NewModel creates a new TUI Model and loads the project from the given path.
// If the path contains a Gemfile.lock, gems.locked, or .gemspec file, analysis starts automatically.
// The version, commit, and date are displayed in the UI header.
func NewModel(version, commit, date, projectPath string, noCache, verbose bool) *Model {
	m := &Model{
		Version:            version,
		Commit:             commit,
		Date:               date,
		CurrentView:        ViewGemList,
		ActiveTab:          ViewGemList,
		SearchInput:        textinput.New(),
		PathInput:          textinput.New(),
		SelectedGroups:     make(map[string]bool),
		NoCache:            noCache,
		Verbose:            verbose,
		HealthChecker:      gemfile.NewHealthChecker(),
		OutdatedChecker:    gemfile.NewOutdatedChecker(),
		HealthPending:      make([]*gemfile.GemStatus, 0),
		CVECacheTTL:        1 * time.Hour,
		CVEVulnerabilities: make([]*gemfile.Vulnerability, 0),
		GemSizes:           make(map[string]int64),
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

// Init implements the tea.Model interface and starts the initial command queue.
// If a Gemfile.lock or .gemspec is found, it begins project analysis automatically.
func (m *Model) Init() tea.Cmd {
	// Auto-start analysis if lock file or gemspec exists
	if _, err := os.Stat(m.GemfileLockPath); err == nil {
		// File exists, start analysis
		m.CurrentView = ViewLoading
		m.ActiveTab = ViewGemList
		m.Loading = true
		m.LoadingMessage = fmt.Sprintf("Parsing %s...", m.GemfileSource)
		m.AnalysisStage = "parsing"
		m.AnalysisPercentage = 0
		m.AnimationFrame = 0

		// If --no-cache flag is set, clear all caches to force fresh data
		if m.NoCache {
			logger.Info("--no-cache flag set, clearing all caches")
			gemfile.ClearVulnerabilityCache()    // Clear CVE cache
			cache.Clear(m.GemfileLockPath)       // Clear analysis cache
			cache.ClearHealth(m.GemfileLockPath) // Clear health cache
		}

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

	// Check if path is a file (Gemfile.lock, gems.locked, etc.) or directory
	fileInfo, err := os.Stat(absPath)
	if err == nil && !fileInfo.IsDir() {
		// It's a file - use it directly
		m.GemfileLockPath = absPath
		m.ProjectPath = filepath.Dir(absPath)
		m.GemfileSource = filepath.Base(absPath)
		logger.Info("Project loaded from explicit file: %s", m.GemfileSource)
		return
	}

	// It's a directory (or doesn't exist yet)
	m.ProjectPath = absPath

	// For gem projects: try to find a .gemspec file FIRST (it's the authoritative source)
	// This ensures we get all production dependencies, not just what's in Gemfile.lock
	files, err := os.ReadDir(m.ProjectPath)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".gemspec") {
				m.GemfileLockPath = filepath.Join(m.ProjectPath, file.Name())
				m.GemfileSource = file.Name()
				logger.Info("Project loaded from gemspec file: %s (gem project)", m.GemfileSource)
				return
			}
		}
	}

	// For Rails/Bundler projects: try to find a lock file (gems.locked or Gemfile.lock)
	lockFile := gemfile.FindLockFile(m.ProjectPath)
	if lockFile != "" {
		m.GemfileLockPath = lockFile
		m.GemfileSource = filepath.Base(lockFile)
		logger.Info("Project loaded from lock file: %s", m.GemfileSource)
		return
	}

	// Fallback to Gemfile.lock (default behavior for backward compatibility)
	m.GemfileLockPath = filepath.Join(m.ProjectPath, "Gemfile.lock")
	m.GemfileSource = "Gemfile.lock"
	logger.Info("No files found, defaulting to: %s", m.GemfileSource)
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
				// Create a fresh outdated checker so health data can be fetched
				return AnalysisCompleteMsg{
					Result:          cacheEntry.Result,
					Error:           nil,
					OutdatedChecker: gemfile.NewOutdatedChecker(),
				}
			}
		}

		// Cache miss or invalid, do full analysis
		// Determine parser based on file type
		var gf *gemfile.Gemfile
		var err error

		if strings.HasSuffix(gemfilePath, ".gemspec") {
			gf, err = gemfile.ParseGemspec(gemfilePath)
		} else {
			gf, err = gemfile.Parse(gemfilePath)
		}

		if err != nil {
			return AnalysisCompleteMsg{
				Result: nil,
				Error:  err,
			}
		}

		// Load group information from Gemfile (only for lock files, not gemspec)
		if !strings.HasSuffix(gemfilePath, ".gemspec") {
			dir := filepath.Dir(gemfilePath)
			gf.LoadGroupsFromGemfile(dir)
		}

		// Create the outdated checker once and reuse it
		outdatedChecker := gemfile.NewOutdatedChecker()

		result := gemfile.Analyze(gf)
		// Lazy load source code URIs during health fetching to keep UI responsive

		return AnalysisCompleteMsg{
			Result:          result,
			Error:           nil,
			OutdatedChecker: outdatedChecker,
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
				Result:          cacheEntry.Result,
				Error:           nil,
				OutdatedChecker: gemfile.NewOutdatedChecker(),
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

		// Warm up outdated checker for health data extraction
		outdatedChecker := gemfile.NewOutdatedChecker()
		if result != nil {
			for _, gem := range result.FirstLevelGems {
				outdatedChecker.GetSourceCodeURI(gem)
			}
		}

		// Stage 3: Return complete results (100%)
		return AnalysisCompleteMsg{
			Result:          result,
			Error:           nil,
			OutdatedChecker: outdatedChecker,
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

// applyCVEFilters applies the current CVE filter state to CVEVulnerabilities
func (m *Model) applyCVEFilters() {
	if m.UnfilteredCVEs == nil || len(m.UnfilteredCVEs) == 0 {
		return
	}

	m.CVEVulnerabilities = make([]*gemfile.Vulnerability, 0)

	for _, vuln := range m.UnfilteredCVEs {
		// Check severity filter
		if !m.CVESelectedSeverities[vuln.Severity] {
			continue
		}

		// Check direct-only filter
		if m.CVEShowOnlyDirect {
			// Check if gem is in first-level gems (direct dependency)
			isDirectDep := false
			for _, gem := range m.FirstLevelGems {
				if gem.Name == vuln.GemName {
					isDirectDep = true
					break
				}
			}
			if !isDirectDep {
				continue
			}
		}

		m.CVEVulnerabilities = append(m.CVEVulnerabilities, vuln)
	}

	// Reset cursor if out of bounds
	if m.CVECursor >= len(m.CVEVulnerabilities) {
		m.CVECursor = 0
	}
	m.CVEOffset = 0
}

// initializeCVEFilters sets up the CVE filter state when vulnerabilities are loaded
func (m *Model) initializeCVEFilters(vulns []*gemfile.Vulnerability) {
	m.UnfilteredCVEs = vulns
	m.CVEVulnerabilities = vulns
	m.CVESelectedSeverities = map[string]bool{
		"CRITICAL": true,
		"HIGH":     true,
		"MODERATE": true,
		"LOW":      true,
	}
	m.CVEShowOnlyDirect = false
	m.CVEFilterMenuCursor = 0
}

// buildUpgradeableList categorizes outdated gems into first-level, framework, and transitive dependencies
func (m *Model) buildUpgradeableList() {
	if m.AnalysisResult == nil {
		return
	}

	// Build a set of first-level gem names for quick lookup
	firstLevelSet := make(map[string]bool)
	for _, gem := range m.FirstLevelGems {
		firstLevelSet[gem.Name] = true
	}

	// Clear the upgradeable lists
	m.UpgradeableGems = make([]*gemfile.GemStatus, 0)
	m.UpgradeableFrameworkGems = make([]*gemfile.GemStatus, 0)
	m.UpgradeableTransitiveDeps = make([]*gemfile.GemStatus, 0)

	// Categorize all outdated gems
	for _, gs := range m.AnalysisResult.GemStatuses {
		if !gs.IsOutdated {
			continue
		}

		// Check if it's first-level
		if firstLevelSet[gs.Name] {
			m.UpgradeableGems = append(m.UpgradeableGems, gs)
		} else if _, isFramework := frameworkGems[gs.Name]; isFramework {
			// It's a framework gem
			m.UpgradeableFrameworkGems = append(m.UpgradeableFrameworkGems, gs)
		} else {
			// It's a transitive dependency
			m.UpgradeableTransitiveDeps = append(m.UpgradeableTransitiveDeps, gs)
		}
	}

	// Sort all slices by name
	sort.Slice(m.UpgradeableGems, func(i, j int) bool {
		return m.UpgradeableGems[i].Name < m.UpgradeableGems[j].Name
	})
	sort.Slice(m.UpgradeableFrameworkGems, func(i, j int) bool {
		return m.UpgradeableFrameworkGems[i].Name < m.UpgradeableFrameworkGems[j].Name
	})
	sort.Slice(m.UpgradeableTransitiveDeps, func(i, j int) bool {
		return m.UpgradeableTransitiveDeps[i].Name < m.UpgradeableTransitiveDeps[j].Name
	})

	// Reset cursor if out of bounds
	if m.UpgradeableCursor >= len(m.allUpgradeableGems()) {
		m.UpgradeableCursor = 0
	}
	m.UpgradeableOffset = 0
}

// allUpgradeableGems returns a combined slice of all upgradeable gems (first-level + framework + transitive)
func (m *Model) allUpgradeableGems() []*gemfile.GemStatus {
	all := append(m.UpgradeableGems, m.UpgradeableFrameworkGems...)
	return append(all, m.UpgradeableTransitiveDeps...)
}

// ============================================================================
// Health Data Loading
// ============================================================================

func fetchSingleHealth(gem *gemfile.GemStatus, hc *gemfile.HealthChecker, outdatedChecker *gemfile.OutdatedChecker) tea.Msg {
	sourceCodeURI := outdatedChecker.GetSourceCodeURI(gem.Name)
	homepageURI := outdatedChecker.GetHomepage(gem.Name)
	versionCreatedAt := outdatedChecker.GetVersionCreatedAt(gem.Name)
	ownersURL := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s/owners.json", gem.Name)

	health, err := hc.FetchHealth(gem.Name, sourceCodeURI, homepageURI, versionCreatedAt, ownersURL)

	if err != nil && isRateLimited(err) {
		return HealthRateLimitedMsg{StoppedAt: gem.Name}
	}

	return HealthItemMsg{GemName: gem.Name, Health: health, Error: err}
}

func fetchNextHealthItem(gems []*gemfile.GemStatus, hc *gemfile.HealthChecker, outdatedChecker *gemfile.OutdatedChecker) tea.Cmd {
	if len(gems) == 0 {
		return func() tea.Msg { return HealthCompleteMsg{} }
	}
	return func() tea.Msg {
		gem := gems[0]
		return fetchSingleHealth(gem, hc, outdatedChecker)
	}
}

// fetchGitHubBatchHealth collects all repo owner/repo pairs and fetches them in a single GraphQL batch
// Runs async in a goroutine to avoid blocking the TUI
func fetchGitHubBatchHealth(gems []*gemfile.GemStatus, oc *gemfile.OutdatedChecker, hc *gemfile.HealthChecker) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			// Immediately return a message to unblock UI
			return GitHubBatchCompleteMsg{Error: nil}
		},
		func() tea.Msg {
			// Run the actual batch fetch in background
			// Collect all unique (owner, repo) pairs from gems
			pairs := make([]gemfile.RepoOwnerPair, 0)
			seenPairs := make(map[string]bool)

			for _, gem := range gems {
				sourceCodeURI := oc.GetSourceCodeURI(gem.Name)
				homepageURI := oc.GetHomepage(gem.Name)

				githubURI := sourceCodeURI
				if githubURI == "" {
					githubURI = homepageURI
				}

				if githubURI != "" {
					owner, repo, ok := gemfile.ExtractGitHubOwnerRepo(githubURI)
					if ok {
						key := strings.ToLower(owner + "/" + repo)
						if !seenPairs[key] {
							seenPairs[key] = true
							pairs = append(pairs, gemfile.RepoOwnerPair{
								GemName: gem.Name,
								Owner:   owner,
								Repo:    repo,
							})
						}
					}
				}
			}

			// Fetch all GitHub data in batch
			// If no GITHUB_TOKEN, this returns immediately
			// If GITHUB_TOKEN is set, this makes the GraphQL call and caches results
			_ = hc.FetchGitHubBatch(pairs)

			// Return completion message after fetch is done
			return GitHubBatchCompleteMsg{Error: nil}
		},
	)
}

func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "429")
}

func fetchNextOutdatedItem(gems []*gemfile.GemStatus, checker *gemfile.OutdatedChecker) tea.Cmd {
	if len(gems) == 0 {
		return func() tea.Msg { return OutdatedCompleteMsg{} }
	}
	return func() tea.Msg {
		gem := gems[0]
		isOutdated, latest, err := checker.IsOutdated(gem.Name, gem.Version)
		if err != nil {
			return OutdatedItemMsg{GemName: gem.Name, Error: err}
		}
		homepage := checker.GetHomepage(gem.Name)
		desc := checker.GetDescription(gem.Name)
		return OutdatedItemMsg{
			GemName:       gem.Name,
			IsOutdated:    isOutdated,
			LatestVersion: latest,
			HomepageURL:   homepage,
			Description:   desc,
		}
	}
}

// performCVEScan queries OSV.dev for vulnerabilities in the project's gems
// Returns cached data if available and not expired, otherwise fetches fresh data
func performCVEScan(gems []*gemfile.Gem) tea.Cmd {
	return func() tea.Msg {
		if gems == nil || len(gems) == 0 {
			logger.Info("CVE scan skipped: no gems to scan")
			return CVECompleteMsg{Vulnerabilities: []*gemfile.Vulnerability{}, Error: nil}
		}

		logger.Info("Starting CVE scan for %d gems", len(gems))

		// Compute gems signature for cache key
		gemsSignature := gemfile.ComputeGemsSignature(gems)
		logger.Info("Gems signature for cache: %s", gemsSignature)

		// Try to load from cache first
		logger.Info("Attempting to load CVE data from cache...")
		cacheEntry, err := gemfile.LoadVulnerabilityCache(gemsSignature)
		if err != nil {
			logger.Warn("Error loading cache: %v", err)
		}

		if cacheEntry != nil && gemfile.IsCacheValid(cacheEntry) {
			// Cache hit! Return cached data
			cacheAge := gemfile.GetCacheAge(cacheEntry)
			cacheTTL := time.Duration(cacheEntry.TTLSeconds) * time.Second

			logger.Info("CVE cache HIT: returning %d cached vulnerabilities (age: %v, TTL: %v)",
				len(cacheEntry.Vulnerabilities), cacheAge.Round(time.Second), cacheTTL)

			// Convert vulnerabilities to pointers
			vulnPtrs := make([]*gemfile.Vulnerability, len(cacheEntry.Vulnerabilities))
			for i := range cacheEntry.Vulnerabilities {
				vulnPtrs[i] = &cacheEntry.Vulnerabilities[i]
			}

			return CVELoadFromCacheMsg{
				Vulnerabilities: vulnPtrs,
				CacheAge:        cacheAge,
				CacheTTL:        cacheTTL,
			}
		}

		// Cache miss or expired, fetch from OSV.dev
		logger.Info("CVE cache MISS or expired, fetching from OSV.dev...")
		osv := gemfile.NewOSVClient()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		vulns, err := osv.QueryBatch(ctx, gems)
		if err != nil {
			logger.Error("CVE scan failed: %v", err)
			return CVECompleteMsg{Vulnerabilities: nil, Error: err}
		}

		logger.Info("CVE scan complete: found %d vulnerabilities", len(vulns))

		// Convert vulnerabilities to pointers for return
		vulnPtrs := make([]*gemfile.Vulnerability, len(vulns))
		for i := range vulns {
			vulnPtrs[i] = &vulns[i]
		}

		// Save to cache
		cacheEntry = &gemfile.CacheEntry{
			GemsSignature:   gemsSignature,
			CachedAt:        time.Now(),
			ScannedAt:       time.Now(),
			TTLSeconds:      int(gemfile.VulnerabilityCacheTTL.Seconds()),
			GemCount:        len(gems),
			ScanStatus:      "success",
			Vulnerabilities: vulns,
		}

		if err := gemfile.SaveVulnerabilityCache(gemsSignature, cacheEntry); err != nil {
			// Log error but don't fail - we still have the fresh data
			logger.Warn("Failed to save CVE cache: %v", err)
		}

		return CVECompleteMsg{Vulnerabilities: vulnPtrs, Error: nil}
	}
}

func loadSanityData(gems []*gemfile.Gem) tea.Cmd {
	return func() tea.Msg {
		// Get gem directory path
		gemDirPath, err := gemfile.GetGemDirPath()
		if err != nil {
			return SanityDataMsg{Error: err}
		}

		// Detect Ruby manager from path
		rubyManager := gemfile.DetectRubyManager(gemDirPath)

		// Calculate project size
		totalSize, sizes, err := gemfile.CalculateProjectSize(gems, gemDirPath)
		if err != nil {
			// Don't fail on error, but log it
			logger.Warn("Failed to calculate gem sizes: %v", err)
		}

		return SanityDataMsg{
			GemDirPath:       gemDirPath,
			RubyManager:      rubyManager,
			ProjectTotalSize: totalSize,
			GemSizes:         sizes,
			Error:            nil,
		}
	}
}

func fetchGemInfo(gemName string) tea.Cmd {
	return func() tea.Msg {
		output, err := gemfile.GetGemInfo(gemName)
		return GemInfoMsg{
			GemName: gemName,
			Output:  output,
			Error:   err,
		}
	}
}

