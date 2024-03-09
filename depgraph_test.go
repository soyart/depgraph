package depgraph_test

import (
	"fmt"
	"reflect"
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
		"1": {"0"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)
	g.AssertRelationships()

	{
		// Test if nil maps crash depgraph
		_ = g.DependsOn("0", "1")
		_ = g.Contains("2")
	}

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
	g.AssertRelationships()

	// Add some circular dependency and expect error
	err := g.Depend("a", "y")
	if err == nil {
		t.Fatal("expecting error from circular dependency")
	}
	g.AssertRelationships()
}

func TestDependencies(t *testing.T) {
	valids := map[string][]string{
		"b": {"a"},
		"c": {"a"},
		"x": {"c"},
		"0": {"c"},
		"1": {"0"},
		"y": {"x"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)
	g.AssertRelationships()

	deps := g.Dependencies("1")
	assertMapContainsValues(t, deps, []string{"0", "a", "c"})

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
		"0": {"c"},
		"y": {"x"},
		"1": {"0"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)

	deps := g.Dependents("1")
	assertMapContainsValues(t, deps, []string{})

	deps = g.Dependents("b")
	assertMapContainsValues(t, deps, []string{})

	deps = g.Dependents("x")
	assertMapContainsValues(t, deps, []string{"y"})

	deps = g.Dependents("a")
	assertMapContainsValues(t, deps, []string{"b", "c", "x", "y", "0", "1"})

	deps = g.Dependents("c")
	assertMapContainsValues(t, deps, []string{"x", "y", "0", "1"})
}

func TestUndepend(t *testing.T) {
	valids := map[string][]string{
		"b": {"a"},
		"c": {"a"},
		"x": {"c"},
		"y": {"x"},
		"1": {"0"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)

	testUndependToLeaf(t, &g, "b", "a")
	testUndependToLeaf(t, &g, "y", "x")
}

func testUndependToLeaf(t *testing.T, g *depgraph.Graph[string], dependent, dependency string) {
	g.Undepend(dependent, dependency)
	if g.DependsOn(dependent, dependency) {
		t.Fatalf("%v should not depend on %v after undepend", dependent, dependency)
	}
	g.AssertRelationships()

	leaves := g.Leaves()
	found := false
	for leaf := range leaves {
		if leaf == dependent {
			found = true
		}
	}

	if !found {
		t.Fatalf("%v not found in leaf", dependent)
	}

	t.Log("graph after undepend", dependent, dependency, fmt.Sprintf("%+v", g), "leaves", g.Leaves(), "layers", g.Layers())
}

func TestLayersSimple(t *testing.T) {
	valids := map[string][]string{
		"b": {"a"},
		"c": {"a"},
		"x": {"c"},
		"y": {"x"},
		"1": {"0"},
	}

	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)
	g.AssertRelationships()
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a", "0"),
		nodeSet("1", "b", "c"),
		nodeSet("x"),
		nodeSet("y"),
	})

	g.Undepend("b", "a")
	g.AssertRelationships()
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a", "b", "0"),
		nodeSet("1", "c"),
		nodeSet("x"),
		nodeSet("y"),
	})

	g.Undepend("c", "a")
	g.AssertRelationships()
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a", "b", "c", "0"),
		nodeSet("1", "x"),
		nodeSet("y"),
	})

	g.Depend("a", "c")
	g.AssertRelationships()
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("b", "c", "0"),
		nodeSet("a", "1", "x"),
		nodeSet("y"),
	})
}

func TestLayersComplex(t *testing.T) {
	// We'll focus more on x in this test
	valids := map[string][]string{
		"b": {"a"},
		"c": {"a"},
		"x": {"c"},
		"y": {"x"},
		"0": {"c"},
		"1": {"0"},
	}

	// x directly depends on c
	// x depends on [c, a]
	g := depgraph.New[string]()
	addValidDependencies(t, g, valids)
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a"),
		nodeSet("b", "c"),
		nodeSet("0", "x"),
		nodeSet("1", "y"),
	})

	// x directly depends on c, 1
	// x depends on [a, c, 0, 1]
	g.Depend("x", "1")
	g.AssertRelationships()
	if !g.DependsOn("x", "c") {
		t.Fatal("x directly depends on c")
	}
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a"),
		nodeSet("b", "c"),
		nodeSet("0"),
		nodeSet("1"),
		nodeSet("x"),
		nodeSet("y"),
	})

	g.Undepend("y", "x")
	g.AssertRelationships()
	if !g.DependsOn("x", "c") {
		t.Fatal("x directly depends on c")
	}
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a", "y"),
		nodeSet("b", "c"),
		nodeSet("0"),
		nodeSet("1"),
		nodeSet("x"),
	})

	// x directly depends on c
	// x depends on [a, c]
	g.Undepend("x", "1")
	g.AssertRelationships()
	if !g.DependsOn("x", "c") {
		t.Fatal("x directly depends on c")
	}
	assertMapContainsValues(t, g.Dependencies("x"), []string{"a", "c"})
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a", "y"),
		nodeSet("b", "c"),
		nodeSet("x", "0"),
		nodeSet("1"),
	})

	// x directly depends on a, c
	// x depends on [a, c]
	g.Depend("x", "a")
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a", "y"),
		nodeSet("b", "c"),
		nodeSet("x", "0"),
		nodeSet("1"),
	})

	g.Undepend("x", "a")
	g.Undepend("x", "c")
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a", "x", "y"),
		nodeSet("b", "c"),
		nodeSet("0"),
		nodeSet("1"),
	})

	g.Depend("y", "0")
	assertLayers(t, &g, []depgraph.Set[string]{
		nodeSet("a", "x"),
		nodeSet("b", "c"),
		nodeSet("0"),
		nodeSet("y", "1"),
	})
}

func TestRemove(t *testing.T) {
	g := initTestGraph(t)
	var err error

	err = g.Remove("y")
	g.AssertRelationships()
	assertNotContains(t, g, []string{"y"})
	if err != nil {
		t.Log("after-remove", "y", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("x")
	g.AssertRelationships()
	assertNotContains(t, g, []string{"x"})
	if err != nil {
		t.Log("after-remove", "x", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("1")
	g.AssertRelationships()
	assertNotContains(t, g, []string{"1"})
	if err != nil {
		t.Log("after-remove", "1", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("0")
	g.AssertRelationships()
	assertNotContains(t, g, []string{"0"})
	if err != nil {
		t.Log("after-remove", "0", g)
		t.Fatal("unexpected error:", err)
	}

	err = g.Remove("a")
	g.AssertRelationships()
	if err == nil {
		t.Log("after-remove", "a", g)
		t.Fatal("expecting error from removing a")
	}
}

func TestRemoveForce(t *testing.T) {
	g := initTestGraph(t)

	// This should remove a, b, c, d,
	// leaving only x, y, 0, 1
	g.RemoveForce("a")
	g.AssertRelationships()
	assertNotContains(t, g, []string{"a", "b", "c", "d"})
	assertContainsAll(t, g, []string{"x", "y", "0", "1"})
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
			nodesRemaining: []string{"a", "b", "x", "y", "0", "1"},
		},
		{
			removes:        []string{"a", "x"},
			willBeEmpty:    false,
			nodesRemaining: []string{"0", "1"},
		},
		{
			removes:     []string{"a", "x", "1"},
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
		g.AssertRelationships()
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
	for leaf := range g.Leaves() {
		g.RemoveAutoRemove(leaf)
		g.AssertRelationships()
	}

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
			err := g.Depend(dependent, dependency)
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

	g.AssertRelationships()
}

func assertMapContainsValues[K comparable, V any](
	t *testing.T,
	m map[K]V,
	keys []K,
) {
	for _, thing := range keys {
		_, ok := m[thing]
		if !ok {
			t.Logf("map %v", m)
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

func assertLayers(
	t *testing.T,
	g *depgraph.Graph[string],
	expecteds []depgraph.Set[string],
) {
	layers := g.Layers()
	if ll, le := len(layers), len(expecteds); ll != le {
		t.Log("expected\t", expecteds)
		t.Log("actual\t", layers)
		t.Fatalf("length differs - expecting %d, got %d", le, ll)
	}

	for i := range expecteds {
		expected := expecteds[i]
		layer := layers[i]

		if reflect.DeepEqual(expected, layer) {
			continue
		}

		t.Log("expected\t", expecteds)
		t.Log("actual\t", layers)
		t.Fatal("unexpected value in layer", i)
	}
}

func nodeSet(nodes ...string) depgraph.Set[string] {
	s := make(depgraph.Set[string])
	for i := range nodes {
		s[nodes[i]] = struct{}{}
	}

	return s
}
