package cbackend_test

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"strings"
	"testing"
)

func TestArgumentLabelsIdenticalC(t *testing.T) {
	for _, input := range []string{
		`module main; fn usize expression(usize minimum, usize depth, bool allow_record) { return if (allow_record) { minimum + depth } else { depth }; } fn i32 main() { assert(expression(1, 0, true) == 1); return 0; }`,
		`module main; struct Counter { i32 value; } fn i32 bump(ref mut Counter n) { n.value = n.value + 1; return n.value; } fn i32 pair(i32 left, i32 right) { return left * 10 + right; } fn i32 main() { var Counter n = Counter { .value = 0 }; return pair(bump(ref mut n), bump(ref mut n)); }`,
		`module main; resource Item { i32 number; cleanup {} } fn i32 take(Item value) { return value.number; } fn i32 main() { let Item item = Item { .number = 42 }; return take(item); }`,
	} {
		labeled := strings.NewReplacer("expression(1, 0, true)", "expression(.minimum = 1, .depth = 0, .allow_record = true)", "pair(bump(ref mut n), bump(ref mut n))", "pair(.left = bump(ref mut n), .right = bump(ref mut n))", "take(item)", "take(.value = item)").Replace(input)
		base, err := cbackend.Emit(lowerSource(t, input), "main::main", cbackend.Stage0Profile())
		if err != nil {
			t.Fatal(err)
		}
		got, err := cbackend.Emit(lowerSource(t, labeled), "main::main", cbackend.Stage0Profile())
		if err != nil {
			t.Fatal(err)
		}
		if got.Source != base.Source {
			t.Fatal("labels changed generated C")
		}
	}
}
