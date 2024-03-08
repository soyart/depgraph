package depgraph

import (
	"errors"
	"fmt"
)

var (
	ErrDependsOnSelf      = errors.New("node depends on self")
	ErrDependentExists    = errors.New("dependent exists")
	ErrCircularDependency = errors.New("circular dependency")
)

type (
	Nodes[T comparable]      map[T]struct{} // Nodes is a set of T, implemented with Go map
	Dependency[T comparable] map[T]Nodes[T] // Dependency either maps a dependent to its dependencies, or a dependency to its dependents
)

type Graph[T comparable] struct {
	nodes        Nodes[T]      // All nodes in a set
	dependents   Dependency[T] // parent -> []child
	dependencies Dependency[T] // child  -> []parent
}

func New[T comparable]() Graph[T] {
	return Graph[T]{
		nodes:        make(Nodes[T]),
		dependents:   make(Dependency[T]),
		dependencies: make(Dependency[T]),
	}
}

func (n Nodes[T]) Slice() []T {
	slice := make([]T, len(n))
	i := 0

	for node := range n {
		slice[i] = node
		i++
	}

	return slice
}

func (n Nodes[T]) Contains(item T) bool {
	return contains(n, item)
}

func (d Dependency[T]) ContainsKey(key T) bool {
	return contains(d, key)
}

func (d Dependency[T]) Contains(key, item T) bool {
	// Note: reading from nil maps will not panic
	_, ok := d[key][item]

	return ok
}

func (g *Graph[T]) Contains(node T) bool {
	return contains(g.nodes, node)
}

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

	// Parent already depends on child
	if g.DependsOn(dependency, dependent) {
		return ErrCircularDependency
	}

	addToDep(g.dependents, dependency, dependent)
	addToDep(g.dependencies, dependent, dependency)

	g.nodes[dependency] = struct{}{}
	g.nodes[dependent] = struct{}{}

	return nil
}

// Undepends remove dependent->dependency edges in the graph,
// but not from the nodes, transforming dependent into a leaf.
func (g *Graph[T]) Undepend(dependent, dependency T) {
	removeFromDep(g.dependents, dependency, dependent)
	removeFromDep(g.dependencies, dependent, dependency)
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
func (g *Graph[T]) Leaves() Nodes[T] {
	leaves := make(Nodes[T])

	for node := range g.nodes {
		if len(g.dependencies[node]) != 0 {
			continue
		}

		leaves[node] = struct{}{}
	}

	return leaves
}

// Dependencies returns all deep dependencies
func (g *Graph[T]) Dependencies(node T) Nodes[T] {
	if _, found := g.nodes[node]; !found {
		return nil
	}

	dependencies := make(Nodes[T])
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
func (g *Graph[T]) Dependents(node T) Nodes[T] {
	if _, found := g.nodes[node]; !found {
		return nil
	}

	dependents := make(Nodes[T])
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
func (g *Graph[T]) Layers() []Nodes[T] {
	var layers []Nodes[T]
	copied := g.Clone()
	for {
		leaves := copied.Leaves()
		if len(leaves) == 0 {
			break
		}

		layers = append(layers, copyMap(leaves))

		for leaf := range leaves {
			copied.Delete(leaf)
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
			g.Undepend(dependent, current)
			queue = append(queue, dependent)
		}

		for dependency := range g.dependencies[current] {
			siblings, ok := g.dependents[dependency]
			if !ok {
				panic("bug")
			}

			g.Undepend(current, dependency)

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
			g.Undepend(dependent, current)

			queue = append(queue, dependent)
		}

		for dependency := range g.dependencies[current] {
			_, ok := g.dependents[dependency]
			if !ok {
				panic("bug")
			}

			g.Undepend(current, dependency)
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

	// Delete node from nodes
	delete(g.nodes, target)
}

func (g *Graph[T]) GraphNodes() Nodes[T]               { return copyMap(g.nodes) }              // Returns a copy of all nodes
func (g *Graph[T]) GraphDependents() Dependency[T]     { return copyMap(g.dependents) }         // Returns a copy of dependent map
func (g *Graph[T]) GraphDependencies() Dependency[T]   { return copyMap(g.dependencies) }       // Returns a copy of dependency map
func (g *Graph[T]) DependentsDirect(node T) Nodes[T]   { return copyMap(g.dependents)[node] }   // Returns a copy of direct dependents of node
func (g *Graph[T]) DependenciesDirect(node T) Nodes[T] { return copyMap(g.dependencies)[node] } // Returns a copy of direct dependencies of node

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

func copyDep[T comparable](deps Dependency[T]) Dependency[T] {
	copied := make(Dependency[T])
	for k, v := range deps {
		copied[k] = copyMap(v)
	}

	return copied
}

func addToDep[T comparable](deps Dependency[T], key, node T) {
	nodes := deps[key]
	if nodes == nil {
		nodes = make(Nodes[T])
		deps[key] = nodes
	}

	nodes[node] = struct{}{}
}

func removeFromDep[T comparable](deps Dependency[T], key, target T) {
	nodes := deps[key]
	if len(nodes) == 1 {
		if !contains(nodes, target) {
			return
		}

		delete(deps, key)
		return
	}

	delete(nodes, target)
}
