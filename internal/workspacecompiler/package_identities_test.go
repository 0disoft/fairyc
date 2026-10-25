package workspacecompiler

import (
	"fmt"
	"maps"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestPackageIdentityMapsMatchWorkspace(t *testing.T) {
	t.Parallel()
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "spec", "fixtures", "workspace-loader", "cases", "pass-feature-distinct"))
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatal(err)
	}
	identities, coordinates, err := packageIdentityMaps(metadata)
	if err != nil {
		t.Fatal(err)
	}
	wantIdentities, err := workspace.PackageIdentities()
	if err != nil {
		t.Fatal(err)
	}
	wantCoordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(identities, wantIdentities) || !maps.Equal(coordinates, wantCoordinates) {
		t.Fatal("metadata maps differ from workspace identities or coordinates")
	}
	// A later metadata snapshot must produce fresh values, not a retained map.
	metadata[0].Instance[0] ^= 1
	metadata[0].Coordinate[0] ^= 1
	nextIdentities, nextCoordinates, err := packageIdentityMaps(metadata)
	if err != nil || identities[metadata[0].ID] == nextIdentities[metadata[0].ID] || coordinates[metadata[0].ID] == nextCoordinates[metadata[0].ID] {
		t.Fatalf("snapshot identities were reused: %v", err)
	}
}

func TestPackageIdentityMapsRejectDefaultPackage(t *testing.T) {
	identities, coordinates, err := packageIdentityMaps([]workspaceloader.SemanticPackage{{ID: hir.DefaultPackage}})
	if err == nil || identities != nil || coordinates != nil {
		t.Fatalf("default package accepted: %v", err)
	}
}

func BenchmarkPackageIdentityPreparation(b *testing.B) {
	for _, count := range []int{32, 256, 1024} {
		workspace := workspaceloader.Workspace{}
		for index := range count {
			name := fmt.Sprintf("package-%04d", index)
			identity := packagegraph.Identity{PackageUID: fmt.Sprintf("bench.package.%04d", index), SourceIdentity: "workspace", VersionIdentity: "1.0.0"}
			instance, err := packagegraph.ComputeInstanceID(identity)
			if err != nil {
				b.Fatal(err)
			}
			workspace.Packages = append(workspace.Packages, workspaceloader.Package{Path: name})
			workspace.Instances = append(workspace.Instances, workspaceloader.Instance{NodeID: name, PackagePath: name})
			workspace.Graph.Nodes = append(workspace.Graph.Nodes, packagegraph.Node{ID: name, Identity: identity})
			workspace.Resolution.InstanceClasses = append(workspace.Resolution.InstanceClasses, packagegraph.InstanceClass{ID: instance, Members: []string{name}})
		}
		for _, reuse := range []bool{false, true} {
			b.Run(fmt.Sprintf("packages=%d/reuse=%t", count, reuse), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					metadata, err := workspace.SemanticPackages()
					if err != nil {
						b.Fatal(err)
					}
					if reuse {
						if _, _, err := packageIdentityMaps(metadata); err != nil {
							b.Fatal(err)
						}
					} else {
						if _, err := workspace.PackageIdentities(); err != nil {
							b.Fatal(err)
						}
						if _, err := workspace.PackageCoordinates(); err != nil {
							b.Fatal(err)
						}
					}
				}
			})
		}
	}
}
