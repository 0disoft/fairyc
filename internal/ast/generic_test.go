package ast

import (
	"slices"
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

func TestGenericAggregateSyntaxRetainsParametersFieldsAndCleanup(t *testing.T) {
	text := `module main;
/// Retained template.
pub struct Pair<T, U,> { T first; U second; }
pub resource Holder<T> { T value; cleanup { let i32 marker = 1; } }
struct Plain { i32 value; }
resource Ordinary { i32 value; }
`
	tree := syntax.Parse(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	module, err := Lower(tree)
	if err != nil || len(tree.Diagnostics) != 0 {
		t.Fatalf("lower: %v diagnostics=%+v", err, tree.Diagnostics)
	}
	pair := module.Structs[0]
	if !pair.Public || !slices.Equal(pair.TypeParameters, []string{"T", "U"}) || len(pair.Fields) != 2 || pair.Fields[0].Type.Path != "T" || pair.Fields[1].Type.Path != "U" {
		t.Fatalf("struct template lost: %+v", pair)
	}
	if pair.DocumentationSpan.EndByte <= pair.DocumentationSpan.StartByte || text[pair.NameSpan.StartByte:pair.NameSpan.EndByte] != "Pair" {
		t.Fatal("template source provenance lost")
	}
	holder := module.Resources[0]
	if !holder.Public || !slices.Equal(holder.TypeParameters, []string{"T"}) || len(holder.Fields) != 1 || holder.Fields[0].Type.Path != "T" || holder.Cleanup == nil || len(holder.Cleanup.Body.Statements) != 1 {
		t.Fatalf("resource template lost: %+v", holder)
	}
	if module.Structs[1].TypeParameters != nil || module.Resources[1].TypeParameters != nil {
		t.Fatal("ordinary aggregate gained parameters")
	}
}

func TestMalformedAggregateTypeParametersAreRejected(t *testing.T) {
	for _, declaration := range []string{
		"struct Bad<> { i32 value; }",
		"resource Bad<> { i32 value; }",
		"struct Bad<T U> { i32 value; }",
		"resource Bad<T { i32 value; }",
		"struct Bad<T,",
	} {
		t.Run(declaration, func(t *testing.T) {
			tree := syntax.Parse(source.NewFile("main.fy", []byte("module main; "+declaration)), source.DefaultLimits())
			if len(tree.Diagnostics) == 0 {
				t.Fatal("malformed parameter list accepted")
			}
		})
	}
}
