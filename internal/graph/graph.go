package graph

import (
	"sort"

	"github.com/atoolz/terraxi/pkg/types"
)

// DependencyGraph tracks relationships between discovered resources.
type DependencyGraph struct {
	nodes map[string]types.Resource
	edges map[string][]string // resource key -> dependent resource keys
}

// New creates an empty dependency graph.
func New() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]types.Resource),
		edges: make(map[string][]string),
	}
}

// resourceKey returns a unique key for a resource.
func resourceKey(r types.Resource) string {
	return r.Type + "." + r.ID
}

// Add inserts a resource into the graph.
func (g *DependencyGraph) Add(r types.Resource) {
	key := resourceKey(r)
	g.nodes[key] = r

	for _, dep := range r.Dependencies {
		depKey := dep.Type + "." + dep.ID
		g.edges[key] = append(g.edges[key], depKey)
	}
}

// AddAll inserts multiple resources into the graph.
func (g *DependencyGraph) AddAll(resources []types.Resource) {
	for _, r := range resources {
		g.Add(r)
	}
}

// DependenciesOf returns all resources that the given resource depends on.
func (g *DependencyGraph) DependenciesOf(r types.Resource) []types.Resource {
	key := resourceKey(r)
	depKeys := g.edges[key]

	var deps []types.Resource
	for _, dk := range depKeys {
		if dep, ok := g.nodes[dk]; ok {
			deps = append(deps, dep)
		}
	}
	return deps
}

// DependentsOf returns all resources that depend on the given resource.
func (g *DependencyGraph) DependentsOf(r types.Resource) []types.Resource {
	targetKey := resourceKey(r)
	var dependents []types.Resource

	for key, edges := range g.edges {
		for _, edge := range edges {
			if edge == targetKey {
				if node, ok := g.nodes[key]; ok {
					dependents = append(dependents, node)
				}
			}
		}
	}
	return dependents
}

// TopologicalSort returns resources in dependency order (dependencies first).
// Circular dependencies are detected and the back-edge is skipped to prevent
// stack overflow (AWS security groups can reference each other).
func (g *DependencyGraph) TopologicalSort() []types.Resource {
	visited := make(map[string]bool) // fully processed
	inStack := make(map[string]bool) // currently on the recursion stack
	var sorted []types.Resource
	var visit func(key string)

	visit = func(key string) {
		if visited[key] {
			return
		}
		if inStack[key] {
			// Cycle detected. Skip this edge to break the cycle.
			return
		}
		inStack[key] = true
		for _, depKey := range g.edges[key] {
			visit(depKey)
		}
		inStack[key] = false
		visited[key] = true
		if node, ok := g.nodes[key]; ok {
			sorted = append(sorted, node)
		}
	}

	// Sort keys for deterministic output across runs
	keys := make([]string, 0, len(g.nodes))
	for key := range g.nodes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		visit(key)
	}
	return sorted
}

// Resources returns all resources in the graph.
func (g *DependencyGraph) Resources() []types.Resource {
	resources := make([]types.Resource, 0, len(g.nodes))
	for _, r := range g.nodes {
		resources = append(resources, r)
	}
	return resources
}

// Len returns the number of resources in the graph.
func (g *DependencyGraph) Len() int {
	return len(g.nodes)
}
