package gemfile

// DependencyNode represents a node in a dependency tree, used for displaying forward
// and reverse dependency chains with version information and nesting depth.
type DependencyNode struct {
	// Name is the gem name
	Name string
	// Version is the gem version at this node
	Version string
	// Children are the direct dependencies (or dependents for reverse trees)
	Children []*DependencyNode
	// Depth is the nesting level in the tree (0 for root, increments for each level)
	Depth int
}

// DependencyInfo contains forward and reverse dependency information for a selected gem,
// including both simple lists and tree structures for visualization.
type DependencyInfo struct {
	// GemName is the selected gem name
	GemName string
	// Version is the selected gem's version
	Version string
	// ForwardDeps lists the gems that this gem depends on (direct dependencies only)
	ForwardDeps []string
	// ReverseDeps lists the gems that depend on this gem (direct dependents only)
	ReverseDeps []string
	// ForwardDepsCount is the count of direct forward dependencies
	ForwardDepsCount int
	// ReverseDepsCount is the count of direct reverse dependencies
	ReverseDepsCount int
	// ForwardTree is a tree structure showing transitive dependencies of this gem
	ForwardTree *DependencyNode
	// ReverseTree is a tree structure showing what depends on this gem (up to 3 levels)
	ReverseTree *DependencyNode
}

// DependencyResult contains the analysis result for a selected gem's dependencies.
type DependencyResult struct {
	// SelectedGem is the name of the gem being analyzed
	SelectedGem string
	// DependencyInfo contains the dependency analysis for the selected gem
	DependencyInfo *DependencyInfo
	// AllGems is a reference to the full gem map for version lookups
	AllGems map[string]*Gem
}

// AnalyzeDependencies analyzes forward and reverse dependencies for a given gem.
// It returns both lists of direct dependencies and tree structures showing transitive relationships.
// Trees are limited to prevent circular dependencies: forward tree is capped at depth 5,
// reverse tree is capped at depth 3.
func AnalyzeDependencies(gemfile *Gemfile, selectedGemName string) *DependencyResult {
	result := &DependencyResult{
		SelectedGem: selectedGemName,
		AllGems:     gemfile.Gems,
	}

	// Get selected gem
	selectedGem, ok := gemfile.Gems[selectedGemName]
	if !ok {
		return result
	}

	// Build forward dependencies (what this gem depends on)
	forwardDeps := selectedGem.Dependencies

	// Build reverse dependencies (what gems depend on this gem)
	reverseDeps := []string{}
	for _, gem := range gemfile.Gems {
		for _, dep := range gem.Dependencies {
			if dep == selectedGemName {
				reverseDeps = append(reverseDeps, gem.Name)
				break
			}
		}
	}

	// Build dependency trees
	visited := make(map[string]bool)
	forwardTree := buildDependencyTree(selectedGemName, gemfile, visited, 0)

	visited = make(map[string]bool)
	reverseTree := buildReverseDependencyTree(selectedGemName, gemfile, visited, 0)

	result.DependencyInfo = &DependencyInfo{
		GemName:          selectedGem.Name,
		Version:          selectedGem.Version,
		ForwardDeps:      forwardDeps,
		ReverseDeps:      reverseDeps,
		ForwardDepsCount: len(forwardDeps),
		ReverseDepsCount: len(reverseDeps),
		ForwardTree:      forwardTree,
		ReverseTree:      reverseTree,
	}

	return result
}

// buildDependencyTree recursively builds a tree of what a gem depends on (forward dependencies).
// It prevents circular dependencies and infinite loops with a visited map and depth limit (max 5).
// Returns nil if the gem is already visited at this branch or depth exceeds the limit.
func buildDependencyTree(gemName string, gemfile *Gemfile, visited map[string]bool, depth int) *DependencyNode {
	if visited[gemName] || depth > 5 { // Prevent infinite loops and limit depth
		return nil
	}
	visited[gemName] = true

	gem, ok := gemfile.Gems[gemName]
	if !ok {
		return &DependencyNode{Name: gemName, Version: "?", Depth: depth}
	}

	node := &DependencyNode{
		Name:     gemName,
		Version:  gem.Version,
		Depth:    depth,
		Children: make([]*DependencyNode, 0),
	}

	// Add direct dependencies as children
	for _, depName := range gem.Dependencies {
		child := buildDependencyTree(depName, gemfile, visited, depth+1)
		if child != nil {
			node.Children = append(node.Children, child)
		}
	}

	return node
}

// GetReverseDependencies returns a list of gems that directly depend on the given gem.
// This is a simple list of direct dependents, useful for quick lookups without building a tree.
func GetReverseDependencies(gemName string, gemfile *Gemfile) []string {
	reverseDeps := []string{}
	for _, gem := range gemfile.Gems {
		for _, dep := range gem.Dependencies {
			if dep == gemName {
				reverseDeps = append(reverseDeps, gem.Name)
				break
			}
		}
	}
	return reverseDeps
}

// buildReverseDependencyTree recursively builds a tree showing what depends on a gem (reverse dependencies).
// It prevents circular dependencies with a visited map and limits depth to 3 (shallower than forward tree
// because reverse trees show less relevant context for deep nesting). For each parent gem, it also shows
// the parent's other dependencies for context.
func buildReverseDependencyTree(gemName string, gemfile *Gemfile, visited map[string]bool, depth int) *DependencyNode {
	if visited[gemName] || depth > 3 { // Limit depth for reverse deps to prevent too much nesting
		return nil
	}
	visited[gemName] = true

	gem, ok := gemfile.Gems[gemName]
	if !ok {
		return &DependencyNode{Name: gemName, Version: "?", Depth: depth}
	}

	node := &DependencyNode{
		Name:     gemName,
		Version:  gem.Version,
		Depth:    depth,
		Children: make([]*DependencyNode, 0),
	}

	// Find all gems that depend on this gem
	directParents := make(map[string]bool)
	for _, parentGem := range gemfile.Gems {
		for _, dep := range parentGem.Dependencies {
			if dep == gemName {
				directParents[parentGem.Name] = true
				break
			}
		}
	}

	// Add parent nodes
	for parentName := range directParents {
		parentGem, ok := gemfile.Gems[parentName]
		if !ok {
			continue
		}

		parentNode := &DependencyNode{
			Name:     parentName,
			Version:  parentGem.Version,
			Depth:    depth + 1,
			Children: make([]*DependencyNode, 0),
		}

		// For the parent's dependencies (other than the current gem), add all as children
		for _, dep := range parentGem.Dependencies {
			if dep == gemName {
				continue
			}
			depGem, ok := gemfile.Gems[dep]
			if ok {
				parentNode.Children = append(parentNode.Children, &DependencyNode{
					Name:     dep,
					Version:  depGem.Version,
					Depth:    depth + 2,
					Children: make([]*DependencyNode, 0),
				})
			}
		}

		node.Children = append(node.Children, parentNode)
	}

	return node
}
