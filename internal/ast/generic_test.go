package ast

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestGenericSyntaxRetainsTypeParametersAndArguments(t *testing.T) {
	text := `module main;
fn T identity<T>(T value) { return value; }
fn i32 main() { return identity<i32>(42); }
fn bool less(i32 a, i32 b) { return a < b; }
`
	tree := syntax.Parse(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	module, err := Lower(tree)
	if err != nil {
		t.Fatalf("lower: %v diagnostics=%+v", err, tree.Diagnostics)
	}
	if len(module.Functions[0].TypeParameters) != 1 || module.Functions[0].TypeParameters[0] != "T" {
		t.Fatal("type parameters lost")
	}
	call := module.Functions[1].Body.Statements[0].Expression
	if call.Kind != syntax.KindCall || call.Children[0].Kind != syntax.KindSpecialize || len(call.Children[0].TypeArguments) != 1 || call.Children[0].TypeArguments[0].Path != "i32" {
		t.Fatal("type arguments lost")
	}
	if module.Functions[2].Body.Statements[0].Expression.Kind != syntax.KindBinary {
		t.Fatal("ordinary comparison changed")
	}
}
