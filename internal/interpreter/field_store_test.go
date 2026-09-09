package interpreter_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestExecuteFieldStoreCopiesAndMutableReceiver(t *testing.T) {
	program := lowerSource(t, `module main;
struct State { i32 count; usize size; bool ready; }
resource Holder { State state; cleanup {} }
fn void State::set(ref mut State this) { this.count = 42; this.size = 9; this.ready = true; }
fn i32 main() {
    var State original = State { .count = 1, .size = 2, .ready = false };
    var State copy = original;
    copy.count = 5;
    assert(original.count == 1);
    copy.set();
    assert(copy.count == 42);
    assert(copy.size == 9);
    assert(copy.ready);
    assert(original.count == 1);
    assert(original.size == 2);
    assert(!original.ready);
    assert(copy.count == 42 && copy.size == 9 && copy.ready && !original.ready);
    let Holder holder = Holder { .state = original };
    original.count = 8;
    assert(holder.state.count == 1);
    let ref i32 view = ref copy.count;
    assert(*view == 42);
    copy.count = 43;
    return copy.count;
}`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 43 {
		t.Fatalf("field store: %+v %v", outcome, err)
	}
}
