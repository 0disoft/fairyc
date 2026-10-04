package interpreter

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"testing"
)

func TestStructDerefStoreBackingAndCopy(t *testing.T) {
	child := copyTestStruct(t)
	definition := &mir.StructType{Type: "main::Token", Fields: []mir.StructField{{Name: "nested", Type: child.typ}}}
	token, e := structValue(definition, []runtimeValue{child})
	if e != nil {
		t.Fatal(e)
	}
	backing := &arrayState{elements: []runtimeValue{token.cloneForValueCopy()}}
	view := runtimeValue{kind: valueView, typ: hir.MutableBorrow(hir.Type("[main::Token]")), view: &indexedViewState{array: backing, length: 1, mutable: true}}
	selected, e := view.borrowStructElement(0, hir.MutableBorrow(token.typ))
	if e != nil {
		t.Fatal(e)
	}
	token.structure.fields["nested"].structure.fields["count"] = i32Value(42)
	locals := map[mir.LocalID]runtimeValue{1: selected, 2: token}
	op := mir.Operation{Kind: mir.OperationDerefStore, Symbol: hir.NoSymbol, DestinationLocal: mir.NoLocal, OwnerType: selected.typ, Type: token.typ, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 1}, {Kind: mir.OperandLocal, Local: 2}}}
	structs := map[hir.Type]*mir.StructType{token.typ: definition, child.typ: child.structure.definition}
	if e := storeDeref(locals, op, structs); e != nil {
		t.Fatal(e)
	}
	token.structure.fields["nested"].structure.fields["count"] = i32Value(99)
	copied, e := view.indexedAt(0)
	if e != nil {
		t.Fatal(e)
	}
	if copied.structure.fields["nested"].structure.fields["count"].i32 != 42 || selected.structure != backing.elements[0].structure {
		t.Fatal("store lost referent or aliased source")
	}
	copied.structure.fields["nested"].structure.fields["count"] = i32Value(100)
	if selected.structure.fields["nested"].structure.fields["count"].i32 != 42 {
		t.Fatal("copy aliases replaced storage")
	}
	shared := selected
	shared.typ = hir.SharedBorrow(token.typ)
	locals[1] = shared
	if e := storeDeref(locals, op, structs); e == nil {
		t.Fatal("shared runtime receiver accepted")
	}
}
