package depgraph

import (
	"errors"
	"fmt"
)

var (
	ErrCircularDependency = errors.New("circular dependency")
	ErrDependsOnSelf      = errors.New("node depends on self")
	ErrDependentExists    = errors.New("dependent exists")
)

type (
	NodeSet map[string]struct{}
	DepMap  map[string]NodeSet
)

type Graph struct {
	nodes        NodeSet // All nodes in a set
	dependents   DepMap  // parent -> []child
	dependencies DepMap  // child  -> []parent
}

func New() Graph {
	return Graph{
		nodes:        make(map[string]struct{}),
		dependencies: make(map[string]NodeSet),
		dependents:   make(map[string]NodeSet),
	}
}

func (g *Graph) GraphNodes() NodeSet       { return copyMap(g.nodes) }
func (g *Graph) GraphDependencies() DepMap { return copyMap(g.dependencies) }
func (g *Graph) GraphDependents() DepMap   { return copyMap(g.dependents) }

func (g *Graph) Clone() Graph {
	return Graph{
		nodes:        copyMap(g.nodes),
		dependencies: copyMap(g.dependencies),
		dependents:   copyMap(g.dependents),
	}
}

func (g *Graph) Contains(node string) bool {
	_, ok := g.nodes[node]

	return ok
}

// AddDependency establishes the dependency relationship between 2 nodes.
// It errs if a node depends on itself, or if circular dependency is found.
func (g *Graph) AddDependency(dependent, dependency string) error {
	if dependent == dependency {
		return ErrDependsOnSelf
	}

	// Parent already depends on child
	if g.DependsOn(dependency, dependent) {
		return ErrCircularDependency
	}

	addToMap(g.dependents, dependency, dependent)
	addToMap(g.dependencies, dependent, dependency)

	g.nodes[dependency] = struct{}{}
	g.nodes[dependent] = struct{}{}

	return nil
}

func (g *Graph) DependsOn(dependent, dependency string) bool {
	depcyDepcies := g.Dependencies(dependent)
	_, ok := depcyDepcies[dependency]

	return ok
}

func (g *Graph) DependsOnDirectly(dependent, dependency string) bool {
	deps := g.dependencies[dependent]
	_, ok := deps[dependency]

	return ok
}

func (g *Graph) Leaves() []string {
	var leaves []string
	for node := range g.nodes {
		if _, ok := g.dependencies[node]; !ok {
			leaves = append(leaves, node)
		}
	}

	return leaves
}

// Dependencies returns all deep dependencies
func (g *Graph) Dependencies(node string) NodeSet {
	if _, found := g.nodes[node]; !found {
		return nil
	}

	dependencies := make(NodeSet)
	searchNext := []string{node}

	for len(searchNext) != 0 {
		var discovered []string
		for _, next := range searchNext {
			deps, ok := g.dependencies[next]
			if !ok {
				continue
			}

			for dep := range deps {
				if _, ok := dependencies[dep]; !ok {
					dependencies[dep] = struct{}{}
					discovered = append(discovered, dep)
				}
			}
		}

		searchNext = discovered
	}

	return dependencies
}

// Dependencies returns all deep dependencts
func (g *Graph) Dependents(node string) NodeSet {
	if _, found := g.nodes[node]; !found {
		return nil
	}

	dependents := make(NodeSet)
	searchNext := []string{node}

	for len(searchNext) != 0 {
		var discovered []string
		for _, next := range searchNext {
			deps, ok := g.dependents[next]
			if !ok {
				continue
			}

			for dep := range deps {
				if _, ok := dependents[dep]; !ok {
					dependents[dep] = struct{}{}
					discovered = append(discovered, dep)
				}
			}
		}

		searchNext = discovered
	}

	return dependents
}

// Layers returns nodes in topological sort order.
// Nodes in each outer slot only depend on prior slots,
// i.e. independent nodes come before dependent ones.
func (g *Graph) Layers() [][]string {
	var layers [][]string
	copied := g.Clone()
	for {
		leaves := copied.Leaves()
		if len(leaves) == 0 {
			break
		}

		layers = append(layers, leaves)

		for i := range leaves {
			copied.Delete(leaves[i])
		}
	}

	return layers
}

// RemoveAutoRemove removes target node as well as its dependents and dependencies,
// like with APT or Homebrew autoremove command.
func (g *Graph) RemoveAutoRemove(target string) {
	queue := []string{target}

	for len(queue) != 0 {
		current := popQueue(&queue)
		for dependent := range g.dependents[current] {
			removeFromDepMap(g.dependents, current, dependent)
			removeFromDepMap(g.dependencies, dependent, current)

			queue = append(queue, dependent)
		}

		for dependency := range g.dependencies[current] {
			siblings, ok := g.dependents[dependency]
			if !ok {
				panic("bug")
			}

			removeFromDepMap(g.dependents, dependency, current)
			removeFromDepMap(g.dependencies, current, dependency)

			// Check if current is the only dependent node on this dependency
			// If so, we can safely remove this dependency from the graph
			_, ok = siblings[current]
			if len(siblings) == 1 && ok {
				queue = append(queue, dependency)
			}
		}

		delete(g.nodes, current)
	}
}

// Remove removes target including its dependents
func (g *Graph) RemoveForce(target string) {
	queue := []string{target}

	for len(queue) != 0 {
		current := popQueue(&queue)
		for dependent := range g.dependents[current] {
			removeFromDepMap(g.dependents, current, dependent)
			removeFromDepMap(g.dependencies, dependent, current)

			queue = append(queue, dependent)
		}

		for dependency := range g.dependencies[current] {
			_, ok := g.dependents[dependency]
			if !ok {
				panic("bug")
			}

			removeFromDepMap(g.dependents, dependency, current)
			removeFromDepMap(g.dependencies, current, dependency)
		}

		delete(g.nodes, current)
	}
}

// Remove removes target with 0 dependents.
// Otherwise it returns ErrDependentExists.
func (g *Graph) Remove(target string) error {
	_, ok := g.dependents[target]
	if ok {
		return ErrDependentExists
	}

	g.Delete(target)

	return nil
}

// Delete removes a node and all of its references
// without checking for or handling dangling references
func (g *Graph) Delete(node string) {
	// Delete all edges to dependents
	for dependent := range g.dependents[node] {
		removeFromDepMap(g.dependencies, dependent, node)
	}
	delete(g.dependents, node)

	// Delete all edges to dependencies
	for dependency := range g.dependencies[node] {
		removeFromDepMap(g.dependents, dependency, node)
	}
	delete(g.dependencies, node)

	// Delete node from nodes
	delete(g.nodes, node)
}

func (g *Graph) AssertRelationships() {
	// Asserts that every node has valid references in all fields
	for parent := range g.dependents {
		_, ok := g.nodes[parent]
		if !ok {
			panic(fmt.Sprintf("dangling dependency: %v", parent))
		}

		for child := range g.dependents[parent] {
			_, ok = g.nodes[child]
			if !ok {
				panic(fmt.Sprintf("dangling dependents for parent %s: child: %v", parent, child))
			}
		}
	}

	for child := range g.dependencies {
		_, ok := g.nodes[child]
		if !ok {
			panic(fmt.Sprintf("dangling dependents: %v", child))
		}

		for parent := range g.dependencies[child] {
			_, ok = g.nodes[parent]
			if !ok {
				panic(fmt.Sprintf("dangling child %v parent: %v", child, parent))
			}
		}
	}
}

func addToMap(m DepMap, key, node string) {
	set := m[key]
	if set == nil {
		set = make(NodeSet)
		m[key] = set
	}

	set[node] = struct{}{}
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	copied := make(map[K]V)
	for k, v := range m {
		copied[k] = v
	}

	return copied
}

func removeFromDepMap(deps DepMap, key, node string) {
	nodes := deps[key]
	if len(nodes) == 1 {
		delete(deps, key)
		return
	}

	delete(nodes, node)
}

func popQueue[T any](p *[]T) T {
	var t T
	if p == nil {
		return t
	}

	queue := *p
	l := len(queue)
	if l == 0 {
		return t
	}

	item := queue[0]
	if l == 1 {
		*p = []T{}
		return item
	}

	*p = queue[1:]
	return item
}
