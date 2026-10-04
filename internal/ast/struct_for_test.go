package ast

import (
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
	"testing"
)

func TestStructForRetainsWrittenTypeAndBody(t *testing.T) {
	for _, typ := range []string{"Token", "tokens::Token"} {
		input := []byte("module main; fn void visit(ref " + typ + "[] values){for(let " + typ + " item in values){continue;}}")
		tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
		module, err := Lower(tree)
		if err != nil || len(tree.Diagnostics) != 0 {
			t.Fatalf("parse: %v %+v", err, tree.Diagnostics)
		}
		statement := module.Functions[0].Body.Statements[0]
		loop := statement.ByteFor
		if loop == nil || loop.Type.Path != typ || string(input[loop.NameSpan.StartByte:loop.NameSpan.EndByte]) != "item" {
			t.Fatal("written loop identity lost")
		}
		if statement.Expression == nil || statement.Expression.Block == nil {
			t.Fatal("missing indexed-loop preparation")
		}
		prepared := statement.Expression.Block
		if len(prepared.Statements) != 2 || len(loop.Body.Statements) != 1 || len(prepared.Statements[1].For.Body.Statements) != 2 || prepared.Statements[1].For.Body.Statements[0].Binding.Type.Path != typ {
			t.Fatal("item type or source body changed")
		}
	}
}

func TestMutableStructForRetainsWrittenReference(t *testing.T) {
	input := []byte("module main;fn void edit(ref mut Token[] values){for(let ref mut Token item in values){item.value=7;}}")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	module, err := Lower(tree)
	if err != nil || len(tree.Diagnostics) != 0 {
		t.Fatalf("parse: %v %+v", err, tree.Diagnostics)
	}
	statement := module.Functions[0].Body.Statements[0]
	loop := statement.ByteFor
	if loop == nil || loop.Type.Kind != TypeKindReference || !loop.Type.Mutable || loop.Type.Element.Path != "Token" || len(loop.Body.Statements) != 1 {
		t.Fatal("written mutable loop lost")
	}
	item := statement.Expression.Block.Statements[1].For.Body.Statements[0].Binding
	if item.Type.Kind != TypeKindReference || !item.Type.Mutable || item.Expression.Kind != syntax.KindPrefix || item.Expression.Text != "&mut" || item.Expression.Children[0].Kind != syntax.KindIndex {
		t.Fatal("missing generated element reference")
	}
	if len(loop.Body.Statements) != 1 || loop.Body.Statements[0].Kind != syntax.KindAssignment {
		t.Fatal("rewrote source body")
	}
}
