package cbackend_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestFieldShorthandEmitsIdenticalC(t *testing.T) {
	for _, input := range []string{
		`module main; struct Pair { i32 left; i32 right; }
fn i32 main() { let i32 left = 40; let i32 right = 2; let Pair pair = Pair { .left = left, .right = right }; return pair.left + pair.right; }`,
		`module main; resource Leaf { i32 number; cleanup {} } resource Owner { Leaf value; cleanup {} }
fn i32 main() { let Leaf value = Leaf { .number = 42 }; let Owner owner = Owner { .value = value }; return 42; }`,
		`module main; enum Choice { Pair { i32 left; i32 right; }; }
fn i32 main() { let i32 left = 40; let i32 right = 2; let Choice item = .Pair { .left = left, .right = right }; return switch (item) { case .Pair { .left = a, .right = b } : { a + b } }; }`,
	} {
		short := strings.NewReplacer(".left = left", ".left", ".right = right", ".right", ".value = value", ".value").Replace(input)
		fullUnit, err := cbackend.Emit(lowerSource(t, input), "main::main", cbackend.Stage0Profile())
		if err != nil {
			t.Fatal(err)
		}
		shortUnit, err := cbackend.Emit(lowerSource(t, short), "main::main", cbackend.Stage0Profile())
		if err != nil {
			t.Fatal(err)
		}
		if fullUnit.Source != shortUnit.Source {
			t.Fatalf("shorthand changed executable C:\nFULL:\n%s\nSHORT:\n%s", fullUnit.Source, shortUnit.Source)
		}
	}
}
