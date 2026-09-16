package syntax

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestFieldShorthandSyntax(t *testing.T) {
	for _, constructor := range []string{"Pair", "remote::Pair", "Pair<i32>", ".Pair"} {
		t.Run(constructor, func(t *testing.T) {
			input := "module main; fn i32 main() { let Pair p = " + constructor + " { .left /* keep */, .right = 2 }; return 0; }"
			file := source.NewFile("main.fy", []byte(input))
			tree := Parse(file, source.DefaultLimits())
			if len(tree.Diagnostics) != 0 {
				t.Fatalf("diagnostics: %+v", tree.Diagnostics)
			}
			found := false
			for _, node := range tree.Nodes {
				if node.Kind != KindField || node.Text != "left" {
					continue
				}
				if len(node.Children) != 1 {
					t.Fatal("field lost value")
				}
				value, ok := tree.Node(node.Children[0])
				if !ok || value.Kind != KindPath || value.Text != "left" || input[value.Span.StartByte:value.Span.EndByte] != "left" {
					t.Fatalf("implicit value must point to written identifier: %+v", value)
				}
				found = true
			}
			if !found {
				t.Fatal("missing shorthand")
			}
			if len(ParsePreviousRevision(file, source.DefaultLimits()).Diagnostics) == 0 {
				t.Fatal("legacy parser admitted shorthand")
			}
		})
	}
	for _, field := range []string{".left 1", ".left + 1", ".left =", ".", ".left .right"} {
		input := "module main; fn i32 main() { let Pair p = Pair { " + field + " }; return 0; }"
		if len(Parse(source.NewFile("main.fy", []byte(input)), source.DefaultLimits()).Diagnostics) == 0 {
			t.Fatalf("accepted malformed %s", field)
		}
	}
}

func BenchmarkFieldShorthandParse(b *testing.B) {
	for _, shorthand := range []bool{false, true} {
		name, field := "explicit", ".value = value"
		if shorthand {
			name, field = "shorthand", ".value"
		}
		input := "module main; struct Box { i32 value; } fn i32 main() { let i32 value = 42;" + strings.Repeat("{ let Box box = Box { "+field+" }; }", 100) + "return 0; }"
		file := source.NewFile("main.fy", []byte(input))
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if tree := Parse(file, source.DefaultLimits()); len(tree.Diagnostics) != 0 {
					b.Fatal(tree.Diagnostics)
				}
			}
		})
	}
}
