package ui

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spaquet/gemtracker/internal/gemfile"
)

type AnalysisCompleteMsg struct {
	Result *gemfile.AnalysisResult
	Error  error
}

type ViewMode string

const (
	ViewMain       ViewMode = "main"
	ViewAnalyzing  ViewMode = "analyzing"
	ViewResults    ViewMode = "results"
	ViewResultsList ViewMode = "results_list"
	ViewHelp       ViewMode = "help"
	ViewError      ViewMode = "error"
	ViewSelectPath ViewMode = "select_path"
)

type gemItem struct {
	Name    string
	Version string
	Status  string
}

func (g gemItem) FilterValue() string { return g.Name }

type gemDelegate struct{}

func (d gemDelegate) Height() int                             { return 1 }
func (d gemDelegate) Spacing() int                            { return 0 }
func (d gemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d gemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i := item.(gemItem)
	str := fmt.Sprintf("%s %-30s  v%s", i.Status, i.Name, i.Version)

	var output string
	if index == m.Index() {
		output = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Background(lipgloss.Color("237")).
			Render("> " + str)
	} else {
		output = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("  " + str)
	}

	fmt.Fprint(w, output)
}

type Command struct {
	Name        string
	Description string
	Execute     func(*Model) tea.Cmd
}

type Model struct {
	// Window dimensions
	Width  int
	Height int

	// UI state
	CurrentView    ViewMode
	Commands       []Command
	CommandList    list.Model
	SearchInput    textinput.Model
	PathInput      textinput.Model
	ResultsFilter  textinput.Model
	GemsList       list.Model
	AllGems        []gemItem
	FilteredGems   []gemItem
	ShowDropdown   bool
	FilteredIndex  int

	// Navigation
	Cursor int

	// Project state
	ProjectPath      string
	GemfileLockPath  string
	GemCount         int
	OutdatedCount    int
	VulnerableCount  int
	LastScanTime     *time.Time
	CurrentMessage   string
	ErrorMessage     string
	AnalysisResult   interface{} // Will hold *gemfile.AnalysisResult

	// App metadata
	Version string
	Commit  string
	Date    string

	// Flag parsing
	ShowHelp    bool
	ShowVersion bool
	Quitting    bool
}

func NewModel(version, commit, date string) *Model {
	m := &Model{
		Version:        version,
		Commit:         commit,
		Date:           date,
		CurrentView:    ViewMain,
		Cursor:         0,
		SearchInput:    textinput.New(),
		PathInput:      textinput.New(),
		CurrentMessage: "Ready",
	}

	m.SearchInput.Placeholder = "Search commands..."
	m.SearchInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	m.SearchInput.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	m.SearchInput.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	m.SearchInput.Cursor.Style = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	m.SearchInput.Focus()

	m.PathInput.Placeholder = "/path/to/project"
	m.PathInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	m.PathInput.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	m.PathInput.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	m.PathInput.Cursor.Style = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	m.ResultsFilter.Placeholder = "Type gem name to search..."
	m.ResultsFilter.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	m.ResultsFilter.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	m.ResultsFilter.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	m.ResultsFilter.Cursor.Style = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	m.ResultsFilter.CharLimit = 50

	// Initialize gems list with custom delegate
	m.GemsList = list.New([]list.Item{}, gemDelegate{}, 0, 0)
	m.GemsList.SetShowTitle(false)
	m.GemsList.SetShowHelp(false)

	m.initializeCommands()
	m.setupCommandList()

	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) loadProject(path string) {
	// Expand ~ to home directory
	if path == "~" || path == "." {
		if path == "." {
			// Current directory
			m.ProjectPath = "./"
			m.GemfileLockPath = "./Gemfile.lock"
		} else {
			// Home directory
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

func performAnalysis(gemfilePath string) tea.Cmd {
	return func() tea.Msg {
		gf, err := gemfile.Parse(gemfilePath)
		if err != nil {
			return AnalysisCompleteMsg{
				Result: nil,
				Error:  err,
			}
		}

		result := gemfile.Analyze(gf)
		return AnalysisCompleteMsg{
			Result: result,
			Error:  nil,
		}
	}
}

func (m *Model) initializeCommands() {
	m.Commands = []Command{
		{
			Name:        "open",
			Description: "Open a different Ruby project",
			Execute: func(m *Model) tea.Cmd {
				m.CurrentView = ViewSelectPath
				m.PathInput.Reset()
				m.PathInput.Focus()
				return nil
			},
		},
		{
			Name:        "analyze",
			Description: "Analyze Gemfile.lock for risks and conflicts",
			Execute: func(m *Model) tea.Cmd {
				m.CurrentView = ViewAnalyzing
				m.CurrentMessage = "Analyzing Gemfile.lock..."
				return performAnalysis(m.GemfileLockPath)
			},
		},
		{
			Name:        "deps",
			Description: "Show dependency tree for a gem",
			Execute: func(m *Model) tea.Cmd {
				m.CurrentMessage = "Show dependency tree (coming soon)"
				m.CurrentView = ViewResults
				return nil
			},
		},
		{
			Name:        "vulnerabilities",
			Description: "Check for known vulnerabilities",
			Execute: func(m *Model) tea.Cmd {
				m.CurrentMessage = "Checking for vulnerabilities (coming soon)"
				m.CurrentView = ViewResults
				return nil
			},
		},
		{
			Name:        "licenses",
			Description: "Generate license compliance report",
			Execute: func(m *Model) tea.Cmd {
				m.CurrentMessage = "Generating license report (coming soon)"
				m.CurrentView = ViewResults
				return nil
			},
		},
		{
			Name:        "help",
			Description: "Show detailed help",
			Execute: func(m *Model) tea.Cmd {
				m.CurrentView = ViewHelp
				return nil
			},
		},
		{
			Name:        "quit",
			Description: "Exit gemtracker",
			Execute: func(m *Model) tea.Cmd {
				m.Quitting = true
				return tea.Quit
			},
		},
	}
}

func (m *Model) setupCommandList() {
	items := make([]list.Item, 0, len(m.Commands))
	for _, cmd := range m.Commands {
		items = append(items, commandItem{
			name:        cmd.Name,
			description: cmd.Description,
		})
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	m.CommandList = list.New(items, delegate, 0, 0)
	m.CommandList.SetShowTitle(false)
	m.CommandList.SetShowHelp(false)
}

type commandItem struct {
	name        string
	description string
}

func (i commandItem) FilterValue() string { return i.name }

