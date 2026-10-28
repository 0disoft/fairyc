package interpreter_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestExecuteSharedFieldBorrow(t *testing.T) {
	program := lowerSource(t, `module main;
struct Inner { i32 number; }
resource Owner { Inner inner; cleanup {} }
fn ref i32 number(ref Owner value) { return ref value.inner.number; }
fn i32 main() {
    let Owner owner = Owner { .inner = Inner { .number = 42 } };
    let ref i32 direct = ref owner.inner.number;
    assert(*direct == 42);
    return *number(ref owner);
}`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("field borrow: %+v %v", outcome, err)
	}
}

func TestExecuteWideAggregateFieldBorrow(t *testing.T) {
	for _, resource := range []bool{false, true} {
		t.Run(fmt.Sprintf("resource_%t", resource), func(t *testing.T) {
			var fields, initializers strings.Builder
			for i := range 17 {
				fmt.Fprintf(&fields, "i32 pad%d;", i)
				fmt.Fprintf(&initializers, ".pad%d = %d,", i, i)
			}
			kind, cleanup := "struct", ""
			if resource {
				kind, cleanup = "resource", "cleanup {}"
			}
			text := fmt.Sprintf(`module main;
%s Record { %s i32 value; %s }
fn ref i32 last(ref Record owner) { return ref owner.value; }
fn i32 main() {
    let Record owner = Record { %s .value = 42 };
    return *last(ref owner);
}`, kind, fields.String(), cleanup, initializers.String())
			outcome, err := interpreter.Execute(lowerSource(t, text), "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
				t.Fatalf("wide field borrow: %+v, %v", outcome, err)
			}
		})
	}
}
