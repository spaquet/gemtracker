package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

// statusBarTotalHeight calculates the total height of the status bar including
// all lines: hints + optional status indicators + optional update notification
func (m *Model) statusBarTotalHeight() int {
	height := 1 // Base height for hints line

	// Add height for status indicators line if any are present
	if m.OutdatedLoading || m.HealthLoading || m.OutdatedRateLimited ||
		m.HealthRateLimited || m.OutdatedErrorCount > 0 {
		height += 1
	}

	// Add height for update notification bar if present
	if m.NewVersionAvailable != "" {
		height += 1
	}

	return height
}

// placeOverlay overlays a foreground view on top of a background view at a specified row/column position.
// It uses ANSI-aware truncation to preserve the background view left of the overlay while placing
// the overlay content in the center and allowing the terminal default background to appear on the right.
func placeOverlay(startRow, startCol int, fg, bg string) string {
	fgLines := strings.Split(fg, "\n")
	bgLines := strings.Split(bg, "\n")

	for i, fgLine := range fgLines {
		row := startRow + i
		if row < 0 || row >= len(bgLines) {
			continue
		}

		// Use ANSI-aware truncation to get the left portion of the background
		left := ansi.Truncate(bgLines[row], startCol, "")
		// Replace the line with left background + foreground content
		bgLines[row] = left + fgLine
	}

	return strings.Join(bgLines, "\n")
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
	case ViewUpgradeable:
		return m.viewUpgradeable()
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

var viewHints = map[ViewMode][]string{
	ViewGemList:     {"↑↓ navigate", "enter select", "f filter", "u upgradable", "c clear", "r refresh", "tab next", "q quit"},
	ViewGemDetail:   {"esc back", "tab section", "↑↓ navigate", "enter select", "o open url", "q quit"},
	ViewSearch:      {"type search", "↑↓ navigate", "enter select", "esc clear"},
	ViewUpgradeable: {"↑↓ navigate", "enter select", "tab next", "q quit"},
	ViewCVE:         {"↑↓ navigate", "enter select", "tab next", "q quit"},
	ViewProjectInfo: {"tab next", "shift+tab prev", "q quit"},
	ViewFilterMenu:  {"↑↓ navigate", "space toggle", "enter back", "q quit"},
	ViewSelectPath:  {"enter confirm", "esc cancel"},
}

func (m *Model) getHintsForView() []string {
	if hints, ok := viewHints[m.CurrentView]; ok {
		return hints
	}
	return []string{"type to filter", "q quit"}
}

func (m *Model) renderHintLine(hints []string) string {
	var rendered []string
	for _, hint := range hints {
		parts := strings.SplitN(hint, " ", 2)
		if len(parts) == 2 {
			key := KeyHintKeyStyle.Render(parts[0])
			desc := KeyHintDescStyle.Render(" " + parts[1])
			rendered = append(rendered, key+desc)
		}
	}
	hintContent := strings.Join(rendered, "  ")
	return StatusBarStyle.Width(m.Width - 4).Render(hintContent)
}

func (m *Model) assembleViewWithChrome(contentString string) string {
	// Helper function to assemble any view with proper header, tabbar, and statusbar
	var allLines []string

	// Add header and tabbar (2 lines)
	allLines = append(allLines, m.renderAppHeader())
	allLines = append(allLines, m.renderTabBar())

	// Calculate available space for content and statusbar
	statusbarLines := m.statusBarTotalHeight()
	availableForContent := m.Height - 2 - statusbarLines
	if availableForContent < 1 {
		availableForContent = 1
	}

	// Add content (split into lines if it's a pre-joined string)
	if contentString != "" {
		contentLines := strings.Split(strings.Trim(contentString, "\n"), "\n")
		// Limit to available space
		if len(contentLines) > availableForContent {
			contentLines = contentLines[:availableForContent]
		}
		allLines = append(allLines, contentLines...)
	}

	// Pad content area to available height (before statusbar)
	contentHeight := len(allLines) - 2 // -2 for header and tabbar
	paddingNeeded := availableForContent - contentHeight
	for i := 0; i < paddingNeeded; i++ {
		allLines = append(allLines, "")
	}

	// Add status bar (can be multi-line)
	statusbarContent := m.renderStatusBar()
	if statusbarContent != "" {
		statusbarLines2 := strings.Split(strings.Trim(statusbarContent, "\n"), "\n")
		// Limit to expected statusbar height
		if len(statusbarLines2) > statusbarLines {
			statusbarLines2 = statusbarLines2[:statusbarLines]
		}
		allLines = append(allLines, statusbarLines2...)
	}

	// Final safety check - ensure we don't exceed terminal height
	if len(allLines) > m.Height {
		allLines = allLines[:m.Height]
	}

	return lipgloss.JoinVertical(lipgloss.Left, allLines...)
}

func (m *Model) renderAppHeader() string {
	appName := fmt.Sprintf("gemtracker %s", m.Version)

	// Build right side: source file info
	rightParts := []string{}

	// Add source file
	if m.GemfileSource != "" {
		if strings.HasSuffix(m.GemfileSource, ".gemspec") {
			rightParts = append(rightParts, fmt.Sprintf("Source: %s (unresolved)", m.GemfileSource))
		} else {
			rightParts = append(rightParts, fmt.Sprintf("Source: %s", m.GemfileSource))
		}
	}

	// Add project path
	projectPath := m.ProjectPath
	if projectPath == "" {
		projectPath = "(no project)"
	}
	rightParts = append(rightParts, projectPath)

	rightContent := strings.Join(rightParts, " • ")

	left := AppHeaderStyle.Render(appName)
	right := ProjectPathStyle.Render(rightContent)

	// Calculate spacing
	totalLen := lipgloss.Width(left) + lipgloss.Width(right)
	spacerCount := m.Width - totalLen
	if spacerCount < 0 {
		spacerCount = 0
	}
	spacer := strings.Repeat(" ", spacerCount)

	// Apply background to spacer to fill full width
	headerStyle := lipgloss.NewStyle().Background(lipgloss.Color(ColorSurface))
	headerSpaceFill := headerStyle.Render(spacer)

	return left + headerSpaceFill + right
}

func (m *Model) renderTabBar() string {
	tabLabels := []string{"Gems", "Search", "Updates", "CVE", "Project"}
	tabModes := []ViewMode{ViewGemList, ViewSearch, ViewUpgradeable, ViewCVE, ViewProjectInfo}

	var tabs []string
	for i, label := range tabLabels {
		mode := tabModes[i]
		// Add count badges
		if mode == ViewUpgradeable {
			upgradableCount := len(m.UpgradeableGems) + len(m.UpgradeableFrameworkGems) + len(m.UpgradeableTransitiveDeps)
			if upgradableCount > 0 {
				label = fmt.Sprintf("%s (%d)", label, upgradableCount)
			}
		} else if mode == ViewCVE && len(m.VulnerableGems) > 0 {
			label = fmt.Sprintf("%s (%d)", label, len(m.VulnerableGems))
		}
		if mode == m.ActiveTab {
			tabs = append(tabs, TabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, TabStyle.Render(label))
		}
	}

	tabContent := strings.Join(tabs, "  ")
	tabWidth := lipgloss.Width(tabContent)

	// Fill remaining width with background
	if tabWidth < m.Width {
		fillStyle := lipgloss.NewStyle().Background(lipgloss.Color(ColorSurface))
		fillWidth := m.Width - tabWidth
		fill := fillStyle.Render(strings.Repeat(" ", fillWidth))
		tabContent = tabContent + fill
	}

	return tabContent
}

func (m *Model) renderStatusBar() string {
	hints := m.getHintsForView()
	hintLine := m.renderHintLine(hints)

	var lines []string
	lines = append(lines, hintLine)

	// Build status indicators on a separate line if needed
	var statusParts []string

	if m.OutdatedLoading {
		doneCount := len(m.FirstLevelGems) - len(m.OutdatedPending)
		outdatedStatus := fmt.Sprintf("Checking updates... (%d/%d)", doneCount, len(m.FirstLevelGems))
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
		statusParts = append(statusParts, statusStyle.Render(outdatedStatus))
	}

	if m.HealthLoading {
		healthStatus := fmt.Sprintf("Fetching health... (%d/%d)", m.HealthLoadedCount, m.HealthTotalCount)
		healthStatusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
		statusParts = append(statusParts, healthStatusStyle.Render(healthStatus))
	}

	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDanger))
	if m.OutdatedRateLimited {
		statusParts = append(statusParts, errorStyle.Render("updates: rate limited"))
	}
	if m.HealthRateLimited {
		statusParts = append(statusParts, errorStyle.Render("health: rate limited"))
	}
	if m.OutdatedErrorCount > 0 {
		errMsg := fmt.Sprintf("%d update errors", m.OutdatedErrorCount)
		statusParts = append(statusParts, errorStyle.Render(errMsg))
	}

	if len(statusParts) > 0 {
		statusContent := strings.Join(statusParts, "  ")
		statusLine := StatusBarStyle.Width(m.Width - 4).Render(statusContent)
		lines = append(lines, statusLine)
	}

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
	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines
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

	return m.assembleViewWithChrome(content)
}

// ============================================================================
// View: Gem List
// ============================================================================

func (m *Model) viewGemList() string {
	statusbarHeight := m.statusBarTotalHeight()
	if statusbarHeight < 1 {
		statusbarHeight = 1
	}
	// Reserve 1 line for footer/statusbar to prevent clipping the last gem
	contentHeight := m.Height - 2 - statusbarHeight - 1
	if contentHeight < 1 {
		contentHeight = 1
	}
	gemContent := m.renderGemListTable(contentHeight)

	return m.assembleViewWithChrome(gemContent)
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

	// Table rows - don't reserve space for padding, show as many gems as will fit
	// The wrapper will pad the content area to fill the terminal
	maxGems := height - len(lines)
	if maxGems < 0 {
		maxGems = 0
	}

	endIdx := m.GemListOffset + maxGems
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

	// Don't pad - let wrapper handle layout
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) gemStatusBadge(gem *gemfile.GemStatus) string {
	if gem.IsVulnerable {
		return BadgeVulnerableStyle.Render("⚠ CVE")
	}
	if gem.IsOutdated {
		return BadgeOutdatedStyle.Render("↑ " + gem.LatestVersion)
	}
	if gem.OutdatedFailed {
		return BadgeErrorStyle.Render("! err")
	}
	if gem.LatestVersion == "" {
		return BadgeLoadingStyle.Render("…")
	}
	return BadgeOKStyle.Render("✓")
}

func (m *Model) formatGemListRow(idx int, gem *gemfile.GemStatus, selected bool) string {
	// Status indicator
	status := m.gemStatusBadge(gem)

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

// buildGemInfoLines builds the header, description, and health info lines for gem detail view
func (m *Model) buildGemInfoLines(descMaxLen int) []string {
	// Format version info
	versionDisplay := "Latest"
	if m.SelectedGem.IsOutdated {
		versionDisplay = m.SelectedGem.LatestVersion
	}

	// Build header line
	updateMarker := ""
	if m.SelectedGem.IsOutdated {
		updateMarker = " (update available)"
	}
	headerLine1 := fmt.Sprintf("%s   Installed: %s  →  %s%s",
		m.SelectedGem.Name,
		m.SelectedGem.Version,
		versionDisplay,
		updateMarker,
	)
	headerLine1Formatted := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimary)).Render(headerLine1)

	var gemInfoLines []string
	gemInfoLines = append(gemInfoLines, headerLine1Formatted)

	// Format description line
	if m.SelectedGem.Description != "" {
		descLine := truncateStr(m.SelectedGem.Description, descMaxLen)
		descLine = "  " + descLine
		descLine = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)).Render(descLine)
		gemInfoLines = append(gemInfoLines, descLine)
	}

	// URL line
	urlLine := "  " + truncateStr(m.SelectedGem.HomepageURL, descMaxLen)
	urlLine = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)).Italic(true).Render(urlLine)
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

	return gemInfoLines
}

// ============================================================================
// View: Gem Detail
// ============================================================================

func (m *Model) viewGemDetail() string {
	if m.SelectedGem == nil {
		return ""
	}

	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Format description line
	descMaxLen := m.Width - 4
	if descMaxLen < 20 {
		descMaxLen = 20
	}

	gemInfoLines := m.buildGemInfoLines(descMaxLen)

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

	return m.assembleViewWithChrome(content)
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
	var scoreStr string
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

	// Search results - account for header (1), tabbar (1), searchLine (1), and statusbar (1-2)
	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 3 - statusbarLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	resultContent := m.renderSearchResults(contentHeight)

	content := lipgloss.JoinVertical(lipgloss.Left,
		searchLine,
		resultContent,
	)

	return m.assembleViewWithChrome(content)
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

	return lipgloss.JoinVertical(lipgloss.Left, lines[:height]...)
}

// ============================================================================
// View: Upgradeable
// ============================================================================

func (m *Model) viewUpgradeable() string {
	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	upgradeContent := m.renderUpgradeableTable(contentHeight)

	return m.assembleViewWithChrome(upgradeContent)
}

func (m *Model) renderUpgradeableTable(height int) string {
	if height < 1 {
		height = 1
	}

	allUpgradeable := m.allUpgradeableGems()
	if len(allUpgradeable) == 0 {
		msg := "All gems are up to date! ✓"
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true).
			Padding(2, 2).
			Render(msg)
	}

	// Build all visible lines first, then apply offset
	var allLines []string

	// Add top spacing
	allLines = append(allLines, "")

	lineIndex := 0 // Track line number for cursor comparison

	// First-level gems section
	if len(m.UpgradeableGems) > 0 {
		allLines = append(allLines, lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary)).
			Render("DIRECT DEPENDENCIES"))

		headerRow := fmt.Sprintf("  %-24s %-11s %-11s %s",
			"Gem Name", "Installed", "Latest", "")
		header := TableHeaderStyle.Render(headerRow)
		allLines = append(allLines, header)

		for _, gem := range m.UpgradeableGems {
			isSelected := lineIndex == m.UpgradeableCursor
			row := fmt.Sprintf("  %-24s %-11s %-11s %s",
				truncateStr(gem.Name, 24),
				gem.Version,
				gem.LatestVersion,
				BadgeOutdatedStyle.Render("↑"),
			)
			if isSelected {
				row = RowSelectedStyle.Render(row)
			} else {
				row = RowNormalStyle.Render(row)
			}
			allLines = append(allLines, row)
			lineIndex++
		}
		allLines = append(allLines, "")
		lineIndex++
	}

	// Framework gems section
	if len(m.UpgradeableFrameworkGems) > 0 {
		allLines = append(allLines, lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary)).
			Render("FRAMEWORK COMPONENTS"))

		headerRow := fmt.Sprintf("  %-24s %-11s %-11s %s",
			"Gem Name", "Installed", "Latest", "")
		header := TableHeaderStyle.Render(headerRow)
		allLines = append(allLines, header)

		for _, gem := range m.UpgradeableFrameworkGems {
			isSelected := lineIndex == m.UpgradeableCursor
			row := fmt.Sprintf("  %-24s %-11s %-11s %s",
				truncateStr(gem.Name, 24),
				gem.Version,
				gem.LatestVersion,
				BadgeOutdatedStyle.Render("↑"),
			)
			if isSelected {
				row = RowSelectedStyle.Render(row)
			} else {
				row = RowNormalStyle.Render(row)
			}
			allLines = append(allLines, row)
			lineIndex++
		}
		allLines = append(allLines, "")
		lineIndex++
	}

	// Transitive dependencies section
	if len(m.UpgradeableTransitiveDeps) > 0 {
		allLines = append(allLines, lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary)).
			Render("TRANSITIVE DEPENDENCIES"))

		headerRow := fmt.Sprintf("  %-24s %-11s %-11s %s",
			"Gem Name", "Installed", "Latest", "")
		header := TableHeaderStyle.Render(headerRow)
		allLines = append(allLines, header)

		for _, gem := range m.UpgradeableTransitiveDeps {
			isSelected := lineIndex == m.UpgradeableCursor
			row := fmt.Sprintf("  %-24s %-11s %-11s %s",
				truncateStr(gem.Name, 24),
				gem.Version,
				gem.LatestVersion,
				BadgeOutdatedStyle.Render("↑"),
			)
			if isSelected {
				row = RowSelectedStyle.Render(row)
			} else {
				row = RowNormalStyle.Render(row)
			}
			allLines = append(allLines, row)
			lineIndex++
		}
	}

	// Apply offset and return visible lines - don't pad, let wrapper handle layout
	visibleLines := allLines[m.UpgradeableOffset:]
	return lipgloss.JoinVertical(lipgloss.Left, visibleLines...)
}

// ============================================================================
// View: CVE
// ============================================================================

func (m *Model) viewCVE() string {
	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	cveContent := m.renderCVETable(contentHeight)

	return m.assembleViewWithChrome(cveContent)
}

func (m *Model) renderCVETable(height int) string {
	if height < 1 {
		height = 1
	}

	// If no vulnerabilities found and not refreshing, show clean state
	if len(m.CVEVulnerabilities) == 0 && !m.CVERefreshInProgress {
		msg := "No vulnerabilities found. Your gems are safe! ✓"
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true).
			Padding(2, 2).
			Render(msg)
	}

	// Build header section with severity summary and cache status
	headerSection := m.renderCVEHeader(height)

	// Calculate available space after header
	headerLines := strings.Count(headerSection, "\n") + 1
	remainingHeight := height - headerLines

	if remainingHeight < 1 {
		return headerSection
	}

	// Build vulnerability list
	vulnList := m.renderCVEVulnerabilitiesList(remainingHeight)

	// Combine header and list - don't pad here, let the wrapper handle it
	return lipgloss.JoinVertical(lipgloss.Left, headerSection, vulnList)
}

func (m *Model) renderCVEHeader(maxHeight int) string {
	lines := []string{}

	// Count vulnerabilities by severity
	critCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for _, vuln := range m.CVEVulnerabilities {
		switch vuln.Severity {
		case "CRITICAL":
			critCount++
		case "HIGH":
			highCount++
		case "MEDIUM":
			mediumCount++
		case "LOW":
			lowCount++
		}
	}

	// Severity summary line with colors
	severityLine := fmt.Sprintf(
		"  Severity: %s CRITICAL (%d)  %s HIGH (%d)  %s MEDIUM (%d)  %s LOW (%d)",
		BadgeCriticalDotStyle.Render("●"), critCount,
		BadgeCriticalDotStyle.Render("●"), highCount,
		BadgeWarningDotStyle.Render("●"), mediumCount,
		BadgeHealthyDotStyle.Render("●"), lowCount,
	)
	lines = append(lines, severityLine)

	// Cache status line
	cacheStatusParts := []string{}

	if m.CVEVulnerabilities != nil && len(m.CVEVulnerabilities) > 0 {
		// Show cache age
		if !m.CVECacheLoadedAt.IsZero() {
			cacheAge := time.Since(m.CVECacheLoadedAt)
			cacheAgeStr := formatDuration(cacheAge)
			cacheStatusParts = append(cacheStatusParts, fmt.Sprintf("Cache: %s old", cacheAgeStr))

			// Show TTL countdown
			remaining := m.CVECacheTTL - cacheAge
			if remaining > 0 {
				remainingStr := formatDuration(remaining)
				cacheStatusParts = append(cacheStatusParts, fmt.Sprintf("expires in %s", remainingStr))
			} else {
				cacheStatusParts = append(cacheStatusParts, "(expired)")
			}
		}

		// Show last scan time
		if !m.CVELastScanTime.IsZero() {
			scanAge := time.Since(m.CVELastScanTime)
			scanAgeStr := formatDuration(scanAge)
			cacheStatusParts = append(cacheStatusParts, fmt.Sprintf("last scanned: %s ago", scanAgeStr))
		}

		// Show gem count scanned
		if m.AnalysisResult != nil && len(m.AnalysisResult.AllGems) > 0 {
			cacheStatusParts = append(cacheStatusParts, fmt.Sprintf("%d gems scanned", len(m.AnalysisResult.AllGems)))
		}
	}

	if len(cacheStatusParts) > 0 {
		cacheLine := "  " + strings.Join(cacheStatusParts, " · ")
		lines = append(lines, cacheLine)
	}

	// Refresh progress line (if refreshing)
	if m.CVERefreshInProgress {
		refreshLine := "  🔄 Refreshing vulnerabilities in background..."
		lines = append(lines, refreshLine)
	}

	// Error message if last scan failed
	if m.CVELastError != "" && !m.CVERefreshInProgress {
		errorLine := fmt.Sprintf("  ⚠️  Could not fetch latest data: %s", m.CVELastError)
		lines = append(lines, errorLine)
	}

	return strings.Join(lines, "\n")
}

func (m *Model) renderCVEVulnerabilitiesList(height int) string {
	if len(m.CVEVulnerabilities) == 0 {
		if m.CVERefreshInProgress {
			return "  ⏳ Scanning for vulnerabilities..."
		}
		return ""
	}

	lines := []string{}

	// Table header
	headerRow := fmt.Sprintf("  %-18s %-14s %-12s %-30s",
		"CVE ID", "Gem", "Severity", "Description")
	lines = append(lines, TableHeaderStyle.Render(headerRow))

	// Render vulnerabilities - don't reserve space for padding
	maxVulns := height - 1 // -1 for header
	if maxVulns < 0 {
		maxVulns = 0
	}

	endIdx := m.CVEOffset + maxVulns
	if endIdx > len(m.CVEVulnerabilities) {
		endIdx = len(m.CVEVulnerabilities)
	}

	for i := m.CVEOffset; i < endIdx; i++ {
		if i >= len(m.CVEVulnerabilities) {
			break
		}

		vuln := m.CVEVulnerabilities[i]

		// Get severity badge style
		severityBadge := ""
		switch vuln.Severity {
		case "CRITICAL", "HIGH":
			severityBadge = BadgeCriticalDotStyle.Render("●")
		case "MEDIUM":
			severityBadge = BadgeWarningDotStyle.Render("●")
		default:
			severityBadge = BadgeHealthyDotStyle.Render("●")
		}

		row := fmt.Sprintf("  %-18s %-14s %s %-12s %-30s",
			truncateStr(vuln.CVE, 18),
			truncateStr(vuln.GemName, 14),
			severityBadge,
			vuln.Severity,
			truncateStr(vuln.Description, 30),
		)

		if i == m.CVECursor {
			row = RowSelectedStyle.Render(row)
		} else {
			row = RowNormalStyle.Render(row)
		}
		lines = append(lines, row)
	}

	// Don't pad - let wrapper handle layout
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// formatDuration converts a duration to a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// ============================================================================
// View: Project Info
// ============================================================================

func (m *Model) viewProjectInfo() string {
	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	projectContent := m.renderProjectInfo(contentHeight)

	return m.assembleViewWithChrome(projectContent)
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
	lines := sections
	for len(lines) < height {
		lines = append(lines, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines[:height]...)
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

	return m.assembleViewWithChrome(content)
}

// ============================================================================
// View: Filter Menu
// ============================================================================

func (m *Model) viewFilterMenu() string {
	// Render gem list as background
	background := m.viewGemList()

	// Create filter modal
	modal := m.renderFilterModalBox()

	// Calculate centered position
	modalLines := strings.Split(modal, "\n")
	modalH := len(modalLines)
	modalW := lipgloss.Width(modal)

	startRow := (m.Height - modalH) / 2
	startCol := (m.Width - modalW) / 2

	if startRow < 2 {
		startRow = 2 // Don't cover header
	}
	if startCol < 0 {
		startCol = 0
	}

	// Overlay modal on background
	return placeOverlay(startRow, startCol, modal, background)
}

func (m *Model) renderFilterModalBox() string {
	// Create checkbox styles
	checkOn := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorSuccess)).
		Render("[✓]")
	checkOff := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextMuted)).
		Render("[ ]")

	// Helper to choose checkbox
	checkbox := func(on bool) string {
		if on {
			return checkOn
		}
		return checkOff
	}

	// Build content lines
	lines := []string{}

	// Title
	title := "Filter Gems"
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, "")

	// Upgradable filter option
	upgradableLabel := "Show only upgradable"
	upgradableLine := checkbox(m.ShowOnlyUpgradable) + " " + upgradableLabel
	if m.FilterMenuCursor == 0 {
		lines = append(lines, RowSelectedStyle.Render("› "+upgradableLine))
	} else {
		lines = append(lines, "  "+upgradableLine)
	}

	lines = append(lines, "")

	// Group filter options
	if len(m.AvailableGroups) > 0 {
		lines = append(lines, "Filter by group:")

		for i, group := range m.AvailableGroups {
			groupLine := checkbox(m.SelectedGroups[group]) + " " + group
			menuIdx := 1 + i
			if m.FilterMenuCursor == menuIdx {
				lines = append(lines, RowSelectedStyle.Render("› "+groupLine))
			} else {
				lines = append(lines, "  "+groupLine)
			}
		}
	}

	// Active filters summary
	lines = append(lines, "")
	lines = append(lines, "Active filters:")

	if !m.hasActiveFilters() {
		lines = append(lines, "  (none)")
	} else {
		if m.ShowOnlyUpgradable {
			lines = append(lines, "  • Upgradable only")
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

	// Footer hint
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextMuted)).
		Italic(true)
	lines = append(lines, hintStyle.Render("↑↓ navigate  space toggle  enter/esc close"))

	// Create the modal box with border
	content := strings.Join(lines, "\n")

	// Calculate width - use enough space but not too much
	modalWidth := lipgloss.Width(content) + 4 // 2 for padding left/right, 2 for border
	if modalWidth < 50 {
		modalWidth = 50
	}
	if modalWidth > m.Width-4 {
		modalWidth = m.Width - 4
	}

	// Apply border and styling
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBorderActive)).
		Background(lipgloss.Color(ColorSurface)).
		Padding(1, 2)

	return boxStyle.Width(modalWidth).Render(content)
}

// ============================================================================
// View: Error
// ============================================================================

func (m *Model) viewError() string {
	errorBox := ErrorBoxStyle.Render("ERROR\n\n" + m.ErrorMessage)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		errorBox,
		"",
		"Press Enter or Esc to continue",
	)

	return m.assembleViewWithChrome(content)
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

func pluralizeGem(count int) string {
	if count == 1 {
		return "gem"
	}
	return "gems"
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
