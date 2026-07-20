package syntax

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/lexer"
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

func TestNestedTypeCloseKeepsShiftTokenLossless(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn wrap(value: Token) -> Option<Option<Token>> { value }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
	foundShiftToken := false
	for _, token := range tree.Tokens {
		if token.Text(tree.Source) == ">>" {
			foundShiftToken = true
			break
		}
	}
	if !foundShiftToken {
		t.Fatal("Parse() did not preserve the original >> token")
	}
	outer, inner := false, false
	for _, node := range tree.Nodes {
		if node.Kind != KindType {
			continue
		}
		switch node.Text {
		case "Option<Option<Token>>":
			outer = true
		case "Option<Token>":
			inner = true
		}
	}
	if !outer || !inner {
		t.Fatalf("Parse() nested type nodes = outer %v, inner %v", outer, inner)
	}
}

func TestNestedTypeCloseChargesContextualTokenBudget(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn wrap(value: Token) -> Option<Option<Token>> { value }\n")
	file := source.NewFile("main.fy", input)
	baseline := source.DefaultLimits()
	lexed := lexer.Lex(file, source.NewBudget(baseline, len(input)))
	if len(lexed.Diagnostics) != 0 {
		t.Fatalf("Lex() diagnostics = %+v", lexed.Diagnostics)
	}
	limits := source.DefaultLimits()
	limits.MaxTokens = len(lexed.Tokens) - 1
	tree := Parse(file, limits)
	if len(tree.Diagnostics) != 1 || tree.Diagnostics[0].Code != "E0010" {
		t.Fatalf("Parse() diagnostics = %+v, want one E0010", tree.Diagnostics)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
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

func TestParseImmutableByteViewSlice(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> &[u8] { b\"fairy\"[1..4] }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var slice *Node
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindSlice {
			slice = &tree.Nodes[index]
			break
		}
	}
	if slice == nil || slice.Text != ".." || len(slice.Children) != 3 {
		t.Fatalf("slice node = %+v", slice)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}
