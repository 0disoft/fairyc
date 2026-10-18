package interpreter

import (
	"testing"

	"github.com/0disoft/fairyc/internal/mir"
)

func TestInertStructDropDoesNotHideOwnedPayloads(t *testing.T) {
	definition := &mir.StructType{Fields: []mir.StructField{{Name: "value"}}}
	wrap := func(v runtimeValue) runtimeValue {
		return runtimeValue{kind: valueStruct, structure: &structState{definition: definition, fields: []runtimeValue{v}}}
	}
	if !inertDropValue(wrap(i32Value(42)), 0) {
		t.Fatal("inert scalar field rejected")
	}
	if !inertDropValue(wrap(wrap(i32Value(42))), 0) {
		t.Fatal("nested inert field rejected")
	}
	if inertDropValue(wrap(runtimeValue{kind: valueResource}), 0) {
		t.Fatal("owned field hidden by struct")
	}
	if inertDropValue(wrap(runtimeValue{kind: valueStruct}), 0) {
		t.Fatal("missing nested state accepted")
	}
	cycle := wrap(i32Value(0))
	cycle.structure.fields[0] = cycle
	if inertDropValue(cycle, 0) {
		t.Fatal("cyclic runtime state accepted")
	}
}
