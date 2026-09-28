package gemfile

import (
	"slices"
	"testing"
)

func TestSanitizeGemOutput(t *testing.T) {
	got := sanitizeGemOutput("A\x1b[31mB\x1b[0m\xff\ufffd")
	if got != "AB??" {
		t.Fatalf("unexpected sanitized output: %q", got)
	}
}

func TestExtractVersionsFromFirstLine(t *testing.T) {
	if got := extractVersionsFromFirstLine("pgvector (0.3.3, 0.3.2) trailing"); !slices.Equal(got, []string{"0.3.3", "0.3.2"}) {
		t.Fatalf("unexpected versions: %v", got)
	}
	if got := extractVersionsFromFirstLine("pg (1.6.3"); got != nil {
		t.Fatalf("expected no versions, got %v", got)
	}
}
