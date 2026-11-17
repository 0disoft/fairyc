package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestExecuteFieldReadSnapshotsBeforeLaterArgumentMutation(t *testing.T) {
	for _, shape := range []struct{ name, declaration, initializer, field string }{
		{"flat", "struct Value { i32 count; }", "Value { .count = 7 }", "count"},
		{"nested", "struct Leaf { i32 count; } struct Value { Leaf child; }", "Value { .child = Leaf { .count = 7 } }", "child.count"},
	} {
		t.Run(shape.name, func(t *testing.T) {
			program := lowerSource(t, fmt.Sprintf(`module main;
%s
struct Owner { Value value; }
fn i32 change(ref mut Owner target) { target.value.%s = 42; return 0; }
fn i32 before(Value snapshot, i32 ignored) { return snapshot.%s; }
fn i32 main() {
    var Owner owner = Owner { .value = %s };
    let ref mut Owner target = ref mut owner;
    let i32 snapshot = before(target.value, change(target));
    assert(target.value.%s == 42);
    return snapshot;
}`, shape.declaration, shape.field, shape.field, shape.initializer, shape.field))
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 7 {
				t.Fatalf("field snapshot: %+v, %v; want return 7", outcome, err)
			}
		})
	}
}
