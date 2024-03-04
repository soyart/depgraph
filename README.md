# depgraph

depgraph is a simple, Go map-based dependency graph implementation.

depgraph supports arbitary adding of dependencies,
prevents circular dependencies and other features,
like package manager style node removal (autoremove).

It also supports dependencies layers, sorted in topological order.

It uses 3 hash maps to implement dependency graphs,
so it's not really space-efficient.
