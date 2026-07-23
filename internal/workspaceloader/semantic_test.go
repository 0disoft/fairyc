package workspaceloader

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestSemanticInputReachesVerifiedCompiler(t *testing.T) {
	t.Parallel()
	workspace, err := Load(filepath.Join("..", "..", "spec", "fixtures", "workspace-loader", "cases", "pass-feature-distinct"))
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	if len(input.Files) != 3 || len(input.Imports) != 0 {
		t.Fatalf("semantic input has %d files and %d imports", len(input.Files), len(input.Imports))
	}
	identities, err := workspace.PackageIdentities()
	if err != nil {
		t.Fatal(err)
	}
	if len(identities) != len(workspace.Resolution.InstanceClasses) {
		t.Fatalf("PackageIdentities() has %d entries for %d classes", len(identities), len(workspace.Resolution.InstanceClasses))
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	if len(coordinates) != len(identities) {
		t.Fatalf("PackageCoordinates() has %d entries for %d identities", len(coordinates), len(identities))
	}
	for _, file := range input.Files {
		if file.Package == hir.DefaultPackage || identities[file.Package] == ([32]byte{}) || coordinates[file.Package] == ([32]byte{}) {
			t.Fatalf("semantic file package %d has no exact and coordinate identities", file.Package)
		}
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil || rootPackage == 0 {
		t.Fatalf("RootPackageID() = %d, %v", rootPackage, err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified {
		t.Fatalf("workspace did not reach verified MIR: phase=%s failed=%s diagnostics=%v unsupported=%v", result.Semantic.CompletedThrough, result.Semantic.FailedPhase, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
}

func TestSemanticFilesRejectsSourceMutationAfterWorkspaceValidation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), validRootSource())
	workspace, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspace.Packages) == 0 || len(workspace.Packages[0].Sources) == 0 || len(workspace.Packages[0].Sources[0].Bytes) == 0 {
		t.Fatalf("workspace source closure = %+v", workspace.Packages)
	}
	workspace.Packages[0].Sources[0].Bytes[0] ^= 0x20
	if _, err := workspace.SemanticFiles(); err == nil || !strings.Contains(err.Error(), "source bytes changed after validation") {
		t.Fatalf("SemanticFiles() error = %v", err)
	}
}

func TestBundledStandardLibraryReachesVerifiedCompiler(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), map[string]string{
		"src/main.fy": "module main;\n\nimport std::alloc as alloc;\n\nfn i32 inspect(ref mut alloc::Arena arena) {\n    return 1;\n}\n",
	})
	workspace, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	if len(input.Files) != 3 || len(input.Imports) != 1 {
		t.Fatalf("semantic input has %d files and %d imports", len(input.Files), len(input.Imports))
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified {
		t.Fatalf("workspace did not reach verified MIR: phase=%s failed=%s diagnostics=%v unsupported=%v", result.Semantic.CompletedThrough, result.Semantic.FailedPhase, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	packages, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 2 || packages[0].Role != packagegraph.RoleRoot || packages[1].Role != packagegraph.RoleStandardLibrary ||
		!slices.Equal(packages[0].Dependencies, []hir.PackageID{packages[1].ID}) || packages[1].RuntimeABI == "" {
		t.Fatalf("SemanticPackages() = %+v", packages)
	}
	identities, err := workspace.PackageIdentities()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	packagePaths := make(map[hir.PackageID]string, len(packages))
	for _, loaded := range packages {
		if loaded.PackageUID == "" || loaded.Name == "" || loaded.Version == "" || loaded.Language != "fairyc-language/0" ||
			loaded.Instance != identities[loaded.ID] || loaded.Coordinate != coordinates[loaded.ID] || loaded.Source == ([32]byte{}) || loaded.Path == "" {
			t.Fatalf("SemanticPackages() package = %+v, identities = %+v, coordinates = %+v", loaded, identities, coordinates)
		}
		packagePaths[loaded.ID] = loaded.Path
	}
	files, err := workspace.SemanticFiles()
	if err != nil {
		t.Fatalf("SemanticFiles() error = %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("SemanticFiles() = %+v", files)
	}
	for index, file := range files {
		if file.Package == hir.DefaultPackage || file.Path == "" || file.Digest == ([32]byte{}) || file.LineMap == ([32]byte{}) || file.Kind != SemanticFileSource ||
			index > 0 && (files[index-1].Package > file.Package || files[index-1].Package == file.Package && files[index-1].Path >= file.Path) {
			t.Fatalf("SemanticFiles() file = %+v", file)
		}
		sourcePackage, found := workspace.Package(packagePaths[file.Package])
		if !found {
			t.Fatalf("SemanticFiles() package %d is not loaded", file.Package)
		}
		matched := false
		for _, sourceFile := range sourcePackage.Sources {
			if sourceFile.Path != file.Path {
				continue
			}
			digest, lineMap := packagecontent.ComputeFileIdentities(sourceFile.Bytes)
			if digest != file.Digest || lineMap != file.LineMap {
				t.Fatalf("SemanticFiles() identity = %+v, want %s / %s", file, digest, lineMap)
			}
			matched = true
		}
		if !matched {
			t.Fatalf("SemanticFiles() file %q has no source bytes", file.Path)
		}
	}
}

func TestBundledStandardLibraryIOReachesVerifiedCompiler(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), map[string]string{
		"src/main.fy": `module main;

import std::io as io;

@effects(console)
fn i32 main() {
    discard io::write("Fairy C says hi\n");
    discard io::write_bytes(b"\x00\xff");
    discard io::write_stderr("Fairy C warns too\n");
    discard io::write_stderr_bytes(b"\x80\x00");
    return 42;
}
`,
	})
	workspace, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	if len(input.Files) != 3 || len(input.Imports) != 1 {
		t.Fatalf("semantic input has %d files and %d imports", len(input.Files), len(input.Imports))
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified {
		t.Fatalf("std::io workspace did not reach verified MIR: phase=%s failed=%s diagnostics=%v unsupported=%v", result.Semantic.CompletedThrough, result.Semantic.FailedPhase, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
}

func TestBundledStandardLibraryIORequiresConsoleEffect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		function string
		argument string
	}{
		{function: "write", argument: `"Fairy C says hi\n"`},
		{function: "write_bytes", argument: `b"\x00\xff"`},
		{function: "write_stderr", argument: `"Fairy C warns too\n"`},
		{function: "write_stderr_bytes", argument: `b"\x80\x00"`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.function, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			source := strings.NewReplacer("FUNCTION", test.function, "ARGUMENT", test.argument).Replace(`module main;

import std::io as io;

fn i32 main() {
    discard io::FUNCTION(ARGUMENT);
    return 0;
}
`)
			writePackage(t, root, ".", rootManifest(""), map[string]string{
				"src/main.fy": source,
			})
			workspace, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			input, err := workspace.SemanticInput()
			if err != nil {
				t.Fatal(err)
			}
			result, err := compiler.CheckPackages(input)
			if err != nil {
				t.Fatal(err)
			}
			if result.Verified || len(result.Semantic.Diagnostics) != 1 || result.Semantic.Diagnostics[0].Code != "E7102" ||
				len(result.Semantic.Diagnostics[0].Related) != 1 || result.Semantic.Diagnostics[0].Related[0].Role != "effect-declaration" {
				t.Fatalf("missing-console-effect result = %+v", result.Semantic)
			}
		})
	}
}
