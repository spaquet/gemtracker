package gemfile

import (
	"slices"
	"testing"
)

func TestExtractVersionsFromFirstLine(t *testing.T) {
	if got := extractVersionsFromFirstLine("pgvector (0.3.3, 0.3.2) trailing"); !slices.Equal(got, []string{"0.3.3", "0.3.2"}) {
		t.Fatalf("unexpected versions: %v", got)
	}
	if got := extractVersionsFromFirstLine("pg (1.6.3"); got != nil {
		t.Fatalf("expected no versions, got %v", got)
	}
}
