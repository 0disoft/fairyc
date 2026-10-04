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

func TestEmitMutableFieldBorrowUsesOwnerAddress(t *testing.T) {
	program := lowerSource(t, `module main;struct Inner{i64 number;}struct Owner{Inner inner;}fn void edit(ref mut Inner value){value.number=4000000002;}fn i32 main(){var Owner owner=Owner{.inner=Inner{.number=4000000000}};let ref mut Inner selected=ref mut owner.inner;edit(selected);let ref mut i64 scalar=ref mut selected.number;*scalar=4000000003;assert(owner.inner.number==4000000003);return 0;}`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(unit.Source, "= &((fy_l_") != 2 {
		t.Fatalf("lost projected addresses: %s", unit.Source)
	}
}
