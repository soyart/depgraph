# depgraph

depgraph is a simple, Go generic map-based dependency graph implementation.
Any `comparable` types can be used as nodes in depgraph.

It uses 3 hash maps to implement dependency graphs,
so it's not really space-efficient.

To address the ever-growing memory use by Go map memory leaks,
depgraph provides `*Graph.Clone() Graph` to create an equivalent
graph with lower memory footprint.

## Features

- Arbitary order of adding dependencies

  Users can arbitarily adds new dependencies to graph, so long
  that new dependencies do not violate any rules (e.g. circular dependency).

  Users can do something like this:

  ```go
  func foo() {
    // New graph with string nodes
    g := depgraph.New[string]()

    _ = g.AddDependency("b", "a") // Nodes b and a are both initialized as it's inserted (b depends on a)
    _ = g.AddDependency("c", "b") // Node c gets initialized to depend on node b
  }
  ```

- Prevents circular dependencies, and self-dependent nodes

  Any circular dependencies, deep or direct, are prevented on insertion:

  ```go
  func foo() {
    var err error

    g := depgraph.New[string]()
    err = g.AddDependency("b", "a") // ok: b -> a
    err = g.AddDependency("c", "b") // ok: c -> b
    err = g.AddDependency("d", "c") // ok: d -> c

    err = g.AddDependency("a", "d") // error! circular dependency! a -> d -> c -> b -> a
  }
  ```

- Auto-removal of dependencies and dependents

  depgraph provides many node removal strategies:

  1. `Remove` (standard)

    Removes target nodes with 0 dependents
  
    ```go
    func foo() {
      var err error

      g := depgraph.New[string]()
      err = g.AddDependency("b", "a")
      err = g.AddDependency("c", "b")
      err = g.AddDependency("d", "c")

      err = g.Remove("d") // ok: d has 0 dependents
      err = g.Remove("c") // ok: c now has 0 dependents (d was just removed)

      err = g.Remove("a") // error! 'a' still has dependent 'b'
    }
    ```

  2. `RemoveForce`

    Removes target as well as its dependents

    ```go
    func foo() {
      g := depgraph.New[string]()

      _ = g.AddDependency("b", "a")
      _ = g.AddDependency("c", "b")
      _ = g.AddDependency("d", "c")

      g.RemoveForce("b") // removes b, c, and d
    }
    ```

  3. `RemoveAutoRemove`

    Removes target, as well as its dependents, as well as its dependencies
    whose only dependent is target. This is recursive, and works like how
    `autoremove` command works in package managers (e.g. `apt autoremove <FOO>`)

    ```go
    func foo() {
      g := depgraph.New[string]()

      _ = g.AddDependency("b", "a")
      _ = g.AddDependency("c", "b")
      _ = g.AddDependency("d", "c")
      _ = g.AddDependency("x", "b")
      _ = g.AddDependency("y", "x")

      g.RemoveAutoRemove("y") // removes y and x
    }
    ```

- Topological sort order in layers

  Discover dependencies in layers (nodes in earlier layer will not
  depend on any nodes that appear in later layer):

  ```go
  func foo() {
    g := depgraph.New[string]()

    _ = g.AddDependency("b", "a")
    _ = g.AddDependency("c", "b")
    _ = g.AddDependency("d", "c")

    _ = g.AddDependency("x", "a")

    layers := g.Layers() // [["a"], ["b", "x"], ["c"], ["d"]]
  }
  ```
