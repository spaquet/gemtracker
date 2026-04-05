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
	"github.com/spaquet/gemtracker/internal/telemetry"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Clamp scroll offsets if needed
		m.clampScrollOffsets()
		return m, nil

	case SpinnerTickMsg:
		if m.Loading {
			m.AnimationFrame = (m.AnimationFrame + 1) % len(spinnerFrames)
			return m, tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
				return SpinnerTickMsg{}
			})
		}
		return m, nil

	case ProgressTickMsg:
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

	case HealthRateLimitedMsg:
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
	}

	return m, nil
}

// ============================================================================
// Key Handling
// ============================================================================

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
		// No keys allowed during loading
		return m, nil

	case ViewGemList:
		return m.handleGemListKeys(msg)

	case ViewGemDetail:
		return m.handleGemDetailKeys(msg)

	case ViewSearch:
		return m.handleSearchKeys(msg)

	case ViewCVE:
		return m.handleCVEKeys(msg)

	case ViewProjectInfo:
		return m.handleProjectInfoKeys(msg)

	case ViewFilterMenu:
		return m.handleFilterMenuKeys(msg)

	case ViewSelectPath:
		return m.handlePathInputKeys(msg)

	case ViewError:
		if msg.String() == "enter" || msg.String() == "esc" {
			m.CurrentView = m.ActiveTab
			m.ErrorMessage = ""
		}
		return m, nil
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
	}

	return m, nil
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
		// Navigate to the selected gem in the tree
		var selectedGemName string
		if m.DetailSection == 0 {
			// Forward dependencies
			if m.DetailTreeCursor < len(m.DetailForwardLines) {
				selectedGemName = m.DetailForwardLines[m.DetailTreeCursor]
			}
		} else {
			// Reverse dependencies (Used By)
			if m.DetailTreeCursor < len(m.DetailReverseLines) {
				selectedGemName = m.DetailReverseLines[m.DetailTreeCursor]
			}
		}

		if selectedGemName != "" {
			// Try to find the gem status for this name
			var targetGem *gemfile.GemStatus
			for _, gem := range m.AnalysisResult.GemStatuses {
				if gem.Name == selectedGemName {
					targetGem = gem
					break
				}
			}
			// Update SelectedGem if found, otherwise use the name anyway
			if targetGem != nil {
				m.SelectedGem = targetGem
			}
			// Always load dependency analysis for the selected gem, even if not in AnalysisResult
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
		// Open the homepage URL in the default browser
		if m.SelectedGem != nil && m.SelectedGem.HomepageURL != "" {
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("open", m.SelectedGem.HomepageURL)
			case "linux":
				cmd = exec.Command("xdg-open", m.SelectedGem.HomepageURL)
			case "windows":
				cmd = exec.Command("cmd", "/c", "start", m.SelectedGem.HomepageURL)
			default:
				return m, nil
			}
			// Run the command in the background
			_ = cmd.Start()
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.CurrentView = ViewCVE
		m.ActiveTab = ViewCVE
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

func (m *Model) handleCVEKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.CurrentView = ViewProjectInfo
		m.ActiveTab = ViewProjectInfo
		return m, nil

	case "shift+tab":
		m.CurrentView = ViewSearch
		m.ActiveTab = ViewSearch
		return m, nil

	case "up":
		if m.CVECursor > 0 {
			m.CVECursor--
			m.ensureCVECursorVisible()
		}
		return m, nil

	case "down":
		if m.CVECursor < len(m.VulnerableGems)-1 {
			m.CVECursor++
			m.ensureCVECursorVisible()
		}
		return m, nil

	case "enter":
		if len(m.VulnerableGems) > 0 && m.CVECursor < len(m.VulnerableGems) {
			m.SelectedGem = m.VulnerableGems[m.CVECursor]
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
		return m, nil
	}

	return m, nil
}

func (m *Model) handleProjectInfoKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.CurrentView = ViewGemList
		m.ActiveTab = ViewGemList
		return m, nil

	case "shift+tab":
		m.CurrentView = ViewCVE
		m.ActiveTab = ViewCVE
		return m, nil
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

	// Extract vulnerable gems and sort alphabetically
	m.VulnerableGems = make([]*gemfile.GemStatus, 0)
	for _, gs := range msg.Result.GemStatuses {
		if gs.IsVulnerable {
			m.VulnerableGems = append(m.VulnerableGems, gs)
		}
	}

	// Sort vulnerable gems alphabetically by name
	sort.Slice(m.VulnerableGems, func(i, j int) bool {
		return m.VulnerableGems[i].Name < m.VulnerableGems[j].Name
	})

	// Populate project info fields
	m.RubyVersion = gemfile.ExtractRubyVersion(m.GemfileLockPath)
	m.BundleVersion = gemfile.ExtractBundleVersion(m.GemfileLockPath)

	// Parse Gemfile for framework detection
	gf, err := gemfile.Parse(m.GemfileLockPath)
	if err == nil {
		framework, version := gemfile.DetectFramework(gf)
		m.FrameworkDetected = framework
		m.RailsVersion = version
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
		fmt.Printf("Warning: Failed to cache analysis: %v\n", err)
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

	// Start outdated checking for first-level gems
	m.OutdatedLoading = true
	m.OutdatedPending = make([]*gemfile.GemStatus, len(m.FirstLevelGems))
	copy(m.OutdatedPending, m.FirstLevelGems)

	// Initialize health data loading queue (but don't start fetching yet)
	// Health checks will start after outdated checking completes to avoid race conditions
	m.HealthLoading = true
	m.HealthTotalCount = len(m.FirstLevelGems)
	m.HealthLoadedCount = 0
	m.HealthPending = make([]*gemfile.GemStatus, len(m.FirstLevelGems))
	copy(m.HealthPending, m.FirstLevelGems)

	return m, fetchNextOutdatedItem(m.OutdatedPending, m.OutdatedChecker)
}

func (m *Model) handleDependencyComplete(msg DependencyCompleteMsg) (tea.Model, tea.Cmd) {
	m.Loading = false

	if msg.Error != nil {
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
	// Report error to Sentry if health check failed
	if msg.Error != nil {
		err := fmt.Errorf("failed to fetch health for gem %q: %w", msg.GemName, msg.Error)
		telemetry.CaptureException(err, sentry.LevelError)
	}

	// Find and update the gem with health data
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

func (m *Model) handleHealthComplete() (tea.Model, tea.Cmd) {
	m.HealthLoading = false

	// Save health data to cache
	healthCache := &cache.HealthCacheEntry{
		Gems:     make(map[string]*gemfile.GemHealth),
		CachedAt: time.Now(),
	}
	for _, gem := range m.FirstLevelGems {
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
			m.OutdatedRateLimited = true
			m.OutdatedLoading = false
			// Report rate limiting to Sentry as a warning
			telemetry.CaptureException(msg.Error, sentry.LevelWarning)
			return m, nil // stop queue on rate limit
		}
		// Network/timeout error: mark gem as failed, continue queue
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

	// Start health checking now that outdated checker cache is fully populated
	// This avoids race conditions with the outdated checker
	if len(m.HealthPending) > 0 {
		return m, fetchNextHealthItem(m.HealthPending, m.HealthChecker, m.OutdatedChecker)
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
	contentHeight := m.Height - FixedChrome - m.updateBarHeight()

	// Clamp gem list offset
	maxOffset := len(m.FirstLevelGems) - contentHeight + 2
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.GemListOffset > maxOffset {
		m.GemListOffset = maxOffset
	}

	// Clamp search offset
	maxSearchOffset := len(m.SearchResults) - contentHeight + 2
	if maxSearchOffset < 0 {
		maxSearchOffset = 0
	}
	if m.SearchOffset > maxSearchOffset {
		m.SearchOffset = maxSearchOffset
	}

	// Clamp CVE offset
	maxCVEOffset := len(m.VulnerableGems) - contentHeight + 2
	if maxCVEOffset < 0 {
		maxCVEOffset = 0
	}
	if m.CVEOffset > maxCVEOffset {
		m.CVEOffset = maxCVEOffset
	}
}

func (m *Model) ensureGemListCursorVisible() {
	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 2
	if m.GemListCursor < m.GemListOffset {
		m.GemListOffset = m.GemListCursor
	} else if m.GemListCursor >= m.GemListOffset+contentHeight {
		m.GemListOffset = m.GemListCursor - contentHeight + 1
	}
}

func (m *Model) ensureSearchCursorVisible() {
	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 2
	if m.SearchCursor < m.SearchOffset {
		m.SearchOffset = m.SearchCursor
	} else if m.SearchCursor >= m.SearchOffset+contentHeight {
		m.SearchOffset = m.SearchCursor - contentHeight + 1
	}
}

func (m *Model) ensureCVECursorVisible() {
	contentHeight := m.Height - FixedChrome - m.updateBarHeight() - 2
	if m.CVECursor < m.CVEOffset {
		m.CVEOffset = m.CVECursor
	} else if m.CVECursor >= m.CVEOffset+contentHeight {
		m.CVEOffset = m.CVECursor - contentHeight + 1
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
