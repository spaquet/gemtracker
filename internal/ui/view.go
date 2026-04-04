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
	spacer := strings.Repeat(" ", m.Width-totalLen-4)

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

	return strings.Join(tabs, "")
}

func (m *Model) renderStatusBar() string {
	var hints []string

	switch m.CurrentView {
	case ViewGemList:
		hints = []string{"↑↓ navigate", "enter select", "tab next", "q quit"}
	case ViewGemDetail:
		hints = []string{"esc back", "tab section", "↑↓ scroll", "q quit"}
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
	contentLines := (contentHeight - 1) / 2
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
	// Table header
	headerRow := fmt.Sprintf("  %-4s %-24s %-11s %-11s %-14s %s",
		"#", "Gem Name", "Installed", "Latest", "Groups", "Status")
	header := TableHeaderStyle.Render(headerRow)

	// Table rows
	lines := []string{header}
	visibleRows := height - 2
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

	contentHeight := m.Height - FixedChrome - 4
	gemInfo := fmt.Sprintf("%s v%s    %s",
		m.SelectedGem.Name,
		m.SelectedGem.Version,
		m.SelectedGem.HomepageURL,
	)
	gemInfoFormatted := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimary)).Render(gemInfo)

	// Two panels: forward deps and reverse deps
	panelHeight := (contentHeight - 2) / 2

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

	content := lipgloss.JoinVertical(lipgloss.Left,
		gemInfoFormatted,
		forwardPanel,
		reversePanel,
	)

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

	lines := m.renderDependencyTree(node, height, 0)

	// Ensure we have exactly `height` lines
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:height], "\n")
}

func (m *Model) renderDependencyTree(node *gemfile.DependencyNode, maxLines int, depth int) []string {
	if node == nil || maxLines <= 0 {
		return []string{}
	}

	var lines []string
	m.renderTreeNode(node, depth, &lines, maxLines)
	return lines
}

func (m *Model) renderTreeNode(node *gemfile.DependencyNode, depth int, lines *[]string, remaining int) {
	if remaining <= 0 || node == nil {
		return
	}

	// Indent based on depth
	indent := strings.Repeat("  ", depth)
	connector := "├── "
	if depth == 0 {
		connector = ""
	}

	name := node.Name
	if node.Version != "" {
		name = fmt.Sprintf("%s (%s)", name, node.Version)
	}

	line := indent + connector + TreeGemNameStyle.Render(name)
	*lines = append(*lines, line)
	remaining--

	// Render children (cap at 3 per node for readability)
	for i, child := range node.Children {
		if i >= 3 && len(node.Children) > 3 {
			*lines = append(*lines, strings.Repeat("  ", depth+1)+"... and "+
				fmt.Sprintf("%d more", len(node.Children)-3))
			break
		}
		m.renderTreeNode(child, depth+1, lines, remaining)
		remaining = len(*lines) - 1
	}
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
	searchInput := SearchBoxStyle.Width(m.Width - 10).Render(m.SearchInput.View())
	searchLine := searchPrompt + searchInput

	// Search results
	contentHeight := m.Height - FixedChrome - 4
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
	if m.SearchQuery == "" {
		return strings.Repeat(" \n", height)
	}

	title := fmt.Sprintf("Gems matching \"%s\" (%d found)", m.SearchQuery, len(m.SearchResults))
	title = lipgloss.NewStyle().Bold(true).Render(title)

	// Header
	headerRow := fmt.Sprintf("  %-30s %-11s %s", "Gem Name", "Version", "Groups")
	header := TableHeaderStyle.Render(headerRow)

	lines := []string{title, header}

	// Result rows
	visibleRows := height - 4
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
