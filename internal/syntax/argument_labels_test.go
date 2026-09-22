package syntax

import (
	"github.com/0disoft/fairyc/internal/source"
	"testing"
)

func TestArgumentLabelsSyntax(t *testing.T) {
	for _, call := range []string{"f(.first = 1, 2, .third = .some(3),)", "f(.value /* label */ = g(2))"} {
		file := source.NewFile("main.fy", []byte("module main; fn i32 main() { return "+call+"; }"))
		tree := Parse(file, source.DefaultLimits())
		if len(tree.Diagnostics) != 0 {
			t.Fatal(tree.Diagnostics)
		}
		if text, ok := tree.Reconstruct(); !ok || string(text) != string(file.Bytes) {
			t.Fatal("lossless reconstruction changed")
		}
		if len(ParsePreviousRevision(file, source.DefaultLimits()).Diagnostics) == 0 {
			t.Fatal("legacy admitted labels")
		}
	}
	for _, call := range []string{"f(.first =)", "f(. = 1)", "f(.first 1)", "f(.first = 1 .second = 2)"} {
		file := source.NewFile("main.fy", []byte("module main; fn i32 main() { return "+call+"; }"))
		if len(Parse(file, source.DefaultLimits()).Diagnostics) == 0 {
			t.Fatalf("accepted malformed %s", call)
		}
	}
}
