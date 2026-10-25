package packagegraph

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func denseDependencyGraph(count int) Graph {
	graph := Graph{Nodes: make([]Node, count)}
	for index := range count {
		id := fmt.Sprintf("p%04d", index)
		role := RoleDependency
		if index == 0 {
			role = RoleRoot
		}
		graph.Nodes[index] = Node{ID: id, Role: role, Identity: testIdentity(id), Modules: []Module{{Path: id}}}
		if index < count-1 {
			graph.Nodes[index].Modules[0].Imports = []string{fmt.Sprintf("p%04d", count-1)}
		}
		for target := index + 1; target < count; target++ {
			targetID := fmt.Sprintf("p%04d", target)
			graph.Dependencies = append(graph.Dependencies, Dependency{Owner: id, Binding: targetID, Target: targetID})
		}
	}
	return graph
}

func TestResolveCanonicalAdjacencyPreservesLookupAndRejectionOrder(t *testing.T) {
	for _, mode := range []string{"valid", "cycle", "ambiguous", "missing"} {
		t.Run(mode, func(t *testing.T) {
			graph := denseDependencyGraph(8)
			// Different bindings can reach one exact instance; this is not a duplicate binding.
			graph.Dependencies = append(graph.Dependencies, Dependency{Owner: "p0000", Binding: "alias", Target: "p0007"})
			switch mode {
			case "cycle":
				graph.Dependencies = append(graph.Dependencies, Dependency{Owner: "p0007", Binding: "root", Target: "p0000"})
			case "ambiguous":
				graph.Nodes[0].Modules = []Module{{Path: "p0007", Imports: []string{"p0007"}}}
			case "missing":
				graph.Nodes[0].Modules[0].Imports = []string{"absent"}
			}
			want, wantRejection, err := Resolve(graph)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "valid" && (wantRejection != nil || len(want.Lookups) != 7) {
				t.Fatalf("valid graph: result=%+v rejection=%+v", want, wantRejection)
			}
			if mode != "valid" && wantRejection == nil {
				t.Fatal("expected rejection")
			}
			slices.Reverse(graph.Dependencies)
			before := slices.Clone(graph.Dependencies)
			got, rejection, err := Resolve(graph)
			if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(rejection, wantRejection) {
				t.Fatalf("dependency order changed resolution: result=%+v rejection=%+v error=%v", got, rejection, err)
			}
			if !slices.Equal(graph.Dependencies, before) {
				t.Fatal("Resolve mutated the caller's dependencies")
			}
		})
	}
}

func TestResolveCanonicalAdjacencyKeepsSelfFirstAfterEarlierProvider(t *testing.T) {
	shared := testIdentity("provider")
	_, rejection, err := Resolve(Graph{
		Nodes: []Node{
			{ID: "provider", Role: RoleDependency, Identity: shared, Modules: []Module{{Path: "shared"}}},
			{ID: "owner", Role: RoleRoot, Identity: testIdentity("owner"), Modules: []Module{{Path: "shared", Imports: []string{"shared"}}}},
			{ID: "alias", Role: RoleDependency, Identity: shared, Modules: []Module{{Path: "shared"}}},
		},
		Dependencies: []Dependency{{Owner: "owner", Binding: "provider", Target: "provider"}, {Owner: "owner", Binding: "alias", Target: "alias"}},
	})
	if err != nil || rejection == nil || rejection.Reason != ReasonAmbiguousVisibleModule || !slices.Equal(rejection.Subjects, []string{"owner", "provider", "alias"}) {
		t.Fatalf("rejection=%+v error=%v", rejection, err)
	}
}

func BenchmarkResolveDenseDependencies(b *testing.B) {
	for _, count := range []int{128, 512} {
		b.Run(fmt.Sprintf("packages_%d", count), func(b *testing.B) {
			graph := denseDependencyGraph(count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result, rejection, err := Resolve(graph)
				if err != nil || rejection != nil || len(result.Lookups) != count-1 {
					b.Fatalf("result=%d rejection=%+v error=%v", len(result.Lookups), rejection, err)
				}
			}
		})
	}
}
