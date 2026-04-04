package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spaquet/gemtracker/internal/gemfile"
)

func (m *Model) View() string {
	if m.Quitting {
		return ""
	}

	switch m.CurrentView {
	case ViewLoading:
		return m.viewLoading()
	case ViewGemList:
		return m.viewGemList()
	case ViewGemDetail:
		return m.viewGemDetail()
	case ViewSearch:
		return m.viewSearch()
	case ViewCVE:
		return m.viewCVE()
	case ViewSelectPath:
		return m.viewSelectPath()
	case ViewError:
		return m.viewError()
	default:
		return m.viewGemList()
	}
}

// ============================================================================
// Chrome Components
// ============================================================================

func (m *Model) renderAppHeader() string {
	appName := fmt.Sprintf("gemtracker v%s", m.Version)
	projectPath := m.ProjectPath
	if projectPath == "" {
		projectPath = "(no project)"
	}

	left := AppHeaderStyle.Render(appName)
	right := ProjectPathStyle.Render(projectPath)

	// Calculate spacing
	totalLen := lipgloss.Width(left) + lipgloss.Width(right)
	spacerCount := m.Width - totalLen - 4
	if spacerCount < 0 {
		spacerCount = 0
	}
	spacer := strings.Repeat(" ", spacerCount)

	return left + spacer + right
}

func (m *Model) renderTabBar() string {
	tabLabels := []string{"Gems", "Search", "CVE"}
	tabModes := []ViewMode{ViewGemList, ViewSearch, ViewCVE}

	var tabs []string
	for i, label := range tabLabels {
		mode := tabModes[i]
		if mode == m.ActiveTab {
			tabs = append(tabs, TabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, TabStyle.Render(label))
		}
	}

	return strings.Join(tabs, "  ")
}

func (m *Model) renderStatusBar() string {
	var hints []string

	switch m.CurrentView {
	case ViewGemList:
		hints = []string{"↑↓ navigate", "enter select", "tab next", "q quit"}
	case ViewGemDetail:
		hints = []string{"esc back", "tab section", "↑↓ navigate", "enter select", "o open url", "q quit"}
	case ViewSearch:
		hints = []string{"type search", "↑↓ navigate", "enter select", "esc clear"}
	case ViewCVE:
		hints = []string{"↑↓ navigate", "enter select", "tab next", "q quit"}
	case ViewSelectPath:
		hints = []string{"enter confirm", "esc cancel"}
	default:
		hints = []string{"type to filter", "q quit"}
	}

	var rendered []string
	for _, hint := range hints {
		parts := strings.SplitN(hint, " ", 2)
		if len(parts) == 2 {
			key := KeyHintKeyStyle.Render(parts[0])
			desc := KeyHintDescStyle.Render(" " + parts[1])
			rendered = append(rendered, key+desc)
		}
	}

	content := strings.Join(rendered, "  ")
	status := StatusBarStyle.Width(m.Width - 4).Render(content)
	return status
}

// ============================================================================
// View: Loading
// ============================================================================

func (m *Model) viewLoading() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	spinner := spinnerFrames[m.AnimationFrame%len(spinnerFrames)]
	spinnerText := SpinnerStyle.Render(spinner + " " + m.LoadingMessage)

	contentHeight := m.Height - FixedChrome - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	contentLines := (contentHeight - 1) / 2
	if contentLines < 0 {
		contentLines = 0
	}

	padding := strings.Repeat("\n", contentLines)

	content := lipgloss.JoinVertical(lipgloss.Center, padding, spinnerText)
	content = lipgloss.NewStyle().Height(contentHeight).Render(content)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		content,
		statusbar,
	)
}

// ============================================================================
// View: Gem List
// ============================================================================

func (m *Model) viewGemList() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	contentHeight := m.Height - FixedChrome - 2
	gemListContent := m.renderGemListTable(contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		gemListContent,
		statusbar,
	)
}

func (m *Model) renderGemListTable(height int) string {
	if height < 1 {
		height = 1
	}

	// Table header
	headerRow := fmt.Sprintf("  %-4s %-24s %-11s %-11s %-14s %s",
		"#", "Gem Name", "Installed", "Latest", "Groups", "Status")
	header := TableHeaderStyle.Render(headerRow)

	// Table rows
	lines := []string{header}
	visibleRows := height - 2
	if visibleRows < 0 {
		visibleRows = 0
	}
	endIdx := m.GemListOffset + visibleRows
	if endIdx > len(m.FirstLevelGems) {
		endIdx = len(m.FirstLevelGems)
	}

	for i := m.GemListOffset; i < endIdx; i++ {
		if i >= len(m.FirstLevelGems) {
			break
		}

		gem := m.FirstLevelGems[i]
		isSelected := i == m.GemListCursor

		line := m.formatGemListRow(i+1, gem, isSelected)
		lines = append(lines, line)
	}

	// Padding
	for len(lines) < height {
		lines = append(lines, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) formatGemListRow(idx int, gem *gemfile.GemStatus, selected bool) string {
	// Status indicator
	var status string
	if gem.IsVulnerable {
		status = BadgeVulnerableStyle.Render("⚠ CVE")
	} else if gem.IsOutdated {
		status = BadgeOutdatedStyle.Render("↑ " + gem.LatestVersion)
	} else {
		status = BadgeOKStyle.Render("✓")
	}

	// Latest version display
	latestDisplay := "latest"
	if gem.IsOutdated {
		latestDisplay = gem.LatestVersion
	}

	// Groups display
	groupsDisplay := strings.Join(gem.Groups, ",")
	if len(groupsDisplay) > 14 {
		groupsDisplay = groupsDisplay[:11] + "..."
	}

	// Format row
	row := fmt.Sprintf("  %-4d %-24s %-11s %-11s %-14s %s",
		idx,
		truncateStr(gem.Name, 24),
		gem.Version,
		latestDisplay,
		groupsDisplay,
		status,
	)

	// Apply selection styling
	if selected {
		return RowSelectedStyle.Render(row)
	}
	return RowNormalStyle.Render(row)
}

// ============================================================================
// View: Gem Detail
// ============================================================================

func (m *Model) viewGemDetail() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	if m.SelectedGem == nil {
		return ""
	}

	contentHeight := m.Height - FixedChrome - 5

	// Format version info
	versionDisplay := "Latest"
	if m.SelectedGem.IsOutdated {
		versionDisplay = m.SelectedGem.LatestVersion
	}

	// Build header lines
	headerLine1 := fmt.Sprintf("%s   Installed: %s  →  %s%s",
		m.SelectedGem.Name,
		m.SelectedGem.Version,
		versionDisplay,
		func() string {
			if m.SelectedGem.IsOutdated {
				return " (update available)"
			}
			return ""
		}(),
	)
	headerLine1Formatted := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimary)).Render(headerLine1)

	// Format description line
	descMaxLen := m.Width - 4
	if descMaxLen < 20 {
		descMaxLen = 20
	}
	descLine := ""
	if m.SelectedGem.Description != "" {
		descLine = truncateStr(m.SelectedGem.Description, descMaxLen)
		descLine = "  " + descLine
		descLine = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)).Render(descLine)
	}

	// URL line
	urlLine := "  " + truncateStr(m.SelectedGem.HomepageURL, descMaxLen)
	urlLine = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)).Italic(true).Render(urlLine)

	var gemInfoLines []string
	gemInfoLines = append(gemInfoLines, headerLine1Formatted)
	if descLine != "" {
		gemInfoLines = append(gemInfoLines, descLine)
	}
	gemInfoLines = append(gemInfoLines, urlLine)

	// Two panels: forward deps and reverse deps
	panelHeight := (contentHeight - len(gemInfoLines) - 2) / 2

	forwardTitle := "  Dependencies (what this gem needs)"
	reverseTitle := "  Used By (what depends on this gem)"

	var forwardContent string
	var reverseContent string

	if m.DependencyResult != nil {
		forwardContent = m.renderDependencyPanel(m.DependencyResult.DependencyInfo.ForwardTree, panelHeight, m.DetailSection == 0)
		reverseContent = m.renderDependencyPanel(m.DependencyResult.DependencyInfo.ReverseTree, panelHeight, m.DetailSection == 1)
	} else {
		forwardContent = strings.Repeat(" \n", panelHeight)
		reverseContent = strings.Repeat(" \n", panelHeight)
	}

	forwardSection := lipgloss.JoinVertical(lipgloss.Left,
		forwardTitle,
		forwardContent,
	)

	reverseSection := lipgloss.JoinVertical(lipgloss.Left,
		reverseTitle,
		reverseContent,
	)

	// Apply borders
	borderStyle := PanelBorderStyle
	if m.DetailSection == 0 {
		borderStyle = PanelBorderActiveStyle
	}

	forwardPanel := borderStyle.Render(forwardSection)
	reverseBorderStyle := PanelBorderStyle
	if m.DetailSection == 1 {
		reverseBorderStyle = PanelBorderActiveStyle
	}
	reversePanel := reverseBorderStyle.Render(reverseSection)

	contentLines := []string{}
	contentLines = append(contentLines, gemInfoLines...)
	contentLines = append(contentLines, forwardPanel, reversePanel)
	content := lipgloss.JoinVertical(lipgloss.Left, contentLines...)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		content,
		statusbar,
	)
}

func (m *Model) renderDependencyPanel(node *gemfile.DependencyNode, height int, focused bool) string {
	if node == nil || node.Name == "" {
		return strings.Repeat(" \n", height)
	}

	// Get the appropriate offset for this panel
	offset := m.DetailForwardOffset
	if !focused {
		offset = m.DetailReverseOffset
	}

	// Get all lines from the tree (this will also populate DetailTreeLines)
	allLines := m.renderDependencyTree(node, 9999, 0, offset)

	// Apply offset to slice
	if offset > len(allLines) {
		offset = len(allLines)
	}
	visibleLines := allLines[offset:]

	// Ensure we have exactly `height` lines
	for len(visibleLines) < height {
		visibleLines = append(visibleLines, "")
	}

	return strings.Join(visibleLines[:height], "\n")
}

func (m *Model) renderDependencyTree(node *gemfile.DependencyNode, maxLines int, depth int, offset int) []string {
	if node == nil || maxLines <= 0 {
		return []string{}
	}

	var lines []string
	var gemNames []string
	m.renderTreeNode(node, depth, &lines, &gemNames, maxLines, 0, offset)

	// Store gem names for later lookup
	m.DetailTreeLines = gemNames

	return lines
}

func (m *Model) renderTreeNode(node *gemfile.DependencyNode, depth int, lines *[]string, gemNames *[]string, maxLines int, lineIdx int, offset int) int {
	if node == nil || len(*lines) >= maxLines {
		return lineIdx
	}

	// Indent based on depth
	indent := strings.Repeat("  ", depth)
	connector := "├── "
	if depth == 0 {
		connector = ""
	}

	name := node.Name
	displayName := name
	if node.Version != "" {
		displayName = fmt.Sprintf("%s (%s)", name, node.Version)
	}

	// Calculate visible line index (accounting for offset)
	visibleLineIdx := lineIdx - offset

	// Check if this line should be highlighted (and is visible)
	isSelected := visibleLineIdx == m.DetailTreeCursor && visibleLineIdx >= 0

	var line string
	if isSelected {
		// Highlight selected line
		line = indent + connector + RowSelectedStyle.Render(displayName)
	} else {
		line = indent + connector + TreeGemNameStyle.Render(displayName)
	}

	*lines = append(*lines, line)
	*gemNames = append(*gemNames, name)
	lineIdx++

	// Render all children (stop if we hit maxLines)
	for _, child := range node.Children {
		if len(*lines) >= maxLines {
			break
		}
		lineIdx = m.renderTreeNode(child, depth+1, lines, gemNames, maxLines, lineIdx, offset)
	}

	return lineIdx
}

// ============================================================================
// View: Search
// ============================================================================

func (m *Model) viewSearch() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	// Search input
	searchPrompt := SearchPromptStyle.Render("/ ")
	promptWidth := lipgloss.Width(searchPrompt)
	// Account for prompt, left margin (2), right margin (2), border (2)
	searchInputWidth := m.Width - promptWidth - 6
	if searchInputWidth < 10 {
		searchInputWidth = 10
	}
	searchInput := SearchBoxStyle.Width(searchInputWidth).Render(m.SearchInput.View())
	searchLine := lipgloss.JoinHorizontal(lipgloss.Top, searchPrompt, searchInput)

	// Search results
	contentHeight := m.Height - FixedChrome - 3
	resultContent := m.renderSearchResults(contentHeight)

	content := lipgloss.JoinVertical(lipgloss.Left,
		searchLine,
		resultContent,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		content,
		statusbar,
	)
}

func (m *Model) renderSearchResults(height int) string {
	if height < 1 {
		height = 1
	}

	if m.SearchQuery == "" {
		padding := strings.Repeat(" \n", height)
		return padding
	}

	title := fmt.Sprintf("Gems matching \"%s\" (%d found)", m.SearchQuery, len(m.SearchResults))
	title = lipgloss.NewStyle().Bold(true).Render(title)

	// Header
	headerRow := fmt.Sprintf("  %-30s %-11s %s", "Gem Name", "Version", "Groups")
	header := TableHeaderStyle.Render(headerRow)

	lines := []string{title, header}

	// Result rows
	visibleRows := height - 4
	if visibleRows < 0 {
		visibleRows = 0
	}
	endIdx := m.SearchOffset + visibleRows
	if endIdx > len(m.SearchResults) {
		endIdx = len(m.SearchResults)
	}

	for i := m.SearchOffset; i < endIdx; i++ {
		if i >= len(m.SearchResults) {
			break
		}

		gem := m.SearchResults[i]
		isSelected := i == m.SearchCursor

		groupsDisplay := strings.Join(gem.Groups, ",")
		if len(groupsDisplay) > 20 {
			groupsDisplay = groupsDisplay[:17] + "..."
		}

		row := fmt.Sprintf("  %-30s %-11s %s",
			truncateStr(gem.Name, 30),
			gem.Version,
			groupsDisplay,
		)

		if isSelected {
			row = RowSelectedStyle.Render(row)
		} else {
			row = RowNormalStyle.Render(row)
		}
		lines = append(lines, row)
	}

	// Padding
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:height], "\n")
}

// ============================================================================
// View: CVE
// ============================================================================

func (m *Model) viewCVE() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	contentHeight := m.Height - FixedChrome - 2
	cveContent := m.renderCVETable(contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		cveContent,
		statusbar,
	)
}

func (m *Model) renderCVETable(height int) string {
	if height < 1 {
		height = 1
	}

	if len(m.VulnerableGems) == 0 {
		msg := "No vulnerabilities found. Your gems are safe! ✓"
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true).
			Padding(2, 2).
			Render(msg)
	}

	title := fmt.Sprintf("Vulnerabilities Found (%d)", len(m.VulnerableGems))
	title = lipgloss.NewStyle().Bold(true).Render(title)

	headerRow := fmt.Sprintf("  %-20s %-16s %-11s %-30s %s",
		"CVE ID", "Gem", "Version", "Description", "Status")
	header := TableHeaderStyle.Render(headerRow)

	lines := []string{title, header}

	visibleRows := height - 3
	if visibleRows < 0 {
		visibleRows = 0
	}
	endIdx := m.CVEOffset + visibleRows
	if endIdx > len(m.VulnerableGems) {
		endIdx = len(m.VulnerableGems)
	}

	for i := m.CVEOffset; i < endIdx; i++ {
		if i >= len(m.VulnerableGems) {
			break
		}

		gem := m.VulnerableGems[i]
		cveID := extractCVEID(gem.VulnerabilityInfo)
		desc := extractCVEDesc(gem.VulnerabilityInfo)

		row := fmt.Sprintf("  %-20s %-16s %-11s %-30s %s",
			cveID,
			truncateStr(gem.Name, 16),
			gem.Version,
			truncateStr(desc, 30),
			"⚠",
		)

		if i == m.CVECursor {
			row = RowSelectedStyle.Render(row)
		} else {
			row = RowNormalStyle.Render(row)
		}
		lines = append(lines, row)
	}

	// Padding
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:height], "\n")
}

// ============================================================================
// View: SelectPath
// ============================================================================

func (m *Model) viewSelectPath() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	title := "Select Ruby Project Directory"
	title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimary)).Render(title)

	prompt := "Path: "
	input := InputBoxStyle.Width(m.Width - 20).Render(m.PathInput.View())
	inputLine := prompt + input

	hint := "Examples: . | ~ | /path/to/project | ~/myapp"
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		inputLine,
		"",
		hintStyle.Render(hint),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		content,
		statusbar,
	)
}

// ============================================================================
// View: Error
// ============================================================================

func (m *Model) viewError() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	errorBox := ErrorBoxStyle.Render("ERROR\n\n" + m.ErrorMessage)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		errorBox,
		"",
		"Press Enter or Esc to continue",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		content,
		statusbar,
	)
}

// ============================================================================
// Helper Functions
// ============================================================================

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func extractCVEID(vulnInfo string) string {
	parts := strings.Split(vulnInfo, ":")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return "Unknown"
}

func extractCVEDesc(vulnInfo string) string {
	parts := strings.Split(vulnInfo, ":")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return vulnInfo
}
