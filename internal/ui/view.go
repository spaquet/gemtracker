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
	searchInput := m.renderSearchInput()

	var dropdown string
	if m.ShowDropdown {
		dropdown = m.renderDropdown()
	}

	footer := m.renderFooter()

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		"",
		searchInput,
		dropdown,
		footer,
	)

	return content
}

func (m *Model) renderHeader() string {
	// Top bar
	topBar := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 2).
		Render(fmt.Sprintf("— gemtracker  v%s —", m.Version))

	// Left column: Diamond + Info
	diamond := `   _________
_ /_|_____|_\ _
  '. \   / .'
    '.\ /.'
      '.'`

	diamondStyled := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Align(lipgloss.Center).
		MarginRight(3).
		Render(diamond)

	// Project info text
	welcome := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Render("Welcome to gemtracker!")

	projectPath := "No Gemfile.lock found"
	if m.GemfileLockPath != "" {
		projectPath = m.ProjectPath
	}

	projectInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render(projectPath)

	infoText := lipgloss.JoinVertical(
		lipgloss.Top,
		welcome,
		projectInfo,
	)

	// Left section: Diamond + Info
	leftSection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		diamondStyled,
		infoText,
	)

	// Right section: Tips
	tipsHeader := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render("Tips for getting started")

	tips := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render(`Use arrow keys to navigate
Press Enter to run commands
Type 'q' to quit anytime

Try 'analyze' to scan your
gems for vulnerabilities!`)

	rightSection := lipgloss.JoinVertical(
		lipgloss.Top,
		tipsHeader,
		tips,
	)

	// Combine left and right
	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftSection,
		lipgloss.NewStyle().Width(4).Render(""),
		rightSection,
	)

	// Full header with border
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(
			lipgloss.Top,
			topBar,
			headerContent,
		))

	return headerBox
}

func (m *Model) renderSearchInput() string {
	inputView := m.SearchInput.View()

	inputBox := lipgloss.NewStyle().
		Width(m.Width - 6).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Render(inputView)

	return lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		Render(inputBox)
}

func (m *Model) renderDropdown() string {
	if !m.ShowDropdown || len(m.Commands) == 0 {
		return ""
	}

	var items []string
	for i, cmd := range m.Commands {
		if i == m.FilteredIndex {
			// Selected item
			item := lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Background(lipgloss.Color("237")).
				Width(m.Width - 8).
				Padding(0, 1).
				Render(fmt.Sprintf("> %-18s  %s", cmd.Name, cmd.Description))
			items = append(items, item)
		} else {
			// Regular item
			item := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Width(m.Width - 8).
				Padding(0, 1).
				Render(fmt.Sprintf("  %-18s  %s", cmd.Name, cmd.Description))
			items = append(items, item)
		}
	}

	dropdownContent := lipgloss.JoinVertical(lipgloss.Top, items...)

	return lipgloss.NewStyle().
		MarginLeft(2).
		MarginRight(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 1).
		Render(dropdownContent)
}

func (m *Model) renderFooter() string {
	keys := "↑/↓: navigate  •  Enter: run  •  Esc: clear  •  q: quit"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true).
		PaddingTop(1).
		Render(keys)
}

func (m *Model) viewAnalyzing() string {
	header := m.renderHeader()
	message := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Align(lipgloss.Center).
		Width(m.Width - 4).
		Padding(3, 0).
		Render("🔄 " + m.CurrentMessage)

	backPrompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true).
		Render("Press Enter to return to main menu")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		message,
		"",
		backPrompt,
	)
}

func (m *Model) viewResults() string {
	header := m.renderHeader()
	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(2, 2).
		Render(m.CurrentMessage)

	backPrompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true).
		Render("Press Enter to return to main menu")

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

	helpText := `COMMANDS:

  analyze          Analyze Gemfile.lock for risks and dependency conflicts
  deps             Show which parent gems are using a specific gem
  vulnerabilities  Check for known vulnerabilities in your gems
  licenses         Generate license compliance report
  help             Show this help message
  quit             Exit gemtracker

KEYBOARD SHORTCUTS:

  ↑/↓, Tab        Navigate commands
  Enter           Run selected command
  Esc             Clear search / return to menu
  q, Ctrl+C       Quit gemtracker`

	content := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(1, 2).
		Render(helpText)

	backPrompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true).
		Render("Press Enter to return to main menu")

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

	message := lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true).
		Padding(2, 2).
		Render("❌ " + m.ErrorMessage)

	backPrompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true).
		Render("Press Enter to return to main menu")

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
