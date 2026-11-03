package packagegraph

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func distinctProviderImports(dependencies, modules, imports int) Graph {
	graph := Graph{Nodes: []Node{{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: []Module{{Path: "main"}}}}}
	for dependency := 0; dependency < dependencies; dependency++ {
		name := fmt.Sprintf("d%04d", dependency)
		node := Node{ID: name, Role: RoleDependency, Identity: testIdentity(name)}
		for module := 0; module < modules; module++ {
			node.Modules = append(node.Modules, Module{Path: fmt.Sprintf("%s::m%04d", name, module)})
		}
		graph.Nodes = append(graph.Nodes, node)
		graph.Dependencies = append(graph.Dependencies, Dependency{Owner: "app", Binding: name, Target: name})
	}
	for index := 0; index < imports; index++ {
		graph.Nodes[0].Modules[0].Imports = append(graph.Nodes[0].Modules[0].Imports,
			graph.Nodes[index/modules+1].Modules[index%modules].Path)
	}
	return graph
}

func TestDistinctProviderImportsPreserveSourceOrder(t *testing.T) {
	graph := distinctProviderImports(8, 8, 64)
	imports := graph.Nodes[0].Modules[0].Imports
	slices.Reverse(imports)
	graph.Nodes[0].Modules[0].Imports = append(imports, imports[0], "main")
	result, rejection, err := Resolve(graph)
	if err != nil || rejection != nil || len(result.Lookups) != 66 {
		t.Fatalf("resolve: lookups=%d rejection=%+v err=%v", len(result.Lookups), rejection, err)
	}
	for index, lookup := range result.Lookups {
		want := graph.Nodes[0].Modules[0].Imports[index]
		if lookup.ImportPath != want {
			t.Fatalf("lookup %d reordered: %q, want %q", index, lookup.ImportPath, want)
		}
	}
	if !slices.Equal(result.Lookups[0].ProviderMembers, []string{"d0007"}) || !slices.Equal(result.Lookups[65].ProviderMembers, []string{"app"}) {
		t.Fatal("dependency or self visibility changed")
	}
	result.Lookups[0].ProviderMembers[0] = "changed"
	if result.Lookups[64].ProviderMembers[0] != "d0007" {
		t.Fatal("repeated lookup results share mutable storage")
	}
}

func TestVisibleProviderLookupMatchesScan(t *testing.T) {
	graph := distinctProviderImports(8, 8, 64)
	graph.Nodes[0].Modules[0].Imports = append(graph.Nodes[0].Modules[0].Imports, "missing", "main")
	classes := make([]resolvedClass, len(graph.Nodes))
	for index := range classes {
		classes[index].nodes = []int{index}
	}
	visible := []int{0, 3, 1, 3}
	got := prepareVisibleProviderLookup(graph, classes, visible, graph.Nodes[0].Modules)
	for _, path := range graph.Nodes[0].Modules[0].Imports {
		want := providerClasses(graph, classes, visible, path, nil)
		values, found := got[path]
		if !found || !reflect.DeepEqual(values, want) {
			t.Fatalf("path=%q: providers=%v found=%v, want %v", path, values, found, want)
		}
	}
	// A later call must observe source edits, never an index retained on Graph.
	graph.Nodes[3].Modules[0].Path = "changed"
	updated := prepareVisibleProviderLookup(graph, classes, visible, graph.Nodes[0].Modules)
	if len(got["d0002::m0000"]) != 2 || updated["d0002::m0000"] != nil {
		t.Fatal("lookup index persisted across calls or modified a prior result")
	}
}

func TestVisibleProviderLookupKeepsSparseCatalogLazy(t *testing.T) {
	graph := distinctProviderImports(1, 4096, 1)
	classes := []resolvedClass{{nodes: []int{0}}, {nodes: []int{1}}}
	got := prepareVisibleProviderLookup(graph, classes, []int{0, 1}, graph.Nodes[0].Modules)
	if len(got) != 0 {
		t.Fatal("sparse catalog should retain lazy binary search")
	}
	result, rejection, err := Resolve(graph)
	if err != nil || rejection != nil || len(result.Lookups) != 1 || !slices.Equal(result.Lookups[0].ProviderMembers, []string{"d0000"}) {
		t.Fatalf("sparse resolution changed: %+v %+v %v", result, rejection, err)
	}
}

func TestVisibleProviderIndexPreservesRejectionPrecedence(t *testing.T) {
	for _, missingFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(missingFirst), func(t *testing.T) {
			graph := distinctProviderImports(8, 8, 64)
			for _, node := range []int{1, 2} {
				graph.Nodes[node].Modules = append(graph.Nodes[node].Modules, Module{Path: "util"})
			}
			imports := graph.Nodes[0].Modules[0].Imports
			failures := []string{"util", "missing"}
			wantReason := ReasonAmbiguousVisibleModule
			wantSubjects := []string{"d0000", "d0001"}
			if missingFirst {
				slices.Reverse(failures)
				wantReason, wantSubjects = ReasonMissingVisibleModule, []string{"app"}
			}
			ordered := []string{imports[0], imports[0]}
			ordered = append(ordered, failures...)
			graph.Nodes[0].Modules[0].Imports = append(ordered, imports...)
			result, rejection, err := Resolve(graph)
			if err != nil || rejection == nil || rejection.Reason != wantReason || rejection.ImportPath != failures[0] ||
				!slices.Equal(rejection.Subjects, wantSubjects) || len(result.Lookups) != 2 {
				t.Fatalf("rejection precedence changed: lookups=%d, rejection=%+v, err=%v", len(result.Lookups), rejection, err)
			}
		})
	}
}

func BenchmarkResolveDistinctProviderImports(b *testing.B) {
	for _, test := range []struct {
		name                         string
		dependencies, modules, paths int
	}{
		{"dense-1024", 16, 64, 1024},
		{"dense-4096", 64, 64, 4096},
		{"sparse-4096", 1, 4096, 1},
	} {
		b.Run(test.name, func(b *testing.B) {
			graph := distinctProviderImports(test.dependencies, test.modules, test.paths)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result, rejection, err := Resolve(graph)
				if err != nil || rejection != nil || len(result.Lookups) != test.paths {
					b.Fatalf("resolve: lookups=%d rejection=%+v err=%v", len(result.Lookups), rejection, err)
				}
			}
		})
	}
}

func BenchmarkVisibleProviderLookup(b *testing.B) {
	for _, paths := range []int{1, 4096} {
		graph := distinctProviderImports(64, 64, paths)
		classes := make([]resolvedClass, len(graph.Nodes))
		visible := make([]int, len(classes))
		for index := range classes {
			classes[index].nodes = []int{index}
			visible[index] = index
		}
		for _, indexed := range []bool{false, true} {
			b.Run(fmt.Sprintf("paths=%d/indexed=%t", paths, indexed), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					cache := make(map[string][]int)
					if indexed {
						cache = prepareVisibleProviderLookup(graph, classes, visible, graph.Nodes[0].Modules)
					}
					for _, path := range graph.Nodes[0].Modules[0].Imports {
						values, found := cache[path]
						if !found {
							values = providerClasses(graph, classes, visible, path, nil)
							cache[path] = values
						}
						if len(values) != 1 {
							b.Fatal("unexpected provider count")
						}
					}
				}
			})
		}
	}
}
