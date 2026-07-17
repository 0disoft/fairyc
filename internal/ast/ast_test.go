package ast

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestLowerPreservesExplicitGrouping(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> i32 {\n  let value = (1 + 2) * 3;\n  value\n}\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Bindings) != 1 {
		t.Fatalf("bindings = %d, want 1", len(module.Bindings))
	}
	if got, want := NormalizeExpression(module.Bindings[0].Expression), "(* (group (+ 1 2)) 3)"; got != want {
		t.Fatalf("NormalizeExpression() = %q, want %q", got, want)
	}
}

func TestLowerRejectsDiagnosticTree(t *testing.T) {
	t.Parallel()
	input := []byte("module wrong;")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if _, err := Lower(tree); err == nil {
		t.Fatal("Lower() accepted diagnostic syntax tree")
	}
}
