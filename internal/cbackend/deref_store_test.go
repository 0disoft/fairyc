package cbackend_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestEmitFieldStoreScalarDerefUsesOnePointerWrite(t *testing.T) {
	program := lowerSource(t, `module main;
fn void set(ref mut i32 slot) { *slot = 42; }
fn i32 main() { var i32 value = 0; set(ref mut value); return value; }`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(unit.Source, "    *(fy_l_") != 1 || !strings.Contains(unit.Source, ") = INT32_C(42);") {
		t.Fatalf("expected one direct pointer write:\n%s", unit.Source)
	}
}
