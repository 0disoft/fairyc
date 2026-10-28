package workspaceloader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func batchTestWorkspace(t testing.TB, count int) Workspace {
	t.Helper()
	root := t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var members []string
	var dependencies strings.Builder
	for index := 1; index < count; index++ {
		name := fmt.Sprintf("p%03d", index)
		directory := "libs/" + name
		members = append(members, fmt.Sprintf("%q", directory))
		fmt.Fprintf(&dependencies, "\n[dependencies.%s]\npath = %q\nversion = \"1.0.0\"\nfeatures = []\n", name, directory)
		write(directory+"/fairy.toml", memberManifest("example."+name, name, "1.0.0", "fairyc-language/20", ""))
		write(directory+"/src/main.fy", "module main;\n")
		write(directory+"/tests/main.fy", "module tests::main;\n@test\nfn void works() {}\n")
	}
	write("fairy.toml", rootManifest("\n[workspace]\nmembers = ["+strings.Join(members, ",")+"]\n"+dependencies.String()))
	write("src/main.fy", "module main;\n")
	write("tests/main.fy", "module tests::main;\n@test\nfn void works() {}\n")
	workspace, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := workspace.RootReachableLocalPackagePaths()
	if err != nil || len(paths) != count {
		t.Fatalf("reachable packages = %v, error = %v", paths, err)
	}
	return workspace
}

func BenchmarkWithWorkspaceTests(b *testing.B) {
	for _, count := range []int{8, 32} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			workspace := batchTestWorkspace(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				result, err := WithWorkspaceTests(workspace)
				if err != nil || len(result.Packages) < count {
					b.Fatalf("WithWorkspaceTests: %v", err)
				}
			}
		})
	}
}

func TestWorkspaceTestBatchMatchesSequentialPreparation(t *testing.T) {
	production := batchTestWorkspace(t, 4)
	// A test-only import must inject the SDK into the final combined graph.
	if err := os.WriteFile(filepath.Join(production.Root, "tests", "main.fy"), []byte("module tests::main;\nimport std::bytes as bytes;\n@test\nfn void works() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths, _ := production.RootReachableLocalPackagePaths()
	sequential := production
	for _, path := range paths {
		var err error
		sequential, err = WithPackageTests(sequential, path)
		if err != nil {
			t.Fatal(err)
		}
	}
	batch, err := WithWorkspaceTests(production)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(batch, sequential) {
		t.Fatal("batch preparation differs from sequential graph, sources or digests")
	}
	foundSDK := false
	for _, loaded := range batch.Packages {
		foundSDK = foundSDK || loaded.SourceKind == SourceKindSDK
	}
	if !foundSDK {
		t.Fatal("test-only SDK import was not resolved")
	}
	for _, path := range paths {
		original, _ := production.Package(path)
		if len(original.Sources) != 1 || original.Sources[0].Test {
			t.Fatal("test preparation mutated production sources")
		}
	}
}

func TestPackageTestTreesShareSourceLimit(t *testing.T) {
	production := batchTestWorkspace(t, 2)
	root, _ := production.Package(".")
	member, _ := production.Package("libs/p001")
	budget := limits{files: MaxSourceFiles - 1}
	if added, err := appendPackageTests(&root, &budget); err != nil || !added {
		t.Fatalf("first tree: added = %t, error = %v", added, err)
	}
	_, err := appendPackageTests(&member, &budget)
	var load *LoadError
	if !errors.As(err, &load) || load.Reason != ReasonSourceLimit {
		t.Fatalf("second tree did not enforce cumulative source limit: %v", err)
	}
}
