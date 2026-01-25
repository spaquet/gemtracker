package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.Quitting {
		return ""
	}

	switch m.CurrentView {
	case ViewMain:
		return m.viewMain()
	case ViewAnalyzing:
		return m.viewAnalyzing()
	case ViewResults:
		return m.viewResults()
	case ViewHelp:
		return m.viewHelp()
	case ViewError:
		return m.viewError()
	default:
		return m.viewMain()
	}
}

func (m *Model) viewMain() string {
	header := m.renderHeader()
	commandList := m.renderCommandList()
	searchInput := m.renderSearchInput()
	footer := m.renderFooter()

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		commandList,
		searchInput,
		footer,
	)

	return content
}

func (m *Model) renderHeader() string {
	title := "💎 gemtracker"
	version := fmt.Sprintf("v%s", m.Version)
	subtitle := "Ruby Gem Dependency Analyzer"

	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		TitleStyle.Render(title),
		lipgloss.NewStyle().Margin(0, 1).Render(version),
	)

	projectInfo := ""
	if m.GemfileLockPath != "" {
		stats := fmt.Sprintf(
			"📁 Project: %s  |  📦 Gems: %d  |  ⚠️  Outdated: %d  |  🔒 Vulnerable: %d",
			m.ProjectPath,
			m.GemCount,
			m.OutdatedCount,
			m.VulnerableCount,
		)
		lastScan := ""
		if m.LastScanTime != nil {
			lastScan = fmt.Sprintf("  |  🕐 Last scan: %s ago", timeAgo(*m.LastScanTime))
		}

		projectInfo = StatusStyle.Render(stats + lastScan)
	} else {
		projectInfo = StatusStyle.Render("📁 No Gemfile.lock found in current directory")
	}

	header := lipgloss.JoinVertical(
		lipgloss.Top,
		titleLine,
		SubtitleStyle.Render(subtitle),
		"",
		projectInfo,
	)

	return HeaderStyle.Render(header)
}

func (m *Model) renderCommandList() string {
	m.CommandList.SetSize(m.Width-4, 6)
	commandsView := m.CommandList.View()
	return CommandListStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			"Available Commands",
			commandsView,
		),
	)
}

func (m *Model) renderSearchInput() string {
	inputView := m.SearchInput.View()
	return SearchInputStyle.Render("Search: " + inputView)
}

func (m *Model) renderFooter() string {
	keys := "↑/↓: navigate  •  Enter: run  •  Esc: clear  •  q: quit"
	return HelpStyle.Render(keys)
}

func (m *Model) viewAnalyzing() string {
	header := m.renderHeader()
	message := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3")).
		Align(lipgloss.Center).
		Width(m.Width - 4).
		Padding(2, 0).
		Render("🔄 " + m.CurrentMessage)

	backPrompt := HelpStyle.Render("Press Enter to return to main menu")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		"",
		message,
		"",
		backPrompt,
	)
}

func (m *Model) viewResults() string {
	header := m.renderHeader()
	message := lipgloss.NewStyle().
		Padding(2, 2).
		Render(m.CurrentMessage)

	backPrompt := HelpStyle.Render("Press Enter to return to main menu")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		"",
		message,
		"",
		backPrompt,
	)
}

func (m *Model) viewHelp() string {
	header := m.renderHeader()

	helpText := `
COMMANDS:

  analyze          - Analyze Gemfile.lock for risks and dependency conflicts
  deps             - Show which parent gems are using a specific gem
  vulnerabilities  - Check for known vulnerabilities in your gems
  licenses         - Generate license compliance report
  help             - Show this help message
  quit             - Exit gemtracker

KEYBOARD SHORTCUTS:

  ↑/↓, Tab        - Navigate commands
  Enter           - Run selected command
  Esc             - Clear search / return to menu
  q, Ctrl+C       - Quit gemtracker

FEATURES:

  • Analyze gem dependencies and identify risks
  • Detect outdated and vulnerable gem versions
  • Check license compatibility
  • Detect version conflicts
  • Interactive command palette interface
  • Fast analysis with beautiful terminal output
`

	content := lipgloss.NewStyle().
		Padding(1, 2).
		Render(helpText)

	backPrompt := HelpStyle.Render("Press Enter to return to main menu")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		"",
		content,
		"",
		backPrompt,
	)
}

func (m *Model) viewError() string {
	header := m.renderHeader()

	message := ErrorStyle.Render("❌ " + m.ErrorMessage)
	backPrompt := HelpStyle.Render("Press Enter to return to main menu")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		"",
		message,
		"",
		backPrompt,
	)
}

func timeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration.Seconds() < 60 {
		return fmt.Sprintf("%.0f seconds", duration.Seconds())
	}
	if duration.Minutes() < 60 {
		return fmt.Sprintf("%.0f minutes", duration.Minutes())
	}
	if duration.Hours() < 24 {
		return fmt.Sprintf("%.0f hours", duration.Hours())
	}
	return fmt.Sprintf("%.0f days", duration.Hours()/24)
}
