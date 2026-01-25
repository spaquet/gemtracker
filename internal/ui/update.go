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
			return m.executeSelectedCommand()
		case ViewResults, ViewHelp, ViewError:
			m.CurrentView = ViewMain
			m.CurrentMessage = "Ready"
			m.ErrorMessage = ""
		}

	case "esc":
		m.SearchInput.Reset()
		if m.CurrentView != ViewMain {
			m.CurrentView = ViewMain
		}

	case "up", "shift+tab":
		if m.CurrentView == ViewMain {
			if m.Cursor > 0 {
				m.Cursor--
			} else {
				m.Cursor = len(m.Commands) - 1
			}
			m.updateCommandList()
		}

	case "down", "tab":
		if m.CurrentView == ViewMain {
			if m.Cursor < len(m.Commands)-1 {
				m.Cursor++
			} else {
				m.Cursor = 0
			}
			m.updateCommandList()
		}

	default:
		if m.CurrentView == ViewMain {
			var cmd tea.Cmd
			m.SearchInput, cmd = m.SearchInput.Update(msg)
			m.filterCommands(m.SearchInput.Value())
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
	if m.Cursor < 0 || m.Cursor >= len(m.Commands) {
		return m, nil
	}

	cmd := m.Commands[m.Cursor]
	return m, cmd.Execute(m)
}
