package gemfile

type DependencyNode struct {
	Name     string
	Version  string
	Children []*DependencyNode
	Depth    int
}

type DependencyInfo struct {
	GemName          string
	Version          string
	ForwardDeps      []string // What this gem depends on
	ReverseDeps      []string // What depends on this gem
	ForwardDepsCount int
	ReverseDepsCount int
	// Tree structures
	ForwardTree *DependencyNode // Tree of what this gem depends on
	ReverseTree *DependencyNode // Tree of what depends on this gem
}

type DependencyResult struct {
	SelectedGem    string
	DependencyInfo *DependencyInfo
	AllGems        map[string]*Gem // For version lookups
}

// AnalyzeDependencies analyzes dependencies for a selected gem
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

// buildDependencyTree recursively builds a tree of dependencies
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

// GetReverseDependencies returns a list of gems that depend on the given gem
// This is useful for local calculations without needing to rebuild the tree
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

// buildReverseDependencyTree recursively builds a tree of what depends on this gem
// For reverse dependencies, we want to show: gem <- parent1 <- grandparent, etc.
// Plus show the parent's OTHER dependencies for context
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
