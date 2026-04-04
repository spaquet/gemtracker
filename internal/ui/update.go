package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spaquet/gemtracker/internal/gemfile"
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

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case AnalysisCompleteMsg:
		return m.handleAnalysisComplete(msg)

	case DependencyCompleteMsg:
		return m.handleDependencyComplete(msg)
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
		m.CurrentView = ViewCVE
		m.ActiveTab = ViewCVE
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
		m.DetailOffset = 0
		return m, nil

	case "up":
		if m.DetailOffset > 0 {
			m.DetailOffset--
		}
		return m, nil

	case "down":
		m.DetailOffset++
		// Clamp to max based on section
		m.clampDetailOffset()
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
		m.CurrentView = ViewGemList
		m.ActiveTab = ViewGemList
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

func (m *Model) handlePathInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := m.PathInput.Value()
		if path != "" {
			m.loadProject(path)
			m.PathInput.Reset()
			m.CurrentView = ViewLoading
			m.Loading = true
			m.LoadingMessage = "Analyzing Gemfile.lock..."
			m.AnimationFrame = 0
			return m, tea.Batch(
				tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
					return SpinnerTickMsg{}
				}),
				performAnalysis(m.GemfileLockPath),
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

	m.GemListCursor = 0
	m.GemListOffset = 0
	m.CurrentView = m.ActiveTab

	return m, nil
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
	m.DetailOffset = 0
	m.CurrentView = ViewGemDetail

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
	contentHeight := m.Height - FixedChrome

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
	contentHeight := m.Height - FixedChrome - 2
	if m.GemListCursor < m.GemListOffset {
		m.GemListOffset = m.GemListCursor
	} else if m.GemListCursor >= m.GemListOffset+contentHeight {
		m.GemListOffset = m.GemListCursor - contentHeight + 1
	}
}

func (m *Model) ensureSearchCursorVisible() {
	contentHeight := m.Height - FixedChrome - 2
	if m.SearchCursor < m.SearchOffset {
		m.SearchOffset = m.SearchCursor
	} else if m.SearchCursor >= m.SearchOffset+contentHeight {
		m.SearchOffset = m.SearchCursor - contentHeight + 1
	}
}

func (m *Model) ensureCVECursorVisible() {
	contentHeight := m.Height - FixedChrome - 2
	if m.CVECursor < m.CVEOffset {
		m.CVEOffset = m.CVECursor
	} else if m.CVECursor >= m.CVEOffset+contentHeight {
		m.CVEOffset = m.CVECursor - contentHeight + 1
	}
}

func (m *Model) clampDetailOffset() {
	// Rough estimate - clamp to prevent too much scrolling
	maxDepth := 20
	if m.DetailOffset > maxDepth {
		m.DetailOffset = maxDepth
	}
}
