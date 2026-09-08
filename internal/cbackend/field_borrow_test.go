package cbackend_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestEmitSharedFieldBorrowUsesOwnerAddress(t *testing.T) {
	program := lowerSource(t, `module main;
struct Inner { i32 number; }
resource Owner { Inner inner; cleanup {} }
fn ref i32 number(ref Owner value) { return ref value.inner.number; }
fn i32 main() {
    let Owner owner = Owner { .inner = Inner { .number = 42 } };
    return *number(ref owner);
}`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(unit.Source, "= &((fy_l_") != 2 {
		t.Fatalf("expected two owner-linked field addresses:\n%s", unit.Source)
	}
}
