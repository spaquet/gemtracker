package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m2, cmd := m.handleKeypress(msg)
		return m2, cmd
	case tea.WindowSizeMsg:
		m2, cmd := m.handleWindowSize(msg)
		return m2, cmd
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
		}

	case "esc":
		if m.ShowDropdown {
			m.ShowDropdown = false
			m.SearchInput.Reset()
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
