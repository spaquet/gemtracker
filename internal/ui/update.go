package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spaquet/gemtracker/internal/gemfile"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AnimationTickMsg:
		// Update animation frame for spinner
		m.AnimationFrame = (m.AnimationFrame + 1) % 4
		// Return a command to send the next tick
		return m, tea.Tick(time.Millisecond*200, func(time.Time) tea.Msg {
			return AnimationTickMsg{}
		})
	case tea.KeyMsg:
		if m.CurrentView == ViewFilterInput {
			switch msg.String() {
			case "esc":
				m.CurrentView = ViewMain
				m.FilterInput.Reset()
				m.FilteredGems = nil
				m.SelectedGemIdx = 0
				m.ScrollOffset = 0
				return m, nil
			case "up":
				// Move selection up
				if m.SelectedGemIdx > 0 {
					m.SelectedGemIdx--
					// Scroll viewport if needed
					if m.SelectedGemIdx < m.ScrollOffset {
						m.ScrollOffset = m.SelectedGemIdx
					}
				}
				return m, nil
			case "down":
				// Move selection down
				if m.SelectedGemIdx < len(m.FilteredGems)-1 {
					m.SelectedGemIdx++
					// Scroll viewport if needed
					visibleLines := m.Height - 12 // Approximate lines for gem list
					if m.SelectedGemIdx >= m.ScrollOffset+visibleLines {
						m.ScrollOffset = m.SelectedGemIdx - visibleLines + 1
					}
				}
				return m, nil
			default:
				// Update filter input for typing
				var cmd tea.Cmd
				m.FilterInput, cmd = m.FilterInput.Update(msg)
				m.updateGemListFilter()
				return m, cmd
			}
		}

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
			m.CurrentView = ViewFilterInput
			m.FilterInput.Reset()
			m.SelectedGemIdx = 0
			m.ScrollOffset = 0
			m.updateGemListFilter()
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) updateGemListFilter() {
	result, ok := m.AnalysisResult.(*gemfile.AnalysisResult)
	if !ok {
		return
	}

	m.CurrentMessage = result.Summary

	filterTerm := strings.ToLower(m.FilterInput.Value())
	filtered := make([]*gemfile.GemStatus, 0)

	for _, gemStatus := range result.GemStatuses {
		// Filter by search term
		if filterTerm != "" && !strings.Contains(strings.ToLower(gemStatus.Name), filterTerm) {
			continue
		}
		filtered = append(filtered, gemStatus)
	}

	m.FilteredGems = filtered
	m.SelectedGemIdx = 0
	m.ScrollOffset = 0
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
		}

	case "down", "tab":
		if m.CurrentView == ViewMain && m.ShowDropdown {
			if m.FilteredIndex < len(m.Commands)-1 {
				m.FilteredIndex++
			} else {
				m.FilteredIndex = 0
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


func (m *Model) executeSelectedCommand() (*Model, tea.Cmd) {
	if m.FilteredIndex < 0 || m.FilteredIndex >= len(m.Commands) {
		return m, nil
	}

	m.ShowDropdown = false
	m.SearchInput.Reset()

	cmd := m.Commands[m.FilteredIndex]
	return m, cmd.Execute(m)
}
