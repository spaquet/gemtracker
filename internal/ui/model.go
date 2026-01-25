package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewMode string

const (
	ViewMain       ViewMode = "main"
	ViewAnalyzing  ViewMode = "analyzing"
	ViewResults    ViewMode = "results"
	ViewHelp       ViewMode = "help"
	ViewError      ViewMode = "error"
)

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
	CurrentView ViewMode
	Commands    []Command
	CommandList list.Model
	SearchInput textinput.Model

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
		CurrentMessage: "Ready",
	}

	m.SearchInput.Placeholder = "Type to search commands..."
	m.SearchInput.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	m.SearchInput.TextStyle = lipgloss.NewStyle().Foreground(ColorPrimary)

	m.initializeCommands()
	m.setupCommandList()

	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) initializeCommands() {
	m.Commands = []Command{
		{
			Name:        "analyze",
			Description: "Analyze Gemfile.lock for risks and conflicts",
			Execute: func(m *Model) tea.Cmd {
				m.CurrentMessage = "Analyzing Gemfile.lock..."
				m.CurrentView = ViewAnalyzing
				return nil
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

