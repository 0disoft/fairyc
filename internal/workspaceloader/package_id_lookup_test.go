package workspaceloader

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func linearWorkspacePackageID(workspace Workspace, path string) (hir.PackageID, error) {
	var result hir.PackageID
	for index, class := range workspace.Resolution.InstanceClasses {
		if len(class.Members) == 0 {
			return hir.DefaultPackage, fmt.Errorf("workspace has an empty package instance class")
		}
		mapping, found := workspace.instance(class.Members[0])
		if !found || mapping.PackagePath != path {
			continue
		}
		if result != hir.DefaultPackage {
			return hir.DefaultPackage, fmt.Errorf("workspace package %q has multiple resolved instances", path)
		}
		result = hir.PackageID(index + 1)
	}
	if result == hir.DefaultPackage {
		return hir.DefaultPackage, fmt.Errorf("workspace package %q is not root-reachable", path)
	}
	return result, nil
}

func TestPackageIDLookupMatchesLinearSelection(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Workspace)
	}{
		{"ordered", func(*Workspace) {}},
		{"instance-order", func(w *Workspace) { slices.Reverse(w.Instances) }},
		{"class-order", func(w *Workspace) { slices.Reverse(w.Resolution.InstanceClasses) }},
		{"missing-mapping", func(w *Workspace) { w.Instances = w.Instances[1:] }},
		{"later-duplicate", func(w *Workspace) {
			w.Instances = append(w.Instances, Instance{NodeID: "package-0000", PackagePath: "package-0001"})
		}},
		{"earlier-duplicate", func(w *Workspace) {
			w.Instances = append([]Instance{{NodeID: "package-0000", PackagePath: "missing"}}, w.Instances...)
		}},
		{"ambiguous", func(w *Workspace) {
			w.Resolution.InstanceClasses = append(w.Resolution.InstanceClasses, w.Resolution.InstanceClasses[0])
		}},
		{"empty-class", func(w *Workspace) { w.Resolution.InstanceClasses[0].Members = nil }},
		{"representative-only", func(w *Workspace) {
			w.Resolution.InstanceClasses[0].Members = []string{"missing", "package-0000"}
		}},
		{"ambiguity-before-empty-class", func(w *Workspace) {
			w.Resolution.InstanceClasses = append(w.Resolution.InstanceClasses, w.Resolution.InstanceClasses[0], packagegraph.InstanceClass{})
		}},
		{"path-edit", func(w *Workspace) { w.Instances[0].PackagePath = "missing" }},
		{"no-classes", func(w *Workspace) { w.Resolution.InstanceClasses = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := metadataWorkspace(t, 4)
			// Lookup state must not survive edits to the public Workspace slices.
			if _, err := workspace.PackageID("package-0000"); err != nil {
				t.Fatal(err)
			}
			test.mutate(&workspace)
			for _, path := range []string{"package-0000", "package-0001", "package-0003", "missing", ""} {
				want, wantErr := linearWorkspacePackageID(workspace, path)
				got, gotErr := workspace.PackageID(path)
				if got != want || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
					t.Fatalf("path=%q: id=%d, error=%v; want %d, %v", path, got, gotErr, want, wantErr)
				}
			}
		})
	}
}

// test --workspace resolves the selected package ID once per local package.
// Measure only that repeated selection, excluding loading and test compilation.
func BenchmarkWorkspacePackageIDSelection(b *testing.B) {
	for _, count := range []int{32, 256, 512} {
		b.Run(fmt.Sprintf("packages-%d", count), func(b *testing.B) {
			workspace := metadataWorkspace(b, count)
			paths := make([]string, count)
			for index := range paths {
				paths[index] = workspace.Packages[index].Path
			}
			b.ReportAllocs()
			for b.Loop() {
				for index, path := range paths {
					id, err := workspace.PackageID(path)
					if err != nil || id != hir.PackageID(index+1) {
						b.Fatalf("package %d: id=%d, error=%v", index, id, err)
					}
				}
			}
		})
	}
}
