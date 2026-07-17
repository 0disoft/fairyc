package syntax

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestParseBudgetFailsClosed(t *testing.T) {
	t.Parallel()
	input := []byte("module main;")
	limits := source.DefaultLimits()
	limits.MaxBytes = len(input) - 1
	tree := Parse(source.NewFile("main.fy", input), limits)
	if len(tree.Diagnostics) != 1 || tree.Diagnostics[0].Code != "E0010" {
		t.Fatalf("Parse() diagnostics = %+v, want one E0010", tree.Diagnostics)
	}
}

func TestMissingSemicolonRecoversWithoutConsumingNextStatement(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> i32 {\n  let value = 1\n  return value;\n}\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 1 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	diagnostic := tree.Diagnostics[0]
	if diagnostic.Code != "E0007" || diagnostic.Repair != "insert" || diagnostic.Token != ";" {
		t.Fatalf("Parse() diagnostic = %+v", diagnostic)
	}
	position := len("module main;\nfn main() -> i32 {\n  let value = 1")
	if diagnostic.Span.StartByte != position || diagnostic.Span.EndByte != position {
		t.Fatalf("Parse() span = %+v, want zero-width at %d", diagnostic.Span, position)
	}
}
