package interpreter_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestExecuteFieldStoreScalarDeref(t *testing.T) {
	program := lowerSource(t, `module main;
fn void increment(ref mut i32 slot) { *slot = *slot + 1; }
fn i32 main() {
    var i32 value = 40;
    increment(ref mut value);
    let ref mut i32 slot = ref mut value;
    *slot = *slot + 1;
    assert(value == 42);
    var bool ready = false; let ref mut bool ready_slot = ref mut ready;
    *ready_slot = true; assert(ready);
    var u8 byte = 0; let ref mut u8 byte_slot = ref mut byte;
    *byte_slot = 255; assert(byte == 255);
    var u32 word = 0; let ref mut u32 word_slot = ref mut word;
    *word_slot = 4294967295; assert(word == 4294967295);
    var usize size = 0; let ref mut usize size_slot = ref mut size;
    *size_slot = 9; assert(size == 9);
    return value;
}`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("scalar dereference store: %+v %v", outcome, err)
	}
}
