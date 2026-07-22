package workspaceloader

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
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
	if len(input.Files) != 2 || len(input.Imports) != 1 {
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
	for _, loaded := range packages {
		if loaded.Instance != identities[loaded.ID] || loaded.Coordinate != coordinates[loaded.ID] || loaded.Source == ([32]byte{}) || loaded.Path == "" {
			t.Fatalf("SemanticPackages() package = %+v, identities = %+v, coordinates = %+v", loaded, identities, coordinates)
		}
	}
}
