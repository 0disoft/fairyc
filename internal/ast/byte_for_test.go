package ast

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestLowerRetainsByteViewForAndPreparesScalarLoop(t *testing.T) {
	input := []byte("module main; fn void show(ref u8[] view) { for (let u8 byte in view) { assert(byte == 7); } }")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("parser diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil || len(module.Functions) != 1 || module.Functions[0].Body == nil {
		t.Fatalf("lowered module = %+v, %v", module, err)
	}
	statements := module.Functions[0].Body.Statements
	if len(statements) != 1 || statements[0].ByteFor == nil || statements[0].For != nil {
		t.Fatalf("statements = %+v", statements)
	}
	loop := statements[0].ByteFor
	if loop.Name != "byte" || loop.Type.Path != "u8" || loop.View == nil || loop.View.Kind != syntax.KindPath || loop.View.Text != "view" || loop.Body == nil || len(loop.Body.Statements) != 1 {
		t.Fatalf("byte-view loop = %+v", loop)
	}
	if got := string(input[loop.NameSpan.StartByte:loop.NameSpan.EndByte]); got != "byte" {
		t.Fatalf("name span = %q", got)
	}
	prepared := statements[0].Expression
	if prepared == nil || prepared.Kind != syntax.KindBlock || prepared.Block == nil || len(prepared.Block.Statements) != 2 {
		t.Fatalf("prepared block = %+v", prepared)
	}
	bound := prepared.Block.Statements[0].Binding
	classic := prepared.Block.Statements[1].For
	if bound == nil || bound.Name != byteForBoundName || classic == nil || classic.Initializer == nil || classic.Initializer.Binding == nil || classic.Initializer.Binding.Name != byteForIndexName || classic.Post == nil || classic.Post.Operator != "+=" {
		t.Fatalf("prepared bound/loop = %+v / %+v", bound, classic)
	}
	if len(classic.Body.Statements) != 2 || classic.Body.Statements[0].Binding == nil || classic.Body.Statements[0].Binding.Name != "byte" || len(loop.Body.Statements) != 1 {
		t.Fatalf("prepared body modified source body: %+v / %+v", classic.Body, loop.Body)
	}
}
