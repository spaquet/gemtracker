package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spaquet/gemtracker/internal/gemfile"
)

// ============================================================================
// Helper Methods
// ============================================================================

func (m *Model) updateBarHeight() int {
	if m.NewVersionAvailable != "" {
		return 1
	}
	return 0
}

// ============================================================================
// View Rendering
// ============================================================================

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
	case ViewProjectInfo:
		return m.viewProjectInfo()
	case ViewFilterMenu:
		return m.viewFilterMenu()
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
	appName := fmt.Sprintf("gemtracker %s", m.Version)
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
	tabLabels := []string{"Gems", "Search", "CVE", "Project"}
	tabModes := []ViewMode{ViewGemList, ViewSearch, ViewCVE, ViewProjectInfo}

	var tabs []string
	for i, label := range tabLabels {
		mode := tabModes[i]
		// Add CVE count to the CVE label
		if mode == ViewCVE && len(m.VulnerableGems) > 0 {
			label = fmt.Sprintf("%s (%d)", label, len(m.VulnerableGems))
		}
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
		hints = []string{"↑↓ navigate", "enter select", "f filter", "u upgradable", "c clear", "tab next", "q quit"}
	case ViewGemDetail:
		hints = []string{"esc back", "tab section", "↑↓ navigate", "enter select", "o open url", "q quit"}
	case ViewSearch:
		hints = []string{"type search", "↑↓ navigate", "enter select", "esc clear"}
	case ViewCVE:
		hints = []string{"↑↓ navigate", "enter select", "tab next", "q quit"}
	case ViewProjectInfo:
		hints = []string{"tab next", "shift+tab prev", "q quit"}
	case ViewFilterMenu:
		hints = []string{"↑↓ navigate", "space toggle", "enter back", "q quit"}
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

	// Add outdated checking indicator if needed
	if m.OutdatedLoading {
		doneCount := len(m.FirstLevelGems) - len(m.OutdatedPending)
		outdatedStatus := fmt.Sprintf("Checking updates... (%d/%d)", doneCount, len(m.FirstLevelGems))
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
		content = content + "  " + statusStyle.Render(outdatedStatus)
	}

	// Add health loading indicator if needed
	if m.HealthLoading {
		healthStatus := fmt.Sprintf("Fetching health... (%d/%d)", m.HealthLoadedCount, m.HealthTotalCount)
		healthStatusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
		content = content + "  " + healthStatusStyle.Render(healthStatus)
	}

	// Add error warnings
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDanger))
	if m.OutdatedRateLimited {
		content = content + "  " + errorStyle.Render("updates: rate limited")
	}
	if m.HealthRateLimited {
		content = content + "  " + errorStyle.Render("health: rate limited")
	}
	if m.OutdatedErrorCount > 0 {
		errMsg := fmt.Sprintf("%d update errors", m.OutdatedErrorCount)
		content = content + "  " + errorStyle.Render(errMsg)
	}

	status := StatusBarStyle.Width(m.Width - 4).Render(content)

	// Add update notification bar if a new version is available
	var lines []string
	lines = append(lines, status)

	if m.NewVersionAvailable != "" {
		updateMsg := m.renderUpdateBar()
		lines = append(lines, updateMsg)
	}

	return strings.Join(lines, "\n")
}

func (m *Model) renderUpdateBar() string {
	var updateMsg string

	switch runtime.GOOS {
	case "darwin":
		updateMsg = fmt.Sprintf("  ↑ New version available (%s) — brew upgrade gemtracker", m.NewVersionAvailable)
	default:
		updateMsg = fmt.Sprintf("  ↑ New version available (%s) — https://github.com/spaquet/gemtracker/releases", m.NewVersionAvailable)
	}

	return UpdateBarStyle.Width(m.Width - 4).Render(updateMsg)
}

// ============================================================================
// View: Loading
// ============================================================================

func (m *Model) viewLoading() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Build progress display
	var progressLines []string

	// Stage indicator
	stageText := m.AnalysisStage
	if stageText == "" {
		stageText = "initializing"
	}
	progressLines = append(progressLines, lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary)).
		Render(fmt.Sprintf("Stage: %s", stageText)))

	// Progress bar
	barWidth := 40
	filledWidth := (m.AnalysisPercentage * barWidth) / 100
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	progressBar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
	progressBar = fmt.Sprintf("[%s] %d%%", progressBar, m.AnalysisPercentage)
	progressLines = append(progressLines, progressBar)

	// Message
	if m.LoadingMessage != "" {
		progressLines = append(progressLines, "")
		progressLines = append(progressLines, SpinnerStyle.Render(m.LoadingMessage))
	}

	// Center the progress display
	contentLines := (contentHeight - len(progressLines)) / 2
	if contentLines < 0 {
		contentLines = 0
	}

	padding := strings.Repeat("\n", contentLines)
	allLines := []string{padding}
	allLines = append(allLines, progressLines...)

	// Pad to fill height
	for len(allLines) < contentHeight {
		allLines = append(allLines, "")
	}

	content := strings.Join(allLines[:contentHeight], "\n")

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

	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 2
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

	lines := []string{}

	// Add filter status line if filters are active
	if m.hasActiveFilters() {
		var filterParts []string
		if m.ShowOnlyUpgradable {
			filterParts = append(filterParts, "upgradable")
		}
		if len(m.SelectedGroups) > 0 {
			var groups []string
			for _, g := range m.AvailableGroups {
				if m.SelectedGroups[g] {
					groups = append(groups, g)
				}
			}
			filterParts = append(filterParts, "group:"+strings.Join(groups, ","))
		}
		filterStatus := fmt.Sprintf("  Filters: %s  (c to clear)", strings.Join(filterParts, " | "))
		filterStatusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning)).
			Italic(true)
		lines = append(lines, filterStatusStyle.Render(filterStatus))
		lines = append(lines, "")
	}

	// Table header
	headerRow := fmt.Sprintf("  %-4s %-24s %-11s %-11s %-14s %-3s %s",
		"#", "Gem Name", "Installed", "Latest", "Groups", "H", "Status ")
	header := TableHeaderStyle.Render(headerRow)
	lines = append(lines, header)

	// Table rows
	visibleRows := height - len(lines) - 2
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
	} else if gem.OutdatedFailed {
		status = BadgeErrorStyle.Render("! err")
	} else if gem.LatestVersion == "" {
		status = BadgeLoadingStyle.Render("…")
	} else {
		status = BadgeOKStyle.Render("✓")
	}

	// Latest version display
	var latestDisplay string
	switch {
	case gem.OutdatedFailed:
		latestDisplay = "-"
	case gem.LatestVersion == "":
		latestDisplay = "…"
	case gem.IsOutdated:
		latestDisplay = gem.LatestVersion
	default:
		latestDisplay = "latest"
	}

	// Groups display
	groupsDisplay := strings.Join(gem.Groups, ",")
	if len(groupsDisplay) > 14 {
		groupsDisplay = groupsDisplay[:11] + "..."
	}

	// Health indicator (only on wide terminals)
	healthDisplay := ""
	if m.Width >= 80 {
		if gem.Health == nil {
			healthDisplay = "   " // 3 spaces for loading state
		} else {
			switch gem.Health.Score {
			case gemfile.HealthHealthy:
				healthDisplay = BadgeHealthyDotStyle.Render("●")
			case gemfile.HealthWarning:
				healthDisplay = BadgeWarningDotStyle.Render("●")
			case gemfile.HealthCritical:
				healthDisplay = BadgeCriticalDotStyle.Render("●")
			default:
				healthDisplay = BadgeErrorStyle.Render("!")
			}
		}
		healthDisplay = fmt.Sprintf("%-3s", healthDisplay)
	}

	// Format row with truncated columns
	var row string
	if m.Width >= 80 {
		row = fmt.Sprintf("  %-4d %-24s %-11s %-11s %-14s %s %s",
			idx,
			truncateStr(gem.Name, 24),
			truncateStr(gem.Version, 11),
			truncateStr(latestDisplay, 11),
			groupsDisplay,
			healthDisplay,
			status,
		)
	} else {
		row = fmt.Sprintf("  %-4d %-24s %-11s %-11s %-14s %s",
			idx,
			truncateStr(gem.Name, 24),
			truncateStr(gem.Version, 11),
			truncateStr(latestDisplay, 11),
			groupsDisplay,
			status,
		)
	}

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

	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 5

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

	// Health section
	if m.SelectedGem.Health != nil {
		healthLines := m.renderHealthSection(m.SelectedGem.Health, descMaxLen)
		gemInfoLines = append(gemInfoLines, healthLines...)
	} else if m.HealthLoading {
		healthLine := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)).Render("  Health: ⠙ fetching...")
		gemInfoLines = append(gemInfoLines, healthLine)
	} else if m.HealthRateLimited {
		healthLine := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning)).Render("  Health: — GitHub rate limited")
		gemInfoLines = append(gemInfoLines, healthLine)
	}

	// Two panels: forward deps and reverse deps (side by side)
	panelHeight := contentHeight - len(gemInfoLines) - 1

	// Calculate panel widths (split screen)
	panelWidth := (m.Width - 4) / 2
	if panelWidth < 20 {
		panelWidth = 20
	}

	var forwardContent string
	var reverseContent string

	if m.DependencyResult != nil {
		forwardContent = m.renderDependencyPanel(m.DependencyResult.DependencyInfo.ForwardTree, panelHeight, true)
		reverseContent = m.renderReverseDepsList(panelHeight)
	} else {
		forwardContent = strings.Repeat(" \n", panelHeight)
		reverseContent = strings.Repeat(" \n", panelHeight)
	}

	// Calculate titles AFTER rendering panels so DetailForwardLines/DetailReverseLines are populated
	forwardTitle := "Dependencies (what this gem needs)"
	reverseTitle := "Used By (what depends on this gem)"

	// Update titles based on currently selected gem in detail view
	if m.DetailSection == 0 && m.DetailTreeCursor < len(m.DetailForwardLines) {
		// If viewing forward dependencies, show what depends on the selected dependency
		currentGem := m.DetailForwardLines[m.DetailTreeCursor]
		reverseTitle = fmt.Sprintf("Used By %s (what depends on it)", currentGem)
	} else if m.DetailSection == 1 && m.DetailTreeCursor < len(m.DetailReverseLines) {
		// If viewing reverse dependencies section, show which forward gem we're looking at
		currentGem := m.DetailReverseLines[m.DetailTreeCursor]
		forwardTitle = fmt.Sprintf("Dependencies of %s", currentGem)
	}

	// Format titles with width constraint
	forwardTitleFormatted := truncateStr(forwardTitle, panelWidth-2)
	reverseTitleFormatted := truncateStr(reverseTitle, panelWidth-2)

	forwardSection := lipgloss.JoinVertical(lipgloss.Left,
		forwardTitleFormatted,
		forwardContent,
	)

	reverseSection := lipgloss.JoinVertical(lipgloss.Left,
		reverseTitleFormatted,
		reverseContent,
	)

	// Apply borders with width
	borderStyle := PanelBorderStyle
	if m.DetailSection == 0 {
		borderStyle = PanelBorderActiveStyle
	}

	forwardPanel := borderStyle.Width(panelWidth).Render(forwardSection)
	reverseBorderStyle := PanelBorderStyle
	if m.DetailSection == 1 {
		reverseBorderStyle = PanelBorderActiveStyle
	}
	reversePanel := reverseBorderStyle.Width(panelWidth).Render(reverseSection)

	// Join panels horizontally
	panelsRow := lipgloss.JoinHorizontal(lipgloss.Top, forwardPanel, "  ", reversePanel)

	contentLines := []string{}
	contentLines = append(contentLines, gemInfoLines...)
	contentLines = append(contentLines, panelsRow)
	content := lipgloss.JoinVertical(lipgloss.Left, contentLines...)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		content,
		statusbar,
	)
}

func (m *Model) renderDependencyPanel(node *gemfile.DependencyNode, height int, isForward bool) string {
	if node == nil || node.Name == "" {
		return strings.Repeat(" \n", height)
	}

	// Get the appropriate offset for this panel
	offset := m.DetailForwardOffset
	if !isForward {
		offset = m.DetailReverseOffset
	}

	// Get all lines from the tree (this will also populate the correct lines list)
	allLines := m.renderDependencyTree(node, 9999, 0, offset, isForward)

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

func (m *Model) renderDependencyTree(node *gemfile.DependencyNode, maxLines int, depth int, offset int, isForward bool) []string {
	if node == nil || maxLines <= 0 {
		return []string{}
	}

	var lines []string
	var gemNames []string
	m.renderTreeNode(node, depth, &lines, &gemNames, maxLines, 0, offset)

	// Store gem names for later lookup in the appropriate list
	if isForward {
		m.DetailForwardLines = gemNames
	} else {
		m.DetailReverseLines = gemNames
	}

	return lines
}

func (m *Model) renderReverseDepsList(height int) string {
	if m.DependencyResult == nil || m.DependencyResult.DependencyInfo == nil {
		return strings.Repeat(" \n", height)
	}

	// Determine which gem's reverse dependencies to show
	// If viewing a dependency in the forward tree, show its reverse deps
	// Otherwise, show the originally selected gem's reverse deps
	var reverseDeps []string

	if m.DetailSection == 0 && m.DetailTreeCursor < len(m.DetailForwardLines) {
		// User is navigating in the forward dependencies tree
		// Get reverse dependencies for the currently selected dependency
		currentGemName := m.DetailForwardLines[m.DetailTreeCursor]

		// Use the AllGems map from DependencyResult to calculate reverse deps locally
		if m.DependencyResult.AllGems != nil {
			reverseDeps = gemfile.GetReverseDependencies(currentGemName, &gemfile.Gemfile{Gems: m.DependencyResult.AllGems})
		}
	} else {
		// Show the originally selected gem's reverse dependencies
		reverseDeps = m.DependencyResult.DependencyInfo.ReverseDeps
	}

	// Clear DetailReverseLines for navigation tracking
	m.DetailReverseLines = []string{}

	if len(reverseDeps) == 0 {
		noMatch := "  No gems depend on this gem"
		noMatchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))
		return noMatchStyle.Render(noMatch)
	}

	var lines []string

	for _, depName := range reverseDeps {
		// Bold gem name
		nameLine := "  " + lipgloss.NewStyle().Bold(true).Render(depName)
		lines = append(lines, nameLine)
		m.DetailReverseLines = append(m.DetailReverseLines, depName)

		// Description from AnalysisResult
		desc := ""
		if m.AnalysisResult != nil {
			for _, gemStatus := range m.AnalysisResult.GemStatuses {
				if gemStatus.Name == depName {
					desc = gemStatus.Description
					break
				}
			}
		}
		if desc != "" {
			descLine := "    " + truncateStr(desc, 50)
			descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))
			lines = append(lines, descStyle.Render(descLine))
			// Repeat gem name in DetailReverseLines for description line
			m.DetailReverseLines = append(m.DetailReverseLines, depName)
		}
	}

	// Apply offset
	offset := m.DetailReverseOffset
	if offset > len(lines) {
		offset = len(lines)
	}
	visibleLines := lines[offset:]

	// Ensure we have exactly `height` lines
	for len(visibleLines) < height {
		visibleLines = append(visibleLines, "")
	}

	return strings.Join(visibleLines[:height], "\n")
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
// Health Section Rendering
// ============================================================================

func (m *Model) renderHealthSection(health *gemfile.GemHealth, maxLen int) []string {
	var lines []string

	// Health header with score
	scoreStr := "●"
	scoreStyle := BadgeHealthyDotStyle
	switch health.Score {
	case gemfile.HealthHealthy:
		scoreStyle = BadgeHealthyDotStyle
		scoreStr = "● HEALTHY"
	case gemfile.HealthWarning:
		scoreStyle = BadgeWarningDotStyle
		scoreStr = "● WARNING"
	case gemfile.HealthCritical:
		scoreStyle = BadgeCriticalDotStyle
		scoreStr = "● CRITICAL"
	default:
		scoreStr = "? UNKNOWN"
	}

	healthHeader := "  Health: " + scoreStyle.Render(scoreStr)
	lines = append(lines, healthHeader)

	// Health details line
	var details []string

	// Last release time
	if !health.LastRelease.IsZero() {
		daysAgo := int(time.Since(health.LastRelease).Hours() / 24)
		var releaseStr string
		if daysAgo < 1 {
			releaseStr = "days ago"
		} else if daysAgo < 30 {
			releaseStr = fmt.Sprintf("%d days ago", daysAgo)
		} else if daysAgo < 365 {
			releaseStr = fmt.Sprintf("%d months ago", daysAgo/30)
		} else {
			releaseStr = fmt.Sprintf("%d years ago", daysAgo/365)
		}
		details = append(details, fmt.Sprintf("Last: %s", releaseStr))
	}

	// Stars
	if health.Stars > 0 {
		starsStr := fmt.Sprintf("⭐ %d", health.Stars)
		details = append(details, starsStr)
	}

	// Open issues
	if health.OpenIssues >= 0 {
		issuesStr := fmt.Sprintf("Issues: %d", health.OpenIssues)
		details = append(details, issuesStr)
	}

	// Archived status
	if health.Archived {
		details = append(details, "❌ Archived")
	}

	// Maintainers
	if health.MaintainerCount > 0 {
		maintStr := fmt.Sprintf("Maintainers: %d", health.MaintainerCount)
		details = append(details, maintStr)
	}

	if len(details) > 0 {
		detailsStr := "    " + strings.Join(details, "    ")
		detailsFormatted := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)).Render(detailsStr)
		lines = append(lines, detailsFormatted)
	}

	return lines
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
	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 3
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

	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 2
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
// View: Project Info
// ============================================================================

func (m *Model) viewProjectInfo() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 2
	projectContent := m.renderProjectInfo(contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		projectContent,
		statusbar,
	)
}

func (m *Model) renderProjectInfo(height int) string {
	if height < 1 {
		height = 1
	}

	title := "Project Information"
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))

	// Build info sections
	var sections []string
	sections = append(sections, titleStyle.Render(title))
	sections = append(sections, "")

	// Ruby version
	sections = append(sections, m.formatInfoLine("Ruby Version", m.RubyVersion))

	// Bundle version
	sections = append(sections, m.formatInfoLine("Bundle Version", m.BundleVersion))

	// Framework info
	if m.FrameworkDetected != "" {
		frameworkLabel := strings.ToTitle(m.FrameworkDetected)
		sections = append(sections, m.formatInfoLine(frameworkLabel+" Version", m.RailsVersion))
	}

	// Gem statistics
	sections = append(sections, "")
	sections = append(sections, lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary)).
		Render("Statistics"))
	sections = append(sections, "")

	sections = append(sections, m.formatInfoLine("Total Gems", fmt.Sprintf("%d", m.TotalGems)))
	sections = append(sections, m.formatInfoLine("Direct Dependencies", fmt.Sprintf("%d", m.FirstLevelCount)))
	sections = append(sections, m.formatInfoLine("Transitive Dependencies", fmt.Sprintf("%d", m.TransitiveDeps)))

	// Vulnerabilities summary
	if len(m.VulnerableGems) > 0 {
		sections = append(sections, "")
		vulnLabel := fmt.Sprintf("⚠ Vulnerabilities Found (%d)", len(m.VulnerableGems))
		vulnStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorDanger))
		sections = append(sections, vulnStyle.Render(vulnLabel))
	}

	// Padding to fill height
	content := strings.Join(sections, "\n")
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:height], "\n")
}

func (m *Model) formatInfoLine(label string, value string) string {
	if value == "" || value == "Unknown" {
		value = "—"
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextMuted))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorText))

	return fmt.Sprintf("  %s: %s",
		labelStyle.Render(label),
		valueStyle.Render(value))
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
// View: Filter Menu
// ============================================================================

func (m *Model) viewFilterMenu() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 2
	filterContent := m.renderFilterMenu(contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabbar,
		filterContent,
		statusbar,
	)
}

func (m *Model) renderFilterMenu(height int) string {
	if height < 1 {
		height = 1
	}

	title := "Filter Gems"
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))

	lines := []string{titleStyle.Render(title), ""}

	// Upgradable filter option
	upgradableLabel := "Show only upgradable"
	if m.ShowOnlyUpgradable {
		upgradableLabel = "☑ " + upgradableLabel
	} else {
		upgradableLabel = "☐ " + upgradableLabel
	}

	if m.FilterMenuCursor == 0 {
		lines = append(lines, RowSelectedStyle.Render("  "+upgradableLabel))
	} else {
		lines = append(lines, "  "+upgradableLabel)
	}

	lines = append(lines, "")

	// Group filter options
	if len(m.AvailableGroups) > 0 {
		groupsTitle := "Filter by group:"
		lines = append(lines, groupsTitle)

		for i, group := range m.AvailableGroups {
			label := group
			if m.SelectedGroups[group] {
				label = "☑ " + label
			} else {
				label = "☐ " + label
			}

			menuIdx := 1 + i
			if m.FilterMenuCursor == menuIdx {
				lines = append(lines, RowSelectedStyle.Render("  "+label))
			} else {
				lines = append(lines, "  "+label)
			}
		}
	}

	// Show active filters summary
	lines = append(lines, "")
	lines = append(lines, "Active filters:")

	if !m.hasActiveFilters() {
		lines = append(lines, "  (none)")
	} else {
		if m.ShowOnlyUpgradable {
			lines = append(lines, "  • Show only upgradable")
		}
		if len(m.SelectedGroups) > 0 {
			var selectedGroups []string
			for _, g := range m.AvailableGroups {
				if m.SelectedGroups[g] {
					selectedGroups = append(selectedGroups, g)
				}
			}
			lines = append(lines, fmt.Sprintf("  • Groups: %s", strings.Join(selectedGroups, ", ")))
		}
	}

	// Padding
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:height], "\n")
}

// ============================================================================
// View: Error
// ============================================================================

func (m *Model) viewError() string {
	header := m.renderAppHeader()
	tabbar := m.renderTabBar()
	statusbar := m.renderStatusBar()

	errorBox := ErrorBoxStyle.Render("ERROR\n\n" + m.ErrorMessage)
	_ = m.updateBarHeight() // Ensure update bar height is considered in layout

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
