package interpreter

import (
	"context"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestRegionBytesPreserveSliceAndReferenceAliases(t *testing.T) {
	r := executionState{steps: 32}
	root, err := r.regionBytes(runtimeValue{kind: valueUSize, typ: hir.TypeUSize, scalar: 8}, false)
	if err != nil {
		t.Fatal(err)
	}
	slice, err := root.sliceIndexedView(2, 6)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := slice.borrowViewElement(1, hir.MutableBorrow(hir.TypeU8))
	if err != nil {
		t.Fatal(err)
	}
	op := mir.Operation{Kind: mir.OperationDerefStore, Symbol: hir.NoSymbol, DestinationLocal: mir.NoLocal,
		OwnerType: hir.MutableBorrow(hir.TypeU8), Type: hir.TypeU8,
		Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 0}, {Kind: mir.OperandU8, Type: hir.TypeU8, U64: 255}}}
	if err := storeScalarDeref(map[mir.LocalID]runtimeValue{0: reference}, op); err != nil {
		t.Fatal(err)
	}
	if b, err := root.byteAt(3); err != nil || b != 255 {
		t.Fatalf("reference store did not reach original backing: %d, %v", b, err)
	}
	if err := root.storeIndexed(3, runtimeValue{kind: valueU8, typ: hir.TypeU8, scalar: 42}); err != nil {
		t.Fatal(err)
	}
	if value, ok := reference.reference().load(); !ok || value.u64() != 42 {
		t.Fatal("reference retained a stale byte value")
	}
	shared := slice
	shared.typ = hir.SharedBorrow(hir.TypeByteSlice)
	sharedReference, err := shared.borrowViewElement(1, hir.SharedBorrow(hir.TypeU8))
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := sharedReference.reference().load(); !ok || value.u64() != 42 {
		t.Fatal("shared reference lost its offset backing")
	}
	if err := shared.storeIndexed(0, runtimeValue{kind: valueU8, typ: hir.TypeU8}); err == nil {
		t.Fatal("shared view acquired write authority")
	}
	if _, err := shared.borrowViewElement(0, hir.MutableBorrow(hir.TypeU8)); err == nil {
		t.Fatal("shared view acquired a mutable element reference")
	}
	if _, err := slice.byteAt(4); err == nil {
		t.Fatal("slice read escaped its declared length")
	}
	if err := slice.storeIndexed(4, runtimeValue{kind: valueU8, typ: hir.TypeU8}); err == nil {
		t.Fatal("slice store escaped its declared length")
	}
}

func TestRegionByteCopiesDetachStaticAndRegionSubviews(t *testing.T) {
	r := executionState{steps: 32}
	input := staticByteView("a\x00\xffz", hir.SharedBorrow(hir.TypeByteSlice))
	source, err := input.sliceIndexedView(1, 4)
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.regionBytes(source, true)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []byte{0, 255, 'z'} {
		if b, err := first.byteAt(uint64(i)); err != nil || b != want {
			t.Fatalf("copied byte %d: %d, %v", i, b, err)
		}
	}
	slice, err := first.sliceIndexedView(1, 3)
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.regionBytes(slice, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.storeIndexed(0, runtimeValue{kind: valueU8, typ: hir.TypeU8, scalar: 7}); err != nil {
		t.Fatal(err)
	}
	if b, err := first.byteAt(1); err != nil || b != 255 {
		t.Fatal("Region copy shared mutable backing with its source")
	}
	if err := first.storeIndexed(2, runtimeValue{kind: valueU8, typ: hir.TypeU8, scalar: 8}); err != nil {
		t.Fatal(err)
	}
	if b, err := second.byteAt(1); err != nil || b != 'z' {
		t.Fatal("source mutation changed an independent Region copy")
	}
	if r.steps != 27 {
		t.Fatalf("copy step charge = %d, want 27 remaining", r.steps)
	}
}

func TestRegionBytesPreserveEmptyBudgetAndCancellation(t *testing.T) {
	r := executionState{}
	empty, err := r.regionBytes(runtimeValue{kind: valueUSize, typ: hir.TypeUSize}, false)
	if err != nil || empty.view().length != 0 || r.steps != 0 {
		t.Fatalf("empty allocation: %v", err)
	}
	if _, err := empty.byteAt(0); err == nil {
		t.Fatal("empty byte view admitted an element")
	}
	r.steps = 1
	_, err = r.regionBytes(runtimeValue{kind: valueUSize, typ: hir.TypeUSize, scalar: 2}, false)
	var capability CapabilityError
	if !errors.As(err, &capability) || capability.Kind != LimitStep || r.steps != 1 {
		t.Fatalf("allocation budget changed: steps=%d, error=%v", r.steps, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.context, r.steps = ctx, 2
	_, err = r.regionBytes(runtimeValue{kind: valueUSize, typ: hir.TypeUSize, scalar: 2}, false)
	if !errors.Is(err, context.Canceled) || r.steps != 2 {
		t.Fatalf("cancellation was not checked before allocation: %v", err)
	}
}
