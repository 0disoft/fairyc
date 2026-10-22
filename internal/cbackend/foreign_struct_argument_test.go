package cbackend_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestEmitRejectsNonReprCStructForeignParameter(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
struct PrivateStatus { u32 status; }
unsafe extern C fn i32 consume(PrivateStatus value);
fn i32 main() { return unsafe { consume(PrivateStatus { .status = 42 }) }; }
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, `foreign parameter struct "main::PrivateStatus" is not repr(C)`) {
		t.Fatalf("private record argument: %v", err)
	}
}
