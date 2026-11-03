package packagegraph

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func ambiguousProviderGraph(count int) Graph {
	graph := Graph{Nodes: []Node{{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: []Module{{Path: "main", Imports: []string{"shared"}}}}}}
	for index := 0; index < count; index++ {
		name := fmt.Sprintf("provider-%05d", index)
		graph.Nodes = append(graph.Nodes, Node{ID: name, Role: RoleDependency, Identity: testIdentity(name), Modules: []Module{{Path: "shared"}}})
		graph.Dependencies = append(graph.Dependencies, Dependency{Owner: "app", Binding: name, Target: name})
	}
	return graph
}

func TestAmbiguousSubjectsPreserveInstanceAndMemberOrder(t *testing.T) {
	graph := ambiguousProviderGraph(2)
	// The first provider has two handles for the same exact instance. Their
	// order is the source node order, independently of dependency edge order.
	alias := graph.Nodes[1]
	alias.ID = "alias-first-provider"
	graph.Nodes = append(graph.Nodes, alias)
	graph.Dependencies = append([]Dependency{{Owner: "app", Binding: "alias", Target: alias.ID}}, graph.Dependencies...)
	beforeNodes := slices.Clone(graph.Nodes)
	beforeEdges := slices.Clone(graph.Dependencies)
	result, rejection, err := Resolve(graph)
	if err != nil || rejection == nil || rejection.Reason != ReasonAmbiguousVisibleModule || rejection.ImportPath != "shared" || len(result.Lookups) != 0 {
		t.Fatalf("unexpected ambiguity: %+v, %+v, %v", result, rejection, err)
	}
	want := []string{"provider-00000", "alias-first-provider", "provider-00001"}
	if !slices.Equal(rejection.Subjects, want) {
		t.Fatalf("subjects=%v, want %v", rejection.Subjects, want)
	}
	rejection.Subjects[0] = "changed result"
	if !reflect.DeepEqual(graph.Nodes, beforeNodes) || !reflect.DeepEqual(graph.Dependencies, beforeEdges) {
		t.Fatal("subject construction exposed caller-owned graph slices")
	}
}

func TestAmbiguousProviderSubjectsMatchOrderedDeduplication(t *testing.T) {
	classes := []resolvedClass{
		{members: []string{"b", "a", "b"}},
		{members: []string{"a", "c"}},
		{members: nil},
	}
	for _, providers := range [][]int{nil, {}, {2}, {0}, {1, 0, 1, 2}, {0, 1, 0}} {
		var want []string
		for _, provider := range providers {
			want = appendUnique(want, classes[provider].members...)
		}
		got := ambiguousProviderSubjects(classes, providers)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("providers=%v: got %v, want %v", providers, got, want)
		}
		if len(got) != 0 {
			got[0] = "changed result"
			if !slices.Equal(classes[0].members, []string{"b", "a", "b"}) || !slices.Equal(classes[1].members, []string{"a", "c"}) {
				t.Fatal("returned subjects share storage with an instance class")
			}
		}
	}
}

func BenchmarkResolveAmbiguousProviders(b *testing.B) {
	for _, count := range []int{256, 1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			graph := ambiguousProviderGraph(count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, rejection, err := Resolve(graph)
				if err != nil || rejection == nil || len(rejection.Subjects) != count {
					b.Fatalf("unexpected graph result: %+v, %v", rejection, err)
				}
			}
		})
	}
}
