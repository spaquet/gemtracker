package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/getsentry/sentry-go"
	"github.com/spaquet/gemtracker/internal/cache"
	"github.com/spaquet/gemtracker/internal/gemfile"
	"github.com/spaquet/gemtracker/internal/logger"
	"github.com/spaquet/gemtracker/internal/telemetry"
)

func (m *Model) handleSpinnerTick() (tea.Model, tea.Cmd) {
	if m.Loading {
		m.AnimationFrame = (m.AnimationFrame + 1) % len(spinnerFrames)
		return m, tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
			return SpinnerTickMsg{}
		})
	}
	return m, nil
}

func (m *Model) handleProgressTick() (tea.Model, tea.Cmd) {
	if m.Loading && m.AnalysisPercentage < 90 {
		// Increment progress with slight acceleration
		increment := 3 + (m.AnalysisPercentage / 20)
		if increment < 1 {
			increment = 1
		}
		m.AnalysisPercentage += increment

		// Update stage based on progress
		if m.AnalysisPercentage < 50 {
			m.AnalysisStage = "parsing"
			m.LoadingMessage = "Parsing Gemfile.lock..."
		} else {
			m.AnalysisStage = "checking-updates"
			m.LoadingMessage = "Checking for updates..."
		}

		// Continue ticking
		return m, tea.Tick(time.Millisecond*200, func(time.Time) tea.Msg {
			return ProgressTickMsg{}
		})
	}
	return m, nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Clamp scroll offsets if needed
		m.clampScrollOffsets()
		return m, nil

	case SpinnerTickMsg:
		return m.handleSpinnerTick()

	case ProgressTickMsg:
		return m.handleProgressTick()

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case AnalysisCompleteMsg:
		return m.handleAnalysisComplete(msg)

	case DependencyCompleteMsg:
		return m.handleDependencyComplete(msg)

	case VersionCheckMsg:
		if msg.HasUpdate {
			m.NewVersionAvailable = msg.LatestVersion
		}
		return m, nil

	case ProgressMsg:
		// Update progress state
		m.AnalysisStage = msg.Stage
		m.AnalysisPercentage = msg.Percentage
		m.LoadingMessage = msg.Message
		return m, nil

	case HealthItemMsg:
		return m.handleHealthItem(msg)

	case HealthCompleteMsg:
		return m.handleHealthComplete()

	case GitHubBatchCompleteMsg:
		return m.handleGitHubBatchComplete(msg)

	case HealthRateLimitedMsg:
		logger.Warn("Health check rate limited at gem: %s", msg.StoppedAt)
		m.HealthRateLimited = true
		m.HealthLoading = false
		// Report rate limiting to Sentry
		err := fmt.Errorf("health check rate limited at gem: %s", msg.StoppedAt)
		telemetry.CaptureException(err, sentry.LevelWarning)
		return m, nil

	case OutdatedItemMsg:
		return m.handleOutdatedItem(msg)

	case OutdatedCompleteMsg:
		return m.handleOutdatedComplete()

	case CVEScanStartedMsg:
		return m.handleCVEScanStarted()

	case CVEProgressMsg:
		return m.handleCVEProgress(msg)

	case CVECompleteMsg:
		return m.handleCVEComplete(msg)

	case CVELoadFromCacheMsg:
		return m.handleCVELoadFromCache(msg)

	case SanityDataMsg:
		return m.handleSanityData(msg)

	case GemInfoMsg:
		return m.handleGemInfo(msg)
	}

	return m, nil
}

// ============================================================================
// Key Handling
// ============================================================================

func (m *Model) handleErrorViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" || msg.String() == "esc" {
		m.CurrentView = m.ActiveTab
		m.ErrorMessage = ""
	}
	return m, nil
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	if msg.String() == "ctrl+c" || msg.String() == "q" {
		m.Quitting = true
		return m, tea.Quit
	}

	// / key jumps to search
	if msg.String() == "/" && m.CurrentView != ViewLoading {
		m.CurrentView = ViewSearch
		m.ActiveTab = ViewSearch
		m.SearchInput.Focus()
		return m, nil
	}

	// View-specific handling
	switch m.CurrentView {
	case ViewLoading:
		return m, nil

	case ViewGemList:
		return m.handleGemListKeys(msg)

	case ViewGemDetail:
		return m.handleGemDetailKeys(msg)

	case ViewSearch:
		return m.handleSearchKeys(msg)

	case ViewUpgradeable:
		return m.handleUpgradeableKeys(msg)

	case ViewCVE:
		return m.handleCVEKeys(msg)

	case ViewSanity:
		if m.ShowingGemInfo {
			return m.handleGemInfoKeys(msg)
		}
		return m.handleSanityKeys(msg)

	case ViewProjectInfo:
		return m.handleProjectInfoKeys(msg)

	case ViewFilterMenu:
		return m.handleFilterMenuKeys(msg)

	case ViewCVEFilterMenu:
		return m.handleCVEFilterMenuKeys(msg)

	case ViewCVEInfo:
		return m.handleCVEInfoKeys(msg)

	case ViewSelectPath:
		return m.handlePathInputKeys(msg)

	case ViewError:
		return m.handleErrorViewKey(msg)
	}

	return m, nil
}

func (m *Model) handleGemListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.GemListCursor > 0 {
			m.GemListCursor--
			m.ensureGemListCursorVisible()
		}
		return m, nil

	case "down":
		if m.GemListCursor < len(m.FirstLevelGems)-1 {
			m.GemListCursor++
			m.ensureGemListCursorVisible()
		}
		return m, nil

	case "enter":
		if len(m.FirstLevelGems) > 0 && m.GemListCursor < len(m.FirstLevelGems) {
			m.SelectedGem = m.FirstLevelGems[m.GemListCursor]
			m.CurrentView = ViewGemDetail
			m.Loading = true
			m.LoadingMessage = "Loading dependencies..."
			// Reset navigation state for new detail view
			m.DetailSection = 0
			m.DetailTreeCursor = 0
			m.DetailForwardOffset = 0
			m.DetailReverseOffset = 0
			return m, tea.Batch(
				tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
					return SpinnerTickMsg{}
				}),
				performDependencyAnalysis(m.GemfileLockPath, m.SelectedGem.Name),
			)
		}
		return m, nil

	case "tab":
		m.CurrentView = ViewSearch
		m.ActiveTab = ViewSearch
		m.SearchInput.Focus()
		return m, nil

	case "shift+tab":
		m.CurrentView = ViewProjectInfo
		m.ActiveTab = ViewProjectInfo
		return m, nil

	case "u":
		m.toggleUpgradableFilter()
		return m, nil

	case "c":
		m.clearFilters()
		return m, nil

	case "f":
		m.CurrentView = ViewFilterMenu
		return m, nil

	case "r":
		// Full refresh: gem list, upgrades, and vulnerabilities
		if m.HealthLoading || m.OutdatedLoading || m.CVERefreshInProgress {
			// Don't start another refresh while already loading
			return m, nil
		}

		logger.Info("User requested full refresh (r key)")
		m.Loading = true
		m.LoadingMessage = "Refreshing all data..."

		// Clear all caches to force fresh data
		cache.Clear(m.GemfileLockPath)       // Clear analysis cache
		cache.ClearHealth(m.GemfileLockPath) // Clear health cache
		gemfile.ClearVulnerabilityCache()    // Clear CVE cache

		// Restart full analysis from scratch
		return m, performAnalysis(m.GemfileLockPath, true) // true = ignore cache
	}

	return m, nil
}

func (m *Model) selectedGemNameFromDetail() string {
	if m.DetailSection == 0 {
		// Forward dependencies
		if m.DetailTreeCursor < len(m.DetailForwardLines) {
			return m.DetailForwardLines[m.DetailTreeCursor]
		}
	} else {
		// Reverse dependencies (Used By)
		if m.DetailTreeCursor < len(m.DetailReverseLines) {
			return m.DetailReverseLines[m.DetailTreeCursor]
		}
	}
	return ""
}

func (m *Model) openGemURL(url string) {
	if url == "" {
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	_ = cmd.Start()
}

func (m *Model) handleGemDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.CurrentView = m.ActiveTab
		return m, nil

	case "tab":
		m.DetailSection = (m.DetailSection + 1) % 2
		m.DetailTreeCursor = 0
		return m, nil

	case "up":
		if m.DetailTreeCursor > 0 {
			m.DetailTreeCursor--
			m.ensureDetailCursorVisible()
		}
		return m, nil

	case "down":
		maxCursor := 0
		if m.DetailSection == 0 {
			maxCursor = len(m.DetailForwardLines) - 1
		} else {
			maxCursor = len(m.DetailReverseLines) - 1
		}
		if m.DetailTreeCursor < maxCursor {
			m.DetailTreeCursor++
			m.ensureDetailCursorVisible()
		}
		return m, nil

	case "enter":
		selectedGemName := m.selectedGemNameFromDetail()
		if selectedGemName != "" {
			var targetGem *gemfile.GemStatus
			for _, gem := range m.AnalysisResult.GemStatuses {
				if gem.Name == selectedGemName {
					targetGem = gem
					break
				}
			}
			if targetGem != nil {
				m.SelectedGem = targetGem
			}
			m.DetailTreeCursor = 0
			m.DetailForwardOffset = 0
			m.DetailReverseOffset = 0
			m.Loading = true
			m.LoadingMessage = "Loading dependencies..."
			return m, tea.Batch(
				tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
					return SpinnerTickMsg{}
				}),
				performDependencyAnalysis(m.GemfileLockPath, selectedGemName),
			)
		}
		return m, nil

	case "o":
		if m.SelectedGem != nil {
			m.openGemURL(m.SelectedGem.HomepageURL)
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.CurrentView = ViewUpgradeable
		m.ActiveTab = ViewUpgradeable
		return m, nil

	case "shift+tab":
		m.CurrentView = ViewGemList
		m.ActiveTab = ViewGemList
		return m, nil

	case "esc":
		m.SearchQuery = ""
		m.SearchResults = nil
		m.SearchCursor = 0
		m.SearchOffset = 0
		m.SearchInput.SetValue("")
		m.SearchInput.Focus()
		return m, nil

	case "up":
		if m.SearchCursor > 0 {
			m.SearchCursor--
			m.ensureSearchCursorVisible()
		}
		return m, nil

	case "down":
		if m.SearchCursor < len(m.SearchResults)-1 {
			m.SearchCursor++
			m.ensureSearchCursorVisible()
		}
		return m, nil

	case "enter":
		if len(m.SearchResults) > 0 && m.SearchCursor < len(m.SearchResults) {
			m.SelectedGem = m.SearchResults[m.SearchCursor]
			m.CurrentView = ViewGemDetail
			m.Loading = true
			m.LoadingMessage = "Loading dependencies..."
			return m, tea.Batch(
				tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
					return SpinnerTickMsg{}
				}),
				performDependencyAnalysis(m.GemfileLockPath, m.SelectedGem.Name),
			)
		}
		return m, nil

	default:
		// Pass to text input
		var cmd tea.Cmd
		m.SearchInput, cmd = m.SearchInput.Update(msg)
		m.SearchQuery = m.SearchInput.Value()
		m.updateSearchResults()
		m.SearchCursor = 0
		m.SearchOffset = 0
		return m, cmd
	}
}

func (m *Model) handleUpgradeableKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.CurrentView = ViewCVE
		m.ActiveTab = ViewCVE
		return m.ensureCVEScanStarted()

	case "shift+tab":
		m.CurrentView = ViewSearch
		m.ActiveTab = ViewSearch
		return m, nil

	case "up":
		if m.UpgradeableCursor > 0 {
			m.UpgradeableCursor--
			m.ensureUpgradeableCursorVisible()
		}
		return m, nil

	case "down":
		allUpgradeable := m.allUpgradeableGems()
		if m.UpgradeableCursor < len(allUpgradeable)-1 {
			m.UpgradeableCursor++
			m.ensureUpgradeableCursorVisible()
		}
		return m, nil

	case "enter":
		allUpgradeable := m.allUpgradeableGems()
		if len(allUpgradeable) > 0 && m.UpgradeableCursor < len(allUpgradeable) {
			m.SelectedGem = allUpgradeable[m.UpgradeableCursor]
			m.CurrentView = ViewGemDetail
			m.ActiveTab = ViewUpgradeable
			m.Loading = true
			m.LoadingMessage = "Loading dependencies..."
			return m, tea.Batch(
				tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
					return SpinnerTickMsg{}
				}),
				performDependencyAnalysis(m.GemfileLockPath, m.SelectedGem.Name),
			)
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleCVEKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.CurrentView = ViewSanity
		m.ActiveTab = ViewSanity
		return m, nil

	case "shift+tab":
		m.CurrentView = ViewUpgradeable
		m.ActiveTab = ViewUpgradeable
		return m, nil

	case "up":
		if m.CVECursor > 0 {
			m.CVECursor--
			m.ensureCVECursorVisible()
		}
		return m, nil

	case "down":
		if m.CVECursor < len(m.CVEVulnerabilities)-1 {
			m.CVECursor++
			m.ensureCVECursorVisible()
		}
		return m, nil

	case "enter":
		if len(m.CVEVulnerabilities) > 0 && m.CVECursor < len(m.CVEVulnerabilities) {
			vuln := m.CVEVulnerabilities[m.CVECursor]
			// Find the gem with this vulnerability to display details
			if m.AnalysisResult != nil {
				for _, gemStatus := range m.AnalysisResult.GemStatuses {
					if gemStatus.Name == vuln.GemName {
						m.SelectedGem = gemStatus
						m.CurrentView = ViewGemDetail
						m.ActiveTab = ViewCVE
						m.Loading = true
						m.LoadingMessage = "Loading dependencies..."
						return m, tea.Batch(
							tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
								return SpinnerTickMsg{}
							}),
							performDependencyAnalysis(m.GemfileLockPath, m.SelectedGem.Name),
						)
					}
				}
			}
		}
		return m, nil

	case "f":
		// Open CVE filter modal
		m.CurrentView = ViewCVEFilterMenu
		m.CVEFilterMenuCursor = 0
		return m, nil

	case "i":
		// Open CVE info modal for current CVE
		if len(m.CVEVulnerabilities) > 0 && m.CVECursor < len(m.CVEVulnerabilities) {
			m.CurrentView = ViewCVEInfo
			m.CVEInfoScroll = 0 // Reset scroll when opening
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleCVEInfoKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.CurrentView = ViewCVE
		m.CVEInfoScroll = 0 // Reset scroll when closing
		return m, nil

	case "up":
		if m.CVEInfoScroll > 0 {
			m.CVEInfoScroll--
		}
		return m, nil

	case "down":
		if m.CVEInfoScroll < m.getCVEInfoMaxScroll() {
			m.CVEInfoScroll++
		}
		return m, nil

	case "home":
		m.CVEInfoScroll = 0
		return m, nil

	case "end":
		m.CVEInfoScroll = m.getCVEInfoMaxScroll()
		return m, nil

	case "o":
		// Open CVE link in browser
		if len(m.CVEVulnerabilities) > 0 && m.CVECursor < len(m.CVEVulnerabilities) {
			vuln := m.CVEVulnerabilities[m.CVECursor]
			if vuln.OSVId != "" {
				url := fmt.Sprintf("https://osv.dev/vulnerability/%s", vuln.OSVId)
				return m, openBrowserCmd(url)
			}
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleSanityKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	allGems := m.allGemsForSanity()

	switch msg.String() {
	case "tab":
		m.CurrentView = ViewProjectInfo
		m.ActiveTab = ViewProjectInfo
		return m, nil

	case "shift+tab":
		m.CurrentView = ViewCVE
		m.ActiveTab = ViewCVE
		return m, nil

	case "up":
		if m.SanityCursor > 0 {
			m.SanityCursor--
			m.ensureSanityCursorVisible()
		}
		return m, nil

	case "down":
		if m.SanityCursor < len(allGems)-1 {
			m.SanityCursor++
			m.ensureSanityCursorVisible()
		}
		return m, nil

	case "enter", "i":
		// Open gem info modal with current gem data
		if len(allGems) > 0 && m.SanityCursor < len(allGems) {
			gem := allGems[m.SanityCursor]
			m.ShowingGemInfo = true
			m.GemInfoLoading = true
			m.CurrentGemInfoOutput = ""
			m.ParsedGemInfo = nil
			m.GemInfoScrollOffset = 0 // Reset scroll position
			// Fetch gem info asynchronously
			return m, fetchGemInfo(gem.Name)
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleGemInfoKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.ShowingGemInfo = false
		m.GemInfoScrollOffset = 0 // Reset scroll position when closing
		return m, nil

	case "up":
		if m.GemInfoScrollOffset > 0 {
			m.GemInfoScrollOffset--
		}
		return m, nil

	case "down":
		// Allow scrolling down (actual max will be checked in rendering)
		m.GemInfoScrollOffset++
		return m, nil
	}

	return m, nil
}

// ensureSanityCursorVisible adjusts the offset so the cursor stays visible
func (m *Model) ensureSanityCursorVisible() {
	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines

	// Account for header and section headers
	visibleRows := contentHeight - 6 // Rough estimate, conservative

	if visibleRows < 1 {
		visibleRows = 1
	}

	// Scroll to keep cursor visible
	if m.SanityCursor < m.SanityOffset {
		m.SanityOffset = m.SanityCursor
	} else if m.SanityCursor >= m.SanityOffset+visibleRows {
		m.SanityOffset = m.SanityCursor - visibleRows + 1
	}
}

// getCVEInfoMaxScroll calculates the maximum scroll position for the CVE info modal
func (m *Model) getCVEInfoMaxScroll() int {
	if len(m.CVEVulnerabilities) == 0 || m.CVECursor >= len(m.CVEVulnerabilities) {
		return 0
	}

	vuln := m.CVEVulnerabilities[m.CVECursor]
	gemType, group := m.getCVEGemInfo(vuln.GemName)

	// Reconstruct the lines to count total content
	lines := []string{}

	// Title
	lines = append(lines, "CVE Details")
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

	// Severity
	severityLine := fmt.Sprintf("Severity: %s", vuln.Severity)
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

	// Workarounds section
	if vuln.Workarounds != "" {
		lines = append(lines, "Workarounds:")
		workaroundLines := strings.Split(vuln.Workarounds, "\n")
		for _, wLine := range workaroundLines {
			trimmed := strings.TrimSpace(wLine)
			if trimmed != "" {
				lines = append(lines, fmt.Sprintf("  %s", trimmed))
			}
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

	// Calculate available height
	availableHeight := m.Height - 8
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Return max scroll position
	maxScroll := len(lines) - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (m *Model) handleProjectInfoKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.CurrentView = ViewGemList
		m.ActiveTab = ViewGemList
		return m, nil

	case "shift+tab":
		m.CurrentView = ViewSanity
		m.ActiveTab = ViewSanity
		return m, nil
	}

	return m, nil
}

// ensureCVEScanStarted checks if CVE scan needs to start and initiates it if needed
func (m *Model) ensureCVEScanStarted() (tea.Model, tea.Cmd) {
	if m.AnalysisResult == nil || len(m.AnalysisResult.AllGems) == 0 {
		return m, nil
	}

	currentGemsSignature := gemfile.ComputeGemsSignature(m.AnalysisResult.AllGems)

	// Check if gems have changed or if we need a refresh
	needsRefresh := currentGemsSignature != m.LastGemsSignature || m.CVECacheLoadedAt.IsZero()

	if needsRefresh {
		// Start CVE scan
		return m, performCVEScan(m.AnalysisResult.AllGems)
	}

	return m, nil
}

func (m *Model) handleFilterMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Calculate total filter options: 1 for upgradable + number of groups
	totalOptions := 1 + len(m.AvailableGroups)

	switch msg.String() {
	case "up":
		if m.FilterMenuCursor > 0 {
			m.FilterMenuCursor--
		}
		return m, nil

	case "down":
		if m.FilterMenuCursor < totalOptions-1 {
			m.FilterMenuCursor++
		}
		return m, nil

	case " ":
		// Toggle the selected filter
		if m.FilterMenuCursor == 0 {
			// Upgradable filter
			m.toggleUpgradableFilter()
		} else {
			// Group filter
			groupIdx := m.FilterMenuCursor - 1
			if groupIdx < len(m.AvailableGroups) {
				m.toggleGroupFilter(m.AvailableGroups[groupIdx])
			}
		}
		return m, nil

	case "enter", "esc":
		m.CurrentView = ViewGemList
		m.FilterMenuCursor = 0
		return m, nil
	}

	return m, nil
}

func (m *Model) handleCVEFilterMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 4 severity options + 1 direct option + 1 separator + 1 close = 7 items
	totalOptions := 6

	switch msg.String() {
	case "up":
		if m.CVEFilterMenuCursor > 0 {
			m.CVEFilterMenuCursor--
		}
		return m, nil

	case "down":
		if m.CVEFilterMenuCursor < totalOptions-1 {
			m.CVEFilterMenuCursor++
		}
		return m, nil

	case " ":
		// Toggle the selected filter
		switch m.CVEFilterMenuCursor {
		case 0: // CRITICAL
			m.CVESelectedSeverities["CRITICAL"] = !m.CVESelectedSeverities["CRITICAL"]
		case 1: // HIGH
			m.CVESelectedSeverities["HIGH"] = !m.CVESelectedSeverities["HIGH"]
		case 2: // MODERATE
			m.CVESelectedSeverities["MODERATE"] = !m.CVESelectedSeverities["MODERATE"]
		case 3: // LOW
			m.CVESelectedSeverities["LOW"] = !m.CVESelectedSeverities["LOW"]
		case 4: // Direct only
			m.CVEShowOnlyDirect = !m.CVEShowOnlyDirect
		}
		// Apply filters immediately
		m.applyCVEFilters()
		return m, nil

	case "enter", "esc":
		m.CurrentView = ViewCVE
		m.CVEFilterMenuCursor = 0
		return m, nil
	}

	return m, nil
}

func (m *Model) handlePathInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := m.PathInput.Value()
		if path != "" {
			m.loadProject(path)
			m.PathInput.Reset()
			m.CurrentView = ViewLoading
			m.Loading = true
			m.LoadingMessage = "Parsing Gemfile.lock..."
			m.AnalysisStage = "parsing"
			m.AnalysisPercentage = 0
			m.AnimationFrame = 0
			return m, tea.Batch(
				tea.Tick(time.Millisecond*200, func(time.Time) tea.Msg {
					return ProgressTickMsg{}
				}),
				performAnalysis(m.GemfileLockPath, m.NoCache),
			)
		}
		return m, nil

	case "esc":
		m.PathInput.Reset()
		m.CurrentView = m.ActiveTab
		return m, nil

	default:
		var cmd tea.Cmd
		m.PathInput, cmd = m.PathInput.Update(msg)
		return m, cmd
	}
}

// ============================================================================
// Message Handlers
// ============================================================================

func (m *Model) handleAnalysisComplete(msg AnalysisCompleteMsg) (tea.Model, tea.Cmd) {
	m.Loading = false

	if msg.Error != nil {
		logger.Error("Gemfile.lock analysis failed: %v", msg.Error)
		m.ErrorMessage = fmt.Sprintf("Error analyzing Gemfile.lock: %v", msg.Error)
		m.CurrentView = ViewError
		return m, nil
	}

	if msg.Result == nil {
		m.ErrorMessage = "No analysis result returned"
		m.CurrentView = ViewError
		return m, nil
	}

	m.AnalysisResult = msg.Result

	// Extract first-level gems
	m.FirstLevelGems = make([]*gemfile.GemStatus, 0)
	firstLevelSet := make(map[string]bool)
	for _, name := range msg.Result.FirstLevelGems {
		firstLevelSet[name] = true
	}

	for _, gs := range msg.Result.GemStatuses {
		if firstLevelSet[gs.Name] {
			m.FirstLevelGems = append(m.FirstLevelGems, gs)
		}
	}

	// Sort first-level gems alphabetically by name
	sort.Slice(m.FirstLevelGems, func(i, j int) bool {
		return m.FirstLevelGems[i].Name < m.FirstLevelGems[j].Name
	})

	// Store unfiltered gems and extract available groups for filtering
	m.UnfilteredGems = make([]*gemfile.GemStatus, len(m.FirstLevelGems))
	copy(m.UnfilteredGems, m.FirstLevelGems)
	m.AvailableGroups = m.extractAvailableGroups(m.FirstLevelGems)

	// Note: m.VulnerableGems is no longer populated from static checker
	// Instead, we use m.CVEVulnerabilities which is populated from OSV.dev
	// This will be built in rebuildVulnerableGemsList() after CVE scan completes
	m.VulnerableGems = make([]*gemfile.GemStatus, 0)

	// Populate project info fields
	m.RubyVersion = gemfile.ExtractRubyVersion(m.GemfileLockPath)
	m.BundleVersion = gemfile.ExtractBundleVersion(m.GemfileLockPath)

	// Parse Gemfile for framework detection
	gf, err := gemfile.Parse(m.GemfileLockPath)
	if err == nil {
		framework, version := gemfile.DetectFramework(gf)
		m.FrameworkDetected = framework
		m.RailsVersion = version
		// Also get insecure sources from the parsed Gemfile
		m.InsecureSourceGems = gf.GetInsecureSourceGems()
	}

	// Calculate statistics
	m.TotalGems = len(msg.Result.GemStatuses)
	m.FirstLevelCount = len(m.FirstLevelGems)
	m.TransitiveDeps = m.TotalGems - m.FirstLevelCount

	// Save to cache for faster subsequent loads
	cacheEntry := &cache.CacheEntry{
		Result:            msg.Result,
		RubyVersion:       m.RubyVersion,
		BundleVersion:     m.BundleVersion,
		FrameworkDetected: m.FrameworkDetected,
		RailsVersion:      m.RailsVersion,
	}
	if err := cache.Write(m.GemfileLockPath, cacheEntry); err != nil {
		// Log but don't fail - caching is optional
		logger.Warn("Failed to cache analysis: %v", err)
	}

	m.GemListCursor = 0
	m.GemListOffset = 0
	m.AnalysisPercentage = 100
	m.AnalysisStage = "complete"
	m.LoadingMessage = "Analysis complete"
	m.CurrentView = m.ActiveTab

	// Store the outdated checker for health data extraction
	if msg.OutdatedChecker != nil {
		m.OutdatedChecker = msg.OutdatedChecker
	}

	// Start outdated checking for all gems (not just first-level)
	m.OutdatedLoading = true
	m.OutdatedPending = make([]*gemfile.GemStatus, len(msg.Result.GemStatuses))
	copy(m.OutdatedPending, msg.Result.GemStatuses)

	// Initialize health data loading queue (but don't start fetching yet)
	// Health checks will start after outdated checking completes to avoid race conditions
	m.HealthLoading = true
	m.HealthTotalCount = len(msg.Result.GemStatuses)
	m.HealthLoadedCount = 0
	m.HealthPending = make([]*gemfile.GemStatus, len(msg.Result.GemStatuses))
	copy(m.HealthPending, msg.Result.GemStatuses)

	// Try to load health data from cache first (fixes issue #29 - dots disappearing on tab switch)
	if healthCache, err := cache.ReadHealth(m.GemfileLockPath); err == nil {
		remaining := m.HealthPending[:0]
		for _, gem := range m.HealthPending {
			if cached, ok := healthCache.Gems[gem.Name]; ok && cached != nil {
				gem.Health = cached
				m.HealthLoadedCount++
			} else {
				remaining = append(remaining, gem)
			}
		}
		m.HealthPending = remaining

		// Also set on UnfilteredGems and FirstLevelGems for consistency
		for _, gem := range m.UnfilteredGems {
			if cached, ok := healthCache.Gems[gem.Name]; ok && cached != nil && gem.Health == nil {
				gem.Health = cached
			}
		}
		for _, gem := range m.FirstLevelGems {
			if cached, ok := healthCache.Gems[gem.Name]; ok && cached != nil && gem.Health == nil {
				gem.Health = cached
			}
		}

		// If all gems loaded from cache, stop health loading now
		if m.HealthLoadedCount == m.HealthTotalCount {
			m.HealthLoading = false
		}
	}

	// Start CVE scanning in background
	// Compute gems signature for later refresh detection
	if len(msg.Result.AllGems) > 0 {
		m.LastGemsSignature = gemfile.ComputeGemsSignature(msg.Result.AllGems)
	}

	// Start Sanity data loading (gem sizes)
	m.SanityLoading = true

	// Batch outdated checking, CVE scanning, and Sanity data loading
	return m, tea.Batch(
		fetchNextOutdatedItem(m.OutdatedPending, m.OutdatedChecker),
		performCVEScan(msg.Result.AllGems),
		loadSanityData(msg.Result.AllGems),
	)
}

func (m *Model) handleDependencyComplete(msg DependencyCompleteMsg) (tea.Model, tea.Cmd) {
	m.Loading = false

	if msg.Error != nil {
		logger.Error("Dependency analysis failed: %v", msg.Error)
		m.ErrorMessage = fmt.Sprintf("Error loading dependencies: %v", msg.Error)
		m.CurrentView = ViewError
		return m, nil
	}

	if msg.Result == nil {
		m.ErrorMessage = "No dependency result returned"
		m.CurrentView = ViewError
		return m, nil
	}

	m.DependencyResult = msg.Result
	m.DetailSection = 0
	m.DetailForwardOffset = 0
	m.DetailReverseOffset = 0
	m.DetailTreeCursor = 0
	m.CurrentView = ViewGemDetail

	return m, nil
}

func (m *Model) handleHealthItem(msg HealthItemMsg) (tea.Model, tea.Cmd) {
	// Report error to Sentry if health check failed (but skip rate limit errors - they're expected)
	if msg.Error != nil && !isRateLimited(msg.Error) {
		err := fmt.Errorf("failed to fetch health for gem %q: %w", msg.GemName, msg.Error)
		telemetry.CaptureException(err, sentry.LevelError)
	}

	// Find and update the gem with health data
	// If health is nil due to rate limit, still set it (with RateLimited flag)
	if msg.Health == nil && msg.Error != nil && isRateLimited(msg.Error) {
		msg.Health = &gemfile.GemHealth{
			Score:       gemfile.HealthUnknown,
			RateLimited: true,
			FetchedAt:   time.Now(),
		}
	}

	// Update all gem lists
	for _, gem := range m.FirstLevelGems {
		if gem.Name == msg.GemName {
			gem.Health = msg.Health
			break
		}
	}
	for _, gem := range m.UnfilteredGems {
		if gem.Name == msg.GemName {
			gem.Health = msg.Health
			break
		}
	}
	for _, gem := range m.AnalysisResult.GemStatuses {
		if gem.Name == msg.GemName {
			gem.Health = msg.Health
			break
		}
	}

	m.HealthLoadedCount++

	// Pop the first pending gem and fetch the next
	if len(m.HealthPending) > 0 {
		m.HealthPending = m.HealthPending[1:]
	}

	if len(m.HealthPending) > 0 {
		return m, fetchNextHealthItem(m.HealthPending, m.HealthChecker, m.OutdatedChecker)
	}

	// All gems processed, emit complete message
	return m, func() tea.Msg { return HealthCompleteMsg{} }
}

func (m *Model) handleGitHubBatchComplete(msg GitHubBatchCompleteMsg) (tea.Model, tea.Cmd) {
	// GitHub batch fetch completed (or was skipped if no token)
	// Now start fetching per-gem data (RubyGems owners)
	if len(m.HealthPending) > 0 {
		return m, fetchNextHealthItem(m.HealthPending, m.HealthChecker, m.OutdatedChecker)
	}

	// All health data already loaded from cache
	m.HealthLoading = false
	return m, nil
}

func (m *Model) handleHealthComplete() (tea.Model, tea.Cmd) {
	m.HealthLoading = false

	// Save health data to cache (including all gems, not just first-level)
	healthCache := &cache.HealthCacheEntry{
		Gems:     make(map[string]*gemfile.GemHealth),
		CachedAt: time.Now(),
	}
	for _, gem := range m.AnalysisResult.GemStatuses {
		if gem.Health != nil {
			healthCache.Gems[gem.Name] = gem.Health
		}
	}

	// Fire-and-forget cache write
	go cache.WriteHealth(m.GemfileLockPath, healthCache)

	return m, nil
}

func (m *Model) handleOutdatedItem(msg OutdatedItemMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		// Check if rate limited
		if isRateLimited(msg.Error) {
			logger.Warn("Outdated version check rate limited at gem: %s", msg.GemName)
			m.OutdatedRateLimited = true
			m.OutdatedLoading = false
			// Report rate limiting to Sentry as a warning
			telemetry.CaptureException(msg.Error, sentry.LevelWarning)
			return m, nil // stop queue on rate limit
		}
		// Network/timeout error: mark gem as failed, continue queue
		logger.Error("Outdated check failed for gem %q: %v", msg.GemName, msg.Error)
		for _, gem := range m.AnalysisResult.GemStatuses {
			if gem.Name == msg.GemName {
				gem.OutdatedFailed = true
				break
			}
		}
		m.OutdatedErrorCount++
		// Report individual gem check failure to Sentry
		err := fmt.Errorf("failed to check outdated version for gem %q: %w", msg.GemName, msg.Error)
		telemetry.CaptureException(err, sentry.LevelError)
	} else {
		// Success: update gem fields
		for _, gem := range m.AnalysisResult.GemStatuses {
			if gem.Name == msg.GemName {
				gem.IsOutdated = msg.IsOutdated
				gem.LatestVersion = msg.LatestVersion
				gem.HomepageURL = msg.HomepageURL
				gem.Description = msg.Description
				break
			}
		}
	}

	// Pop the first pending gem
	if len(m.OutdatedPending) > 0 {
		m.OutdatedPending = m.OutdatedPending[1:]
	}

	if len(m.OutdatedPending) > 0 {
		return m, fetchNextOutdatedItem(m.OutdatedPending, m.OutdatedChecker)
	}

	return m, func() tea.Msg { return OutdatedCompleteMsg{} }
}

func (m *Model) handleOutdatedComplete() (tea.Model, tea.Cmd) {
	m.OutdatedLoading = false

	// Rebuild the upgradeable gems list with updated outdated status
	m.buildUpgradeableList()

	// Start GitHub batch fetch first (collects all repos and fetches in one GraphQL call)
	// Then health checking will continue after batch complete
	if len(m.HealthPending) > 0 {
		// Use all gems (not just pending) to collect all unique GitHub repos for batching
		return m, fetchGitHubBatchHealth(m.AnalysisResult.GemStatuses, m.OutdatedChecker, m.HealthChecker)
	}

	return m, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

func (m *Model) updateSearchResults() {
	if m.AnalysisResult == nil || m.SearchQuery == "" {
		m.SearchResults = nil
		return
	}

	m.SearchResults = make([]*gemfile.GemStatus, 0)
	query := strings.ToLower(m.SearchQuery)

	for _, gs := range m.AnalysisResult.GemStatuses {
		if strings.Contains(strings.ToLower(gs.Name), query) {
			m.SearchResults = append(m.SearchResults, gs)
		}
	}

	// Sort search results alphabetically by name
	sort.Slice(m.SearchResults, func(i, j int) bool {
		return m.SearchResults[i].Name < m.SearchResults[j].Name
	})
}

func (m *Model) clampScrollOffsets() {
	// Use dynamic statusbar height instead of hardcoded FixedChrome
	// This ensures offset calculation matches the actual rendered content
	statusbarHeight := m.statusBarTotalHeight()
	// Reserve 1 line for footer/statusbar buffer (matches viewGemList calculation)
	contentHeight := m.Height - 2 - statusbarHeight - 1

	// Account for header row and any filter status lines
	// renderGemListTable shows: [filter (2)] + header (1) + gems (rest)
	availableGemsRows := contentHeight - 1 // -1 for header
	if m.hasActiveFilters() {
		availableGemsRows -= 2 // -2 for filter status line + blank line
	}
	if availableGemsRows < 1 {
		availableGemsRows = 1 // Minimum 1 gem row
	}

	// Clamp gem list offset
	// Allow scrolling to show all gems with the last gem fully visible
	maxOffset := len(m.FirstLevelGems) - availableGemsRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.GemListOffset > maxOffset {
		m.GemListOffset = maxOffset
	}

	// Clamp search offset (has title + header lines above results)
	availableSearchRows := contentHeight - 2
	maxSearchOffset := len(m.SearchResults) - availableSearchRows
	if maxSearchOffset < 0 {
		maxSearchOffset = 0
	}
	if m.SearchOffset > maxSearchOffset {
		m.SearchOffset = maxSearchOffset
	}

	// Clamp CVE offset (has title + header lines above results)
	availableCVERows := contentHeight - 2
	maxCVEOffset := len(m.VulnerableGems) - availableCVERows
	if maxCVEOffset < 0 {
		maxCVEOffset = 0
	}
	if m.CVEOffset > maxCVEOffset {
		m.CVEOffset = maxCVEOffset
	}
}

func (m *Model) ensureGemListCursorVisible() {
	// Use dynamic statusbar height instead of hardcoded FixedChrome
	// This ensures cursor visibility calculation matches the actual rendered content
	statusbarHeight := m.statusBarTotalHeight()
	// Reserve 1 line for footer/statusbar buffer (matches viewGemList and clampScrollOffsets)
	contentHeight := m.Height - 2 - statusbarHeight - 1
	// renderGemListTable shows: [filter (2)] + header (1) + gems (rest)
	availableGemsRows := contentHeight - 1 // -1 for header
	if m.hasActiveFilters() {
		availableGemsRows -= 2 // -2 for filter status line + blank line
	}
	if availableGemsRows < 1 {
		availableGemsRows = 1
	}
	if m.GemListCursor < m.GemListOffset {
		m.GemListOffset = m.GemListCursor
	} else if m.GemListCursor >= m.GemListOffset+availableGemsRows {
		m.GemListOffset = m.GemListCursor - availableGemsRows + 1
	}
}

func (m *Model) ensureSearchCursorVisible() {
	contentHeight := m.Height - FixedChrome - m.updateBarHeight()
	// renderSearchResults shows: title (1) + header (1) + results (contentHeight - 2)
	availableSearchRows := contentHeight - 2
	if m.SearchCursor < m.SearchOffset {
		m.SearchOffset = m.SearchCursor
	} else if m.SearchCursor >= m.SearchOffset+availableSearchRows {
		m.SearchOffset = m.SearchCursor - availableSearchRows + 1
	}
}

func (m *Model) ensureCVECursorVisible() {
	statusbarHeight := m.statusBarTotalHeight()
	// Reserve 1 line for footer/statusbar buffer (matches viewCVE and clampScrollOffsets)
	contentHeight := m.Height - 2 - statusbarHeight - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	// renderCVETable shows: header (1-4 lines) + table_header (1 line) + vulnerabilities (rest)
	// Use 5 as a safe estimate for the CVE header section
	availableCVERows := contentHeight - 5

	if availableCVERows < 1 {
		availableCVERows = 1
	}

	// Keep cursor in bounds
	if m.CVECursor >= len(m.CVEVulnerabilities) {
		m.CVECursor = len(m.CVEVulnerabilities) - 1
	}
	if m.CVECursor < 0 {
		m.CVECursor = 0
	}

	// Scroll to keep cursor visible
	if m.CVECursor < m.CVEOffset {
		m.CVEOffset = m.CVECursor
	} else if m.CVECursor >= m.CVEOffset+availableCVERows {
		m.CVEOffset = m.CVECursor - availableCVERows + 1
	}

	// Keep offset in bounds
	if m.CVEOffset < 0 {
		m.CVEOffset = 0
	}
	if m.CVEOffset >= len(m.CVEVulnerabilities) && len(m.CVEVulnerabilities) > 0 {
		m.CVEOffset = len(m.CVEVulnerabilities) - 1
	}
}

func (m *Model) ensureUpgradeableCursorVisible() {
	statusbarLines := m.statusBarTotalHeight()
	contentHeight := m.Height - 2 - statusbarLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	// renderUpgradeableTable consumes lines for headers and spacing, so actual gem rows < contentHeight
	// Conservative estimate: subtract 4 lines per section header (title + blank + header + spacing)
	// In practice, we show maybe 75-80% of contentHeight as actual gems
	availableRows := contentHeight * 3 / 4
	if availableRows < 1 {
		availableRows = 1
	}
	if m.UpgradeableCursor < m.UpgradeableOffset {
		m.UpgradeableOffset = m.UpgradeableCursor
	} else if m.UpgradeableCursor >= m.UpgradeableOffset+availableRows {
		m.UpgradeableOffset = m.UpgradeableCursor - availableRows + 1
	}
}

func (m *Model) ensureDetailCursorVisible() {
	// Get the appropriate offset and total lines for current panel
	var offset *int
	var totalLines int

	if m.DetailSection == 0 {
		offset = &m.DetailForwardOffset
		totalLines = len(m.DetailForwardLines)
	} else {
		offset = &m.DetailReverseOffset
		totalLines = len(m.DetailReverseLines)
	}

	// Estimate panel height
	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 5
	panelHeight := (contentHeight - 2) / 2

	// Clamp cursor to visible range
	if m.DetailTreeCursor >= panelHeight {
		// Cursor is beyond visible area, scroll down
		*offset = m.DetailTreeCursor - panelHeight + 1
	} else if m.DetailTreeCursor < 0 {
		m.DetailTreeCursor = 0
		*offset = 0
	} else {
		// Cursor is within visible area - reset offset to show from top if possible
		*offset = 0
	}

	// Ensure offset doesn't go past the end
	maxOffset := totalLines - panelHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if *offset > maxOffset {
		*offset = maxOffset
		// Adjust cursor if offset was clamped
		visibleLines := totalLines - *offset
		if m.DetailTreeCursor >= visibleLines {
			m.DetailTreeCursor = visibleLines - 1
		}
	}

	// Ensure offset is not negative
	if *offset < 0 {
		*offset = 0
	}
}

// ============================================================================
// CVE Scan Handlers
// ============================================================================

func (m *Model) handleCVEScanStarted() (tea.Model, tea.Cmd) {
	m.CVERefreshInProgress = true
	return m, nil
}

func (m *Model) handleCVEProgress(msg CVEProgressMsg) (tea.Model, tea.Cmd) {
	// Update progress in UI if needed
	// For now, just acknowledge the progress
	return m, nil
}

func (m *Model) handleCVELoadFromCache(msg CVELoadFromCacheMsg) (tea.Model, tea.Cmd) {
	// Initialize CVE filters with loaded vulnerabilities
	m.initializeCVEFilters(msg.Vulnerabilities)
	m.CVELastScanTime = time.Now().Add(-msg.CacheAge)
	m.CVECacheLoadedAt = time.Now()

	// Merge CVE results into gem statuses so Gems tab shows vulnerability indicators
	m.mergeVulnerabilityDataIntoGems(msg.Vulnerabilities)

	// Rebuild vulnerable gems list (only gems with vulnerabilities)
	m.rebuildVulnerableGemsList()

	return m, nil
}

func (m *Model) handleSanityData(msg SanityDataMsg) (tea.Model, tea.Cmd) {
	m.SanityLoading = false

	if msg.Error != nil {
		logger.Warn("Failed to load Sanity data: %v", msg.Error)
		// Don't fail, just show error state in UI
		return m, nil
	}

	m.GemDirPath = msg.GemDirPath
	m.RubyManager = msg.RubyManager
	m.ProjectTotalSizeBytes = msg.ProjectTotalSize
	m.GemSizes = msg.GemSizes

	return m, nil
}

func (m *Model) handleGemInfo(msg GemInfoMsg) (tea.Model, tea.Cmd) {
	m.GemInfoLoading = false

	if msg.Error != nil {
		logger.Warn("Failed to get gem info for %s: %v", msg.GemName, msg.Error)
		// Still show output even if error occurred
		m.CurrentGemInfoOutput = msg.Output
	} else {
		m.CurrentGemInfoOutput = msg.Output
	}

	// Store parsed gem info data (versions and paths)
	m.ParsedGemInfo = msg.Parsed

	return m, nil
}

func (m *Model) handleCVEComplete(msg CVECompleteMsg) (tea.Model, tea.Cmd) {
	m.CVERefreshInProgress = false

	if msg.Error != nil {
		// If error, keep old data if available, show error
		m.CVELastError = msg.Error.Error()
		logger.Warn("CVE scan failed: %v", msg.Error)
		return m, nil
	}

	// Initialize CVE filters with fresh data
	m.initializeCVEFilters(msg.Vulnerabilities)
	m.CVELastScanTime = time.Now()
	m.CVECacheLoadedAt = time.Now()
	m.CVELastError = ""

	// Update gems signature for next check
	if m.AnalysisResult != nil && len(m.AnalysisResult.AllGems) > 0 {
		m.LastGemsSignature = gemfile.ComputeGemsSignature(m.AnalysisResult.AllGems)
	}

	// Merge CVE results into gem statuses so Gems tab shows vulnerability indicators
	m.mergeVulnerabilityDataIntoGems(msg.Vulnerabilities)

	// Rebuild vulnerable gems list (only gems with vulnerabilities)
	m.rebuildVulnerableGemsList()

	return m, nil
}

// mergeVulnerabilityDataIntoGems updates gem statuses with vulnerability information from OSV.dev
// This ensures the Gems tab shows the same vulnerability data as the CVE tab
func (m *Model) mergeVulnerabilityDataIntoGems(vulnerabilities []*gemfile.Vulnerability) {
	if m.AnalysisResult == nil {
		return
	}

	// Build a map of vulnerabilities by gem name
	vulnByGem := make(map[string][]*gemfile.Vulnerability)
	for _, vuln := range vulnerabilities {
		vulnByGem[vuln.GemName] = append(vulnByGem[vuln.GemName], vuln)
	}

	// Update gem statuses with vulnerability info
	for _, gemStatus := range m.AnalysisResult.GemStatuses {
		if vulns, hasVulns := vulnByGem[gemStatus.Name]; hasVulns && len(vulns) > 0 {
			gemStatus.IsVulnerable = true
			// Use the first vulnerability for the summary info
			vuln := vulns[0]
			gemStatus.VulnerabilityInfo = fmt.Sprintf("%s: %s", vuln.CVE, vuln.Description)
		} else {
			gemStatus.IsVulnerable = false
			gemStatus.VulnerabilityInfo = ""
		}
	}
}

// rebuildVulnerableGemsList updates the VulnerableGems list to match CVEVulnerabilities
// Only includes gems that actually have vulnerabilities
func (m *Model) rebuildVulnerableGemsList() {
	// Build a map of gem names with vulnerabilities
	gemNameMap := make(map[string]bool)
	for _, vuln := range m.CVEVulnerabilities {
		gemNameMap[vuln.GemName] = true
	}

	// Filter gem statuses to only those with vulnerabilities
	vulnerableGems := make([]*gemfile.GemStatus, 0)
	if m.AnalysisResult != nil {
		for _, gemStatus := range m.AnalysisResult.GemStatuses {
			if gemNameMap[gemStatus.Name] {
				vulnerableGems = append(vulnerableGems, gemStatus)
			}
		}
	}

	m.VulnerableGems = vulnerableGems
	m.CVECursor = 0
	m.CVEOffset = 0
}

// openBrowserCmd returns a BubbleTea command that opens a URL in the default browser
func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", url)
		}

		if cmd != nil {
			_ = cmd.Run()
		}
		return nil
	}
}
