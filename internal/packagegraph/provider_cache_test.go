package packagegraph

import (
	"fmt"
	"slices"
	"testing"
)

func TestRepeatedProviderLookupKeepsImportOrderAndOwnsMembers(t *testing.T) {
	graph := repeatedDependencyImports(2, 16)
	result, rejection, err := Resolve(graph)
	if err != nil || rejection != nil || len(result.Lookups) != 32 {
		t.Fatalf("resolve: error=%v rejection=%+v lookups=%d", err, rejection, len(result.Lookups))
	}
	for index, lookup := range result.Lookups {
		want := fmt.Sprintf("d%04d", index%16)
		if lookup.ImportPath != want || !slices.Equal(lookup.ProviderMembers, []string{want}) {
			t.Fatalf("lookup %d reordered or changed provider: %+v", index, lookup)
		}
	}
	result.Lookups[0].ProviderMembers[0] = "changed"
	if result.Lookups[16].ProviderMembers[0] != "d0000" {
		t.Fatal("repeated lookup receipts share mutable member slices")
	}
}

func TestRepeatedProviderLookupKeepsInstanceVisibility(t *testing.T) {
	graph := Graph{}
	for _, name := range []string{"app", "left", "right", "a", "b"} {
		path := name
		if name == "a" || name == "b" {
			path = "util"
		}
		graph.Nodes = append(graph.Nodes, Node{ID: name, Role: RoleDependency, Identity: testIdentity(name), Modules: []Module{{Path: path}}})
	}
	graph.Nodes[0].Role = RoleRoot
	graph.Nodes[0].Modules[0].Imports = []string{"left", "right"}
	graph.Nodes[1].Modules[0].Imports = []string{"util", "util"}
	graph.Nodes[2].Modules[0].Imports = []string{"util", "util"}
	graph.Dependencies = []Dependency{
		{Owner: "app", Binding: "left", Target: "left"},
		{Owner: "app", Binding: "right", Target: "right"},
		{Owner: "left", Binding: "util", Target: "a"},
		{Owner: "right", Binding: "util", Target: "b"},
	}
	result, rejection, err := Resolve(graph)
	if err != nil || rejection != nil || len(result.Lookups) != 6 {
		t.Fatalf("resolve: error=%v rejection=%+v lookups=%d", err, rejection, len(result.Lookups))
	}
	for index, want := range []string{"left", "right", "a", "a", "b", "b"} {
		if !slices.Equal(result.Lookups[index].ProviderMembers, []string{want}) {
			t.Fatalf("lookup %d crossed package visibility: %+v", index, result.Lookups[index])
		}
	}
}

func TestRepeatedProviderLookupStopsAtMissingImport(t *testing.T) {
	graph := repeatedDependencyImports(1, 16)
	graph.Nodes[0].Modules[0].Imports = []string{"d0000", "d0000", "missing", "d0000"}
	result, rejection, err := Resolve(graph)
	if err != nil || rejection == nil || rejection.Reason != ReasonMissingVisibleModule || rejection.ImportPath != "missing" || len(result.Lookups) != 2 {
		t.Fatalf("missing import changed preceding lookups: %+v %+v %v", result, rejection, err)
	}
}

func repeatedDependencyImports(modules, dependencies int) Graph {
	graph := Graph{Nodes: []Node{{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: make([]Module, modules)}}}
	for index := range dependencies {
		name := fmt.Sprintf("d%04d", index)
		graph.Nodes = append(graph.Nodes, Node{ID: name, Role: RoleDependency, Identity: testIdentity(name), Modules: []Module{{Path: name}}})
		graph.Dependencies = append(graph.Dependencies, Dependency{Owner: "app", Binding: name, Target: name})
	}
	imports := make([]string, 16)
	for index := range imports {
		imports[index] = graph.Nodes[dependencies-len(imports)+index+1].ID
	}
	for index := range graph.Nodes[0].Modules {
		graph.Nodes[0].Modules[index] = Module{Path: fmt.Sprintf("m%04d", index), Imports: imports}
	}
	return graph
}

func BenchmarkResolveRepeatedDependencyImports(b *testing.B) {
	for _, modules := range []int{1024, 3072} {
		b.Run(fmt.Sprint(modules), func(b *testing.B) {
			graph := repeatedDependencyImports(modules, 64)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				result, rejection, err := Resolve(graph)
				if err != nil || rejection != nil || len(result.Lookups) != modules*16 {
					b.Fatalf("resolve: error=%v rejection=%+v lookups=%d", err, rejection, len(result.Lookups))
				}
			}
		})
	}
}
