package interpreter

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestDerefCopyChargesAndDetachesBorrowedStorage(t *testing.T) {
	for _, mutable := range []bool{false, true} {
		original := copyTestStruct(t)
		borrowed := original
		borrowed.typ = hir.SharedBorrow(original.typ)
		if mutable {
			borrowed.typ = hir.MutableBorrow(original.typ)
		}
		locals := map[mir.LocalID]runtimeValue{0: borrowed}
		op := mir.Operation{Operator: "*", OwnerType: borrowed.typ, Type: original.typ,
			Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 0}}}
		// One struct: visit + state + slice + slot + scalar visit = 5 units.
		if _, _, err := copyBudgetState(4).evaluateUnary(locals, op); err == nil {
			t.Fatal("dereference bypassed copy budget")
		} else {
			requireCopyLimit(t, err)
		}
		state := copyBudgetState(5)
		copied, trap, err := state.evaluateUnary(locals, op)
		if err != nil || trap != "" || state.copyUnits != 0 || copied.typ != original.typ || copied.structure() == original.structure() {
			t.Fatalf("dereference copy: %+v, %q, %v; remaining units %d", copied, trap, err, state.copyUnits)
		}
		original.structure().fields[0] = i32Value(42)
		if copied.structure().fields[0].i32() != 7 || locals[0] != borrowed {
			t.Fatal("dereference lost the snapshot or changed the borrow")
		}
	}
}
