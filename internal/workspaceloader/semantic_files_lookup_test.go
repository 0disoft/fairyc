package workspaceloader

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecontent"
)

func fileCatalogWorkspace(tb testing.TB, count int) Workspace {
	tb.Helper()
	workspace := metadataWorkspace(tb, count)
	contents := []byte("module main;\n// A small package source used to measure metadata preparation.\nfn i32 main() { return 42; }\n")
	for index := range workspace.Packages {
		loaded := &workspace.Packages[index]
		loaded.Sources = []SourceFile{{Path: "src/main.fy", Bytes: slices.Clone(contents)}}
		var err error
		loaded.Digest, err = packagecontent.Compute([]packagecontent.File{{Path: "src/main.fy", Bytes: contents}})
		if err != nil {
			tb.Fatal(err)
		}
	}
	return workspace
}

func TestSemanticFilesCatalogOrderAndDuplicates(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprintf("reverse-%v", reverse), func(t *testing.T) {
			workspace := fileCatalogWorkspace(t, 4)
			var want []SemanticFile
			for index, loaded := range workspace.Packages {
				digest, lineMap := packagecontent.ComputeFileIdentities(loaded.Sources[0].Bytes)
				want = append(want, SemanticFile{Package: hir.PackageID(index + 1), Path: "src/main.fy", Digest: digest, LineMap: lineMap, Kind: SemanticFileSource})
			}
			if reverse {
				slices.Reverse(workspace.Packages)
				slices.Reverse(workspace.Instances)
			}
			// Later duplicates must not override first-match mappings or validated bytes.
			workspace.Instances = append(workspace.Instances, Instance{NodeID: "package-0000", PackagePath: "missing"})
			workspace.Packages = append(workspace.Packages, Package{Path: "package-0000"})
			workspace.Resolution.InstanceClasses[0].Members = append(workspace.Resolution.InstanceClasses[0].Members, "unresolved-alias")
			got, err := workspace.SemanticFiles()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("files=%+v, error=%v; want %+v", got, err, want)
			}
		})
	}
}

func TestSemanticFilesCatalogErrorsAndFreshness(t *testing.T) {
	for _, test := range []struct {
		name, message string
		mutate        func(*Workspace)
	}{
		{"missing-instance", "has no package mapping", func(w *Workspace) { w.Instances = w.Instances[1:] }},
		{"missing-package", "is missing", func(w *Workspace) { w.Packages = w.Packages[1:] }},
		{"empty-class", "empty package instance class", func(w *Workspace) { w.Resolution.InstanceClasses[0].Members = nil }},
		{"first-duplicate-instance", "is missing", func(w *Workspace) {
			w.Instances = append([]Instance{{NodeID: "package-0000", PackagePath: "missing"}}, w.Instances...)
		}},
		{"first-duplicate-package", "has no files", func(w *Workspace) {
			w.Packages = append([]Package{{Path: "package-0000"}}, w.Packages...)
		}},
		{"error-order", "has no package mapping", func(w *Workspace) {
			w.Instances = w.Instances[1:]
			w.Resolution.InstanceClasses[1].Members = nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := fileCatalogWorkspace(t, 4)
			if _, err := workspace.SemanticFiles(); err != nil {
				t.Fatal(err)
			}
			test.mutate(&workspace)
			got, err := workspace.SemanticFiles()
			if got != nil || err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("files=%v, error=%v; want %q", got, err, test.message)
			}
		})
	}
	files, err := (Workspace{}).SemanticFiles()
	if err != nil || files == nil || len(files) != 0 {
		t.Fatalf("empty workspace: files=%v, error=%v", files, err)
	}
}

func BenchmarkSemanticFilesCatalog(b *testing.B) {
	for _, count := range []int{16, 1024, 4096} {
		b.Run(fmt.Sprintf("packages-%d", count), func(b *testing.B) {
			workspace := fileCatalogWorkspace(b, count)
			b.ReportAllocs()
			for b.Loop() {
				files, err := workspace.SemanticFiles()
				if err != nil || len(files) != count {
					b.Fatalf("files=%d, error=%v", len(files), err)
				}
			}
		})
	}
}
