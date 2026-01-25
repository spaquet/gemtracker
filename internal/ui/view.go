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
	// Top bar with title and version
	topBar := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		PaddingTop(0).
		PaddingBottom(0).
		PaddingLeft(2).
		PaddingRight(2).
		Render(fmt.Sprintf("— gemtracker  v%s —", m.Version))

	// Left column content
	leftColumn := m.renderHeaderLeft()

	// Right column content
	rightColumn := m.renderHeaderRight()

	// Combine columns with proper spacing
	contentLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		"  ",
		rightColumn,
	)

	// Full header with top bar
	header := lipgloss.JoinVertical(
		lipgloss.Top,
		topBar,
		contentLine,
	)

	// Add border around entire header
	headerStyled := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Render(header)

	return headerStyled
}

func (m *Model) renderHeaderLeft() string {
	// ASCII gem art
	gemArt := `
    ◆
   ◆ ◆
  ◆ ◆ ◆
 ◆ ◆ ◆ ◆
  ◆ ◆ ◆
   ◆ ◆
    ◆`

	gemStyled := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Align(lipgloss.Center).
		Render(gemArt)

	// Welcome message
	welcome := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		MarginTop(1).
		Render("Welcome to gemtracker!")

	// Project info
	projectPath := "No Gemfile.lock found"
	if m.GemfileLockPath != "" {
		projectPath = m.ProjectPath
	}

	projectInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		MarginTop(1).
		Render(projectPath)

	// Stats line
	statsLine := ""
	if m.GemfileLockPath != "" {
		statsLine = fmt.Sprintf("📦 %d gems  |  ⚠️  %d outdated  |  🔒 %d vulnerable",
			m.GemCount,
			m.OutdatedCount,
			m.VulnerableCount,
		)
	}

	statsStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		MarginTop(1).
		Render(statsLine)

	leftContent := lipgloss.JoinVertical(
		lipgloss.Center,
		gemStyled,
		welcome,
		projectInfo,
		statsStyled,
	)

	return lipgloss.NewStyle().
		Width(35).
		MaxHeight(12).
		Render(leftContent)
}

func (m *Model) renderHeaderRight() string {
	// Tips section
	tipsHeader := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		MarginBottom(1).
		Render("Tips for getting started")

	tips := `Use arrow keys to navigate
Press Enter to run commands
Type 'q' to quit anytime

Try 'analyze' to scan your
gems for vulnerabilities and
outdated versions!`

	tipsContent := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render(tips)

	rightContent := lipgloss.JoinVertical(
		lipgloss.Top,
		tipsHeader,
		tipsContent,
	)

	return lipgloss.NewStyle().
		Width(40).
		MaxHeight(12).
		Render(rightContent)
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
	searchBox := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Render("🔍 " + inputView)
	return searchBox
}

func (m *Model) renderFooter() string {
	keys := "↑/↓: navigate  •  Enter: run  •  Esc: clear  •  q: quit"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		MarginTop(1).
		Render(keys)
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
