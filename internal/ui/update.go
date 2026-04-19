package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/getsentry/sentry-go"
	"github.com/spaquet/gemtracker/internal/cache"
	"github.com/spaquet/gemtracker/internal/gemfile"
	"github.com/spaquet/gemtracker/internal/logger"
	"github.com/spaquet/gemtracker/internal/telemetry"
)

func (m *Model) handleVersionCheck(msg VersionCheckMsg) (tea.Model, tea.Cmd) {
	if msg.HasUpdate {
		m.NewVersionAvailable = msg.LatestVersion
	}
	return m, nil
}

func (m *Model) handleProgress(msg ProgressMsg) (tea.Model, tea.Cmd) {
	m.AnalysisStage = msg.Stage
	m.AnalysisPercentage = msg.Percentage
	m.LoadingMessage = msg.Message
	return m, nil
}

func (m *Model) handleHealthRateLimited(msg HealthRateLimitedMsg) (tea.Model, tea.Cmd) {
	logger.Warn("Health check rate limited at gem: %s", msg.StoppedAt)
	m.HealthRateLimited = true
	m.HealthLoading = false

	// Report rate limiting to Sentry
	err := fmt.Errorf("health check rate limited at gem: %s", msg.StoppedAt)
	telemetry.CaptureException(err, sentry.LevelWarning)
	return m, nil
}

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

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case AnalysisCompleteMsg:
		return m.handleAnalysisComplete(msg)

	case DependencyCompleteMsg:
		return m.handleDependencyComplete(msg)

	case VersionCheckMsg:
		return m.handleVersionCheck(msg)

	case ProgressMsg:
		return m.handleProgress(msg)

	case HealthItemMsg, HealthCompleteMsg, GitHubBatchCompleteMsg, HealthRateLimitedMsg:
		return m.dispatchHealthMessages(msg)

	case OutdatedItemMsg, OutdatedCompleteMsg:
		return m.dispatchOutdatedMessages(msg)

	case UpdateableItemMsg, UpdateableCompleteMsg:
		return m.dispatchUpdateableMessages(msg)

	case CVEScanStartedMsg, CVEProgressMsg, CVECompleteMsg, CVELoadFromCacheMsg, CVECommentsLoadedMsg, CVEEnrichmentCompleteMsg:
		return m.dispatchCVEMessages(msg)

	case SanityDataMsg:
		return m.handleSanityData(msg)

	case GemInfoMsg:
		return m.handleGemInfo(msg)
	}

	return m, nil
}

func (m *Model) dispatchHealthMessages(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case HealthItemMsg:
		return m.handleHealthItem(msg)
	case HealthCompleteMsg:
		return m.handleHealthComplete()
	case GitHubBatchCompleteMsg:
		return m.handleGitHubBatchComplete(msg)
	case HealthRateLimitedMsg:
		return m.handleHealthRateLimited(msg)
	}
	return m, nil
}

func (m *Model) dispatchOutdatedMessages(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case OutdatedItemMsg:
		return m.handleOutdatedItem(msg)
	case OutdatedCompleteMsg:
		return m.handleOutdatedComplete()
	case UpdateableItemMsg:
		return m.handleUpdateableItem(msg)
	case UpdateableCompleteMsg:
		return m.handleUpdateableComplete()
	case UpgradeResultMsg:
		return m.handleUpgradeResult(msg)
	}
	return m, nil
}

func (m *Model) dispatchCVEMessages(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CVEScanStartedMsg:
		return m.handleCVEScanStarted()
	case CVEProgressMsg:
		return m.handleCVEProgress(msg)
	case CVECompleteMsg:
		return m.handleCVEComplete(msg)
	case CVELoadFromCacheMsg:
		return m.handleCVELoadFromCache(msg)
	case CVECommentsLoadedMsg:
		return m.handleCVECommentsLoaded(msg)
	case CVEEnrichmentCompleteMsg:
		return m.handleCVEEnrichmentComplete(msg)
	}
	return m, nil
}

// ============================================================================
// Key Handling
// ============================================================================

func (m *Model) handleErrorViewKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" || msg.String() == "esc" {
		m.CurrentView = m.ActiveTab
		m.ErrorMessage = ""
	}
	return m, nil
}

// moveCursorUp moves the cursor up within bounds.
func (m *Model) moveCursorUp(cursor *int, maxItems int) {
	if *cursor > 0 {
		*cursor--
	}
}

// moveCursorDown moves the cursor down within bounds.
func (m *Model) moveCursorDown(cursor *int, maxItems int) {
	if *cursor < maxItems-1 {
		*cursor++
	}
}

// switchViewTo switches the current view and active tab.
func (m *Model) switchViewTo(view ViewMode) {
	m.CurrentView = view
	m.ActiveTab = view
}

// selectGemFromList handles selecting a gem from the list.
func (m *Model) selectGemFromList() (tea.Model, tea.Cmd) {
	if len(m.FirstLevelGems) == 0 || m.GemListCursor >= len(m.FirstLevelGems) {
		return nil, nil
	}

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

// performRefresh handles a full refresh of all data.
func (m *Model) performRefresh() (tea.Model, tea.Cmd) {
	if m.isLoadingOtherData() {
		return m, nil
	}

	logger.Info("User requested full refresh (r key)")
	m.Loading = true
	m.LoadingMessage = "Refreshing all data..."

	// Clear all caches to force fresh data
	cache.Clear(m.GemfileLockPath)
	cache.ClearHealth(m.GemfileLockPath)
	gemfile.ClearVulnerabilityCache()

	return m, performAnalysis(m.GemfileLockPath, true)
}

// isLoadingOtherData checks if any other loading operations are in progress.
func (m *Model) isLoadingOtherData() bool {
	return m.HealthLoading || m.OutdatedLoading || m.CVERefreshInProgress
}

func (m *Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Handle global keys — skip "q" quit when Search view is active (user typing search query)
	// Don't quit on "q" when user is typing in a text input
	textInputViews := m.CurrentView == ViewSearch || m.CurrentView == ViewCVEComment || m.CurrentView == ViewSelectPath
	if isQuitKey(msg) && !(msg.String() == "q" && textInputViews) {
		m.Quitting = true
		return m, tea.Quit
	}

	if isSearchKey(msg) && m.CurrentView != ViewLoading {
		m.switchViewTo(ViewSearch)
		m.SearchInput.Focus()
		return m, nil
	}

	// Dispatch to view-specific handler
	return m.handleViewKeys(msg)
}

// isQuitKey returns true if the key is a quit command.
func isQuitKey(msg tea.KeyPressMsg) bool {
	return msg.String() == "ctrl+c" || msg.String() == "q"
}

// isSearchKey returns true if the key is a search command.
func isSearchKey(msg tea.KeyPressMsg) bool {
	return msg.String() == "/"
}

// handleViewKeys dispatches key handling to view-specific handlers.
func (m *Model) handleViewKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		return m.handleSanityViewKeys(msg)

	case ViewProjectInfo:
		return m.handleProjectInfoKeys(msg)

	case ViewFilterMenu:
		return m.handleFilterMenuKeys(msg)

	case ViewCVEFilterMenu:
		return m.handleCVEFilterMenuKeys(msg)

	case ViewCVEInfo:
		return m.handleCVEInfoKeys(msg)

	case ViewCVEComment:
		return m.handleCVECommentKeys(msg)

	case ViewSelectPath:
		return m.handlePathInputKeys(msg)

	case ViewError:
		return m.handleErrorViewKey(msg)
	}

	return m, nil
}

// handleSanityViewKeys handles keys in the sanity view.
func (m *Model) handleSanityViewKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.ShowingGemInfo {
		return m.handleGemInfoKeys(msg)
	}
	return m.handleSanityKeys(msg)
}

func (m *Model) handleGemListKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.moveCursorUp(&m.GemListCursor, len(m.FirstLevelGems))
		m.ensureGemListCursorVisible()
		return m, nil

	case "down":
		m.moveCursorDown(&m.GemListCursor, len(m.FirstLevelGems))
		m.ensureGemListCursorVisible()
		return m, nil

	case "enter":
		if model, cmd := m.selectGemFromList(); cmd != nil {
			return model, cmd
		}
		return m, nil

	case "tab":
		m.switchViewTo(ViewSearch)
		m.SearchInput.Focus()
		return m, nil

	case "shift+tab":
		m.switchViewTo(ViewProjectInfo)
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
		return m.performRefresh()
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

func (m *Model) handleGemDetailKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m *Model) handleSearchKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m *Model) handleUpgradeableKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Handle modal close for any key
	if m.UpgradeResultModalOpen {
		return m.closeUpgradeResultModal()
	}

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

	case " ":
		m.ToggleSelectionAtCursor()
		return m, nil

	case "space":
		m.ToggleSelectionAtCursor()
		return m, nil

	case "ctrl+a":
		m.SelectAllUpgradeable()
		return m, nil

	case "ctrl+d":
		m.DeselectAllUpgradeable()
		return m, nil

	case "u":
		return m.startUpgrade()
	}

	return m, nil
}

func (m *Model) handleCVEKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.switchViewTo(ViewSanity)
		return m, nil

	case "shift+tab":
		m.switchViewTo(ViewUpgradeable)
		return m, nil

	case "up":
		m.moveCursorUp(&m.CVECursor, len(m.CVEVulnerabilities))
		m.ensureCVECursorVisible()
		return m, nil

	case "down":
		m.moveCursorDown(&m.CVECursor, len(m.CVEVulnerabilities))
		m.ensureCVECursorVisible()
		return m, nil

	case "enter":
		if model, cmd := m.selectVulnerableGem(); cmd != nil {
			return model, cmd
		}
		return m, nil

	case "f":
		m.CurrentView = ViewCVEFilterMenu
		m.CVEFilterMenuCursor = 0
		return m, nil

	case "i":
		m.openCVEInfoModal()
		return m, nil

	case "c":
		m.openCVECommentModal()
		return m, nil
	}

	return m, nil
}

// selectVulnerableGem selects the vulnerable gem and shows its details.
func (m *Model) selectVulnerableGem() (tea.Model, tea.Cmd) {
	if !m.hasValidCVECursor() {
		return nil, nil
	}

	vuln := m.CVEVulnerabilities[m.CVECursor]
	if m.AnalysisResult == nil {
		return nil, nil
	}

	// Find the gem with this vulnerability
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
	return nil, nil
}

// openCVEInfoModal opens the CVE info modal.
func (m *Model) openCVEInfoModal() {
	if m.hasValidCVECursor() {
		m.CurrentView = ViewCVEInfo
		m.CVEInfoScroll = 0 // Reset scroll when opening
	}
}

// hasValidCVECursor checks if the CVE cursor is valid.
func (m *Model) hasValidCVECursor() bool {
	return len(m.CVEVulnerabilities) > 0 && m.CVECursor < len(m.CVEVulnerabilities)
}

func (m *Model) handleCVEInfoKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeCVEInfoModal()
		return m, nil

	case "up":
		m.scrollCVEInfoUp()
		return m, nil

	case "down":
		m.scrollCVEInfoDown()
		return m, nil

	case "home":
		m.CVEInfoScroll = 0
		return m, nil

	case "end":
		m.CVEInfoScroll = m.getCVEInfoMaxScroll()
		return m, nil

	case "o":
		if cmd := m.openCVELinkInBrowser(); cmd != nil {
			return m, cmd
		}
		return m, nil

	case "c":
		m.openCVECommentModal()
		return m, nil
	}

	return m, nil
}

// closeCVEInfoModal closes the CVE info modal.
func (m *Model) closeCVEInfoModal() {
	m.CurrentView = ViewCVE
	m.CVEInfoScroll = 0
	m.CVEInfoCachedCVEID = ""
}

// scrollCVEInfoUp scrolls the CVE info up.
func (m *Model) scrollCVEInfoUp() {
	if m.hasValidCVECursor() && m.CVEInfoScroll > 0 {
		m.CVEInfoScroll--
	}
}

// scrollCVEInfoDown scrolls the CVE info down.
func (m *Model) scrollCVEInfoDown() {
	if m.hasValidCVECursor() && m.CVEInfoScroll < m.getCVEInfoMaxScroll() {
		m.CVEInfoScroll++
	}
}

// openCVELinkInBrowser opens the CVE link in the browser.
func (m *Model) openCVELinkInBrowser() tea.Cmd {
	if !m.hasValidCVECursor() {
		return nil
	}

	vuln := m.CVEVulnerabilities[m.CVECursor]
	if vuln.OSVId == "" {
		return nil
	}

	url := fmt.Sprintf("https://osv.dev/vulnerability/%s", vuln.OSVId)
	return openBrowserCmd(url)
}

func (m *Model) openCVECommentModal() {
	if len(m.CVEVulnerabilities) == 0 || m.CVECursor >= len(m.CVEVulnerabilities) {
		return
	}

	vuln := m.CVEVulnerabilities[m.CVECursor]
	key := gemfile.GetCVECommentKey(vuln)

	// Clear input and reset to defaults
	m.CVECommentInput.Reset()
	m.CVECommentDecision = gemfile.DecisionAcknowledged
	m.CVECommentDecisionIdx = 0

	// If there's an existing comment, load it
	if m.CVEComments != nil && len(m.CVEComments.Entries) > 0 {
		if comment, ok := m.CVEComments.Entries[key]; ok && comment != nil {
			m.CVECommentInput.SetValue(comment.Comment)
			m.CVECommentDecision = comment.Decision
			if comment.Decision == gemfile.DecisionIgnored {
				m.CVECommentDecisionIdx = 1
			} else {
				m.CVECommentDecisionIdx = 0
			}
		}
	}

	m.CurrentView = ViewCVEComment
	m.CVECommentInput.Focus()
}

func (m *Model) handleCVECommentKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.CurrentView = ViewCVE
		m.CVECommentInput.Reset()
		return m, nil

	case "tab":
		// Toggle between Acknowledged (0) and Ignored (1)
		m.CVECommentDecisionIdx = 1 - m.CVECommentDecisionIdx
		if m.CVECommentDecisionIdx == 0 {
			m.CVECommentDecision = gemfile.DecisionAcknowledged
		} else {
			m.CVECommentDecision = gemfile.DecisionIgnored
		}
		return m, nil

	case "enter":
		// Save the comment
		if len(m.CVEVulnerabilities) > 0 && m.CVECursor < len(m.CVEVulnerabilities) {
			vuln := m.CVEVulnerabilities[m.CVECursor]
			key := gemfile.GetCVECommentKey(vuln)

			if m.CVEComments == nil {
				m.CVEComments = &gemfile.CVEComments{
					Version: 1,
					Entries: make(map[string]*gemfile.CVEComment),
				}
			}

			now := time.Now()
			installedVersion := m.getInstalledGemVersion(vuln.GemName)

			m.CVEComments.Entries[key] = &gemfile.CVEComment{
				Decision:   m.CVECommentDecision,
				Comment:    m.CVECommentInput.Value(),
				GemName:    vuln.GemName,
				GemVersion: installedVersion,
				CreatedAt:  now,
				UpdatedAt:  now,
			}

			// Save to file
			projectDir := filepath.Dir(m.GemfileLockPath)
			if err := gemfile.SaveCVEComments(projectDir, m.CVEComments); err != nil {
				logger.Error("Failed to save CVE comments: %v", err)
			}
		}

		m.CurrentView = ViewCVE
		m.CVECommentInput.Reset()
		return m, nil

	default:
		// Delegate to textinput
		var cmd tea.Cmd
		m.CVECommentInput, cmd = m.CVECommentInput.Update(msg)
		return m, cmd
	}
}

func (m *Model) getInstalledGemVersion(gemName string) string {
	// Search in analysis result first (all gems)
	if m.AnalysisResult != nil {
		for _, gem := range m.AnalysisResult.GemStatuses {
			if gem.Name == gemName && gem.Version != "" {
				return gem.Version
			}
		}
	}

	// Fall back to first-level gems
	for _, gem := range m.FirstLevelGems {
		if gem.Name == gemName && gem.Version != "" {
			return gem.Version
		}
	}

	return ""
}

func (m *Model) handleSanityKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

	case "enter":
		// Navigate to gem detail view (same as other tabs)
		if len(allGems) > 0 && m.SanityCursor < len(allGems) {
			m.SelectedGem = allGems[m.SanityCursor]
			m.CurrentView = ViewGemDetail
			m.ActiveTab = ViewSanity
			m.Loading = true
			m.LoadingMessage = "Loading dependencies..."
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

	case "i":
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

func (m *Model) handleGemInfoKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	if !m.hasValidCVECursor() {
		return 0
	}

	vuln := m.CVEVulnerabilities[m.CVECursor]
	lineCount := m.buildCVEInfoLineCount(vuln)

	availableHeight := m.Height - 8
	if availableHeight < 10 {
		availableHeight = 10
	}

	maxScroll := lineCount - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

// buildCVEInfoLineCount calculates the number of lines needed to display CVE info.
func (m *Model) buildCVEInfoLineCount(vuln *gemfile.Vulnerability) int {
	count := 0

	// Header
	count += 2 // Title + blank line

	// Basic info
	count += 1 // CVE ID
	count += 1 // Gem
	count += 1 // Severity
	count += 1 // Published (if present)
	count += 1 // Group
	count += 1 // blank line

	// Remediation
	if vuln.FixedVersion != "" {
		count += 3 // Remediation header + details + blank
	}

	// Workarounds
	if vuln.Workarounds != "" {
		count += m.countWorkaroundLines(vuln)
	}

	// OSV link
	if vuln.OSVId != "" {
		count += 2 // Link + blank
	}

	// Affected versions
	if len(vuln.AffectedVersions) > 0 {
		count += 1 + len(vuln.AffectedVersions) + 1 // Header + versions + blank
	}

	// Parent gems (transitive)
	gemType, _ := m.getCVEGemInfo(vuln.GemName)
	if gemType == "Transitive" {
		parentGems := m.findParentGems(vuln.GemName)
		if len(parentGems) > 0 {
			count += 1 + len(parentGems) + 1 // Header + parents + blank
		}
	}

	return count
}

// countWorkaroundLines counts the rendered workaround lines.
func (m *Model) countWorkaroundLines(vuln *gemfile.Vulnerability) int {
	estimatedWidth := 60
	if m.Width > 80 {
		estimatedWidth = m.Width - 20
	}

	renderer := NewMarkdownRenderer(estimatedWidth)
	renderedWorkarounds, err := renderer.Render(vuln.Workarounds)
	if err == nil {
		renderedLines := strings.Split(strings.TrimSpace(renderedWorkarounds), "\n")
		return len(renderedLines) + 1 // rendered lines + blank
	}

	// Fallback: estimate plain lines
	count := 1 // Workarounds header
	for _, wLine := range strings.Split(vuln.Workarounds, "\n") {
		if strings.TrimSpace(wLine) != "" {
			count++
		}
	}
	return count + 1 // lines + blank
}

func (m *Model) handleProjectInfoKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		// Set flag and start CVE scan
		m.CVERefreshInProgress = true
		return m, performCVEScan(m.AnalysisResult.AllGems)
	}

	return m, nil
}

func (m *Model) handleFilterMenuKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

	case "space":
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

func (m *Model) handleCVEFilterMenuKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 4 severity options + 1 direct option + 3 acknowledgment options
	totalOptions := 8

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

	case "space":
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
		case 5: // Acknowledged
			m.CVEAcknowledgmentFilters["acknowledged"] = !m.CVEAcknowledgmentFilters["acknowledged"]
		case 6: // Ignored
			m.CVEAcknowledgmentFilters["ignored"] = !m.CVEAcknowledgmentFilters["ignored"]
		case 7: // Unacknowledged
			m.CVEAcknowledgmentFilters["unacknowledged"] = !m.CVEAcknowledgmentFilters["unacknowledged"]
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

func (m *Model) handlePathInputKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

	if err := m.validateAnalysisResult(msg); err != nil {
		return m, nil
	}

	m.AnalysisResult = msg.Result
	m.processAnalysisGems(msg.Result)
	m.populateProjectInfo(msg.Result)
	m.updateAnalysisState(msg)
	m.setupOutdatedChecking(msg.Result)
	m.setupHealthChecking()
	m.setupUpdateableChecking(msg.Result)

	// Compute gems signature for later refresh detection
	if len(msg.Result.AllGems) > 0 {
		m.LastGemsSignature = gemfile.ComputeGemsSignature(msg.Result.AllGems)
	}

	m.SanityLoading = true

	// Batch outdated checking, CVE scanning, Sanity data loading, and load CVE comments
	return m, tea.Batch(
		fetchNextOutdatedItem(m.OutdatedPending, m.OutdatedChecker),
		fetchNextUpdateableItem(m.UpdateablePending, m.ConstraintResolver),
		performCVEScan(msg.Result.AllGems),
		loadSanityData(msg.Result.AllGems),
		m.loadCVECommentsCmd(),
	)
}

// validateAnalysisResult checks if the analysis result is valid.
func (m *Model) validateAnalysisResult(msg AnalysisCompleteMsg) error {
	if msg.Error != nil {
		logger.Error("Gemfile.lock analysis failed: %v", msg.Error)
		m.ErrorMessage = fmt.Sprintf("Error analyzing Gemfile.lock: %v", msg.Error)
		m.CurrentView = ViewError
		return msg.Error
	}

	if msg.Result == nil {
		m.ErrorMessage = "No analysis result returned"
		m.CurrentView = ViewError
		return fmt.Errorf("no result")
	}

	return nil
}

// processAnalysisGems extracts and sorts first-level gems from the analysis result.
func (m *Model) processAnalysisGems(result *gemfile.AnalysisResult) {
	m.FirstLevelGems = make([]*gemfile.GemStatus, 0)
	firstLevelSet := make(map[string]bool)
	for _, name := range result.FirstLevelGems {
		firstLevelSet[name] = true
	}

	for _, gs := range result.GemStatuses {
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
	m.VulnerableGems = make([]*gemfile.GemStatus, 0)
}

// populateProjectInfo gathers project information and caches the analysis result.
func (m *Model) populateProjectInfo(result *gemfile.AnalysisResult) {
	m.RubyVersion = gemfile.ExtractRubyVersion(m.GemfileLockPath)
	m.BundleVersion = gemfile.ExtractBundleVersion(m.GemfileLockPath)

	// Parse Gemfile for framework detection
	gf, err := gemfile.Parse(m.GemfileLockPath)
	if err == nil {
		framework, version := gemfile.DetectFramework(gf)
		m.FrameworkDetected = framework
		m.RailsVersion = version
		m.InsecureSourceGems = gf.GetInsecureSourceGems()
	}

	// Calculate statistics
	m.TotalGems = len(result.GemStatuses)
	m.FirstLevelCount = len(m.FirstLevelGems)
	m.TransitiveDeps = m.TotalGems - m.FirstLevelCount

	// Save to cache for faster subsequent loads
	cacheEntry := &cache.CacheEntry{
		Result:            result,
		RubyVersion:       m.RubyVersion,
		BundleVersion:     m.BundleVersion,
		FrameworkDetected: m.FrameworkDetected,
		RailsVersion:      m.RailsVersion,
	}
	if err := cache.Write(m.GemfileLockPath, cacheEntry); err != nil {
		logger.Warn("Failed to cache analysis: %v", err)
	}
}

// updateAnalysisState updates the UI state after analysis completes.
func (m *Model) updateAnalysisState(msg AnalysisCompleteMsg) {
	m.GemListCursor = 0
	m.GemListOffset = 0
	m.AnalysisPercentage = 100
	m.AnalysisStage = "complete"
	m.LoadingMessage = "Analysis complete"
	m.CurrentView = m.ActiveTab

	if msg.OutdatedChecker != nil {
		m.OutdatedChecker = msg.OutdatedChecker
	}
}

// setupOutdatedChecking initializes the outdated gem checking queue.
func (m *Model) setupOutdatedChecking(result *gemfile.AnalysisResult) {
	m.OutdatedLoading = true
	m.OutdatedPending = make([]*gemfile.GemStatus, len(result.GemStatuses))
	copy(m.OutdatedPending, result.GemStatuses)
}

// setupHealthChecking initializes health data loading from cache or pending queue.
func (m *Model) setupHealthChecking() {
	m.HealthLoading = true
	m.HealthTotalCount = len(m.AnalysisResult.GemStatuses)
	m.HealthLoadedCount = 0
	m.HealthPending = make([]*gemfile.GemStatus, len(m.AnalysisResult.GemStatuses))
	copy(m.HealthPending, m.AnalysisResult.GemStatuses)

	// Try to load health data from cache first
	if healthCache, err := cache.ReadHealth(m.GemfileLockPath); err == nil {
		m.applyHealthCache(healthCache)
	}
}

// setupUpdateableChecking initializes the updateable version checking queue for gems with constraints.
func (m *Model) setupUpdateableChecking(result *gemfile.AnalysisResult) {
	// Reset the queue (same as outdated checking)
	m.UpdateableLoading = true
	m.UpdateablePending = make([]*gemfile.GemStatus, 0, len(result.GemStatuses))

	// Queue gems with constraints, set no-constraint gems directly
	for _, gem := range result.GemStatuses {
		if gem.Constraint != "" {
			// Initialize to "..." (loading indicator) so we can see if async task updates it
			gem.UpdateableVersion = "…"
			m.UpdateablePending = append(m.UpdateablePending, gem)
		} else if gem.LatestVersion != "" {
			// No constraint: updateable = latest (only if we have latest version)
			gem.UpdateableVersion = gem.LatestVersion
		}
	}

	logger.Info("Updateable checking: queued %d gems with constraints", len(m.UpdateablePending))
}

// applyHealthCache applies cached health data to gems.
func (m *Model) applyHealthCache(healthCache *cache.HealthCacheEntry) {
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

func (m *Model) dispatchUpdateableMessages(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case UpdateableItemMsg:
		return m.handleUpdateableItem(msg)
	case UpdateableCompleteMsg:
		return m.handleUpdateableComplete()
	}
	return m, nil
}

func (m *Model) handleUpdateableItem(msg UpdateableItemMsg) (tea.Model, tea.Cmd) {
	logger.Info("Updateable version resolved: %s -> %s", msg.GemName, msg.UpdateableVersion)

	// Update the gem's cached UpdateableVersion in GemStatuses
	for _, gem := range m.AnalysisResult.GemStatuses {
		if gem.Name == msg.GemName {
			gem.UpdateableVersion = msg.UpdateableVersion
			// Also ensure it's set in FirstLevelGems and UnfilteredGems for consistency
			for _, fg := range m.FirstLevelGems {
				if fg.Name == msg.GemName {
					fg.UpdateableVersion = msg.UpdateableVersion
					break
				}
			}
			for _, ug := range m.UnfilteredGems {
				if ug.Name == msg.GemName {
					ug.UpdateableVersion = msg.UpdateableVersion
					break
				}
			}
			break
		}
	}

	// Continue with next pending gem if any
	if len(m.UpdateablePending) > 0 {
		nextGem := m.UpdateablePending[0]
		m.UpdateablePending = m.UpdateablePending[1:]
		return m, fetchUpdateableVersion(nextGem, m.ConstraintResolver)
	}

	// All done
	return m, func() tea.Msg { return UpdateableCompleteMsg{} }
}

func (m *Model) handleUpdateableComplete() (tea.Model, tea.Cmd) {
	m.UpdateableLoading = false
	logger.Info("Updateable version checking complete")
	return m, nil
}

func (m *Model) startUpgrade() (tea.Model, tea.Cmd) {
	selectedGems := m.GetSelectedGemNames()
	if len(selectedGems) == 0 {
		return m, nil
	}

	m.UpgradeInProgress = true
	m.UpgradeSuccessCount = 0
	m.UpgradeErrors = nil

	return m, func() tea.Msg {
		results, _ := gemfile.UpgradeGems(selectedGems, m.GemfileLockPath)
		return UpgradeResultMsg{Results: results}
	}
}

func (m *Model) handleUpgradeResult(msg UpgradeResultMsg) (tea.Model, tea.Cmd) {
	m.UpgradeInProgress = false
	m.UpgradeResultModalOpen = true

	successCount := 0
	var errors []string

	for _, result := range msg.Results {
		if result.Success {
			successCount++
		} else {
			errors = append(errors, fmt.Sprintf("%s: %s", result.GemName, result.Error))
		}
	}

	m.UpgradeSuccessCount = successCount
	m.UpgradeErrors = errors

	return m, nil
}

func (m *Model) closeUpgradeResultModal() (tea.Model, tea.Cmd) {
	m.UpgradeResultModalOpen = false
	m.DeselectAllUpgradeable()
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
	m.CVERefreshInProgress = true // Set flag to show enrichment is in progress

	// Merge CVE results into gem statuses so Gems tab shows vulnerability indicators
	m.mergeVulnerabilityDataIntoGems(msg.Vulnerabilities)

	// Rebuild vulnerable gems list (only gems with vulnerabilities)
	m.rebuildVulnerableGemsList()

	// Send command to enrich vulnerabilities in background
	return m, enrichCachedVulnerabilitiesCmd(msg.Vulnerabilities)
}

func (m *Model) handleCVEEnrichmentComplete(msg CVEEnrichmentCompleteMsg) (tea.Model, tea.Cmd) {
	// Enrichment complete - update the vulnerabilities with enriched data
	m.CVERefreshInProgress = false

	if msg.Error != nil {
		logger.Warn("CVE enrichment failed: %v", msg.Error)
		// Keep the cached data even if enrichment failed
		return m, nil
	}

	// Update with enriched vulnerabilities
	m.initializeCVEFilters(msg.Vulnerabilities)

	// Merge CVE results into gem statuses
	m.mergeVulnerabilityDataIntoGems(msg.Vulnerabilities)

	// Rebuild vulnerable gems list
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

func (m *Model) handleCVECommentsLoaded(msg CVECommentsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		logger.Warn("Failed to load CVE comments: %v", msg.Error)
		// Initialize with empty comments on error
		m.CVEComments = &gemfile.CVEComments{
			Version: 1,
			Entries: make(map[string]*gemfile.CVEComment),
		}
		return m, nil
	}

	if msg.Comments == nil {
		m.CVEComments = &gemfile.CVEComments{
			Version: 1,
			Entries: make(map[string]*gemfile.CVEComment),
		}
	} else {
		m.CVEComments = msg.Comments
	}

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
