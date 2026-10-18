package interpreter_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestUnusedNativeOnlyOperationIsRejected(t *testing.T) {
	program := lowerSource(t, `module main;
fn i32 main() {
    let ref str text = "fairy";
    let ptr u8 address = unsafe { text as ptr u8 };
    return 42;
}`)
	if failures := mir.Verify(program); len(failures) != 0 {
		t.Fatalf("invalid test input: %+v", failures)
	}
	_, err := interpreter.Execute(program, "main::main")
	var capability interpreter.CapabilityError
	if !errors.As(err, &capability) || capability.Kind != "str-data-address" || !strings.Contains(err.Error(), "requires a native backend") {
		t.Fatalf("unused native operation was silently ignored: %v", err)
	}
}
