package depgraph

import (
	"errors"
	"fmt"
)

var (
	ErrDependsOnSelf      = errors.New("node depends on self")
	ErrDependentExists    = errors.New("dependent exists")
	ErrNoSuchDependency   = errors.New("no such direct dependency")
	ErrCircularDependency = errors.New("circular dependency")
)

type (
	Set[T comparable]   map[T]struct{} // Set is a set of T nodes, implemented with Go map
	Edges[T comparable] map[T]Set[T]   // Edges either maps a dependent node to its set of dependencies, or a dependency to its dependents
)

type Graph[T comparable] struct {
	nodes        Set[T]   // All nodes in Graph
	dependents   Edges[T] // dependency -> []dependents
	dependencies Edges[T] // dependent  -> []dependencies
}

func New[T comparable]() Graph[T] {
	return Graph[T]{
		nodes:        make(Set[T]),
		dependents:   make(Edges[T]),
		dependencies: make(Edges[T]),
	}
}

func NodeSet[T comparable](nodes ...T) Set[T] {
	set := make(Set[T])
	for i := range nodes {
		set[nodes[i]] = struct{}{}
	}

	return set
}

func (s Set[T]) Slice() []T {
	slice := make([]T, len(s))
	i := 0

	for node := range s {
		slice[i] = node
		i++
	}

	return slice
}

func (s Set[T]) Contains(item T) bool {
	return contains(s, item)
}

func (e Edges[T]) ContainsKey(key T) bool {
	return contains(e, key)
}

func (e Edges[T]) Contains(key, item T) bool {
	// Note: reading from nil maps will not panic
	_, ok := e[key][item]

	return ok
}

func (g *Graph[T]) Contains(node T) bool {
	return contains(g.nodes, node)
}

func (g *Graph[T]) GraphNodes() Set[T]               { return copyMap(g.nodes) }              // Returns a copy of all nodes
func (g *Graph[T]) GraphDependents() Edges[T]        { return copyMap(g.dependents) }         // Returns a copy of dependent map
func (g *Graph[T]) GraphDependencies() Edges[T]      { return copyMap(g.dependencies) }       // Returns a copy of dependency map
func (g *Graph[T]) DependentsDirect(node T) Set[T]   { return copyMap(g.dependents)[node] }   // Returns a copy of direct dependents of node
func (g *Graph[T]) DependenciesDirect(node T) Set[T] { return copyMap(g.dependencies)[node] } // Returns a copy of direct dependencies of node

func (g *Graph[T]) Clone() Graph[T] {
	return Graph[T]{
		nodes:        copyMap(g.nodes),
		dependencies: copyDep(g.dependencies),
		dependents:   copyDep(g.dependents),
	}
}

// Depend establishes the dependency relationship between 2 nodes.
// It errs if a node depends on itself, or if circular dependency is found.
func (g *Graph[T]) Depend(dependent, dependency T) error {
	if dependent == dependency {
		return ErrDependsOnSelf
	}

	if g.DependsOn(dependency, dependent) {
		return ErrCircularDependency
	}

	addToDep(g.dependents, dependency, dependent)
	addToDep(g.dependencies, dependent, dependency)

	g.nodes[dependency] = struct{}{}
	g.nodes[dependent] = struct{}{}

	return nil
}

// Undepend removes dependent->dependency edges in g.
// It returns ErrNoSuchDependency if the relationship is indirect.
func (g *Graph[T]) Undepend(dependent, dependency T) error {
	if !g.DependsOnDirectly(dependent, dependency) {
		return ErrNoSuchDependency
	}

	removeFromDep(g.dependents, dependency, dependent)
	removeFromDep(g.dependencies, dependent, dependency)

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
func (g *Graph[T]) Leaves() Set[T] {
	leaves := make(Set[T])

	for node := range g.nodes {
		if len(g.dependencies[node]) != 0 {
			continue
		}

		leaves[node] = struct{}{}
	}

	return leaves
}

// Dependencies returns all deep dependencies
func (g *Graph[T]) Dependencies(node T) Set[T] {
	if !g.nodes.Contains(node) {
		return nil
	}

	dependencies := make(Set[T])
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
func (g *Graph[T]) Dependents(node T) Set[T] {
	if !g.nodes.Contains(node) {
		return nil
	}

	dependents := make(Set[T])
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
func (g *Graph[T]) Layers() []Set[T] {
	var layers []Set[T]
	copied := g.Clone()

	for len(copied.nodes) != 0 {
		leaves := copied.Leaves()

		for leaf := range leaves {
			copied.Delete(leaf)
		}

		layers = append(layers, copyMap(leaves))
	}

	return layers
}

// RemoveAutoRemove removes target node as well as its dependents and dependencies,
// like with pacman -Rns, or APT autoremove commands.
func (g *Graph[T]) RemoveAutoRemove(target T) {
	queue := []T{target}

	for len(queue) != 0 {
		current := popQueue(&queue)

		for dependent := range g.dependents[current] {
			err := g.Undepend(dependent, current)
			if err != nil {
				panic("bug autoremove undepend dependent->current")
			}

			queue = append(queue, dependent)
		}

		for dependency := range g.dependencies[current] {
			siblings, ok := g.dependents[dependency]
			if !ok {
				panic("bug")
			}

			err := g.Undepend(current, dependency)
			if err != nil {
				panic("bug autoremove undepend current->dependency")
			}

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
			err := g.Undepend(dependent, current)
			if err != nil {
				panic("bug: remove-force undepend dependent->current")
			}

			queue = append(queue, dependent)
		}

		for dependency := range g.dependencies[current] {
			_, ok := g.dependents[dependency]
			if !ok {
				panic("bug")
			}

			err := g.Undepend(current, dependency)
			if err != nil {
				panic("bug: remove-force undepend current->dependency")
			}
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
func (g *Graph[T]) Delete(target T) {
	// Delete all edges to dependencies
	for dependency := range g.dependencies[target] {
		removeFromDep(g.dependents, dependency, target)
	}
	delete(g.dependencies, target)

	// Delete all edges to dependents
	for dependent := range g.dependents[target] {
		removeFromDep(g.dependencies, dependent, target)
	}
	delete(g.dependents, target)

	// Delete target from nodes
	delete(g.nodes, target)
}

// Realloc allocates a new internal maps of g, and drop the old maps,
// hopefully to reduce memory footprints from map memory leaks
// after multiple deletion in large maps.
//
// This can also be done with Clone.
func (g *Graph[T]) Realloc() {
	g.nodes = copyMap(g.nodes)
	g.dependents = copyDep(g.dependents)
	g.dependencies = copyDep(g.dependencies)
}

// AssertRelationship asserts that every node has valid references in all fields.
// Panics if invalid references are found. Currently only used in tests.
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

func popQueue[T any](p *[]T) T {
	if p == nil {
		panic("nil p")
	}

	queue := *p
	l := len(queue)
	if l == 0 {
		panic("empty queue")
	}

	item := queue[0]
	if l == 1 {
		*p = []T{}
		return item
	}

	*p = queue[1:]
	return item
}

func contains[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]

	return ok
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	copied := make(map[K]V)
	for k, v := range m {
		copied[k] = v
	}

	return copied
}

func copyDep[T comparable](deps Edges[T]) Edges[T] {
	copied := make(Edges[T])
	for k, v := range deps {
		copied[k] = copyMap(v)
	}

	return copied
}

func addToDep[T comparable](deps Edges[T], key, node T) {
	nodes := deps[key]
	if nodes == nil {
		nodes = make(Set[T])
		deps[key] = nodes
	}

	nodes[node] = struct{}{}
}

func removeFromDep[T comparable](deps Edges[T], key, target T) {
	nodes := deps[key]
	if !contains(nodes, target) {
		return
	}

	if len(nodes) != 1 {
		delete(nodes, target)
		return
	}

	delete(deps, key)
}
