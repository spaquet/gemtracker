package ui

import (
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	FirstLevelGems []*gemfile.GemStatus
	GemListCursor  int
	GemListOffset  int

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

	// Path selection modal
	PathInput textinput.Model

	// Loading state
	Loading        bool
	LoadingMessage string
	AnimationFrame int

	// Error state
	ErrorMessage string

	// Project state
	ProjectPath     string
	GemfileLockPath string

	// App metadata
	Version  string
	Commit   string
	Date     string
	Quitting bool
}

// ============================================================================
// Initialization
// ============================================================================

func NewModel(version, commit, date, projectPath string) *Model {
	m := &Model{
		Version:     version,
		Commit:      commit,
		Date:        date,
		CurrentView: ViewGemList,
		ActiveTab:   ViewGemList,
		SearchInput: textinput.New(),
		PathInput:   textinput.New(),
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
		m.LoadingMessage = "Analyzing Gemfile.lock..."
		m.AnimationFrame = 0

		return tea.Batch(
			tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
				return SpinnerTickMsg{}
			}),
			performAnalysis(m.GemfileLockPath),
		)
	}

	// File doesn't exist, show path selection
	m.CurrentView = ViewSelectPath
	m.PathInput.Focus()
	return nil
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

func performAnalysis(gemfilePath string) tea.Cmd {
	return func() tea.Msg {
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
