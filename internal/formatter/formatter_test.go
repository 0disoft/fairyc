package formatter

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestFormatIsIdempotentAndPreservesComments(t *testing.T) {
	input := []byte("module main;  \r\n/// keep me\r\nfn i32 main() {  \r\n    // exact comment\r\n    return 0;   \r\n}\r\n")
	limits := source.DefaultLimits()
	first, err := Format("main.fy", input, limits)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("module main;\n/// keep me\nfn i32 main() {\n    // exact comment\n    return 0;\n}\n")
	if !bytes.Equal(first.Bytes, want) {
		t.Fatalf("formatted = %q, want %q", first.Bytes, want)
	}
	if !bytes.Contains(first.Bytes, []byte("/// keep me")) || !bytes.Contains(first.Bytes, []byte("// exact comment")) {
		t.Fatal("formatter lost comments")
	}
	second, err := Format("main.fy", first.Bytes, limits)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(second.Bytes, first.Bytes) {
		t.Fatalf("second format changed bytes: %q", second.Bytes)
	}
	if second.Tree.Identity() != first.Tree.Identity() {
		t.Fatal("idempotent formatting changed green identity")
	}
}

func TestFormatPreservesParserShape(t *testing.T) {
	input := []byte("module main;\r\nfn i32 add(i32 left, i32 right) { return left + right; }\r\nfn i32 main() { return add(20, 22) - 42; }\r\n")
	limits := source.DefaultLimits()
	before, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", input), limits)
	if err != nil || !before.Tree.ErrorFree() {
		t.Fatalf("before parse = (%v, %+v)", err, before.Tree.Diagnostics)
	}
	formatted, err := Format("main.fy", input, limits)
	if err != nil {
		t.Fatal(err)
	}
	after, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", formatted.Bytes), limits)
	if err != nil || !after.Tree.ErrorFree() {
		t.Fatalf("after parse = (%v, %+v)", err, after.Tree.Diagnostics)
	}
	if syntaxShape(before.Tree) != syntaxShape(after.Tree) {
		t.Fatal("formatted parse shape differs")
	}
}

func TestFormatTrimsFinalWhitespaceBeforeAddingNewline(t *testing.T) {
	result, err := Format("main.fy", []byte("module main;   "), source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("module main;\n"); !bytes.Equal(result.Bytes, want) {
		t.Fatalf("formatted = %q, want %q", result.Bytes, want)
	}
}

func TestFormatMalformedSourceFailsClosed(t *testing.T) {
	input := []byte("module main;\nfn i32 main( { return 0; }   \n")
	result, err := Format("main.fy", input, source.DefaultLimits())
	if !errors.Is(err, ErrMalformedSource) {
		t.Fatalf("error = %v", err)
	}
	if !result.Malformed || !bytes.Equal(result.Bytes, input) {
		t.Fatalf("malformed result = %+v", result)
	}
	if result.Tree.TokenCount() == 0 || result.Tree.NodeCount() == 0 {
		t.Fatal("malformed source lost green-tree facts")
	}
}

func TestGreenTreeIsDetachedAndTriviaSensitive(t *testing.T) {
	limits := source.DefaultLimits()
	firstSyntax, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte("module main;\nfn i32 main() { return 0; }\n")), limits)
	if err != nil {
		t.Fatal(err)
	}
	secondSyntax, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte("module main;\nfn i32 main() {  return 0; }\n")), limits)
	if err != nil {
		t.Fatal(err)
	}
	first, err := BuildGreen(firstSyntax.Tree)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildGreen(secondSyntax.Tree)
	if err != nil {
		t.Fatal(err)
	}
	if first.ShapeIdentity() != second.ShapeIdentity() {
		t.Fatal("trivia-only edit changed parser shape")
	}
	if first.Identity() == second.Identity() || first.SourceIdentity() == second.SourceIdentity() {
		t.Fatal("trivia-only edit did not change exact green/source identity")
	}
	tokens := first.Tokens()
	if len(tokens) == 0 {
		t.Fatal("missing green tokens")
	}
	tokens[0].Text = "mutated"
	again := first.Tokens()
	if again[0].Text == "mutated" {
		t.Fatal("GreenTree exposed mutable token storage")
	}
	root, ok := first.Node(first.Root())
	if !ok {
		t.Fatal("missing green root")
	}
	children := root.Children()
	if len(children) > 0 {
		children[0] = ^uint32(0)
		rootAgain, _ := first.Node(first.Root())
		if len(rootAgain.Children()) > 0 && rootAgain.Children()[0] == ^uint32(0) {
			t.Fatal("GreenTree exposed mutable child storage")
		}
	}
}

func TestFormatLargeFileWithinDefaultBudget(t *testing.T) {
	var input bytes.Buffer
	input.WriteString("module main;\n")
	for index := 0; index < 256; index++ {
		fmt.Fprintf(&input, "fn i32 f%d() { return %d; }   \r\n", index, index)
	}
	input.WriteString("fn i32 main() { return 0; }\r\n")
	result, err := Format("main.fy", input.Bytes(), source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if result.Malformed || bytes.Contains(result.Bytes, []byte(" \n")) || bytes.Contains(result.Bytes, []byte("\r\n")) {
		t.Fatal("large-file formatting did not canonicalize whitespace")
	}
}

func BenchmarkFormatRepresentativeFile(b *testing.B) {
	var input bytes.Buffer
	input.WriteString("module main;\n")
	for index := 0; index < 128; index++ {
		fmt.Fprintf(&input, "fn i32 f%d(i32 value) { return value + %d; }\n", index, index)
	}
	input.WriteString("fn i32 main() { return 0; }\n")
	data := input.Bytes()
	limits := source.DefaultLimits()
	b.ReportMetric(float64(len(data)), "source-bytes")
	b.ResetTimer()
	for range b.N {
		if _, err := Format("main.fy", data, limits); err != nil {
			b.Fatal(err)
		}
	}
}
