package workspaceloader

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func sharedSourceWorkspace(tb testing.TB, instances int) Workspace {
	tb.Helper()
	contents := bytes.Repeat([]byte("// shared source\n"), 1<<16)
	files := []packagecontent.File{{Path: "src/main.fy", Bytes: contents}}
	digest, err := packagecontent.Compute(files)
	if err != nil {
		tb.Fatal(err)
	}
	workspace := Workspace{Packages: []Package{{
		Path: "shared", Digest: digest,
		Sources: []SourceFile{{Path: files[0].Path, Bytes: contents}},
	}}}
	for index := range instances {
		id := fmt.Sprintf("feature-%d", index)
		identity := packagegraph.Identity{PackageUID: "shared", SourceIdentity: digest.String(), VersionIdentity: "1", Features: []string{id}}
		instance, err := packagegraph.ComputeInstanceID(identity)
		if err != nil {
			tb.Fatal(err)
		}
		workspace.Graph.Nodes = append(workspace.Graph.Nodes, packagegraph.Node{ID: id, Identity: identity})
		workspace.Instances = append(workspace.Instances, Instance{NodeID: id, PackagePath: "shared", Features: []string{id}})
		workspace.Resolution.InstanceClasses = append(workspace.Resolution.InstanceClasses, packagegraph.InstanceClass{ID: instance, Members: []string{id}})
	}
	return workspace
}

func TestSemanticFilesReuseMatchesIndependentInstanceResults(t *testing.T) {
	workspace, err := Load(filepath.Join("..", "..", "spec", "fixtures", "workspace-loader", "cases", "pass-feature-distinct"))
	if err != nil {
		t.Fatal(err)
	}
	// Exercise multiple files, test classification, and result slice growth.
	for index := range workspace.Packages {
		loaded := &workspace.Packages[index]
		loaded.Sources = append(loaded.Sources, SourceFile{Path: "src/extra.fy", Bytes: []byte("module extra;\n"), Test: true})
		var files []packagecontent.File
		for _, file := range loaded.Sources {
			files = append(files, packagecontent.File{Path: file.Path, Bytes: file.Bytes})
		}
		loaded.Digest, err = packagecontent.Compute(files)
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := workspace.SemanticFiles()
	if err != nil {
		t.Fatal(err)
	}
	var want []SemanticFile
	for index, class := range workspace.Resolution.InstanceClasses {
		isolated := workspace
		isolated.Resolution.InstanceClasses = []packagegraph.InstanceClass{class}
		files, err := isolated.SemanticFiles()
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			file.Package = hir.PackageID(index + 1)
			want = append(want, file)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("combined identities differ from independent instance results:\n got %+v\nwant %+v", got, want)
	}
}

func TestSemanticFilesReuseStillRevalidatesEachCall(t *testing.T) {
	workspace := sharedSourceWorkspace(t, 8)
	if _, err := workspace.SemanticFiles(); err != nil {
		t.Fatal(err)
	}
	workspace.Packages[0].Sources[0].Bytes[0] ^= 0x20
	if _, err := workspace.SemanticFiles(); err == nil || !strings.Contains(err.Error(), "source bytes changed after validation") {
		t.Fatalf("changed source rejection = %v", err)
	}
}

func BenchmarkSemanticFilesSharedSource(b *testing.B) {
	for _, count := range []int{1, 16} {
		b.Run(fmt.Sprintf("instances-%d", count), func(b *testing.B) {
			workspace := sharedSourceWorkspace(b, count)
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
