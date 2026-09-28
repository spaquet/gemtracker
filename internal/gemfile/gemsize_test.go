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

func TestParseGemInfoFormats(t *testing.T) {
	legacy := ParseGemInfo("rack (3.2.6, 3.2.5)\nInstalled at (3.2.6): /new\n(3.2.5): /old")
	if want := []InstalledVersion{{Version: "3.2.6", Path: "/new"}, {Version: "3.2.5", Path: "/old"}}; !slices.Equal(legacy.Versions, want) {
		t.Fatalf("legacy format: got %v, want %v", legacy.Versions, want)
	}
	current := ParseGemInfo("rack (3.2.6)\nInstalled at: /new")
	if want := []InstalledVersion{{Version: "3.2.6", Path: "/new"}}; !slices.Equal(current.Versions, want) {
		t.Fatalf("current format: got %v, want %v", current.Versions, want)
	}
}
