package workspaceloader

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

// Benchmark freshness only: source bytes are immutable cache inputs; parsing and
// AST construction are deliberately outside this allocation measurement.
func BenchmarkFileSnapshotUnchanged64MiB(b *testing.B) {
	root := b.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		b.Fatal(err)
	}
	manifestBytes := []byte(rootManifest(""))
	if err := os.WriteFile(filepath.Join(root, "fairy.toml"), manifestBytes, 0600); err != nil {
		b.Fatal(err)
	}
	manifest, err := projectmanifest.Parse(manifestBytes)
	if err != nil {
		b.Fatal(err)
	}
	pkg := Package{Path: ".", SourceKind: SourceKindLocal, Manifest: manifest}
	for i := 0; i < 8; i++ {
		name := fmt.Sprintf("chunk%d.fy", i)
		data := make([]byte, 8<<20)
		for index := range data {
			data[index] = ' '
		}
		copy(data, fmt.Sprintf("module chunk%d; /*", i))
		copy(data[len(data)-2:], "*/")
		if err := os.WriteFile(filepath.Join(root, "src", name), data, 0600); err != nil {
			b.Fatal(err)
		}
		pkg.Sources = append(pkg.Sources, SourceFile{Path: name, Bytes: data})
	}
	snapshot, err := CaptureFileSnapshot(b.Context(), Workspace{Root: root, Packages: []Package{pkg}}, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(64 << 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches, err := snapshot.Matches(b.Context(), nil)
		if err != nil || !matches {
			b.Fatalf("freshness: %v %v", matches, err)
		}
	}
}
