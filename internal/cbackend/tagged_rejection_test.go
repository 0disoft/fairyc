package cbackend_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestEmitStillRejectsUnsupportedFloatingTaggedPayload(t *testing.T) {
	program := lowerSource(t, `module main;
fn Option<f64> choose() { return .none; }
fn i32 main() {
    let Option<f64> value = choose();
    return 42;
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	_, err = cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, `"Option<f64>"`) {
		t.Fatalf("unsupported payload rejection changed: %v", err)
	}
}
