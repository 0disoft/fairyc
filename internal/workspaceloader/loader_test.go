package workspaceloader

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

func TestLoadBuildsFeatureDistinctLocalGraph(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(`
[workspace]
members = ["libs/net"]

[dependencies.net-base]
path = "libs/net"
version = "1.2.0"
features = ["base"]

[dependencies.net-plain]
path = "libs/net"
version = "1.2.0"
features = []
`), map[string]string{"src/main.fy": "module main;\n"})
	writePackage(t, root, "libs/net", memberManifest("example.libs.net", "net", "1.2.0", "fairyc-language/8", ""), map[string]string{"src/net.fy": "module net;\n"})

	workspace, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(workspace.Packages) != 2 || len(workspace.Instances) != 3 || len(workspace.Graph.Nodes) != 3 || len(workspace.Graph.Dependencies) != 2 {
		t.Fatalf("workspace sizes = packages %d, nodes %d, dependencies %d", len(workspace.Packages), len(workspace.Graph.Nodes), len(workspace.Graph.Dependencies))
	}
	loaded, found := workspace.Package("libs/net")
	if !found || len(loaded.Sources) != 1 || loaded.Digest.String() == "" {
		t.Fatalf("member = %+v, found = %v", loaded, found)
	}
	var featureSets [][]string
	for _, node := range workspace.Graph.Nodes {
		if node.Identity.PackageUID == "example.libs.net" {
			featureSets = append(featureSets, node.Identity.Features)
		}
		if !strings.HasPrefix(node.Identity.SourceIdentity, "fysrc1-sha256-") {
			t.Fatalf("source identity = %q", node.Identity.SourceIdentity)
		}
	}
	if len(featureSets) != 2 || !slices.Equal(featureSets[0], []string{"base"}) || len(featureSets[1]) != 0 {
		t.Fatalf("feature sets = %v", featureSets)
	}
}

func TestLoadInjectsExactCompilerBundledStandardLibrary(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), map[string]string{
		"src/main.fy": "module main;\n\nimport std::alloc as alloc;\n\nfn i32 inspect(ref mut alloc::Arena arena) {\n    return 1;\n}\n",
	})

	workspace, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(workspace.Packages) != 2 || len(workspace.Graph.Nodes) != 2 || len(workspace.Graph.Dependencies) != 1 {
		t.Fatalf("workspace sizes = packages %d, nodes %d, dependencies %d", len(workspace.Packages), len(workspace.Graph.Nodes), len(workspace.Graph.Dependencies))
	}
	standardLibrary, found := workspace.Package(sdkstdlib.PackagePath)
	if !found || standardLibrary.SourceKind != SourceKindSDK || standardLibrary.SDKIdentity != sdkstdlib.SDKIdentity ||
		standardLibrary.RuntimeABI != sdkstdlib.RuntimeABI || standardLibrary.Directory != "" || standardLibrary.SourceRoot != "" ||
		standardLibrary.Manifest.Package.UID != sdkstdlib.PackageUID || standardLibrary.Manifest.Package.Version != sdkstdlib.Version ||
		len(standardLibrary.Sources) != 6 || standardLibrary.Sources[0].Module.Path != "std::alloc" ||
		standardLibrary.Sources[1].Module.Path != "std::bytes" || standardLibrary.Sources[2].Module.Path != "std::fs" ||
		standardLibrary.Sources[3].Module.Path != "std::io" || standardLibrary.Sources[4].Module.Path != "std::process" ||
		standardLibrary.Sources[5].Module.Path != "std::text" {
		t.Fatalf("standard library = %+v, found = %v", standardLibrary, found)
	}
	if workspace.Graph.Nodes[1].Role != packagegraph.RoleStandardLibrary || workspace.Graph.Nodes[1].Identity.SourceIdentity != sdkstdlib.SourceIdentity(sdkstdlib.SDKIdentity, sdkstdlib.RuntimeABI, standardLibrary.Digest) {
		t.Fatalf("standard library node = %+v", workspace.Graph.Nodes[1])
	}
	edge := workspace.Graph.Dependencies[0]
	if edge.Owner != "pkg:root[]" || edge.Binding != "stdlib" || edge.Target != "sdk:stdlib[]" {
		t.Fatalf("standard library edge = %+v", edge)
	}
	if len(workspace.Resolution.Lookups) != 2 ||
		workspace.Resolution.Lookups[0].ImportPath != "std::alloc" ||
		!slices.Equal(workspace.Resolution.Lookups[0].ProviderMembers, []string{"sdk:stdlib[]"}) ||
		workspace.Resolution.Lookups[1].ImportPath != "std::bytes" ||
		!slices.Equal(workspace.Resolution.Lookups[1].ProviderMembers, []string{"sdk:stdlib[]"}) {
		t.Fatalf("standard library lookups = %+v", workspace.Resolution.Lookups)
	}
}

func TestLoadRejectsWorkspaceContractFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		reason string
		setup  func(*testing.T, string)
	}{
		{
			name: "dependency version", reason: ReasonDependencyVersion,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/net\"]\n\n[dependencies.net]\npath = \"libs/net\"\nversion = \"9.0.0\"\nfeatures = []\n"), validRootSource())
				writePackage(t, root, "libs/net", memberManifest("example.libs.net", "net", "1.0.0", "fairyc-language/8", ""), validMemberSource("net"))
			},
		},
		{
			name: "duplicate uid", reason: ReasonDuplicatePackageUID,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/net\"]\n"), validRootSource())
				writePackage(t, root, "libs/net", memberManifest("example.app", "net", "1.0.0", "fairyc-language/8", ""), validMemberSource("net"))
			},
		},
		{
			name: "nested workspace", reason: ReasonNestedWorkspace,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/net\"]\n"), validRootSource())
				writePackage(t, root, "libs/net", memberManifest("example.libs.net", "net", "1.0.0", "fairyc-language/8", "\n[workspace]\nmembers = []\n"), validMemberSource("net"))
			},
		},
		{
			name: "unknown dependency", reason: ReasonUnknownDependency,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest("\n[dependencies.net]\npath = \"libs/net\"\nversion = \"1.0.0\"\nfeatures = []\n"), validRootSource())
			},
		},
		{
			name: "language mismatch", reason: ReasonLanguageMismatch,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/net\"]\n"), validRootSource())
				writePackage(t, root, "libs/net", memberManifest("example.libs.net", "net", "1.0.0", "fairyc-language/0", ""), validMemberSource("net"))
			},
		},
		{
			name: "empty source root", reason: ReasonSourceRoot,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest(""), nil)
			},
		},
		{
			name: "invalid syntax", reason: ReasonSourceSyntax,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest(""), map[string]string{"src/main.fy": "module ;\n"})
			},
		},
		{
			name: "member path case", reason: ReasonPathCase,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/NET\"]\n"), validRootSource())
				writePackage(t, root, "libs/net", memberManifest("example.libs.net", "net", "1.0.0", "fairyc-language/8", ""), validMemberSource("net"))
			},
		},
		{
			name: "overlapping members", reason: ReasonOverlappingMember,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs\", \"libs/net\"]\n"), validRootSource())
			},
		},
		{
			name: "reserved standard library uid", reason: ReasonStandardLibrary,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", strings.Replace(rootManifest(""), `uid = "example.app"`, `uid = "fairyc.std"`, 1), validRootSource())
			},
		},
		{
			name: "reserved standard library dependency binding", reason: ReasonStandardLibrary,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/net\"]\n\n[dependencies.stdlib]\npath = \"libs/net\"\nversion = \"1.0.0\"\nfeatures = []\n"), validRootSource())
				writePackage(t, root, "libs/net", memberManifest("example.libs.net", "net", "1.0.0", "fairyc-language/8", ""), validMemberSource("net"))
			},
		},
		{
			name: "reserved standard library module namespace", reason: ReasonStandardLibrary,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest(""), map[string]string{"src/std/alloc.fy": "module std::alloc;\n"})
			},
		},
		{
			name: "missing standard library module", reason: ReasonPackageGraph,
			setup: func(t *testing.T, root string) {
				writePackage(t, root, ".", rootManifest(""), map[string]string{"src/main.fy": "module main;\nimport std::missing as missing;\n"})
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			test.setup(t, root)
			_, err := Load(root)
			var loadError *LoadError
			if !errors.As(err, &loadError) || loadError.Reason != test.reason {
				t.Fatalf("Load() error = %#v, want reason %q", err, test.reason)
			}
		})
	}
}

func TestLoadRejectsReachableDependencyCycle(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/a\", \"libs/b\"]\n\n[dependencies.a]\npath = \"libs/a\"\nversion = \"1.0.0\"\nfeatures = []\n"), validRootSource())
	writePackage(t, root, "libs/a", memberManifest("example.libs.a", "a", "1.0.0", "fairyc-language/8", "\n[dependencies.b]\npath = \"libs/b\"\nversion = \"1.0.0\"\nfeatures = []\n"), validMemberSource("a"))
	writePackage(t, root, "libs/b", memberManifest("example.libs.b", "b", "1.0.0", "fairyc-language/8", "\n[dependencies.a]\npath = \"libs/a\"\nversion = \"1.0.0\"\nfeatures = []\n"), validMemberSource("b"))
	_, err := Load(root)
	var loadError *LoadError
	if !errors.As(err, &loadError) || loadError.Reason != ReasonPackageGraph || !strings.Contains(loadError.Detail, "package-dependency-cycle") {
		t.Fatalf("Load() error = %#v", err)
	}
}

func TestLoadRejectsSymlinkMember(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	target := t.TempDir()
	writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"linked\"]\n"), validRootSource())
	writePackage(t, target, ".", memberManifest("example.libs.linked", "linked", "1.0.0", "fairyc-language/8", ""), validMemberSource("linked"))
	if err := os.Symlink(target, filepath.Join(root, "linked")); err != nil {
		t.Skipf("host cannot create directory symlink: %v", err)
	}
	_, err := Load(root)
	var loadError *LoadError
	if !errors.As(err, &loadError) || loadError.Reason != ReasonSymlink {
		t.Fatalf("Load() error = %#v", err)
	}
}

func rootManifest(extra string) string {
	return `schema = "fairyc.manifest/v1"
[package]
uid = "example.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/8"
source = "src"
` + extra
}

func memberManifest(uid, name, version, language, extra string) string {
	return `schema = "fairyc.manifest/v1"
[package]
uid = "` + uid + `"
name = "` + name + `"
version = "` + version + `"
language = "` + language + `"
source = "src"
` + extra
}

func validRootSource() map[string]string {
	return map[string]string{"src/main.fy": "module main;\n"}
}

func validMemberSource(module string) map[string]string {
	return map[string]string{"src/" + module + ".fy": "module " + module + ";\n"}
}

func writePackage(t *testing.T, root, relative, manifest string, sources map[string]string) {
	t.Helper()
	directory := root
	if relative != "." {
		directory = filepath.Join(root, filepath.FromSlash(relative))
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "fairy.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	for path, contents := range sources {
		absolute := filepath.Join(directory, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
