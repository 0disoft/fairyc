package interpreter

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestMutableStructViewCopiesAndBounds(t *testing.T) {
	child := copyTestStruct(t)
	token, err := structValue(&mir.StructType{Type: "main::Token", Fields: []mir.StructField{{Name: "nested", Type: child.typ}}}, []runtimeValue{child})
	if err != nil {
		t.Fatal(err)
	}
	typ := hir.MutableBorrow(hir.Type("[main::Token]"))
	backing := &arrayState{elements: []runtimeValue{token.cloneForValueCopy(), token.cloneForValueCopy()}}
	view := runtimeValue{kind: valueView, typ: typ, view: &indexedViewState{array: backing, length: 2, mutable: true}}
	if err := view.storeIndexed(0, token); err != nil {
		t.Fatal(err)
	}
	token.structure.fields["nested"].structure.fields["count"] = i32Value(42)
	loaded, err := view.indexedAt(0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.structure.fields["nested"].structure.fields["count"].i32 != 7 {
		t.Fatal("stored value aliases source")
	}
	loaded.structure.fields["nested"].structure.fields["count"] = i32Value(99)
	if backing.elements[0].structure.fields["nested"].structure.fields["count"].i32 != 7 {
		t.Fatal("copied load aliases owner")
	}
	sub, err := view.sliceIndexedView(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := sub.storeIndexed(0, token); err != nil {
		t.Fatal(err)
	}
	if backing.elements[1].structure.fields["nested"].structure.fields["count"].i32 != 42 {
		t.Fatal("subview store missed backing")
	}
	for _, index := range []uint64{2, ^uint64(0)} {
		if err := view.storeIndexed(index, token); err == nil {
			t.Fatal("out of bounds store accepted")
		}
	}
	wrong := token
	wrong.typ = "main::Other"
	if err := view.storeIndexed(0, wrong); err == nil {
		t.Fatal("wrong nominal accepted")
	}
	shared := view
	shared.typ = hir.SharedBorrow(hir.Type("[main::Token]"))
	if err := shared.storeIndexed(0, token); err == nil {
		t.Fatal("shared store accepted")
	}
}
