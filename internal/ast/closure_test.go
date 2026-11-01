package ast_test

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
	"testing"
)

func TestCopiedClosureSyntaxRetainsSource(t *testing.T) {
	text := `module main;fn i32 main(){let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(2);}`
	tree := syntax.Parse(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if !tree.ErrorFree() {
		t.Fatal(tree.Diagnostics)
	}
	m, err := ast.Lower(tree)
	if err != nil {
		t.Fatal(err)
	}
	v := m.Functions[0].Body.Statements[1].Binding.Expression.Closure
	if v == nil || len(v.Captures) != 1 || v.Captures[0].Name != "offset" || len(v.Function.Parameters) != 1 || v.Function.Parameters[0].Name != "x" || v.Function.ResultType.Path != "i32" {
		t.Fatalf("lost capture/signature: %+v", v)
	}
	if text[v.Captures[0].NameSpan.StartByte:v.Captures[0].NameSpan.EndByte] != "offset" {
		t.Fatal("capture source span")
	}
}
