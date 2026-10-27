package workspaceloader

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// Keep the original linear lookup as an independent compatibility oracle.
func linearPackageCoordinates(workspace Workspace) (map[hir.PackageID]packagegraph.CoordinateID, error) {
	coordinates := make(map[hir.PackageID]packagegraph.CoordinateID, len(workspace.Resolution.InstanceClasses))
	for index, class := range workspace.Resolution.InstanceClasses {
		if len(class.Members) == 0 {
			return nil, fmt.Errorf("workspace has an empty package instance class")
		}
		var node *packagegraph.Node
		for nodeIndex := range workspace.Graph.Nodes {
			if candidate := &workspace.Graph.Nodes[nodeIndex]; candidate.ID == class.Members[0] {
				node = candidate
				break
			}
		}
		if node == nil {
			return nil, fmt.Errorf("workspace graph node %q is missing", class.Members[0])
		}
		coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.CoordinateOf(node.Identity))
		if err != nil {
			return nil, fmt.Errorf("compute package %q coordinate: %w", node.ID, err)
		}
		coordinates[hir.PackageID(index+1)] = coordinate
	}
	return coordinates, nil
}

func TestPackageCoordinatesMatchesLinearLookup(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Workspace)
	}{
		{"ordered", func(*Workspace) {}},
		{"unordered", func(w *Workspace) { slices.Reverse(w.Graph.Nodes) }},
		{"class-order", func(w *Workspace) { slices.Reverse(w.Resolution.InstanceClasses) }},
		{"first-duplicate-wins", func(w *Workspace) {
			w.Graph.Nodes = append(w.Graph.Nodes, packagegraph.Node{ID: w.Graph.Nodes[0].ID})
		}},
		{"invalid-first-duplicate", func(w *Workspace) {
			w.Graph.Nodes = append([]packagegraph.Node{{ID: w.Graph.Nodes[0].ID}}, w.Graph.Nodes...)
		}},
		{"representative-member", func(w *Workspace) {
			w.Resolution.InstanceClasses[0].Members = append(w.Resolution.InstanceClasses[0].Members, "missing-alias")
		}},
		{"empty-class", func(w *Workspace) { w.Resolution.InstanceClasses[0].Members = nil }},
		{"missing-node", func(w *Workspace) { w.Graph.Nodes = w.Graph.Nodes[1:] }},
		{"invalid-coordinate", func(w *Workspace) { w.Graph.Nodes[0].Identity.VersionIdentity = "" }},
		{"error-order", func(w *Workspace) {
			w.Graph.Nodes[0].Identity.VersionIdentity = ""
			w.Resolution.InstanceClasses[1].Members = nil
		}},
		{"unreferenced-invalid-node", func(w *Workspace) {
			w.Graph.Nodes = append(w.Graph.Nodes, packagegraph.Node{ID: "unreferenced"})
		}},
		{"no-classes", func(w *Workspace) { w.Resolution.InstanceClasses = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := metadataWorkspace(t, 6)
			test.mutate(&workspace)
			want, wantErr := linearPackageCoordinates(workspace)
			got, gotErr := workspace.PackageCoordinates()
			if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) {
				t.Fatalf("coordinates=%v, error=%v; want %v, %v", got, gotErr, want, wantErr)
			}
		})
	}
}

func TestPackageCoordinatesReflectsWorkspaceEdits(t *testing.T) {
	workspace := metadataWorkspace(t, 2)
	before, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	workspace.Graph.Nodes[0].Identity.VersionIdentity = "2"
	workspace.Graph.Nodes[0].Identity.Features = []string{"feature"}
	after, err := workspace.PackageCoordinates()
	if err != nil || after[1] == before[1] || after[2] != before[2] {
		t.Fatalf("updated coordinates=%v, error=%v; before=%v", after, err, before)
	}
	after[1] = packagegraph.CoordinateID{}
	next, err := workspace.PackageCoordinates()
	if err != nil || next[1] == after[1] {
		t.Fatalf("caller mutation leaked into coordinates=%v, error=%v", next, err)
	}
	workspace.Graph.Nodes[0].ID = "renamed"
	if _, err := workspace.PackageCoordinates(); err == nil {
		t.Fatal("removed node was reused after workspace mutation")
	}
}

func BenchmarkPackageCoordinates(b *testing.B) {
	for _, count := range []int{16, 1024, 4096} {
		b.Run(fmt.Sprintf("packages-%d", count), func(b *testing.B) {
			workspace := metadataWorkspace(b, count)
			b.ReportAllocs()
			for b.Loop() {
				coordinates, err := workspace.PackageCoordinates()
				if err != nil || len(coordinates) != count {
					b.Fatalf("coordinates=%d, error=%v", len(coordinates), err)
				}
			}
		})
	}
}
