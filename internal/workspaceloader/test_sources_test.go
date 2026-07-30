package workspaceloader

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithRootTestsAddsAnIsolatedRootSourceSet(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(`
[workspace]
members = ["libs/answer"]

[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`), map[string]string{
		"src/main.fy": "module main;\npub fn i32 value() { return 42; }\n",
	})
	writePackage(t, root, "libs/answer", memberManifest("example.libs.answer", "answer", "1.0.0", "fairyc-language/4", ""), map[string]string{
		"src/answer.fy": "module answer;\npub fn i32 value() { return 42; }\n",
	})
	writeWorkspaceLoaderFile(t, root, "tests/answer.fy", `module tests::answer;
import answer as answer;
@test
fn void dependency_returns_answer() { assert(answer::value() == 42); }
`)

	production, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	productionRoot, _ := production.Package(".")
	if len(productionRoot.Sources) != 1 || productionRoot.Sources[0].Test {
		t.Fatalf("production sources = %+v", productionRoot.Sources)
	}

	testWorkspace, err := WithRootTests(production)
	if err != nil {
		t.Fatalf("WithRootTests() error = %v", err)
	}
	testRoot, _ := testWorkspace.Package(".")
	if len(testRoot.Sources) != 2 || testRoot.Sources[1].Path != "tests/answer.fy" || !testRoot.Sources[1].Test || testRoot.Sources[1].Module.Path != "tests::answer" {
		t.Fatalf("test sources = %+v", testRoot.Sources)
	}
	if testRoot.Digest == productionRoot.Digest {
		t.Fatal("test source bytes did not change the ephemeral package digest")
	}
	if len(productionRoot.Sources) != 1 {
		t.Fatalf("WithRootTests() mutated production sources = %+v", productionRoot.Sources)
	}
	foundLookup := false
	for _, lookup := range testWorkspace.Resolution.Lookups {
		if lookup.ImportPath == "answer" {
			foundLookup = true
		}
	}
	if !foundLookup {
		t.Fatalf("test dependency lookup missing from %+v", testWorkspace.Resolution.Lookups)
	}
	files, err := testWorkspace.SemanticFiles()
	if err != nil {
		t.Fatalf("SemanticFiles() error = %v", err)
	}
	rootID, err := testWorkspace.RootPackageID()
	if err != nil {
		t.Fatalf("RootPackageID() error = %v", err)
	}
	foundTestFile := false
	for _, file := range files {
		if file.Package == rootID && file.Path == "tests/answer.fy" && file.Kind == SemanticFileTest &&
			file.Digest != ([32]byte{}) && file.LineMap != ([32]byte{}) {
			foundTestFile = true
		}
	}
	if !foundTestFile {
		t.Fatalf("root package lacks exact test-file metadata: %+v", files)
	}
}

func TestWithRootTestsLeavesAnAbsentOrEmptyTreeOutOfTheWorkspace(t *testing.T) {
	t.Parallel()
	for _, createEmpty := range []bool{false, true} {
		root := t.TempDir()
		writePackage(t, root, ".", rootManifest(""), validRootSource())
		if createEmpty {
			if err := os.Mkdir(filepath.Join(root, TestSourceDirectory), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		production, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		withTests, err := WithRootTests(production)
		if err != nil {
			t.Fatalf("WithRootTests(empty=%t) error = %v", createEmpty, err)
		}
		loaded, _ := withTests.Package(".")
		if len(loaded.Sources) != 1 || loaded.Sources[0].Test {
			t.Fatalf("WithRootTests(empty=%t) sources = %+v", createEmpty, loaded.Sources)
		}
	}
}

func TestOrdinaryLoadIgnoresInvalidTestSourceButTestLoadingRejectsIt(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), validRootSource())
	writeWorkspaceLoaderFile(t, root, "tests/broken.fy", "module tests::broken;\n@test\nfn void broken( {\n")

	production, err := Load(root)
	if err != nil {
		t.Fatalf("ordinary Load() read isolated test source: %v", err)
	}
	_, err = WithRootTests(production)
	var loadError *LoadError
	if !errors.As(err, &loadError) || loadError.Reason != ReasonSourceSyntax || loadError.Path != "tests/broken.fy" || len(loadError.Diagnostics) == 0 {
		t.Fatalf("WithRootTests() error = %#v", err)
	}
}

func TestWithRootTestsRejectsAmbiguousOrOverlappingTrees(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		setup  func(*testing.T, string) Workspace
		reason string
	}{
		{
			name: "production source overlap", reason: ReasonTestSourceRoot,
			setup: func(t *testing.T, root string) Workspace {
				writePackage(t, root, ".", strings.Replace(rootManifest(""), `source = "src"`, `source = "tests"`, 1), map[string]string{
					"tests/main.fy": "module main;\n",
				})
				workspace, err := Load(root)
				if err != nil {
					t.Fatal(err)
				}
				return workspace
			},
		},
		{
			name: "workspace member overlap", reason: ReasonTestSourceRoot,
			setup: func(t *testing.T, root string) Workspace {
				writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"tests/member\"]\n"), validRootSource())
				writePackage(t, root, "tests/member", memberManifest("example.tests.member", "member", "1.0.0", "fairyc-language/4", ""), validMemberSource("member"))
				workspace, err := Load(root)
				if err != nil {
					t.Fatal(err)
				}
				return workspace
			},
		},
		{
			name: "wrong directory case", reason: ReasonPathCase,
			setup: func(t *testing.T, root string) Workspace {
				writePackage(t, root, ".", rootManifest(""), validRootSource())
				writeWorkspaceLoaderFile(t, root, "Tests/example.fy", "module Tests::example;\n")
				workspace, err := Load(root)
				if err != nil {
					t.Fatal(err)
				}
				return workspace
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			workspace := test.setup(t, root)
			_, err := WithRootTests(workspace)
			var loadError *LoadError
			if !errors.As(err, &loadError) || loadError.Reason != test.reason {
				t.Fatalf("WithRootTests() error = %#v, want reason %q", err, test.reason)
			}
		})
	}
}

func writeWorkspaceLoaderFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absolute, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
