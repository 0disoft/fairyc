package ast

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestLowerRetainsByteViewForWithoutSyntheticCounter(t *testing.T) {
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
}
