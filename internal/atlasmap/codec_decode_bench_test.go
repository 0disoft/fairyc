package atlasmap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Compile and encode outside the timer; measure only the public decode path.
func BenchmarkDecodePartition(b *testing.B) {
	for _, count := range []int{32, 512} {
		b.Run(fmt.Sprintf("functions=%d", count), func(b *testing.B) {
			partition := decodeBenchmarkPartition(b, count)
			for _, version := range []struct {
				name   string
				schema string
				encode func(Partition, uint64) ([]byte, error)
				decode func([]byte, uint64) (Partition, error)
			}{
				{"v1", LogicalSchema, Encode, Decode},
				{"v2", LogicalSchemaV2, EncodeV2, DecodeV2},
				{"v3", LogicalSchemaV3, EncodeV3, DecodeV3},
			} {
				b.Run(version.name, func(b *testing.B) {
					partition.data.header.LogicalSchema = version.schema
					encoded, err := version.encode(partition, testMapLimit)
					if err != nil {
						b.Fatal(err)
					}
					b.SetBytes(int64(len(encoded)))
					b.ReportAllocs()
					for b.Loop() {
						decoded, err := version.decode(encoded, uint64(len(encoded)))
						if err != nil || len(decoded.data.functions) != count {
							b.Fatalf("decode: functions=%d, error=%v", len(decoded.data.functions), err)
						}
					}
				})
			}
		})
	}
}

func decodeBenchmarkPartition(tb testing.TB, count int) Partition {
	tb.Helper()
	root := tb.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		tb.Fatal(err)
	}
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.atlas-decode\"\nname = \"atlas-decode\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
	var source strings.Builder
	source.WriteString("module main; fn i32 main() { return f_0(42); }\n")
	for i := 0; i < count-1; i++ {
		fmt.Fprintf(&source, "fn i32 f_%d(i32 value) { return value; }\n", i)
	}
	for path, contents := range map[string]string{"fairy.toml": manifest, "src/main.fy": source.String()} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(contents), 0o600); err != nil {
			tb.Fatal(err)
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		tb.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		tb.Fatal(err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil || !checked.Verified {
		tb.Fatalf("compile: verified=%t, error=%v", checked.Verified, err)
	}
	_, partitions, err := PrepareWorkspace(workspace, checked, testCompilerBuild())
	if err != nil || len(partitions) != 1 {
		tb.Fatalf("prepare: partitions=%d, error=%v", len(partitions), err)
	}
	return partitions[0]
}
