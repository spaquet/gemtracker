package gemfile

import (
	"sort"
	"testing"
)

func TestAnalyzeDependencies_ForwardDeps(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := AnalyzeDependencies(gf, "rails")

	if result.SelectedGem != "rails" {
		t.Errorf("expected selected gem 'rails', got %q", result.SelectedGem)
	}

	if result.DependencyInfo == nil {
		t.Fatal("expected DependencyInfo, got nil")
	}

	if result.DependencyInfo.GemName != "rails" {
		t.Errorf("expected gem name 'rails', got %q", result.DependencyInfo.GemName)
	}

	// Rails has forward dependencies
	if result.DependencyInfo.ForwardDepsCount == 0 {
		t.Error("expected rails to have forward dependencies")
	}

	// Check specific forward dependencies
	expectedDeps := []string{"actionpack", "activesupport", "railties"}
	for _, expected := range expectedDeps {
		found := false
		for _, actual := range result.DependencyInfo.ForwardDeps {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected forward dependency %q not found", expected)
		}
	}
}

func TestAnalyzeDependencies_ReverseDeps(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// concurrent-ruby is a dependency of activesupport and tzinfo
	result := AnalyzeDependencies(gf, "concurrent-ruby")

	if result.DependencyInfo == nil {
		t.Fatal("expected DependencyInfo, got nil")
	}

	// concurrent-ruby should have reverse dependencies
	if result.DependencyInfo.ReverseDepsCount == 0 {
		t.Error("expected concurrent-ruby to have reverse dependencies")
	}
}

func TestAnalyzeDependencies_NonexistentGem(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := AnalyzeDependencies(gf, "nonexistent-gem")

	if result.SelectedGem != "nonexistent-gem" {
		t.Errorf("expected selected gem 'nonexistent-gem', got %q", result.SelectedGem)
	}

	// DependencyInfo should still be nil for nonexistent gem
	if result.DependencyInfo != nil {
		t.Error("expected nil DependencyInfo for nonexistent gem")
	}
}

func TestBuildDependencyTree(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := AnalyzeDependencies(gf, "rails")

	if result.DependencyInfo.ForwardTree == nil {
		t.Fatal("expected ForwardTree, got nil")
	}

	// Root should be rails
	if result.DependencyInfo.ForwardTree.Name != "rails" {
		t.Errorf("expected root 'rails', got %q", result.DependencyInfo.ForwardTree.Name)
	}

	// Root should have depth 0
	if result.DependencyInfo.ForwardTree.Depth != 0 {
		t.Errorf("expected root depth 0, got %d", result.DependencyInfo.ForwardTree.Depth)
	}

	// Root should have children
	if len(result.DependencyInfo.ForwardTree.Children) == 0 {
		t.Error("expected rails to have child dependencies")
	}

	// Check that children have correct depth
	for _, child := range result.DependencyInfo.ForwardTree.Children {
		if child.Depth != 1 {
			t.Errorf("expected child depth 1, got %d", child.Depth)
		}
	}
}

func TestBuildReverseDependencyTree(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := AnalyzeDependencies(gf, "rack")

	if result.DependencyInfo.ReverseTree == nil {
		t.Fatal("expected ReverseTree, got nil")
	}

	// Root should be rack
	if result.DependencyInfo.ReverseTree.Name != "rack" {
		t.Errorf("expected root 'rack', got %q", result.DependencyInfo.ReverseTree.Name)
	}

	// Root should have parents (gems that depend on it)
	if len(result.DependencyInfo.ReverseTree.Children) == 0 {
		t.Error("expected rack to have reverse dependencies")
	}
}

func TestDependencyTreeDepthLimit(t *testing.T) {
	// Create a test gemfile with deep nesting
	gf := &Gemfile{
		Gems: map[string]*Gem{
			"gem-a": {
				Name:         "gem-a",
				Version:      "1.0.0",
				Dependencies: []string{"gem-b"},
			},
			"gem-b": {
				Name:         "gem-b",
				Version:      "1.0.0",
				Dependencies: []string{"gem-c"},
			},
			"gem-c": {
				Name:         "gem-c",
				Version:      "1.0.0",
				Dependencies: []string{"gem-d"},
			},
			"gem-d": {
				Name:         "gem-d",
				Version:      "1.0.0",
				Dependencies: []string{"gem-e"},
			},
			"gem-e": {
				Name:         "gem-e",
				Version:      "1.0.0",
				Dependencies: []string{},
			},
		},
	}

	result := AnalyzeDependencies(gf, "gem-a")

	// Traverse the tree and check max depth
	maxDepth := getMaxDepth(result.DependencyInfo.ForwardTree)

	// Forward tree has depth limit of 5, so we should not exceed that
	if maxDepth > 5 {
		t.Errorf("expected max depth <= 5, got %d", maxDepth)
	}
}

func getMaxDepth(node *DependencyNode) int {
	if node == nil {
		return 0
	}

	maxChildDepth := 0
	for _, child := range node.Children {
		if d := getMaxDepth(child); d > maxChildDepth {
			maxChildDepth = d
		}
	}

	return maxChildDepth + 1
}

func TestDependencyInfo_Version(t *testing.T) {
	path := "../../testdata/Gemfile.lock"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := AnalyzeDependencies(gf, "rails")

	if result.DependencyInfo.Version != "7.0.0" {
		t.Errorf("expected version 7.0.0, got %q", result.DependencyInfo.Version)
	}
}

func TestSimpleDependencies(t *testing.T) {
	path := "../../testdata/Gemfile.lock.simple"
	gf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := AnalyzeDependencies(gf, "another-gem")

	if result.DependencyInfo == nil {
		t.Fatal("expected DependencyInfo, got nil")
	}

	// another-gem depends on simple-gem
	if result.DependencyInfo.ForwardDepsCount != 1 {
		t.Errorf("expected 1 forward dependency, got %d", result.DependencyInfo.ForwardDepsCount)
	}

	if result.DependencyInfo.ForwardDeps[0] != "simple-gem" {
		t.Errorf("expected forward dependency 'simple-gem', got %q", result.DependencyInfo.ForwardDeps[0])
	}
}

func TestGetReverseDependencies(t *testing.T) {
	// Create a simple gemfile
	gf := &Gemfile{
		Gems: map[string]*Gem{
			"rails": {
				Name:         "rails",
				Version:      "7.0.0",
				Dependencies: []string{"actionpack", "activesupport"},
			},
			"actionpack": {
				Name:         "actionpack",
				Version:      "7.0.0",
				Dependencies: []string{"rack"},
			},
			"activesupport": {
				Name:         "activesupport",
				Version:      "7.0.0",
				Dependencies: []string{"concurrent-ruby"},
			},
			"rack": {
				Name:         "rack",
				Version:      "2.2.3",
				Dependencies: []string{},
			},
			"concurrent-ruby": {
				Name:         "concurrent-ruby",
				Version:      "1.2.0",
				Dependencies: []string{},
			},
		},
	}

	tests := []struct {
		name     string
		gemName  string
		expected []string
	}{
		{
			name:     "rack has actionpack as reverse dep",
			gemName:  "rack",
			expected: []string{"actionpack"},
		},
		{
			name:     "concurrent-ruby has activesupport as reverse dep",
			gemName:  "concurrent-ruby",
			expected: []string{"activesupport"},
		},
		{
			name:     "rails has no reverse deps",
			gemName:  "rails",
			expected: []string{},
		},
		{
			name:     "actionpack has rails as reverse dep",
			gemName:  "actionpack",
			expected: []string{"rails"},
		},
		{
			name:     "activesupport has rails as reverse dep",
			gemName:  "activesupport",
			expected: []string{"rails"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetReverseDependencies(tt.gemName, gf)
			
			// Sort for comparison
			sort.Strings(result)
			sort.Strings(tt.expected)
			
			if len(result) != len(tt.expected) {
				t.Errorf("got %d reverse dependencies, want %d", len(result), len(tt.expected))
			}
			
			for i, dep := range result {
				if i >= len(tt.expected) || dep != tt.expected[i] {
					t.Errorf("got %v, want %v", result, tt.expected)
					break
				}
			}
		})
	}
}
