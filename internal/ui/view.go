package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spaquet/gemtracker/internal/gemfile"
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
	case ViewFilterInput:
		return m.viewFilterInput()
	case ViewDependencySearch:
		return m.viewDependencySearch()
	case ViewDependencyTree:
		return m.viewDependencyTree()
	case ViewHelp:
		return m.viewHelp()
	case ViewError:
		return m.viewError()
	case ViewSelectPath:
		return m.viewSelectPath()
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
		Render(`Type / to list all commands
Use ↑/↓ arrows to navigate
Press Enter to run
Type 'q' to quit anytime`)

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
	return ""
}

func (m *Model) viewAnalyzing() string {
	header := m.renderHeader()

	// Get spinner frame
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸"}
	spinnerFrame := spinnerFrames[m.AnimationFrame%len(spinnerFrames)]

	message := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Align(lipgloss.Center).
		Width(m.Width - 4).
		Padding(3, 0).
		Render(spinnerFrame + " " + m.CurrentMessage)

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

func (m *Model) viewFilterInput() string {
	header := m.renderHeader()

	// Summary line
	summary := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		PaddingLeft(2).
		PaddingRight(2).
		MarginTop(1).
		Render(m.CurrentMessage)

	// Gem list display - viewport style
	listContent := m.renderGemListViewport(10) // Fixed height of 10 for analyze view

	// Search input with clear separation
	filterInput := m.FilterInput.View()
	filterInputBox := lipgloss.NewStyle().
		Width(m.Width - 6).
		PaddingLeft(2).
		PaddingRight(2).
		PaddingTop(1).
		MarginTop(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Render(filterInput)

	// Debug: Show what's in the search field
	debugInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingLeft(2).
		Render(fmt.Sprintf("[Input: '%s'] [Gems: %d]", m.FilterInput.Value(), len(m.FilteredGems)))

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		PaddingLeft(2).
		MarginTop(1).
		Render("↑/↓: navigate  •  Type to search  •  Esc: back")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		summary,
		listContent,
		filterInputBox,
		debugInfo,
		instructions,
	)
}

func (m *Model) renderGemListViewport(maxHeight int) string {
	if m.FilteredGems == nil || len(m.FilteredGems) == 0 {
		return lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2).
			MarginTop(1).
			Render("(no gems match filter)")
	}

	// Use provided max height
	availableHeight := maxHeight
	if availableHeight < 3 {
		availableHeight = 3
	}

	// Build viewport
	var lines []string
	endIdx := m.ScrollOffset + availableHeight
	if endIdx > len(m.FilteredGems) {
		endIdx = len(m.FilteredGems)
	}

	for i := m.ScrollOffset; i < endIdx; i++ {
		gemStatus := m.FilteredGems[i]
		line := m.formatGemLine(gemStatus, i == m.SelectedGemIdx)
		lines = append(lines, line)
	}

	// Join lines and add padding
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		MarginTop(1).
		Render(content)
}

func (m *Model) formatGemLine(gemStatus *gemfile.GemStatus, isSelected bool) string {
	// Determine status icon
	statusIcon := "✓"
	if gemStatus.IsVulnerable {
		statusIcon = "🔒"
	} else if gemStatus.IsOutdated {
		statusIcon = "⚠"
	}

	// Build the main line: status, name, version
	line := fmt.Sprintf("%s %-30s v%-8s", statusIcon, gemStatus.Name, gemStatus.Version)

	// Add groups if present
	if len(gemStatus.Groups) > 0 {
		groupStr := strings.Join(gemStatus.Groups, ", ")
		line += fmt.Sprintf("  [%s]", groupStr)
	}

	// Add additional info (vulnerabilities or outdated)
	if gemStatus.IsVulnerable {
		line += fmt.Sprintf("  %s", gemStatus.VulnerabilityInfo)
	} else if gemStatus.IsOutdated && gemStatus.LatestVersion != "" {
		line += fmt.Sprintf("  → v%s", gemStatus.LatestVersion)
	}

	// Apply styling based on selection
	if isSelected {
		// Selected: green background with dark text
		return lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("0")).
			Render(line)
	}

	// Normal line: just show the text
	return line
}

func (m *Model) viewDependencyTree() string {
	header := m.renderHeader()

	if m.DependencyResult == nil {
		return lipgloss.JoinVertical(
			lipgloss.Top,
			header,
			lipgloss.NewStyle().
				Foreground(ColorError).
				Padding(2, 2).
				Render("No dependency result available"),
		)
	}

	depInfo := m.DependencyResult.DependencyInfo
	if depInfo == nil {
		return lipgloss.JoinVertical(
			lipgloss.Top,
			header,
			lipgloss.NewStyle().
				Foreground(ColorError).
				Padding(2, 2).
				Render("Gem not found"),
		)
	}

	// Title
	titleText := fmt.Sprintf("Dependencies for: %s (v%s)", depInfo.GemName, depInfo.Version)
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		PaddingLeft(2).
		PaddingRight(2).
		MarginTop(1).
		Render(titleText)

	// Forward dependencies tree (what this gem depends on)
	forwardSection := m.renderDependencyTreeSection("Dependencies This Gem Needs", depInfo.ForwardTree)

	// Reverse dependencies tree (what depends on this gem)
	reverseSection := m.renderDependencyTreeSection("Gems That Depend On This", depInfo.ReverseTree)

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		PaddingLeft(2).
		MarginTop(2).
		Render("Press Enter to return to main menu")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		title,
		forwardSection,
		"",
		reverseSection,
		"",
		instructions,
	)
}

func (m *Model) renderDependencyTreeSection(title string, root *gemfile.DependencyNode) string {
	sectionTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary).
		PaddingLeft(2).
		PaddingRight(2).
		MarginTop(1).
		Render(title)

	if root == nil || len(root.Children) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			PaddingLeft(4).
			Render("(none)")
		return lipgloss.JoinVertical(lipgloss.Top, sectionTitle, emptyMsg)
	}

	// Smart rendering: show direct children + their direct children (limited depth)
	var lines []string
	seen := make(map[string]bool)

	for i, child := range root.Children {
		isLast := i == len(root.Children)-1
		lines = append(lines, m.renderSmartDependencyNode(child, "", isLast, seen, 0)...)
	}

	depContent := strings.Join(lines, "\n")
	depBox := lipgloss.NewStyle().
		PaddingLeft(3).
		PaddingRight(2).
		Render(depContent)

	return lipgloss.JoinVertical(lipgloss.Top, sectionTitle, depBox)
}

// renderSmartDependencyNode renders a dependency node with smart depth limiting
// Shows up to 2 levels: direct deps and their children, with context
func (m *Model) renderSmartDependencyNode(node *gemfile.DependencyNode, prefix string, isLast bool, seen map[string]bool, depth int) []string {
	var lines []string

	// Current node line
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	// Mark if this gem was already seen (circular reference)
	circularMark := ""
	if seen[node.Name] && depth > 0 {
		circularMark = " ↩"
	}
	seen[node.Name] = true

	nodeLine := fmt.Sprintf("%s%s (v%s)%s", connector, node.Name, node.Version, circularMark)
	lines = append(lines, prefix+nodeLine)

	// Only show children if depth < 2 (limit to 2 levels)
	if len(node.Children) > 0 && depth < 1 {
		extension := "│   "
		if isLast {
			extension = "    "
		}

		// Limit to showing first 3 children for readability
		childrenToShow := len(node.Children)
		if childrenToShow > 3 {
			childrenToShow = 3
		}

		for i := 0; i < childrenToShow; i++ {
			child := node.Children[i]
			childIsLast := i == childrenToShow-1 && childrenToShow == len(node.Children)
			childLines := m.renderSmartDependencyNode(child, prefix+extension, childIsLast, seen, depth+1)
			lines = append(lines, childLines...)
		}

		// If there are more children than we showed, indicate that
		if childrenToShow < len(node.Children) {
			remainingCount := len(node.Children) - childrenToShow
			moreMsg := fmt.Sprintf("%s    └─ ... and %d more", prefix, remainingCount)
			lines = append(lines, moreMsg)
		}
	} else if len(node.Children) > 0 && depth >= 1 {
		// Indicate there are more deps but we're limiting depth
		moreMsg := fmt.Sprintf("%s└─ → %d more dependencies", prefix, len(node.Children))
		lines = append(lines, moreMsg)
	}

	return lines
}

func (m *Model) viewHelp() string {
	header := m.renderHeader()

	helpText := `GETTING STARTED:

  Type / to list all available commands
  Use arrow keys (↑/↓) to navigate through the list
  Press Enter to run the selected command
  Type to search for specific commands

AVAILABLE COMMANDS:

  analyze          Analyze Gemfile.lock for risks and dependency conflicts
  deps             Show which parent gems are using a specific gem
  vulnerabilities  Check for known vulnerabilities in your gems
  licenses         Generate license compliance report
  help             Show this help message
  quit             Exit gemtracker

KEYBOARD SHORTCUTS:

  /               List all commands
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

func (m *Model) viewSelectPath() string {
	header := m.renderHeader()

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		MarginBottom(1).
		Render("Enter project path:")

	pathInput := m.PathInput.View()
	pathBox := lipgloss.NewStyle().
		Width(m.Width - 6).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		MarginLeft(2).
		MarginRight(2).
		MarginTop(1).
		MarginBottom(2).
		Render(pathInput)

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		MarginLeft(2).
		Render("Examples: /path/to/project  or  ~/Sites/myapp  or  .")

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		MarginLeft(2).
		MarginTop(1).
		Render("Press Enter to open project  •  Esc to cancel")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		"",
		title,
		pathBox,
		hint,
		instructions,
	)
}

func (m *Model) viewDependencySearch() string {
	header := m.renderHeader()

	// Summary line
	summary := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		PaddingLeft(2).
		PaddingRight(2).
		MarginTop(1).
		Render(m.CurrentMessage)

	// Search input with clear separation - FIXED AT TOP
	filterInput := m.FilterInput.View()
	filterInputBox := lipgloss.NewStyle().
		Width(m.Width - 6).
		PaddingLeft(2).
		PaddingRight(2).
		PaddingTop(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Render(filterInput)

	// Calculate viewport height using consistent calculation
	viewportHeight := m.calculateDepsViewportHeight()

	// Gem list display - viewport style with calculated height
	listContent := m.renderGemListViewport(viewportHeight)

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		PaddingLeft(2).
		MarginTop(1).
		Render("↑/↓: navigate  •  Type to search  •  Enter: select  •  Esc: back")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		summary,
		filterInputBox,
		listContent,
		instructions,
	)
}

func (m *Model) calculateDepsViewportHeight() int {
	const (
		headerHeight      = 10
		summaryHeight     = 2
		searchBoxHeight   = 3
		instructionsHeight = 2
		spacing           = 2
	)

	viewportHeight := m.Height - (headerHeight + summaryHeight + searchBoxHeight + instructionsHeight + spacing)
	if viewportHeight < 3 {
		viewportHeight = 3
	}
	return viewportHeight
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
