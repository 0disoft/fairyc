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
