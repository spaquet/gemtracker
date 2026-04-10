package ui

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/spaquet/gemtracker/internal/gemfile"
)

// ============================================================================
// Helper Methods
// ============================================================================

// wrapText wraps a string to the specified width, maintaining word boundaries
func wrapText(text string, width int) []string {
	var result []string
	words := strings.Fields(text)
	var currentLine string

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			if currentLine != "" {
				result = append(result, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		result = append(result, currentLine)
	}

	return result
}

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
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC in placeOverlay: %v\n", r)
		}
	}()

	if startRow < 0 || startCol < 0 {
		return bg // Invalid positioning, return background
	}

	fgLines := strings.Split(fg, "\n")
	bgLines := strings.Split(bg, "\n")

	if len(bgLines) == 0 {
		return fg // No background, return foreground
	}

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
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC in View(): %v\n", r)
		}
	}()

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
	case ViewSanity:
		return m.viewSanity()
	case ViewProjectInfo:
		return m.viewProjectInfo()
	case ViewFilterMenu:
		return m.viewFilterMenu()
	case ViewCVEFilterMenu:
		return m.viewCVEFilterMenu()
	case ViewCVEInfo:
		return m.viewCVEInfo()
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
	ViewGemList:       {"↑↓ navigate", "enter select", "f filter", "u upgradable", "c clear", "r refresh", "tab next", "q quit"},
	ViewGemDetail:     {"esc back", "tab section", "↑↓ navigate", "enter select", "o open url", "q quit"},
	ViewSearch:        {"type search", "↑↓ navigate", "enter select", "esc clear"},
	ViewUpgradeable:   {"↑↓ navigate", "enter select", "tab next", "q quit"},
	ViewCVE:           {"↑↓ navigate", "enter select", "f filter", "i info", "tab next", "q quit"},
	ViewSanity:        {"↑↓ navigate", "enter select", "i info", "tab next", "q quit"},
	ViewProjectInfo:   {"tab next", "shift+tab prev", "q quit"},
	ViewFilterMenu:    {"↑↓ navigate", "space toggle", "enter back", "q quit"},
	ViewCVEFilterMenu: {"↑↓ navigate", "space toggle", "enter back", "q quit"},
	ViewCVEInfo:       {"esc close", "q quit"},
	ViewSelectPath:    {"enter confirm", "esc cancel"},
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
	tabLabels := []string{"Gems", "Search", "Updates", "CVE", "Sanity", "Project"}
	tabModes := []ViewMode{ViewGemList, ViewSearch, ViewUpgradeable, ViewCVE, ViewSanity, ViewProjectInfo}

	var tabs []string
	for i, label := range tabLabels {
		mode := tabModes[i]
		// Add count badges
		if mode == ViewUpgradeable {
			upgradableCount := len(m.UpgradeableGems) + len(m.UpgradeableFrameworkGems) + len(m.UpgradeableTransitiveDeps)
			if upgradableCount > 0 {
				label = fmt.Sprintf("%s (%d)", label, upgradableCount)
			}
		} else if mode == ViewCVE && len(m.CVEVulnerabilities) > 0 {
			label = fmt.Sprintf("%s (%d)", label, len(m.CVEVulnerabilities))
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

	var lines []string

	// Add top spacing
	lines = append(lines, "")

	// Map gem index to section info
	// Track which section each gem index belongs to
	directCount := len(m.UpgradeableGems)
	frameworkCount := len(m.UpgradeableFrameworkGems)

	// Track which section we're currently rendering
	var lastSection string

	// Render gems starting from UpgradeableOffset
	for gemIdx := m.UpgradeableOffset; gemIdx < len(allUpgradeable); gemIdx++ {
		if len(lines) >= height {
			break
		}

		// Determine which section this gem belongs to
		var currentSection string
		if gemIdx < directCount {
			currentSection = "DIRECT DEPENDENCIES"
		} else if gemIdx < directCount+frameworkCount {
			currentSection = "FRAMEWORK COMPONENTS"
		} else {
			currentSection = "TRANSITIVE DEPENDENCIES"
		}

		// Add section header when entering a new section
		if currentSection != lastSection {
			// Add blank line before section (except for the very first section)
			if lastSection != "" && len(lines) < height {
				lines = append(lines, "")
			}

			if len(lines) < height {
				lines = append(lines, lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color(ColorPrimary)).
					Render(currentSection))
			}

			if len(lines) < height {
				headerRow := fmt.Sprintf("  %-24s %-11s %-11s %s",
					"Gem Name", "Installed", "Latest", "")
				header := TableHeaderStyle.Render(headerRow)
				lines = append(lines, header)
			}

			lastSection = currentSection
		}

		if len(lines) >= height {
			break
		}

		gem := allUpgradeable[gemIdx]
		isSelected := gemIdx == m.UpgradeableCursor
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
		lines = append(lines, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// View: CVE
// ============================================================================

func (m *Model) viewCVE() string {
	statusbarLines := m.statusBarTotalHeight()
	// Reserve 1 line for footer/statusbar to prevent clipping the last CVE
	contentHeight := m.Height - 2 - statusbarLines - 1
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
		case "MODERATE":
			mediumCount++
		case "LOW":
			lowCount++
		}
	}

	// Severity summary line with colors
	severityLine := fmt.Sprintf(
		"  Severity: %s CRITICAL (%d)  %s HIGH (%d)  %s MODERATE (%d)  %s LOW (%d)",
		BadgeCriticalDotStyle.Render("●"), critCount,
		BadgeHighDotStyle.Render("●"), highCount,
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

// getCVEGemInfo returns the type (Direct/Transitive) and group for a gem in a vulnerability
func (m *Model) getCVEGemInfo(gemName string) (gemType string, group string) {
	// Check if gem is in first-level gems (direct dependency)
	for _, gem := range m.FirstLevelGems {
		if gem.Name == gemName {
			gemType = "Direct"
			if len(gem.Groups) > 0 {
				group = strings.Join(gem.Groups, ",")
			} else {
				group = "default"
			}
			return
		}
	}
	// Not found in first-level, so it's transitive
	// For transitive gems, try to find which first-level gems depend on it
	// and show their groups as context
	gemType = "Transitive"
	parentGems := m.findParentGems(gemName)
	if len(parentGems) > 0 {
		// Collect groups from all parent gems
		groupsMap := make(map[string]bool)
		for _, parentName := range parentGems {
			for _, gem := range m.FirstLevelGems {
				if gem.Name == parentName && len(gem.Groups) > 0 {
					for _, g := range gem.Groups {
						groupsMap[g] = true
					}
				}
			}
		}
		if len(groupsMap) > 0 {
			// Sort and join groups
			var groupList []string
			for g := range groupsMap {
				groupList = append(groupList, g)
			}
			sort.Strings(groupList)
			group = strings.Join(groupList, ",")
		} else {
			group = "default"
		}
	} else {
		group = "—"
	}
	return
}

// isFrameworkGem checks if a gem is part of a known framework
func (m *Model) isFrameworkGem(gemName string) bool {
	_, isFramework := frameworkGems[gemName]
	return isFramework
}

func (m *Model) renderCVEVulnerabilitiesList(height int) string {
	if len(m.CVEVulnerabilities) == 0 {
		if m.CVERefreshInProgress {
			return "  ⏳ Scanning for vulnerabilities..."
		}
		return ""
	}

	lines := []string{}

	// Table header: # | CVE ID | Gem | ● Severity | Type | Group
	headerRow := fmt.Sprintf("  %3s %-18s %-14s %s %-12s %-10s %-15s",
		"#", "CVE ID", "Gem", " ", "Severity", "Type", "Group")
	lines = append(lines, TableHeaderStyle.Render(headerRow))

	// Calculate how many vulnerabilities can fit (like Gems tab does)
	// height is already adjusted by caller, maxVulns = height - lines_already_added
	maxVulns := height - len(lines)
	if maxVulns < 0 {
		maxVulns = 0
	}

	// Calculate end index
	endIdx := m.CVEOffset + maxVulns
	if endIdx > len(m.CVEVulnerabilities) {
		endIdx = len(m.CVEVulnerabilities)
	}

	for i := m.CVEOffset; i < endIdx; i++ {
		if i >= len(m.CVEVulnerabilities) {
			break
		}

		vuln := m.CVEVulnerabilities[i]
		isSelected := i == m.CVECursor
		rowNum := i + 1 // 1-based line number

		line := m.formatCVERow(vuln, isSelected, rowNum)
		lines = append(lines, line)
	}

	// Don't pad - let wrapper handle layout
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// formatCVERow formats a single CVE vulnerability row - mirrors formatGemListRow pattern
func (m *Model) formatCVERow(vuln *gemfile.Vulnerability, selected bool, rowNum int) string {
	// Get severity badge color
	var severityBadge string
	switch vuln.Severity {
	case "CRITICAL":
		severityBadge = BadgeCriticalDotStyle.Render("●")
	case "HIGH":
		severityBadge = BadgeHighDotStyle.Render("●")
	case "MODERATE":
		severityBadge = BadgeWarningDotStyle.Render("●")
	default:
		severityBadge = BadgeHealthyDotStyle.Render("●")
	}

	// Standardize badge width to prevent ANSI codes from breaking fmt.Sprintf
	severityBadge = fmt.Sprintf("%s", severityBadge)

	// Get gem type (Direct/Transitive) and group
	gemType, group := m.getCVEGemInfo(vuln.GemName)

	// Add framework tag to gem name if applicable
	gemDisplay := vuln.GemName
	if m.isFrameworkGem(vuln.GemName) {
		gemDisplay = gemDisplay + " [fw]"
	}

	// Build plain text row matching Gems tab pattern
	row := fmt.Sprintf("  %3d %-18s %-14s %s %-12s %-10s %-15s",
		rowNum,
		truncateStr(vuln.CVE, 18),
		truncateStr(gemDisplay, 14),
		severityBadge,
		vuln.Severity,
		gemType,
		truncateStr(group, 15),
	)

	// Apply selection styling - mirrors formatGemListRow
	if selected {
		return RowSelectedStyle.Render(row)
	}
	return RowNormalStyle.Render(row)
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
// View: Sanity (Gem Sizes)
// ============================================================================

func (m *Model) viewSanity() string {
	if m.ShowingGemInfo {
		return m.viewGemInfoModal()
	}

	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	sanityContent := m.renderSanityTable(contentHeight)

	return m.assembleViewWithChrome(sanityContent)
}

func (m *Model) renderSanityTable(height int) string {
	if height < 1 {
		height = 1
	}

	// Show loading state
	if m.SanityLoading {
		msg := "Checking gem sizes..."
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)).
			Padding(2, 2).
			Render(msg)
	}

	// Show error if gem dir not found
	if m.GemDirPath == "" {
		msg := "Unable to detect gem directory. Make sure Ruby is properly installed."
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDanger)).
			Padding(2, 2).
			Render(msg)
	}

	allGems := m.allGemsForSanity()
	if len(allGems) == 0 {
		msg := "No gems found."
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)).
			Padding(2, 2).
			Render(msg)
	}

	var lines []string

	// Add header with Ruby manager and total size
	lines = append(lines, "")
	headerLine := fmt.Sprintf("Ruby Manager: %s  |  Total Size: %s",
		m.RubyManager,
		gemfile.FormatBytes(m.ProjectTotalSizeBytes))
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPrimary)).
		Bold(true)
	lines = append(lines, headerStyle.Render(headerLine))
	lines = append(lines, "")

	// Pre-calculate section counts for index-based determination
	directCount := len(m.FirstLevelGems)

	// Track which section we're currently rendering
	var lastSection string

	// Render gems starting from SanityOffset
	for gemIdx := m.SanityOffset; gemIdx < len(allGems); gemIdx++ {
		if len(lines) >= height {
			break
		}

		// Determine which section this gem belongs to based on index
		var currentSection string
		if gemIdx < directCount {
			currentSection = "DIRECT DEPENDENCIES"
		} else {
			currentSection = "TRANSITIVE DEPENDENCIES"
		}

		// Add section header when entering a new section
		if currentSection != lastSection {
			// Add blank line before section (except for the very first section)
			if lastSection != "" && len(lines) < height {
				lines = append(lines, "")
			}

			if len(lines) < height {
				lines = append(lines, lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color(ColorPrimary)).
					Render(currentSection))
			}

			if len(lines) < height {
				headerRow := fmt.Sprintf("  %-3s %-24s %-11s %s",
					"ID", "Gem Name", "Installed", "Size")
				header := TableHeaderStyle.Render(headerRow)
				lines = append(lines, header)
			}

			lastSection = currentSection
		}

		if len(lines) >= height {
			break
		}

		gem := allGems[gemIdx]
		size := m.GemSizes[gem.Name]
		sizeStr := gemfile.FormatBytes(size)

		isSelected := gemIdx == m.SanityCursor
		row := fmt.Sprintf("  %-3d %-24s %-11s %s",
			gemIdx+1, // Display ID: 1-based index
			truncateStr(gem.Name, 24),
			gem.Version,
			sizeStr,
		)
		if isSelected {
			row = RowSelectedStyle.Render(row)
		} else {
			row = RowNormalStyle.Render(row)
		}
		lines = append(lines, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) viewGemInfoModal() string {
	// Render Sanity list as background (directly call renderSanityTable to avoid recursion)
	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	sanityContent := m.renderSanityTable(contentHeight)
	background := m.assembleViewWithChrome(sanityContent)

	// Create info modal
	modal := m.renderGemInfoModalBox()

	// Safety checks for invalid terminal size
	if m.Height <= 0 || m.Width <= 0 {
		return background // Can't render modal if terminal is invalid
	}

	// Calculate centered position
	modalLines := strings.Split(modal, "\n")
	modalH := len(modalLines)

	// Calculate width safely, with fallback to reasonable default
	modalW := lipgloss.Width(modal)
	if modalW <= 0 {
		modalW = 80 // Fallback to default width
	}

	// Limit modal height to prevent exceeding screen bounds
	maxModalH := m.Height - 4
	if maxModalH < 10 {
		maxModalH = 10
	}
	if modalH > maxModalH {
		modalH = maxModalH
	}

	// Safety check to prevent division issues
	if modalH <= 0 || modalW <= 0 {
		return background
	}

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

func (m *Model) renderGemInfoModalBox() string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC in renderGemInfoModalBox: %v\n", r)
		}
	}()

	allGems := m.allGemsForSanity()
	if len(allGems) == 0 || m.SanityCursor < 0 || m.SanityCursor >= len(allGems) {
		return ""
	}

	gem := allGems[m.SanityCursor]
	if gem == nil {
		return ""
	}

	// Build content lines
	lines := []string{}

	// Title
	title := fmt.Sprintf("Gem Info: %s", gem.Name)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, "")

	// Gem details
	lines = append(lines, fmt.Sprintf("Name: %s", gem.Name))
	lines = append(lines, fmt.Sprintf("Version: %s", gem.Version))

	size := m.GemSizes[gem.Name]
	lines = append(lines, fmt.Sprintf("Size: %s", gemfile.FormatBytes(size)))
	lines = append(lines, "")

	// Show gem type
	isDirect := false
	for _, direct := range m.FirstLevelGems {
		if direct.Name == gem.Name {
			isDirect = true
			break
		}
	}

	if isDirect {
		lines = append(lines, "Type: Direct Dependency")
	} else {
		lines = append(lines, "Type: Transitive Dependency")
	}

	// Show description if available
	if gem.Description != "" {
		lines = append(lines, "")
		lines = append(lines, "Description:")
		// Wrap description text
		descLines := wrapText(gem.Description, 60)
		for _, line := range descLines {
			lines = append(lines, fmt.Sprintf("  %s", line))
		}
	}

	// Show installed versions and paths
	lines = append(lines, "")
	lines = append(lines, "Installed Versions:")

	if m.GemInfoLoading {
		// Show loading indicator while fetching
		loadingFrame := spinnerFrames[m.AnimationFrame%len(spinnerFrames)]
		lines = append(lines, fmt.Sprintf("  %s Fetching version info...", loadingFrame))
	} else if m.ParsedGemInfo != nil && len(m.ParsedGemInfo.Versions) > 0 {
		// Display parsed versions and paths
		for _, ver := range m.ParsedGemInfo.Versions {
			versionLine := fmt.Sprintf("  %-8s  %s", ver.Version, ver.Path)
			// Truncate long paths if needed to fit in modal
			if len(versionLine) > 76 {
				versionLine = versionLine[:73] + "..."
			}
			lines = append(lines, versionLine)
		}
	} else if m.CurrentGemInfoOutput != "" {
		// Fallback: show that fetch completed but no versions were found
		lines = append(lines, "  (no versions found)")
	} else {
		// No data yet
		lines = append(lines, "  —")
	}

	// Apply scrolling if content exceeds available modal height
	// Calculate available height for content (modal height minus borders/padding)
	maxModalHeight := m.Height - 6 // Leave room for header, footer, and padding
	if maxModalHeight < 5 {
		maxModalHeight = 5
	}

	// Ensure scroll offset doesn't exceed content
	if m.GemInfoScrollOffset >= len(lines) {
		m.GemInfoScrollOffset = len(lines) - 1
	}
	if m.GemInfoScrollOffset < 0 {
		m.GemInfoScrollOffset = 0
	}

	// Slice lines based on scroll offset
	visibleLines := lines
	if len(lines) > maxModalHeight {
		endIdx := m.GemInfoScrollOffset + maxModalHeight
		if endIdx > len(lines) {
			endIdx = len(lines)
		}
		visibleLines = lines[m.GemInfoScrollOffset:endIdx]

		// Add scroll indicator at the bottom if there's more content
		if endIdx < len(lines) {
			visibleLines = append(visibleLines, "")
			visibleLines = append(visibleLines, "  ↓ scroll for more")
		}
	}

	// Create the modal box with border
	content := strings.Join(visibleLines, "\n")

	// Calculate width - limit to reasonable size
	modalWidth := 80
	if modalWidth > m.Width-4 {
		modalWidth = m.Width - 4
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	// Apply border and styling
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBorderActive)).
		Background(lipgloss.Color(ColorSurface)).
		Padding(1, 2)

	return boxStyle.Width(modalWidth).Render(content)
}

// Helper function to get all gems for Sanity tab display
func (m *Model) allGemsForSanity() []*gemfile.GemStatus {
	// Combine direct and transitive gems
	allGems := make([]*gemfile.GemStatus, 0)

	// Add direct dependencies first
	if m.FirstLevelGems != nil {
		allGems = append(allGems, m.FirstLevelGems...)
	}

	// Add transitive dependencies (gems in GemStatuses but not in FirstLevelGems)
	if m.AnalysisResult != nil && m.AnalysisResult.GemStatuses != nil {
		// Build directMap from the gems we already added (safe approach)
		directMap := make(map[string]bool)
		for _, gem := range allGems {
			directMap[gem.Name] = true
		}

		for _, gem := range m.AnalysisResult.GemStatuses {
			if !directMap[gem.Name] {
				allGems = append(allGems, gem)
			}
		}
	}

	return allGems
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

	// Insecure sources summary
	if len(m.InsecureSourceGems) > 0 {
		sections = append(sections, "")
		insecureLabel := fmt.Sprintf("🔓 Insecure Gem Sources (%d)", len(m.InsecureSourceGems))
		insecureStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorWarning))
		sections = append(sections, insecureStyle.Render(insecureLabel))
		sections = append(sections, "")
		for _, gem := range m.InsecureSourceGems {
			sourceInfo := fmt.Sprintf("  • %s @ %s", gem.Name, gem.Source)
			sections = append(sections, sourceInfo)
		}
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

func (m *Model) viewCVEFilterMenu() string {
	// Render CVE list as background
	background := m.viewCVE()

	// Create filter modal
	modal := m.renderCVEFilterModalBox()

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

func (m *Model) renderCVEFilterModalBox() string {
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
	title := "Filter CVEs"
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, "")

	// Severity filter options
	severities := []string{"CRITICAL", "HIGH", "MODERATE", "LOW"}
	for i, severity := range severities {
		severityLine := checkbox(m.CVESelectedSeverities[severity]) + " " + severity + " only"
		if m.CVEFilterMenuCursor == i {
			lines = append(lines, RowSelectedStyle.Render("› "+severityLine))
		} else {
			lines = append(lines, "  "+severityLine)
		}
	}

	lines = append(lines, "")

	// Direct-only filter
	directLine := checkbox(m.CVEShowOnlyDirect) + " Direct only"
	if m.CVEFilterMenuCursor == 4 {
		lines = append(lines, RowSelectedStyle.Render("› "+directLine))
	} else {
		lines = append(lines, "  "+directLine)
	}

	// Footer hint
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextMuted)).
		Italic(true)
	lines = append(lines, hintStyle.Render("↑↓ navigate  space toggle  enter/esc close"))

	// Create the modal box with border
	content := strings.Join(lines, "\n")

	// Calculate width
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

func (m *Model) viewCVEInfo() string {
	if len(m.CVEVulnerabilities) == 0 || m.CVECursor >= len(m.CVEVulnerabilities) {
		return m.viewCVE()
	}

	// Render CVE list as background
	background := m.viewCVE()

	// Create info modal
	modal := m.renderCVEInfoModalBox()

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

func (m *Model) renderCVEInfoModalBox() string {
	vuln := m.CVEVulnerabilities[m.CVECursor]
	gemType, group := m.getCVEGemInfo(vuln.GemName)

	// Build content lines
	lines := []string{}

	// Title
	title := "CVE Details"
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, "")

	// CVE ID
	lines = append(lines, fmt.Sprintf("ID:       %s", vuln.CVE))

	// Gem name and type
	gemLine := vuln.GemName
	if m.isFrameworkGem(vuln.GemName) {
		gemLine += " [fw]"
	}
	gemLine += fmt.Sprintf(" — %s", gemType)
	lines = append(lines, fmt.Sprintf("Gem:      %s", gemLine))

	// Severity with badge
	severityBadge := ""
	switch vuln.Severity {
	case "CRITICAL":
		severityBadge = BadgeCriticalDotStyle.Render("●")
	case "HIGH":
		severityBadge = BadgeHighDotStyle.Render("●")
	case "MODERATE":
		severityBadge = BadgeWarningDotStyle.Render("●")
	default:
		severityBadge = BadgeHealthyDotStyle.Render("●")
	}
	severityLine := fmt.Sprintf("Severity: %s %s", severityBadge, vuln.Severity)
	if vuln.CVSS > 0 {
		severityLine += fmt.Sprintf(" (CVSS: %.1f)", vuln.CVSS)
	}
	lines = append(lines, severityLine)

	// Published date
	if !vuln.PublishedDate.IsZero() {
		lines = append(lines, fmt.Sprintf("Published: %s", vuln.PublishedDate.Format("2006-01-02")))
	}

	// Group
	lines = append(lines, fmt.Sprintf("Group:    %s", group))

	lines = append(lines, "")

	// Remediation section
	if vuln.FixedVersion != "" {
		lines = append(lines, "Remediation:")
		lines = append(lines, fmt.Sprintf("  Upgrade %s to version %s or later", vuln.GemName, vuln.FixedVersion))
		lines = append(lines, "")
	}

	// Workarounds section rendered with glamour for markdown formatting
	if vuln.Workarounds != "" {
		// Estimate modal width for glamour rendering
		estimatedWidth := 60
		if m.Width > 80 {
			estimatedWidth = m.Width - 20
		}

		// Render workarounds markdown with glamour
		renderer := NewMarkdownRenderer(estimatedWidth)
		renderedWorkarounds, err := renderer.Render(vuln.Workarounds)
		if err != nil {
			// Fallback to plain text if rendering fails
			lines = append(lines, "Workarounds:")
			workaroundLines := strings.Split(vuln.Workarounds, "\n")
			for _, wLine := range workaroundLines {
				trimmed := strings.TrimSpace(wLine)
				if trimmed != "" {
					wrapped := wrapText(trimmed, 60)
					for _, wrappedLine := range wrapped {
						lines = append(lines, fmt.Sprintf("  %s", wrappedLine))
					}
				}
			}
		} else {
			// Add rendered markdown (trim trailing newlines)
			renderedLines := strings.Split(strings.TrimSpace(renderedWorkarounds), "\n")
			lines = append(lines, renderedLines...)
		}
		lines = append(lines, "")
	}

	// OSV link
	if vuln.OSVId != "" {
		osvLink := fmt.Sprintf("https://osv.dev/vulnerability/%s", vuln.OSVId)
		lines = append(lines, fmt.Sprintf("Link:      %s", osvLink))
		lines = append(lines, "")
	}

	// Affected versions
	if len(vuln.AffectedVersions) > 0 {
		lines = append(lines, "Affected versions:")
		for _, version := range vuln.AffectedVersions {
			lines = append(lines, fmt.Sprintf("  • %s", version))
		}
		lines = append(lines, "")
	}

	// For transitive gems, show pulling-in parents
	if gemType == "Transitive" {
		parentGems := m.findParentGems(vuln.GemName)
		if len(parentGems) > 0 {
			lines = append(lines, "Pulled in by:")
			for _, parent := range parentGems {
				lines = append(lines, fmt.Sprintf("  › %s", parent))
			}
			lines = append(lines, "")
		}
	}

	// Create the modal box with border
	content := strings.Join(lines, "\n")

	// Calculate width - use enough space
	modalWidth := lipgloss.Width(content) + 4 // 2 for padding left/right, 2 for border
	if modalWidth < 60 {
		modalWidth = 60
	}
	if modalWidth > m.Width-4 {
		modalWidth = m.Width - 4
	}

	// Calculate available height for modal (leave space for header + footer + margins)
	// Reserve at least 4 lines for header/footer/margins
	availableHeight := m.Height - 8
	if availableHeight < 10 {
		availableHeight = 10
	}

	// If content fits within available height, no scrolling needed
	if len(lines) <= availableHeight {
		// Apply border and styling
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorBorderActive)).
			Background(lipgloss.Color(ColorSurface)).
			Padding(1, 2)

		return boxStyle.Width(modalWidth).Render(content)
	}

	// Clip content to fit within available height
	clippedLines := lines
	if m.CVEInfoScroll > 0 {
		if m.CVEInfoScroll >= len(lines) {
			m.CVEInfoScroll = len(lines) - availableHeight
			if m.CVEInfoScroll < 0 {
				m.CVEInfoScroll = 0
			}
		}
		clippedLines = lines[m.CVEInfoScroll:]
	}

	if len(clippedLines) > availableHeight {
		clippedLines = clippedLines[:availableHeight]
	}

	clippedContent := strings.Join(clippedLines, "\n")

	// Add scroll indicator if needed
	scrollHint := ""
	if m.CVEInfoScroll > 0 || (m.CVEInfoScroll+availableHeight < len(lines)) {
		scrollPercent := (m.CVEInfoScroll * 100) / len(lines)
		scrollHint = fmt.Sprintf(" [%d%%]", scrollPercent)
	}

	// Apply border and styling with height constraint
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBorderActive)).
		Background(lipgloss.Color(ColorSurface)).
		Padding(1, 2)

	rendered := boxStyle.Width(modalWidth).Height(availableHeight + 2).Render(clippedContent)

	// Add scroll hint to the output if content is scrollable
	if scrollHint != "" {
		renderedLines := strings.Split(rendered, "\n")
		if len(renderedLines) > 0 {
			// Append scroll hint to the last line of the box
			lastIdx := len(renderedLines) - 1
			if lastIdx >= 0 && strings.Contains(renderedLines[lastIdx], "╰") {
				renderedLines[lastIdx] = strings.TrimSuffix(renderedLines[lastIdx], "╯") + scrollHint + "╯"
			}
		}
		rendered = strings.Join(renderedLines, "\n")
	}

	return rendered
}

// findParentGems returns a list of direct gems that transitively depend on the given gem
// Note: This is a simplified version that checks if the gem appears in any first-level's dependency tree
func (m *Model) findParentGems(gemName string) []string {
	parents := []string{}

	if m.AnalysisResult == nil || len(m.AnalysisResult.AllGems) == 0 {
		return parents
	}

	// Build a map for quick gem lookup
	gemMap := make(map[string]*gemfile.Gem)
	for _, gem := range m.AnalysisResult.AllGems {
		gemMap[gem.Name] = gem
	}

	// Look through all first-level gems to find which transitively depend on the target gem
	for _, firstLevel := range m.FirstLevelGems {
		// Check if this gem transitively depends on the target
		if m.gemDependsOn(firstLevel.Name, gemName, gemMap) {
			parents = append(parents, firstLevel.Name)
		}
	}

	return parents
}

// gemDependsOn checks if gem A transitively depends on gem B using gemMap
func (m *Model) gemDependsOn(gemA, gemB string, gemMap map[string]*gemfile.Gem) bool {
	// BFS through dependency tree using gemMap
	queue := []string{gemA}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		if current == gemB {
			return true
		}

		// Add direct dependencies to queue
		if gem, ok := gemMap[current]; ok && gem.Dependencies != nil {
			for _, depName := range gem.Dependencies {
				if !visited[depName] {
					queue = append(queue, depName)
				}
			}
		}
	}

	return false
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
