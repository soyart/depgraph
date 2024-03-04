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
	NodeSet[T comparable] map[T]struct{}
	DepMap[T comparable]  map[T]NodeSet[T]
)

type Graph[T comparable] struct {
	nodes        NodeSet[T] // All nodes in a set
	dependents   DepMap[T]  // parent -> []child
	dependencies DepMap[T]  // child  -> []parent
}

func New[T comparable]() Graph[T] {
	return Graph[T]{
		nodes:        make(NodeSet[T]),
		dependents:   make(DepMap[T]),
		dependencies: make(DepMap[T]),
	}
}

func (s NodeSet[T]) Contains(item T) bool {
	_, ok := s[item]

	return ok
}

func (d DepMap[T]) ContainsKey(key T) bool {
	_, ok := d[key]

	return ok
}

func (d DepMap[T]) Contains(key, item T) bool {
	_, ok := d[key][item]

	return ok
}

func (g *Graph[T]) Contains(node T) bool {
	_, ok := g.nodes[node]

	return ok
}

func (g *Graph[T]) GraphNodes() NodeSet[T]       { return copyMap(g.nodes) }        // Returns a copy of all nodes
func (g *Graph[T]) GraphDependents() DepMap[T]   { return copyMap(g.dependents) }   // Returns a copy of dependent map
func (g *Graph[T]) GraphDependencies() DepMap[T] { return copyMap(g.dependencies) } // Returns a copy of dependency map

func (g *Graph[T]) Clone() Graph[T] {
	return Graph[T]{
		nodes:        copyMap(g.nodes),
		dependencies: copyMap(g.dependencies),
		dependents:   copyMap(g.dependents),
	}
}

// AddDependency establishes the dependency relationship between 2 nodes.
// It errs if a node depends on itself, or if circular dependency is found.
func (g *Graph[T]) AddDependency(dependent, dependency T) error {
	if dependent == dependency {
		return ErrDependsOnSelf
	}

	// Parent already depends on child
	if g.DependsOn(dependency, dependent) {
		return ErrCircularDependency
	}

	addToDepMap(g.dependents, dependency, dependent)
	addToDepMap(g.dependencies, dependent, dependency)

	g.nodes[dependency] = struct{}{}
	g.nodes[dependent] = struct{}{}

	return nil
}

// DependsOn checks if all deep dependencies of dependent contain dependency
func (g *Graph[T]) DependsOn(dependent, dependency T) bool {
	return g.Dependencies(dependent).Contains(dependency)
}

// DependsOnDirectly returns a boolean indicating
// if dependency is a direct dependency of dependent.
func (g *Graph[T]) DependsOnDirectly(dependent, dependency T) bool {
	return g.dependencies.Contains(dependent, dependency)
}

// Leaves returns leave nodes,
// i.e. nodes that do not depend on any other nodes.
func (g *Graph[T]) Leaves() []T {
	var leaves []T
	for node := range g.nodes {
		if !g.dependencies.ContainsKey(node) {
			leaves = append(leaves, node)
		}
	}

	return leaves
}

// Dependencies returns all deep dependencies
func (g *Graph[T]) Dependencies(node T) NodeSet[T] {
	if _, found := g.nodes[node]; !found {
		return nil
	}

	dependencies := make(NodeSet[T])
	searchNext := []T{node}

	for len(searchNext) != 0 {
		var discovered []T
		for _, next := range searchNext {
			deps, ok := g.dependencies[next]
			if !ok {
				continue
			}

			for dep := range deps {
				if dependencies.Contains(dep) {
					continue
				}

				dependencies[dep] = struct{}{}
				discovered = append(discovered, dep)
			}
		}

		searchNext = discovered
	}

	return dependencies
}

// Dependencies returns all deep dependencies
func (g *Graph[T]) Dependents(node T) NodeSet[T] {
	if _, found := g.nodes[node]; !found {
		return nil
	}

	dependents := make(NodeSet[T])
	searchNext := []T{node}

	for len(searchNext) != 0 {
		var discovered []T
		for _, next := range searchNext {
			deps, ok := g.dependents[next]
			if !ok {
				continue
			}

			for dep := range deps {
				if dependents.Contains(dep) {
					continue
				}

				dependents[dep] = struct{}{}
				discovered = append(discovered, dep)
			}
		}

		searchNext = discovered
	}

	return dependents
}

// Layers returns nodes in topological sort order.
// Nodes in each outer slot only depend on prior slots,
// i.e. independent nodes come before dependent ones.
func (g *Graph[T]) Layers() [][]T {
	var layers [][]T
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
func (g *Graph[T]) RemoveAutoRemove(target T) {
	queue := []T{target}

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
func (g *Graph[T]) RemoveForce(target T) {
	queue := []T{target}

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
func (g *Graph[T]) Remove(target T) error {
	if g.dependents.ContainsKey(target) {
		return ErrDependentExists
	}

	g.Delete(target)

	return nil
}

// Delete removes a node and all of its references
// without checking for or handling dangling references
func (g *Graph[T]) Delete(node T) {
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

// AssertRelationship asserts that every node has valid references in all fields.
// Panics if invalid references are found.
func (g *Graph[T]) AssertRelationships() {
	for dependency := range g.dependents {
		if !g.nodes.Contains(dependency) {
			panic(fmt.Sprintf("dangling dependency: %v", dependency))
		}

		for dependent := range g.dependents[dependency] {
			if !g.nodes.Contains(dependent) {
				panic(fmt.Sprintf("dangling dependents for parent %v: child: %v", dependency, dependent))
			}

			if !g.dependencies.Contains(dependent, dependency) {
				panic(fmt.Sprintf("dangling dependents for parent %v: child: %v", dependency, dependent))
			}
		}
	}

	for dependent := range g.dependencies {
		if !g.nodes.Contains(dependent) {
			panic(fmt.Sprintf("dangling dependents: %v", dependent))
		}

		for dependency := range g.dependencies[dependent] {
			if !g.nodes.Contains(dependency) {
				panic(fmt.Sprintf("dangling child %v parent: %v", dependent, dependency))
			}

			if !g.dependents.Contains(dependency, dependent) {
				panic(fmt.Sprintf("dangling child %v parent: %v", dependent, dependency))
			}
		}
	}
}

func addToDepMap[T comparable](m DepMap[T], key, node T) {
	set := m[key]
	if set == nil {
		set = make(NodeSet[T])
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

func removeFromDepMap[T comparable](deps DepMap[T], key, node T) {
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
