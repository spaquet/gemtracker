package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spaquet/gemtracker/internal/gemfile"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m2, cmd := m.handleKeypress(msg)
		return m2, cmd
	case tea.WindowSizeMsg:
		m2, cmd := m.handleWindowSize(msg)
		return m2, cmd
	case AnalysisCompleteMsg:
		if msg.Error != nil {
			m.CurrentView = ViewError
			m.ErrorMessage = msg.Error.Error()
		} else {
			m.AnalysisResult = msg.Result
			m.populateGemsList(msg.Result)
			m.CurrentView = ViewResultsList
			// Don't call Focus() as it can cause nil pointer panic
			// Input will accept keystrokes naturally when view is active
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) handleKeypress(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.Quitting = true
		return m, tea.Quit

	case "enter":
		switch m.CurrentView {
		case ViewMain:
			if m.ShowDropdown && m.FilteredIndex >= 0 && m.FilteredIndex < len(m.Commands) {
				return m.executeSelectedCommand()
			}
		case ViewResults, ViewHelp, ViewError:
			m.CurrentView = ViewMain
			m.CurrentMessage = "Ready"
			m.ErrorMessage = ""
		case ViewResultsList:
			m.CurrentView = ViewMain
			m.ResultsFilter.Reset()
		case ViewSelectPath:
			path := m.PathInput.Value()
			if path != "" {
				m.loadProject(path)
				m.CurrentView = ViewMain
				m.PathInput.Reset()
			}
		}

	case "esc":
		if m.ShowDropdown {
			m.ShowDropdown = false
			m.SearchInput.Reset()
		} else if m.CurrentView == ViewSelectPath {
			m.CurrentView = ViewMain
			m.PathInput.Reset()
		} else if m.CurrentView == ViewResultsList {
			if m.ResultsFilter.Value() != "" {
				m.ResultsFilter.Reset()
				m.filterGems("")
			} else {
				m.CurrentView = ViewMain
			}
		} else if m.CurrentView != ViewMain {
			m.CurrentView = ViewMain
		}

	case "up", "shift+tab":
		if m.CurrentView == ViewMain && m.ShowDropdown {
			if m.FilteredIndex > 0 {
				m.FilteredIndex--
			} else {
				m.FilteredIndex = len(m.Commands) - 1
			}
		} else if m.CurrentView == ViewResultsList {
			// Navigate gem list
			if m.GemsList.Index() > 0 {
				m.GemsList.CursorUp()
			}
		}

	case "down", "tab":
		if m.CurrentView == ViewMain && m.ShowDropdown {
			if m.FilteredIndex < len(m.Commands)-1 {
				m.FilteredIndex++
			} else {
				m.FilteredIndex = 0
			}
		} else if m.CurrentView == ViewResultsList {
			// Navigate gem list
			if m.GemsList.Index() < len(m.FilteredGems)-1 {
				m.GemsList.CursorDown()
			}
		}

	default:
		if m.CurrentView == ViewMain {
			oldValue := m.SearchInput.Value()
			var cmd tea.Cmd
			m.SearchInput, cmd = m.SearchInput.Update(msg)
			newValue := m.SearchInput.Value()

			// Show dropdown when user starts typing
			if newValue != "" && !m.ShowDropdown {
				m.ShowDropdown = true
				m.FilteredIndex = 0
			}

			// Hide dropdown when input is cleared
			if newValue == "" && m.ShowDropdown {
				m.ShowDropdown = false
			}

			if newValue != oldValue {
				m.filterCommands(newValue)
			}

			return m, cmd
		} else if m.CurrentView == ViewSelectPath {
			var cmd tea.Cmd
			m.PathInput, cmd = m.PathInput.Update(msg)
			return m, cmd
		} else if m.CurrentView == ViewResultsList {
			oldValue := m.ResultsFilter.Value()
			var cmd tea.Cmd
			m.ResultsFilter, cmd = m.ResultsFilter.Update(msg)
			newValue := m.ResultsFilter.Value()

			if newValue != oldValue {
				m.filterGems(newValue)
			}

			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) (*Model, tea.Cmd) {
	m.Width = msg.Width
	m.Height = msg.Height

	headerHeight := 8
	commandListHeight := (m.Height - headerHeight) / 2
	if commandListHeight < 3 {
		commandListHeight = 3
	}

	m.CommandList.SetSize(m.Width-4, commandListHeight)

	// Also update gems list size
	listHeight := m.Height - 16
	if listHeight < 5 {
		listHeight = 5
	}
	m.GemsList.SetSize(m.Width-4, listHeight)

	return m, nil
}

func (m *Model) updateCommandList() {
	items := make([]list.Item, 0, len(m.Commands))
	for _, cmd := range m.Commands {
		items = append(items, commandItem{
			name:        cmd.Name,
			description: cmd.Description,
		})
	}
	m.CommandList.SetItems(items)
}

func (m *Model) filterCommands(query string) {
	// For now, just keep all commands visible
	// TODO: Implement search filtering
	items := make([]list.Item, 0, len(m.Commands))
	for _, cmd := range m.Commands {
		items = append(items, commandItem{
			name:        cmd.Name,
			description: cmd.Description,
		})
	}
	m.CommandList.SetItems(items)
}

func (m *Model) populateGemsList(result *gemfile.AnalysisResult) {
	// Convert gems to gemItem for display
	m.AllGems = make([]gemItem, 0, len(result.AllGems))
	for _, gem := range result.AllGems {
		status := "✓"
		// Mark as outdated if in the list
		for _, outdated := range result.OutdatedGems {
			if outdated == gem.Name {
				status = "⚠"
				break
			}
		}

		m.AllGems = append(m.AllGems, gemItem{
			Name:    gem.Name,
			Version: gem.Version,
			Status:  status,
		})
	}

	m.FilteredGems = m.AllGems
	m.updateGemsListItems()
}

func (m *Model) updateGemsListItems() {
	items := make([]list.Item, 0, len(m.FilteredGems))
	for _, gem := range m.FilteredGems {
		items = append(items, gem)
	}
	m.GemsList.SetItems(items)
}

func (m *Model) filterGems(query string) {
	if query == "" {
		m.FilteredGems = m.AllGems
	} else {
		m.FilteredGems = []gemItem{}
		for _, gem := range m.AllGems {
			if strings.Contains(strings.ToLower(gem.Name), strings.ToLower(query)) {
				m.FilteredGems = append(m.FilteredGems, gem)
			}
		}
	}
	m.updateGemsListItems()
}

func (m *Model) executeSelectedCommand() (*Model, tea.Cmd) {
	if m.FilteredIndex < 0 || m.FilteredIndex >= len(m.Commands) {
		return m, nil
	}

	m.ShowDropdown = false
	m.SearchInput.Reset()

	cmd := m.Commands[m.FilteredIndex]
	return m, cmd.Execute(m)
}
