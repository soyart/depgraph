package depgraph_test

import (
	"testing"

	"github.com/soyart/depgraph"
)

func initTestGraph(t *testing.T) depgraph.Graph[string] {
	valids := map[string][]string{
		// b -> a
		// c -> a, b
		// x -> c
		// y -> x, b
		"b": {"a"},
		"c": {"a"},
		"d": {"c"},
		"y": {"x"},
		"ข": {"ก"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)

	return g
}

func TestAddDependencies(t *testing.T) {
	valids := map[string][]string{
		// b -> a
		// c -> a, b
		// x -> c
		// y -> x, b
		"b": {"a"},
		"c": {"a", "b"},
		"x": {"c"},
		"y": {"x", "b"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)

	// Add some circular dependency and expect error
	err := g.AddDependency("a", "y")
	if err == nil {
		t.Fatal("expecting error from circular dependency")
	}
}

func TestDependencies(t *testing.T) {
	valids := map[string][]string{
		"b": {"a"},
		"c": {"a"},
		"x": {"c"},
		"ก": {"c"},
		"y": {"x"},
		"ข": {"ก"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)

	deps := g.Dependencies("ข")
	assertMapContainsValues(t, deps, []string{"ก", "a", "c"})

	deps = g.Dependencies("c")
	assertMapContainsValues(t, deps, []string{"a"})

	deps = g.Dependencies("x")
	assertMapContainsValues(t, deps, []string{"a", "c"})

	deps = g.Dependencies("y")
	assertMapContainsValues(t, deps, []string{"a", "c", "x"})
}

func TestDependents(t *testing.T) {
	valids := map[string][]string{
		"b": {"a"},
		"c": {"a"},
		"x": {"c"},
		"ก": {"c"},
		"y": {"x"},
		"ข": {"ก"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)

	deps := g.Dependents("ข")
	assertMapContainsValues(t, deps, []string{})

	deps = g.Dependents("b")
	assertMapContainsValues(t, deps, []string{})

	deps = g.Dependents("x")
	assertMapContainsValues(t, deps, []string{"y"})

	deps = g.Dependents("a")
	assertMapContainsValues(t, deps, []string{"b", "c", "x", "y", "ก", "ข"})

	deps = g.Dependents("c")
	assertMapContainsValues(t, deps, []string{"x", "y", "ก", "ข"})
}

func TestRemove(t *testing.T) {
	g := initTestGraph(t)
	var err error

	err = g.Remove("y")
	assertNotContains(t, g, []string{"y"})
	if err != nil {
		t.Log("after-remove", "y", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("x")
	assertNotContains(t, g, []string{"x"})
	if err != nil {
		t.Log("after-remove", "x", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("ข")
	assertNotContains(t, g, []string{"ข"})
	if err != nil {
		t.Log("after-remove", "ข", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("ก")
	assertNotContains(t, g, []string{"ก"})
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

func TestRemoveForce(t *testing.T) {
	g := initTestGraph(t)

	// This should remove a, b, c, d,
	// leaving only x, y, ก, ข
	g.RemoveForce("a")
	assertNotContains(t, g, []string{"a", "b", "c", "d"})
	assertContainsAll(t, g, []string{"x", "y", "ก", "ข"})
}

func TestRemoveAutoRemove(t *testing.T) {
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
		g := initTestGraph(t)
		testAutoRemove(t, g, testCase.removes)

		if testCase.willBeEmpty {
			assertEmptyGraph(t, g)
			continue
		}

		assertNotEmptyGraph(t, g)
		assertRemainingNode(t, g, testCase.nodesRemaining)
	}
}

func TestDelete(t *testing.T) {
	g := initTestGraph(t)

	for _, node := range []string{"y", "a", "x", "c"} {
		g.Delete(node)
		assertNotContains(t, g, []string{node})
	}
}

func TestAutoRemoveLeaves(t *testing.T) {
	testAutoRemoveLeaves(t, initTestGraph(t))
}

func testAutoRemove(t *testing.T, g depgraph.Graph[string], toRemoves []string) {
	t.Log("testAutoRemove-before", g, "removes", toRemoves)
	for _, rm := range toRemoves {
		g.RemoveAutoRemove(rm)
		g.AssertRelationships()
	}

	t.Log("testAutoRemove-after", g)
}

func testAutoRemoveLeaves(t *testing.T, g depgraph.Graph[string]) {
	for _, leaf := range g.Leaves() {
		g.RemoveAutoRemove(leaf)
	}

	g.AssertRelationships()
	assertEmptyGraph(t, g)
}

func TestDebugDepGraph(t *testing.T) {
	g := initTestGraph(t)
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

func addValidDependencies(t *testing.T, g depgraph.Graph[string], valids map[string][]string) {
	for dependent, dependencies := range valids {
		for _, dependency := range dependencies {
			err := g.AddDependency(dependent, dependency)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}

			if !g.DependsOn(dependent, dependency) {
				t.Fatalf("dependent %v should depend on %v", dependent, dependency)
			}
		}

		added := g.Dependencies(dependent)
		for _, dependency := range dependencies {
			_, ok := added[dependency]
			if !ok {
				t.Fatal("added dependency not found", dependency)
			}
		}
	}
}

func assertMapContainsValues[K comparable, V any](
	t *testing.T,
	m map[K]V,
	keys []K,
) {
	for _, thing := range keys {
		_, ok := m[thing]
		if !ok {
			t.Fatalf("thing %v is not in map", thing)
		}
	}
}

func assertEmptyGraph(t *testing.T, g depgraph.Graph[string]) {
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

func assertNotEmptyGraph(t *testing.T, g depgraph.Graph[string]) {
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

func assertNotContains(t *testing.T, g depgraph.Graph[string], nodes []string) {
	if len(nodes) == 0 {
		t.Fatal("remainingNodes is null")
	}

	graphNodes := g.GraphNodes()
	graphDependents := g.GraphDependents()
	graphDependencies := g.GraphDependencies()

	for _, node := range nodes {
		if graphNodes.Contains(node) {
			t.Fatal("found node", node)
		}

		if graphDependencies.ContainsKey(node) {
			t.Fatal("found node as dependent", node)
		}

		if graphDependents.ContainsKey(node) {
			t.Fatal("found node as dependency", node)
		}

		for _, v := range graphDependencies {
			if v.Contains(node) {
				t.Fatal("found node as dependency", node)
			}
		}

		for _, v := range graphDependents {
			if v.Contains(node) {
				t.Fatal("found node as dependent", node)
			}
		}
	}
}

func assertContainsAll(t *testing.T, g depgraph.Graph[string], nodes []string) {
	if len(nodes) == 0 {
		t.Fatal("remainingNodes is null")
	}

	graphNodes := g.GraphNodes()
	graphDependents := g.GraphDependents()
	graphDependencies := g.GraphDependencies()

	for _, node := range nodes {
		if !graphNodes.Contains(node) {
			t.Fatal("found node", node)
		}

		var found bool

		if graphDependencies.ContainsKey(node) {
			found = true
		}

		if graphDependents.ContainsKey(node) {
			found = true
		}

		if !found {
			t.Fatal("node not found", node)
		}
	}
}

func assertRemainingNode(t *testing.T, g depgraph.Graph[string], nodes []string) {
	if len(nodes) == 0 {
		t.Fatal("remainingNodes is null")
	}

	graphNodes := g.GraphNodes()
	graphDependents := g.GraphDependents()
	graphDependencies := g.GraphDependencies()

	for _, node := range nodes {
		if !graphNodes.Contains(node) {
			t.Fatalf("missing node %v", node)
		}

		found := graphDependents.ContainsKey(node)
		if !found {
			for dependency, dependents := range graphDependents {
				if !dependents.Contains(node) {
					continue
				}

				if !graphDependencies.Contains(node, dependency) {
					continue
				}

				found = true
				break
			}
		}

		found = graphDependencies.ContainsKey(node)
		if !found {
			for dependent, dependencies := range graphDependencies {
				if !dependencies.Contains(node) {
					continue
				}

				if !graphDependents.Contains(node, dependent) {
					continue
				}

				found = true
				break
			}
		}

		if !found {
			t.Fatalf("node %v not found", node)
		}
	}
}
