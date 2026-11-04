package packagegraph

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func cycleSubjectGraph(count, aliases int) (Graph, []string) {
	graph := Graph{Nodes: make([]Node, 0, count*aliases), Dependencies: make([]Dependency, 0, count)}
	id := func(class, alias int) string { return fmt.Sprintf("node_%05d_%03d", class, alias) }
	// Interleave the classes in input order to exercise instance deduplication.
	for alias := range aliases {
		for class := range count {
			role := RoleDependency
			if class == 0 && alias == 0 {
				role = RoleRoot
			}
			graph.Nodes = append(graph.Nodes, Node{ID: id(class, alias), Role: role,
				Identity: testIdentity(fmt.Sprintf("cycle_%05d", class)),
				Modules:  []Module{{Path: "main", Imports: []string{"missing"}}}})
		}
	}
	var subjects []string
	for class := range count {
		graph.Dependencies = append(graph.Dependencies, Dependency{
			Owner: id(class, 0), Binding: "next", Target: id((class+1)%count, aliases-1),
		})
		for alias := range aliases {
			subjects = append(subjects, id(class, alias))
		}
	}
	return graph, subjects
}

func TestCycleSubjectsPreserveDeduplicatedClassAndMemberOrder(t *testing.T) {
	for _, count := range []int{1, 2, 128} {
		for _, aliases := range []int{1, 3} {
			graph, want := cycleSubjectGraph(count, aliases)
			before, _ := cycleSubjectGraph(count, aliases)
			result, rejection, err := Resolve(graph)
			if err != nil || rejection == nil || rejection.Reason != ReasonDependencyCycle || !slices.Equal(rejection.Subjects, want) || len(result.Lookups) != 0 {
				t.Fatalf("classes=%d aliases=%d: rejection=%+v error=%v", count, aliases, rejection, err)
			}
			if !reflect.DeepEqual(graph, before) {
				t.Fatal("cycle diagnostic modified caller-owned graph")
			}
			rejection.Subjects[0] = "changed"
			if result.InstanceClasses[0].Members[0] != want[0] || graph.Nodes[0].ID != want[0] {
				t.Fatal("cycle diagnostic aliases instance-class or input members")
			}
		}
	}
}

func TestCycleSubjectsExcludeAcyclicPrefixAndLaterCycles(t *testing.T) {
	graph, all := cycleSubjectGraph(4, 2)
	// 0 -> 1 -> 2 -> 1 is the first discovered cycle; 3 -> 3 is later.
	graph.Dependencies[2].Target = graph.Nodes[1].ID
	graph.Dependencies[3].Target = graph.Nodes[3].ID
	_, rejection, err := Resolve(graph)
	if err != nil || rejection == nil || rejection.Reason != ReasonDependencyCycle || !slices.Equal(rejection.Subjects, all[2:6]) {
		t.Fatalf("first simple cycle: rejection=%+v error=%v", rejection, err)
	}
}

func BenchmarkResolveCycleSubjects(b *testing.B) {
	for _, count := range []int{2048, 8192} {
		b.Run(fmt.Sprintf("classes=%d", count), func(b *testing.B) {
			graph, want := cycleSubjectGraph(count, 2)
			b.ReportAllocs()
			for b.Loop() {
				_, rejection, err := Resolve(graph)
				if err != nil || rejection == nil || rejection.Reason != ReasonDependencyCycle || len(rejection.Subjects) != len(want) {
					b.Fatalf("resolve cycle: rejection=%+v error=%v", rejection, err)
				}
			}
		})
	}
}
