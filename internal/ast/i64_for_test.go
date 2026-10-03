package ast

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestI64ForRetainsWrittenTypeAndSpan(t *testing.T) {
	input := []byte("module main; fn void sum(ref i64[] view) { for (let i64 value in view) { assert(value != 0); } }")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	module, err := Lower(tree)
	if err != nil || len(tree.Diagnostics) != 0 || len(module.Functions) != 1 {
		t.Fatalf("parse/lower: %v %+v", err, tree.Diagnostics)
	}
	statement := module.Functions[0].Body.Statements[0]
	if statement.ByteFor == nil || statement.ByteFor.Type.Path != "i64" || string(input[statement.ByteFor.NameSpan.StartByte:statement.ByteFor.NameSpan.EndByte]) != "value" {
		t.Fatal("written i64 loop lost source identity")
	}
	prepared := statement.Expression
	if prepared == nil || prepared.Block == nil || len(prepared.Block.Statements) != 2 {
		t.Fatal("missing scalar-loop preparation")
	}
	body := prepared.Block.Statements[1].For.Body
	if len(body.Statements) != 2 || body.Statements[0].Binding.Type.Path != "i64" || len(statement.ByteFor.Body.Statements) != 1 {
		t.Fatal("item type changed or written body mutated")
	}
}
