package packagegraph

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func duplicateBindingSubjectGraph(targets, aliases int) (Graph, []string) {
	graph, subjects := cycleSubjectGraph(targets+1, aliases)
	graph.Dependencies = nil
	for target := 1; target <= targets; target++ {
		graph.Dependencies = append(graph.Dependencies, Dependency{
			Owner: graph.Nodes[0].ID, Binding: "shared", Target: graph.Nodes[target].ID,
		})
	}
	for len(graph.Dependencies) < 2 {
		graph.Dependencies = append(graph.Dependencies, Dependency{
			Owner: graph.Nodes[0].ID, Binding: "shared", Target: graph.Nodes[targets].ID,
		})
	}
	return graph, subjects
}

func TestDuplicateBindingSubjectsPreserveMembersAndOwnership(t *testing.T) {
	for _, targets := range []int{0, 1, 2, 128} {
		for _, aliases := range []int{1, 3} {
			graph, want := duplicateBindingSubjectGraph(targets, aliases)
			before, _ := duplicateBindingSubjectGraph(targets, aliases)
			result, rejection, err := Resolve(graph)
			if err != nil || rejection == nil || rejection.Reason != ReasonDuplicateBinding || !slices.Equal(rejection.Subjects, want) || len(result.Lookups) != 0 {
				t.Fatalf("targets=%d aliases=%d: rejection=%+v error=%v", targets, aliases, rejection, err)
			}
			if !reflect.DeepEqual(graph, before) {
				t.Fatal("duplicate binding diagnostic modified caller-owned graph")
			}
			rejection.Subjects[0] = "changed"
			if graph.Nodes[0].ID != want[0] || result.InstanceClasses[0].Members[0] != want[0] {
				t.Fatal("diagnostic subjects alias input or instance-class members")
			}
		}
	}
}

func TestDuplicateBindingSubjectsSkipOwnerAndRepeatedTargetClasses(t *testing.T) {
	graph, all := cycleSubjectGraph(4, 2)
	graph.Dependencies = []Dependency{
		{Owner: graph.Nodes[4].ID, Binding: "shared", Target: graph.Nodes[4].ID},
		{Owner: graph.Nodes[0].ID, Binding: "shared", Target: graph.Nodes[3].ID},
		{Owner: graph.Nodes[4].ID, Binding: "shared", Target: graph.Nodes[7].ID},
		{Owner: graph.Nodes[0].ID, Binding: "shared", Target: graph.Nodes[2].ID},
		{Owner: graph.Nodes[0].ID, Binding: "other", Target: graph.Nodes[1].ID},
	}
	want := append(slices.Clone(all[:2]), all[6:8]...)
	want = append(want, all[4:6]...)
	_, rejection, err := Resolve(graph)
	if err != nil || rejection == nil || rejection.Reason != ReasonDuplicateBinding || !slices.Equal(rejection.Subjects, want) {
		t.Fatalf("owner-first then target encounter order: rejection=%+v error=%v", rejection, err)
	}
	// Dependency admission still checks later edges before constructing any
	// duplicate-binding diagnostic, even though a self-cycle also exists.
	graph.Dependencies = append(graph.Dependencies, Dependency{Owner: "absent", Binding: "last", Target: graph.Nodes[0].ID})
	_, rejection, err = Resolve(graph)
	if err == nil || rejection != nil || !strings.Contains(err.Error(), `dependency owner "absent" is not a declared node`) {
		t.Fatalf("late malformed edge lost precedence: rejection=%+v error=%v", rejection, err)
	}
	// The disjoint-member invariant is established by node admission, before
	// either dependency admission or diagnostic construction can run.
	graph.Nodes[1].ID = graph.Nodes[0].ID
	_, rejection, err = Resolve(graph)
	if err == nil || rejection != nil || !strings.Contains(err.Error(), "duplicate package node") {
		t.Fatalf("duplicate node ID lost admission priority: rejection=%+v error=%v", rejection, err)
	}
}

func BenchmarkResolveDuplicateBindingSubjects(b *testing.B) {
	for _, targets := range []int{2048, 8192} {
		b.Run(fmt.Sprintf("targets=%d", targets), func(b *testing.B) {
			graph, want := duplicateBindingSubjectGraph(targets, 2)
			b.ReportAllocs()
			for b.Loop() {
				_, rejection, err := Resolve(graph)
				if err != nil || rejection == nil || rejection.Reason != ReasonDuplicateBinding || len(rejection.Subjects) != len(want) {
					b.Fatalf("resolve duplicate binding: rejection=%+v error=%v", rejection, err)
				}
			}
		})
	}
}
