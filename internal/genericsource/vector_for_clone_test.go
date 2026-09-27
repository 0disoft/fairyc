package genericsource

import (
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestVectorForCandidateCloneSubstitutesWithoutPlanningCalls(t *testing.T) {
	input := []byte("module main; import std::vec as items; " +
		"fn void visit<T>(ref items::Vec<T> values) { for (let ref T item in values) {} }")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("parser diagnostics = %+v", tree.Diagnostics)
	}
	module, err := ast.Lower(tree)
	if err != nil || len(module.Functions) != 1 {
		t.Fatalf("source AST = %+v, %v", module, err)
	}
	ast.PrepareSDKVectorLoopBody(module.Functions[0].Body, "items")
	original := module.Functions[0].Body.Statements[0].ByteFor
	if original == nil || original.PreparedVector == nil {
		t.Fatalf("missing original candidate = %+v", original)
	}
	requests := 0
	clone, err := (transform{
		bindings: map[string]ast.TypeReference{"T": ast.NamedTypeReference("i32", original.NameSpan)},
		expression: func(expression *ast.Expression) error {
			if expression.Kind == syntax.KindSpecialize {
				requests++
			}
			return nil
		},
	}).function(module.Functions[0])
	if err != nil {
		t.Fatal(err)
	}
	loop := clone.Body.Statements[0].ByteFor
	if loop == nil || loop == original || loop.View == original.View || loop.Body == original.Body ||
		loop.PreparedVector == original.PreparedVector || requests != 0 ||
		loop.PreparedVector.Block.Statements[0].Binding.Expression != loop.View {
		t.Fatalf("source was shared or candidate planned: loop=%+v requests=%d", loop, requests)
	}
	if loop.Type.Element == nil || loop.Type.Element.Path != "i32" ||
		original.Type.Element == nil || original.Type.Element.Path != "T" {
		t.Fatalf("item substitution changed source: cloned=%+v original=%+v", loop.Type, original.Type)
	}
	owner := loop.PreparedVector.Block.Statements[0].Binding
	bound := loop.PreparedVector.Block.Statements[1].Binding
	item := loop.PreparedVector.Block.Statements[2].For.Body.Statements[0].Binding
	if owner == nil || owner.Type == nil || owner.Type.Element == nil ||
		len(owner.Type.Element.Arguments) != 1 || owner.Type.Element.Arguments[0].Path != "i32" ||
		bound == nil || bound.Expression == nil || item == nil || item.Expression == nil ||
		bound.Expression.Children[0].TypeArguments[0].Path != "i32" ||
		item.Expression.Children[0].TypeArguments[0].Path != "i32" {
		t.Fatalf("candidate types were not substituted: owner=%+v bound=%+v item=%+v", owner, bound, item)
	}
	if original.PreparedVector.Block.Statements[1].Binding.Expression.Children[0].TypeArguments[0].Path != "T" {
		t.Fatal("original candidate was mutated")
	}
}

func TestVectorForCandidateCloneRejectsDetachedOwner(t *testing.T) {
	input := []byte("module main; import std::vec as items; " +
		"fn void visit<T>(ref items::Vec<T> values) { for (let ref T item in values) {} }")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	module, err := ast.Lower(tree)
	if err != nil || len(tree.Diagnostics) != 0 {
		t.Fatalf("source AST = %+v, %v", tree.Diagnostics, err)
	}
	ast.PrepareSDKVectorLoopBody(module.Functions[0].Body, "items")
	loop := module.Functions[0].Body.Statements[0].ByteFor
	loop.PreparedVector.Block.Statements[0].Binding.Expression = &ast.Expression{
		Kind: syntax.KindPath, Text: "other", Span: loop.View.Span,
	}
	if _, err := (transform{}).function(module.Functions[0]); err == nil {
		t.Fatal("detached candidate owner was accepted")
	}
}
