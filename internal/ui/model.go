package ui

import (
	"os"
	"path/filepath"

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
	AnalysisResult  *gemfile.AnalysisResult
	DependencyResult *gemfile.DependencyResult

	// Gem List screen state
	FirstLevelGems []*gemfile.GemStatus
	GemListCursor  int
	GemListOffset  int

	// Gem Detail screen state
	SelectedGem   *gemfile.GemStatus
	DetailSection int // 0 = forward deps, 1 = reverse deps
	DetailOffset  int

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

func NewModel(version, commit, date string) *Model {
	m := &Model{
		Version:        version,
		Commit:         commit,
		Date:           date,
		CurrentView:    ViewGemList,
		ActiveTab:      ViewGemList,
		ProjectPath:    "./",
		GemfileLockPath: "./Gemfile.lock",
		SearchInput:    textinput.New(),
		PathInput:      textinput.New(),
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

	return m
}

func (m *Model) Init() tea.Cmd {
	// Start by showing path selection if needed
	if m.ProjectPath == "./" {
		return nil
	}
	return nil
}

// ============================================================================
// Project Loading
// ============================================================================

func (m *Model) loadProject(path string) {
	if path == "~" || path == "." {
		if path == "." {
			m.ProjectPath = "./"
			m.GemfileLockPath = "./Gemfile.lock"
		} else {
			m.ProjectPath = "~/"
			m.GemfileLockPath = "~/Gemfile.lock"
		}
	} else if len(path) > 0 && path[0] == '~' {
		home := os.Getenv("HOME")
		path = home + path[1:]
		m.ProjectPath = path
		m.GemfileLockPath = path + "/Gemfile.lock"
	} else {
		m.ProjectPath = path
		m.GemfileLockPath = path + "/Gemfile.lock"
	}
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
