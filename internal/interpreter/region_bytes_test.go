package interpreter

import (
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestRegionByteAllocationBudgetAndBacking(t *testing.T) {
	r := executionState{steps: 8}
	view, err := r.regionBytes(runtimeValue{kind: valueUSize, typ: hir.TypeUSize, scalar: 4})
	if err != nil {
		t.Fatal(err)
	}
	if r.steps != 4 || view.view().length != 4 {
		t.Fatalf("budget or length: %+v %d", view, r.steps)
	}
	for i := uint64(0); i < 4; i++ {
		b, err := view.byteAt(i)
		if err != nil || b != 0 {
			t.Fatalf("byte %d: %d %v", i, b, err)
		}
	}
	if err := view.storeIndexed(1, runtimeValue{kind: valueU8, typ: hir.TypeU8, scalar: 42}); err != nil {
		t.Fatal(err)
	}
	other, err := r.regionBytes(runtimeValue{kind: valueUSize, typ: hir.TypeUSize, scalar: 4})
	if err != nil {
		t.Fatal(err)
	}
	if b, err := other.byteAt(1); err != nil || b != 0 {
		t.Fatal("independent Region allocations share storage")
	}
	_, err = r.regionBytes(runtimeValue{kind: valueUSize, typ: hir.TypeUSize, scalar: ^uint64(0)})
	var capability CapabilityError
	if !errors.As(err, &capability) || capability.Kind != LimitStep {
		t.Fatalf("unbounded allocation was not rejected: %v", err)
	}
}
