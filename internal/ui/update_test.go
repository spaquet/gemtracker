package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/spaquet/gemtracker/internal/gemfile"
)

func TestGemDetailNavigation(t *testing.T) {
	gem := &gemfile.GemStatus{Name: "rack"}
	m := &Model{CurrentView: ViewGemList, FirstLevelGems: []*gemfile.GemStatus{gem}}
	_, cmd := m.selectGemFromList()
	if cmd == nil || m.SelectedGem != gem || m.CurrentView != ViewGemDetail || !m.Loading {
		t.Fatalf("opening gem from list did not set detail state: %+v", m)
	}

	m.AnalysisResult = &gemfile.AnalysisResult{GemStatuses: []*gemfile.GemStatus{gem}}
	m.DetailSection = 1
	m.DetailReverseLines = []string{"rack"}
	m.DetailTreeCursor = 0
	_, cmd = m.selectDetailGem()
	if cmd == nil || m.SelectedGem != gem || m.DetailSection != 1 || m.DetailTreeCursor != 0 {
		t.Fatalf("nested detail navigation lost selection or section: %+v", m)
	}
}

func TestPasteIntoSearchField(t *testing.T) {
	m := NewModel("dev", "", "", "Gemfile.lock", true, false)
	m.switchViewTo(ViewSearch)
	m.SearchInput.Focus()

	m.Update(tea.PasteMsg{Content: "rails"})
	if got := m.SearchInput.Value(); got != "rails" {
		t.Fatalf("search input = %q, want %q", got, "rails")
	}
	if m.SearchQuery != "rails" {
		t.Fatalf("SearchQuery = %q, want %q", m.SearchQuery, "rails")
	}

	// Paste outside search view is ignored.
	m.switchViewTo(ViewGemList)
	m.Update(tea.PasteMsg{Content: "x"})
	if got := m.SearchInput.Value(); got != "rails" {
		t.Fatalf("paste outside search changed input to %q", got)
	}
}

func TestPasteIntoOtherInputs(t *testing.T) {
	m := NewModel("dev", "", "", "Gemfile.lock", true, false)

	m.CurrentView = ViewCVEComment
	m.CVECommentInput.Focus()
	m.Update(tea.PasteMsg{Content: "accepted risk"})
	if got := m.CVECommentInput.Value(); got != "accepted risk" {
		t.Fatalf("comment input = %q", got)
	}

	m.CurrentView = ViewSelectPath
	m.PathInput.Focus()
	m.Update(tea.PasteMsg{Content: "/tmp/proj"})
	if got := m.PathInput.Value(); got != "/tmp/proj" {
		t.Fatalf("path input = %q", got)
	}
}
