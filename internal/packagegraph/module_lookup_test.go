package packagegraph

import (
	"fmt"
	"slices"
	"testing"
)

func TestResolveModuleLookupBoundaries(t *testing.T) {
	for _, query := range []string{"A", "a", "a::c", "a::child", "a_z", "b", "z", "zz"} {
		t.Run(query, func(t *testing.T) {
			modules := []Module{{Path: "a"}, {Path: "a::child"}, {Path: "a_z"}, {Path: "z"}}
			graph := Graph{
				Nodes: []Node{
					{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: []Module{{Path: "main", Imports: []string{"a", query}}}},
					{ID: "dep", Role: RoleDependency, Identity: testIdentity("dep"), Modules: modules},
				},
				Dependencies: []Dependency{{Owner: "app", Binding: "dep", Target: "dep"}},
			}
			result, rejection, err := Resolve(graph)
			if err != nil {
				t.Fatal(err)
			}
			// A preceding successful lookup must not leave stale provider indexes.
			if slices.Contains([]string{"a", "a::child", "a_z", "z"}, query) {
				if rejection != nil || len(result.Lookups) != 2 || result.Lookups[1].ModulePath != query || !slices.Equal(result.Lookups[1].ProviderMembers, []string{"dep"}) {
					t.Fatalf("existing module lookup: %+v %+v", result, rejection)
				}
			} else if rejection == nil || rejection.Reason != ReasonMissingVisibleModule || rejection.ImportPath != query || len(result.Lookups) != 1 {
				t.Fatalf("missing module lookup: %+v %+v", result, rejection)
			}
		})
	}
}

func BenchmarkResolveModuleImports(b *testing.B) {
	const importsPerModule = 16
	for _, count := range []int{128, 1024, 4096} {
		b.Run(fmt.Sprintf("modules_%d", count), func(b *testing.B) {
			modules := make([]Module, count)
			for n := range modules {
				modules[n].Path = fmt.Sprintf("m%04d", n)
			}
			imports := make([]string, importsPerModule)
			for n := range imports {
				imports[n] = modules[count-importsPerModule+n].Path
			}
			for n := range count - importsPerModule {
				modules[n].Imports = imports
			}
			graph := Graph{Nodes: []Node{{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: modules}}}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result, rejection, err := Resolve(graph)
				if err != nil || rejection != nil || len(result.Lookups) != (count-importsPerModule)*importsPerModule {
					b.Fatalf("resolve: error=%v rejection=%+v lookups=%d", err, rejection, len(result.Lookups))
				}
			}
		})
	}
}
