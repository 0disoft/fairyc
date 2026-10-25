package workspaceloader

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func semanticInputWorkspace(tb testing.TB, count int) Workspace {
	tb.Helper()
	workspace := metadataWorkspace(tb, count)
	for index := range workspace.Packages {
		workspace.Packages[index].Sources = []SourceFile{{
			Path: fmt.Sprintf("src/file-%d.fy", index), Module: &ast.Module{}, Test: index%2 == 0,
		}}
	}
	return workspace
}

func TestSemanticInputLookupPreservesFirstMatchesAndFreshSources(t *testing.T) {
	workspace := semanticInputWorkspace(t, 4)
	slices.Reverse(workspace.Instances)
	slices.Reverse(workspace.Packages)
	alias := workspace.Instances[0]
	alias.NodeID = "alias"
	workspace.Instances = append(workspace.Instances, alias)
	workspace.Resolution.InstanceClasses[3].Members = append(workspace.Resolution.InstanceClasses[3].Members, alias.NodeID)
	workspace.Instances = append(workspace.Instances, Instance{NodeID: workspace.Instances[0].NodeID, PackagePath: "missing"})
	workspace.Packages = append(workspace.Packages, Package{Path: workspace.Packages[0].Path})
	check := func() {
		t.Helper()
		input, err := workspace.SemanticInput()
		if err != nil || len(input.Files) != len(workspace.Resolution.InstanceClasses) {
			t.Fatalf("files=%d, error=%v", len(input.Files), err)
		}
		for index, class := range workspace.Resolution.InstanceClasses {
			// The original linear lookup methods remain the first-match oracle.
			mapping, found := workspace.instance(class.Members[0])
			if !found {
				t.Fatal("fixture instance is missing")
			}
			loaded, found := workspace.Package(mapping.PackagePath)
			if !found {
				t.Fatal("fixture package is missing")
			}
			file, want := input.Files[index], loaded.Sources[0]
			if file.Package != hir.PackageID(index+1) || file.Path != want.Path || file.Module != want.Module || file.Test != want.Test {
				t.Fatalf("file %d differs from the first matching source: %+v", index, file)
			}
		}
	}
	check()
	workspace.Packages[0].Sources[0] = SourceFile{Path: "src/replaced.fy", Module: &ast.Module{}}
	check()
}

func TestSemanticInputLookupKeepsMappingErrors(t *testing.T) {
	for _, test := range []struct {
		name, message string
		mutate        func(*Workspace)
	}{
		{"empty class", "workspace has an empty package instance class", func(w *Workspace) {
			w.Resolution.InstanceClasses[0].Members = nil
		}},
		{"duplicate node", `workspace node "package-0000" belongs to multiple instance classes`, func(w *Workspace) {
			w.Resolution.InstanceClasses[1].Members = []string{w.Resolution.InstanceClasses[0].Members[0]}
		}},
		{"missing instance", `workspace node "package-0000" has no package mapping`, func(w *Workspace) {
			w.Instances = w.Instances[1:]
		}},
		{"missing package", `workspace package "package-0000" is missing`, func(w *Workspace) {
			w.Packages = w.Packages[1:]
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := semanticInputWorkspace(t, 4)
			test.mutate(&workspace)
			if _, err := workspace.SemanticInput(); err == nil || err.Error() != test.message {
				t.Fatalf("error = %v, want %q", err, test.message)
			}
		})
	}
}

func BenchmarkSemanticInputLookup(b *testing.B) {
	for _, count := range []int{16, 1024} {
		b.Run(fmt.Sprintf("packages-%d", count), func(b *testing.B) {
			workspace := semanticInputWorkspace(b, count)
			b.ReportAllocs()
			for b.Loop() {
				input, err := workspace.SemanticInput()
				if err != nil || len(input.Files) != count {
					b.Fatalf("files=%d, error=%v", len(input.Files), err)
				}
			}
		})
	}
}
