package depgraph_test

import (
	"errors"
	"testing"

	"github.com/soyart/depgraph"
)

func testGraph() depgraph.Graph {
	g := depgraph.New()
	g.AddDependency("b", "a")
	g.AddDependency("c", "a")
	g.AddDependency("d", "c")

	g.AddDependency("y", "x")

	g.AddDependency("ข", "ก")

	return g
}

func TestAddDependency(t *testing.T) {
	// b,c -> a
	// ก   -> ข

	g := depgraph.New()
	err := g.AddDependency("b", "a")
	if err != nil {
		t.Error(err)
	}

	err = g.AddDependency("c", "a")
	if err != nil {
		t.Error(err)
	}

	err = g.AddDependency("ข", "ก")
	if err != nil {
		t.Error(err)
	}

	t.Log(g)

	if !g.DependsOn("b", "a") {
		t.Fatal("b should depend on a")
	}
	if !g.DependsOn("c", "a") {
		t.Fatal("c should depend on a")
	}
	if g.DependsOn("c", "b") {
		t.Fatal("c should not depend on b")
	}
	if g.DependsOn("a", "d") {
		t.Fatal("a should be leave")
	}

	err = g.AddDependency("d", "c")
	if err != nil {
		t.Error(err)
	}

	err = g.AddDependency("a", "d")
	if !errors.Is(err, depgraph.ErrCircularDependency) {
		t.Fatal("expecting circular dependency error")
	}
}

func TestDependencies(t *testing.T) {
	g := depgraph.New()
	g.AddDependency("b", "a")
	g.AddDependency("c", "a")
	g.AddDependency("x", "c")
	g.AddDependency("ก", "c")
	g.AddDependency("y", "x")
	g.AddDependency("ข", "ก")

	deps := g.Dependencies("ข")
	assertContainsAll(t, deps, []string{"ก", "a", "c"})

	deps = g.Dependencies("c")
	assertContainsAll(t, deps, []string{"a"})

	deps = g.Dependencies("x")
	assertContainsAll(t, deps, []string{"a", "c"})

	deps = g.Dependencies("y")
	assertContainsAll(t, deps, []string{"a", "c", "x"})
}

func TestDependents(t *testing.T) {
	g := depgraph.New()
	g.AddDependency("b", "a")
	g.AddDependency("c", "a")
	g.AddDependency("x", "c")
	g.AddDependency("ก", "c")
	g.AddDependency("y", "x")
	g.AddDependency("ข", "ก")

	deps := g.Dependents("ข")
	assertContainsAll(t, deps, []string{})

	deps = g.Dependents("b")
	assertContainsAll(t, deps, []string{})

	deps = g.Dependents("x")
	assertContainsAll(t, deps, []string{"y"})

	deps = g.Dependents("a")
	assertContainsAll(t, deps, []string{"b", "c", "x", "y", "ก", "ข"})

	deps = g.Dependents("c")
	assertContainsAll(t, deps, []string{"x", "y", "ก", "ข"})
}

func TestRemove(t *testing.T) {
	g := testGraph()
	var err error

	err = g.Remove("y")
	if err != nil {
		t.Log("after-remove", "y", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("x")
	if err != nil {
		t.Log("after-remove", "x", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("ข")
	if err != nil {
		t.Log("after-remove", "ข", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("ก")
	if err != nil {
		t.Log("after-remove", "ก", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("a")
	if err == nil {
		t.Log("after-remove", "a", g)
		t.Fatal("expecting error from removing a")
	}
}

func TestAutoRemove(t *testing.T) {
	type testCaseDefaultGraphs struct {
		removes        []string
		nodesRemaining []string
		willBeEmpty    bool
	}

	tests := []testCaseDefaultGraphs{
		{
			removes:        []string{"d"},
			willBeEmpty:    false,
			nodesRemaining: []string{"a", "b", "x", "y", "ก", "ข"},
		},
		{
			removes:        []string{"a", "x"},
			willBeEmpty:    false,
			nodesRemaining: []string{"ก", "ข"},
		},
		{
			removes:     []string{"a", "x", "ข"},
			willBeEmpty: true,
		},
	}

	for i := range tests {
		testCase := &tests[i]
		g := testGraph()
		testAutoRemove(t, g, testCase.removes)

		if testCase.willBeEmpty {
			assertEmptyGraph(t, g)
			continue
		}

		assertNotEmptyGraph(t, g)
		assertRemainingNode(t, g, testCase.nodesRemaining)
	}
}

func TestAutoRemoveLeaves(t *testing.T) {
	testAutoRemoveLeaves(t, testGraph())
}

func testAutoRemove(t *testing.T, g depgraph.Graph, toRemoves []string) {
	t.Log("testAutoRemove-before", g, "removes", toRemoves)
	for _, rm := range toRemoves {
		g.RemoveAutoRemove(rm)
		g.AssertRelationships()
	}

	t.Log("testAutoRemove-after", g)
}

func testAutoRemoveLeaves(t *testing.T, g depgraph.Graph) {
	for _, leaf := range g.Leaves() {
		g.RemoveAutoRemove(leaf)
	}

	g.AssertRelationships()
	assertEmptyGraph(t, g)
}

func TestDebugDepGraph(t *testing.T) {
	g := testGraph()
	t.Log("DebugTest: Leaves=", g.Leaves())
	t.Logf("DebugTest: Graph=%+v", g)

	layers := g.Layers()
	t.Log("DebugTest:Layers=", layers)

	deletes := []string{"c"}

	for _, del := range deletes {
		// t.Log("DebugTest-before: Removing=", del)
		// t.Log("DebugTest-before: Leaves=", g.Leaves())
		// t.Logf("DebugTest-before: Graph=%+v", g)

		g.RemoveAutoRemove(del)

		// t.Log("DebugTest-after: Leaves=", g.Leaves())
		// t.Logf("DebugTest-after: Graph=%+v", g)

		g.AssertRelationships()
	}
}

func assertContainsAll[K comparable, V any](t *testing.T, m map[K]V, things []K) {
	for _, thing := range things {
		_, ok := m[thing]
		if !ok {
			t.Fatalf("thing %v is not in map", thing)
		}
	}
}

func assertEmptyGraph(t *testing.T, g depgraph.Graph) {
	if len(g.GraphNodes()) != 0 {
		t.Fatalf("graph not empty after all leaves removed: nodes=%v", g.GraphNodes())
	}

	if len(g.GraphDependencies()) != 0 {
		t.Fatalf("parents not null after all leaves removed: parents=%v", g.GraphDependencies())
	}

	if len(g.GraphDependents()) != 0 {
		t.Fatalf("children not null after all leaves removed: parents=%v", g.GraphDependents())
	}
}

func assertNotEmptyGraph(t *testing.T, g depgraph.Graph) {
	if len(g.GraphNodes()) == 0 {
		t.Fatalf("graph not empty after all leaves removed: nodes=%v", g.GraphNodes())
	}

	if len(g.GraphDependencies()) == 0 {
		t.Fatalf("parents not null after all leaves removed: parents=%v", g.GraphDependencies())
	}

	if len(g.GraphDependents()) == 0 {
		t.Fatalf("children not null after all leaves removed: parents=%v", g.GraphDependents())
	}
}

func assertRemainingNode(t *testing.T, g depgraph.Graph, nodes []string) {
	if len(nodes) == 0 {
		t.Fatal("remainingNodes is null")
	}

	for _, node := range nodes {
		var found bool
		_, ok := g.GraphNodes()[node]
		if !ok {
			t.Fatalf("missing node %v", node)
		}

		_, ok = g.GraphDependents()[node]
		if ok {
			found = true
		} else {
			for _, children := range g.GraphDependents() {
				_, ok := children[node]
				if ok {
					found = true
				}
			}
		}

		_, ok = g.GraphDependencies()[node]
		if ok {
			found = true
		} else {
			for _, parents := range g.GraphDependencies() {
				_, ok := parents[node]
				if ok {
					found = true
				}
			}
		}

		if !found {
			t.Fatalf("node %v not found", node)
		}
	}
}
