package interpreter

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"testing"
)

func TestIntegerElementBorrowBacking(t *testing.T) {
	for _, value := range []runtimeValue{{kind: valueU8, typ: hir.TypeU8, u64: 7}, runtimeValue{kind: valueI32, typ: hir.TypeI32, i32: 7}, {kind: valueI64, typ: hir.TypeI64, i64: 4000000000}, {kind: valueU32, typ: hir.TypeU32, u64: 4000000000}, {kind: valueUSize, typ: hir.TypeUSize, u64: 4000000000}} {
		t.Run(value.typ.String(), func(t *testing.T) {
			backing := &arrayState{elements: []runtimeValue{value, value}}
			view := runtimeValue{kind: valueView, typ: hir.MutableBorrow(hir.Slice(value.typ)), view: &indexedViewState{array: backing, offset: 1, length: 1, mutable: true}}
			borrowed, e := view.borrowViewElement(0, hir.MutableBorrow(value.typ))
			if e != nil {
				t.Fatal(e)
			}
			locals := map[mir.LocalID]runtimeValue{1: borrowed, 2: value}
			op := mir.Operation{Kind: mir.OperationDerefStore, Symbol: hir.NoSymbol, DestinationLocal: mir.NoLocal, OwnerType: borrowed.typ, Type: value.typ, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 1}, {Kind: mir.OperandLocal, Local: 2}}}
			changed := value
			changed.i32 = 9
			changed.i64 = 9
			changed.u64 = 9
			locals[2] = changed
			if e := storeScalarDeref(locals, op); e != nil {
				t.Fatal(e)
			}
			if !scalarStoreValueMatches(backing.elements[1], value.typ) || backing.elements[1] != changed || backing.elements[0] != value {
				t.Fatal("subview reference did not select stored cell")
			}
			shared, e := view.borrowViewElement(0, hir.SharedBorrow(value.typ))
			if e != nil {
				t.Fatal(e)
			}
			loaded, ok := shared.reference.load()
			if !ok || loaded != changed {
				t.Fatal("shared reference missed updated cell")
			}
			for _, index := range []uint64{1, ^uint64(0)} {
				if _, e := view.borrowViewElement(index, hir.MutableBorrow(value.typ)); e == nil {
					t.Fatal("bounds accepted")
				}
			}
			view.typ = hir.SharedBorrow(hir.Slice(value.typ))
			if _, e := view.borrowViewElement(0, hir.MutableBorrow(value.typ)); e == nil {
				t.Fatal("shared receiver became mutable")
			}
		})
	}
	static := staticByteView("Az", hir.SharedBorrow(hir.TypeByteSlice))
	sub, e := static.sliceIndexedView(1, 2)
	if e != nil {
		t.Fatal(e)
	}
	letter, e := sub.borrowViewElement(0, hir.SharedBorrow(hir.TypeU8))
	if e != nil {
		t.Fatal(e)
	}
	value, ok := letter.reference.load()
	if !ok || value.u64 != 122 {
		t.Fatal("static subview reference offset")
	}
	if _, e := sub.borrowViewElement(0, hir.MutableBorrow(hir.TypeU8)); e == nil {
		t.Fatal("static bytes became mutable")
	}
}
