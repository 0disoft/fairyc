package workspaceloader

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestExactSourceReturnsDetachedValidatedFile(t *testing.T) {
	workspace := exactSourceWorkspace(t, 3)
	selected := &workspace.Packages[0].Sources[1]
	selected.Test = true
	instance := workspace.Resolution.InstanceClasses[0].ID.String()
	resolve := func() ExactSource {
		t.Helper()
		result, err := workspace.ResolveExactSource(instance, selected.Path)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	result := resolve()
	digest, lineMap := packagecontent.ComputeFileIdentities(selected.Bytes)
	if !bytes.Equal(result.Bytes, selected.Bytes) || result.File.Path != selected.Path ||
		result.File.Digest != digest || result.File.LineMap != lineMap || result.File.Kind != SemanticFileTest ||
		result.Package.Source != workspace.Packages[0].Digest || result.File.Package != result.Package.ID {
		t.Fatalf("exact source bytes or metadata differ from selected file: %+v", result.File)
	}
	original := slices.Clone(selected.Bytes)
	result.Bytes[0] ^= 0x20
	if !bytes.Equal(selected.Bytes, original) {
		t.Fatal("writing returned bytes changed the retained workspace")
	}
	result = resolve()
	selected.Bytes[0] ^= 0x20
	if !bytes.Equal(result.Bytes, original) {
		t.Fatal("writing retained bytes changed an already returned source")
	}
}

func TestExactSourceRevalidatesWholePackage(t *testing.T) {
	for _, test := range []struct {
		name, want string
		mutate     func(*Package)
	}{
		{"selected bytes", "retained package bytes are stale", func(p *Package) { p.Sources[0].Bytes[0] ^= 0x20 }},
		{"other bytes", "retained package bytes are stale", func(p *Package) { p.Sources[2].Bytes[0] ^= 0x20 }},
		{"digest", "retained package bytes are stale", func(p *Package) { p.Digest[0] ^= 1 }},
		{"duplicate selected", "source path is duplicated", func(p *Package) { p.Sources = append(p.Sources, p.Sources[0]) }},
		{"duplicate other", "duplicate path", func(p *Package) { p.Sources[2].Path = p.Sources[1].Path }},
		{"invalid other path", "invalid canonical", func(p *Package) { p.Sources[2].Path = "../escape.fy" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := exactSourceWorkspace(t, 3)
			instance := workspace.Resolution.InstanceClasses[0].ID.String()
			path := workspace.Packages[0].Sources[0].Path
			if _, err := workspace.ResolveExactSource(instance, path); err != nil {
				t.Fatal(err)
			}
			test.mutate(&workspace.Packages[0])
			if _, err := workspace.ResolveExactSource(instance, path); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func BenchmarkResolveExactSourceSmallFile(b *testing.B) {
	for _, count := range []int{1024, MaxSourceFiles} {
		b.Run(fmt.Sprintf("files-%d", count), func(b *testing.B) {
			workspace := exactSourceWorkspace(b, count)
			instance := workspace.Resolution.InstanceClasses[0].ID.String()
			path := workspace.Packages[0].Sources[0].Path
			b.ReportAllocs()
			for b.Loop() {
				result, err := workspace.ResolveExactSource(instance, path)
				if err != nil || len(result.Bytes) != 128 {
					b.Fatalf("exact source bytes=%d, error=%v", len(result.Bytes), err)
				}
			}
		})
	}
}

func exactSourceWorkspace(tb testing.TB, count int) Workspace {
	tb.Helper()
	workspace := metadataWorkspace(tb, 1)
	loaded := &workspace.Packages[0]
	files := make([]packagecontent.File, count)
	for index := range count {
		size := 16 << 10
		if index == 0 {
			size = 128
		}
		path := fmt.Sprintf("src/file_%04d.fy", index)
		contents := bytes.Repeat([]byte{' '}, size)
		copy(contents, fmt.Sprintf("module file_%04d;\n", index))
		loaded.Sources = append(loaded.Sources, SourceFile{Path: path, Bytes: contents})
		files[index] = packagecontent.File{Path: path, Bytes: contents}
	}
	var err error
	loaded.Digest, err = packagecontent.Compute(files)
	if err != nil {
		tb.Fatal(err)
	}
	workspace.Graph.Nodes[0].Identity.SourceIdentity = loaded.Digest.String()
	workspace.Resolution.InstanceClasses[0].ID, err = packagegraph.ComputeInstanceID(workspace.Graph.Nodes[0].Identity)
	if err != nil {
		tb.Fatal(err)
	}
	return workspace
}
