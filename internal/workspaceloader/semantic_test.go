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
	result, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified {
		t.Fatalf("workspace did not reach verified MIR: phase=%s failed=%s diagnostics=%v unsupported=%v", result.Semantic.CompletedThrough, result.Semantic.FailedPhase, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
}
