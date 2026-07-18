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

func TestLowerPreservesFunctionSemanticStructure(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn choose(flag: bool) -> i32 {\n  let value: i32 = 1;\n  return value;\n}\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Functions) != 1 {
		t.Fatalf("functions = %+v", module.Functions)
	}
	function := module.Functions[0]
	if function.Name != "choose" || function.ResultType.Text != "i32" || len(function.Parameters) != 1 || function.Body == nil {
		t.Fatalf("function = %+v", function)
	}
	if function.Parameters[0].Name != "flag" || function.Parameters[0].Type.Text != "bool" {
		t.Fatalf("parameters = %+v", function.Parameters)
	}
	if len(function.Body.Statements) != 2 || function.Body.Statements[0].Binding == nil || function.Body.Statements[1].Kind != syntax.KindReturn {
		t.Fatalf("body = %+v", function.Body)
	}
}

func TestLowerCapturesResourceFields(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nresource Pair { left: i32; right: i32; drop fn(self: &mut Self) {} }\n")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	if len(module.Resources) != 1 || module.Resources[0].Name != "Pair" || len(module.Resources[0].Fields) != 2 {
		t.Fatalf("resources = %+v", module.Resources)
	}
	if module.Resources[0].Fields[0].Name != "left" || module.Resources[0].Fields[0].Type.Text != "i32" {
		t.Fatalf("first field = %+v", module.Resources[0].Fields[0])
	}
}
