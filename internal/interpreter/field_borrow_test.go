package interpreter_test

import (
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
