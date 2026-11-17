package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestExecuteDerefSnapshotsBeforeLaterArgumentMutation(t *testing.T) {
	for _, shape := range []struct{ name, declaration, initializer, field string }{
		{"flat", "struct State { i32 count; }", "State { .count = 7 }", "count"},
		{"nested", "struct Leaf { i32 count; } struct State { Leaf child; }", "State { .child = Leaf { .count = 7 } }", "child.count"},
	} {
		t.Run(shape.name, func(t *testing.T) {
			program := lowerSource(t, fmt.Sprintf(`module main;
%s
fn i32 change(ref mut State target) { target.%s = 42; return 0; }
fn i32 before(State snapshot, i32 ignored) { return snapshot.%s; }
fn i32 main() {
	var State owner = %s;
	let ref mut State target = ref mut owner;
	return before(*target, change(target));
}`, shape.declaration, shape.field, shape.field, shape.initializer))
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 7 {
				t.Fatalf("dereference snapshot: %+v, %v; want return 7", outcome, err)
			}
		})
	}
}
