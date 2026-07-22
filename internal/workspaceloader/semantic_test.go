package workspaceloader

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
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
}
