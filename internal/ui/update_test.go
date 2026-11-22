package ui

import (
	"strings"
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

func TestSelectionText(t *testing.T) {
	lines := []string{"hello world", "  second line  ", "third"}
	tests := []struct {
		name string
		sel  selection
		want string
	}{
		{"single line", selection{startX: 6, startY: 0, endX: 10, endY: 0}, "world"},
		{"reversed drag", selection{startX: 10, startY: 0, endX: 6, endY: 0}, "world"},
		{"multi line", selection{startX: 6, startY: 0, endX: 4, endY: 2}, "world\n  second line\nthird"},
	}
	for _, tt := range tests {
		if got := tt.sel.text(lines); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestMouseDragCopiesAndNotifies(t *testing.T) {
	m := NewModel("dev", "", "", "Gemfile.lock", true, false)
	m.Width, m.Height = 80, 24
	m.switchViewTo(ViewSearch)

	m.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	m.Update(tea.MouseMotionMsg{X: 10, Y: 1, Button: tea.MouseLeft})
	_, cmd := m.Update(tea.MouseReleaseMsg{X: 10, Y: 1, Button: tea.MouseLeft})

	if cmd == nil {
		t.Fatal("expected clipboard command after drag")
	}
	if !strings.Contains(m.Notice, "Copied") {
		t.Fatalf("Notice = %q, want copy notification", m.Notice)
	}

	// Stale expiry must not clear a newer notice; matching expiry clears it.
	m.Update(NoticeExpiredMsg{Seq: m.noticeSeq - 1})
	if m.Notice == "" {
		t.Fatal("stale expiry cleared notice")
	}
	m.Update(NoticeExpiredMsg{Seq: m.noticeSeq})
	if m.Notice != "" || !m.Selection.empty() {
		t.Fatal("expiry did not clear notice and selection")
	}
}

func TestMouseClickWithoutDragDoesNotCopy(t *testing.T) {
	m := NewModel("dev", "", "", "Gemfile.lock", true, false)
	m.Width, m.Height = 80, 24
	m.Update(tea.MouseClickMsg{X: 3, Y: 3, Button: tea.MouseLeft})
	_, cmd := m.Update(tea.MouseReleaseMsg{X: 3, Y: 3, Button: tea.MouseLeft})
	if cmd != nil || m.Notice != "" {
		t.Fatal("plain click must not copy")
	}
}

func TestGemLinks(t *testing.T) {
	gh := &gemfile.GemStatus{Name: "rails", HomepageURL: "https://github.com/rails/rails"}
	none := &gemfile.GemStatus{Name: "foo"}
	other := &gemfile.GemStatus{Name: "bar", HomepageURL: "https://bar.example.com"}

	tests := []struct{ name, got, want string }{
		{"page github", gemPageURL(gh), "https://github.com/rails/rails"},
		{"page fallback", gemPageURL(none), "https://rubygems.org/gems/foo"},
		{"release github", gemReleaseURL(gh), "https://github.com/rails/rails/releases"},
		{"release non-github", gemReleaseURL(other), "https://rubygems.org/gems/bar/versions"},
		{"release no url", gemReleaseURL(none), "https://rubygems.org/gems/foo/versions"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestOpenKeyOnListsWithEmptyData(t *testing.T) {
	m := NewModel("dev", "", "", "Gemfile.lock", true, false)
	m.CurrentView = ViewGemList
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"}); cmd != nil {
		t.Fatal("o on empty gem list must be a no-op")
	}
	m.CurrentView = ViewUpgradeable
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"}); cmd != nil {
		t.Fatal("o on empty upgradable list must be a no-op")
	}
	m.FirstLevelGems = []*gemfile.GemStatus{{Name: "rails"}}
	m.CurrentView = ViewGemList
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"}); cmd == nil {
		t.Fatal("o on gem list should open the browser")
	}
}
