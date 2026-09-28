package ui

import (
	"testing"

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
