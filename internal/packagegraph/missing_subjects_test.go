package packagegraph

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func missingProviderGraph(hidden, visible int) Graph {
	graph := Graph{Nodes: []Node{{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: []Module{{Path: "main", Imports: []string{"main", "shared", "main"}}}}}}
	for index := range hidden + visible {
		name := fmt.Sprintf("provider-%05d", index)
		module := "shared"
		if index < visible {
			module = "other"
			graph.Dependencies = append(graph.Dependencies, Dependency{Owner: "app", Binding: name, Target: name})
		}
		graph.Nodes = append(graph.Nodes, Node{ID: name, Role: RoleDependency, Identity: testIdentity(name), Modules: []Module{{Path: module}}})
	}
	return graph
}

func TestMissingSubjectsPreserveInstanceOrderAndOwnStorage(t *testing.T) {
	graph := missingProviderGraph(2, 1)
	for _, position := range []int{0, 2, 1} {
		alias := graph.Nodes[position]
		alias.ID += "-alias"
		graph.Nodes = append(graph.Nodes, alias)
	}
	beforeNodes := slices.Clone(graph.Nodes)
	beforeEdges := slices.Clone(graph.Dependencies)
	result, rejection, err := Resolve(graph)
	want := []string{"app", "app-alias", "provider-00001", "provider-00001-alias", "provider-00002"}
	if err != nil || rejection == nil || rejection.Reason != ReasonMissingVisibleModule || rejection.ImportPath != "shared" ||
		!slices.Equal(rejection.Subjects, want) || len(result.Lookups) != 1 {
		t.Fatalf("missing import changed results: %+v %+v %v", result, rejection, err)
	}
	rejection.Subjects[0] = "changed"
	if result.InstanceClasses[0].Members[0] != "app" || !reflect.DeepEqual(graph.Nodes, beforeNodes) || !reflect.DeepEqual(graph.Dependencies, beforeEdges) {
		t.Fatal("diagnostic shares member storage or mutated the graph")
	}
}

func TestMissingSubjectsWithNoHiddenProvider(t *testing.T) {
	for _, visible := range []int{0, 1, 64} {
		graph := missingProviderGraph(0, visible)
		result, rejection, err := Resolve(graph)
		if err != nil || rejection == nil || rejection.Reason != ReasonMissingVisibleModule ||
			!slices.Equal(rejection.Subjects, []string{"app"}) || len(result.Lookups) != 1 {
			t.Fatalf("visible=%d: result=%+v rejection=%+v err=%v", visible, result, rejection, err)
		}
	}
}

func TestMissingSubjectsOwnerAfterProviders(t *testing.T) {
	graph := missingProviderGraph(2, 0)
	root := graph.Nodes[0]
	graph.Nodes = append(slices.Clone(graph.Nodes[1:]), root)
	result, rejection, err := Resolve(graph)
	if err != nil || rejection == nil || rejection.Reason != ReasonMissingVisibleModule ||
		!slices.Equal(rejection.Subjects, []string{"app", "provider-00000", "provider-00001"}) || len(result.Lookups) != 1 {
		t.Fatalf("owner after providers: result=%+v rejection=%+v err=%v", result, rejection, err)
	}
}

func BenchmarkResolveMissingProviders(b *testing.B) {
	for _, hidden := range []int{1024, 4096, 8192} {
		b.Run(fmt.Sprint(hidden), func(b *testing.B) {
			graph := missingProviderGraph(hidden, hidden/2)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result, rejection, err := Resolve(graph)
				if err != nil || rejection == nil || rejection.Reason != ReasonMissingVisibleModule || len(rejection.Subjects) != hidden+1 || len(result.Lookups) != 1 {
					b.Fatalf("unexpected result: %+v %v", rejection, err)
				}
			}
		})
	}
}
