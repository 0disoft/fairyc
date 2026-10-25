package workspaceloader

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func metadataWorkspace(tb testing.TB, count int) Workspace {
	tb.Helper()
	var workspace Workspace
	for index := range count {
		id := fmt.Sprintf("package-%04d", index)
		identity := packagegraph.Identity{PackageUID: id, SourceIdentity: id, VersionIdentity: "1"}
		instanceID, err := packagegraph.ComputeInstanceID(identity)
		if err != nil {
			tb.Fatal(err)
		}
		loaded := Package{Path: id, SourceKind: SourceKindLocal}
		loaded.Manifest.Package.UID = id
		workspace.Packages = append(workspace.Packages, loaded)
		workspace.Instances = append(workspace.Instances, Instance{NodeID: id, PackagePath: id})
		workspace.Graph.Nodes = append(workspace.Graph.Nodes, packagegraph.Node{ID: id, Identity: identity})
		workspace.Resolution.InstanceClasses = append(workspace.Resolution.InstanceClasses, packagegraph.InstanceClass{ID: instanceID, Members: []string{id}})
		for target := index + 1; target < count && target <= index+4; target++ {
			workspace.Graph.Dependencies = append(workspace.Graph.Dependencies, packagegraph.Dependency{
				Owner: id, Binding: fmt.Sprintf("dep-%d", target), Target: fmt.Sprintf("package-%04d", target),
			})
		}
	}
	return workspace
}

func TestSemanticPackagesMetadataPreservesAliasesAndFirstMatches(t *testing.T) {
	workspace := metadataWorkspace(t, 4)
	alias := workspace.Instances[1]
	alias.NodeID = "alias"
	workspace.Instances = append(workspace.Instances, alias)
	workspace.Resolution.InstanceClasses[1].Members = append(workspace.Resolution.InstanceClasses[1].Members, alias.NodeID)
	workspace.Graph.Dependencies = append(workspace.Graph.Dependencies,
		packagegraph.Dependency{Owner: alias.NodeID, Target: workspace.Instances[3].NodeID},
		packagegraph.Dependency{Owner: "unresolved-owner", Target: "unresolved-target"})
	slices.Reverse(workspace.Graph.Dependencies)
	// Existing lookups select the first matching entry even on manually assembled workspaces.
	workspace.Instances = append(workspace.Instances, Instance{NodeID: workspace.Instances[0].NodeID, PackagePath: "missing"})
	workspace.Packages = append(workspace.Packages, Package{Path: workspace.Packages[0].Path})
	workspace.Graph.Nodes = append(workspace.Graph.Nodes, packagegraph.Node{ID: workspace.Graph.Nodes[0].ID})
	packages, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatal(err)
	}
	for index, loaded := range packages {
		want := []hir.PackageID{}
		for target := index + 1; target < 4; target++ {
			want = append(want, hir.PackageID(target+1))
		}
		if loaded.ID != hir.PackageID(index+1) || loaded.PackageUID != workspace.Packages[index].Manifest.Package.UID || !slices.Equal(loaded.Dependencies, want) {
			t.Fatalf("package %d: %+v, dependencies want %v", index, loaded, want)
		}
	}
}

func TestSemanticPackagesMetadataRejectsInvalidMappings(t *testing.T) {
	for _, test := range []struct {
		name, message string
		mutate        func(*Workspace)
	}{
		{"instance", "has no package mapping", func(w *Workspace) { w.Instances = w.Instances[1:] }},
		{"package", "is missing", func(w *Workspace) { w.Packages = w.Packages[1:] }},
		{"node", "graph node", func(w *Workspace) { w.Graph.Nodes = w.Graph.Nodes[1:] }},
		{"target", "has no package id", func(w *Workspace) { w.Graph.Dependencies[0].Target = "missing" }},
		{"self", "depends on itself", func(w *Workspace) { w.Graph.Dependencies[0].Target = w.Graph.Dependencies[0].Owner }},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := metadataWorkspace(t, 4)
			test.mutate(&workspace)
			if _, err := workspace.SemanticPackages(); err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %v, want %q", err, test.message)
			}
		})
	}
}

func BenchmarkSemanticPackagesMetadata(b *testing.B) {
	for _, count := range []int{16, 1024} {
		b.Run(fmt.Sprintf("packages-%d", count), func(b *testing.B) {
			workspace := metadataWorkspace(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				packages, err := workspace.SemanticPackages()
				if err != nil || len(packages) != count {
					b.Fatalf("packages = %d, error = %v", len(packages), err)
				}
			}
		})
	}
}
