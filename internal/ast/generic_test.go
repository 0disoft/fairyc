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

func TestGenericRecordRetainsTypesSeparatelyFromFields(t *testing.T) {
	text := `module main;
fn i32 main() {
    let remote::Holder<Result<i32, u8>> nested = remote::Holder<Result<i32, u8>> { .value = identity<i32>(42) };
    let Pair<i32, bool> pair = Pair<i32, bool,> { .first = 42, .second = true, };
    let Plain plain = Plain { .value = 7 };
    return 0;
}
fn bool less(i32 a, i32 b) { return a < b; }
fn bool greater(i32 a, i32 b) { return a > b; }
`
	tree := syntax.Parse(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	module, err := Lower(tree)
	if err != nil || len(tree.Diagnostics) != 0 {
		t.Fatalf("lower: %v diagnostics=%+v", err, tree.Diagnostics)
	}
	nested := module.Functions[0].Body.Statements[0].Binding.Expression
	if nested.Kind != syntax.KindRecord || len(nested.Children) != 2 || nested.Children[0].Text != "remote::Holder" || nested.Children[1].Text != "value" {
		t.Fatalf("record value children changed: %+v", nested)
	}
	if len(nested.TypeArguments) != 1 || nested.TypeArguments[0].Path != "Result" || len(nested.TypeArguments[0].Arguments) != 2 || nested.TypeArguments[0].Arguments[0].Path != "i32" || nested.TypeArguments[0].Arguments[1].Path != "u8" {
		t.Fatalf("nested type arguments lost: %+v", nested.TypeArguments)
	}
	argument := nested.TypeArguments[0]
	if text[argument.Span.StartByte:argument.Span.EndByte] != "Result<i32, u8>" {
		t.Fatal("type argument source span changed")
	}
	call := nested.Children[1].Children[0]
	if call.Kind != syntax.KindCall || call.Children[0].Kind != syntax.KindSpecialize || call.Children[0].TypeArguments[0].Path != "i32" {
		t.Fatal("generic call inside record changed")
	}
	pair := module.Functions[0].Body.Statements[1].Binding.Expression
	if len(pair.TypeArguments) != 2 || pair.TypeArguments[0].Path != "i32" || pair.TypeArguments[1].Path != "bool" || len(pair.Children) != 3 {
		t.Fatalf("ordered record arguments or fields lost: %+v", pair)
	}
	plain := module.Functions[0].Body.Statements[2].Binding.Expression
	if plain.Kind != syntax.KindRecord || plain.TypeArguments != nil || len(plain.Children) != 2 {
		t.Fatal("ordinary record changed")
	}
	for _, fn := range module.Functions[1:] {
		if fn.Body.Statements[0].Expression.Kind != syntax.KindBinary {
			t.Fatalf("comparison changed: %s", fn.Name)
		}
	}
}
