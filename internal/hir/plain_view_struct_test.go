package hir

import (
	"fmt"
	"testing"
)

func TestPlainViewStructGraph(t *testing.T) {
	for _, item := range []struct {
		name  string
		graph map[Type][]Type
		want  bool
	}{
		{"scalar", map[Type][]Type{"Root": {TypeBool, TypeU8, TypeI32, TypeI64, TypeU32, TypeUSize}}, true},
		{"diamond", map[Type][]Type{"Root": {"Left", "Right"}, "Left": {"Leaf"}, "Right": {"Leaf", "Leaf"}, "Leaf": {TypeI64}}, true},
		{"empty", map[Type][]Type{"Root": {}}, false},
		{"missing", map[Type][]Type{"Root": {"Missing"}}, false},
		{"self-cycle", map[Type][]Type{"Root": {"Root"}}, false},
		{"indirect-cycle", map[Type][]Type{"Root": {"Child"}, "Child": {"Root"}}, false},
		{"borrow", map[Type][]Type{"Root": {"Child"}, "Child": {SharedBorrow(TypeI32)}}, false},
		{"pointer", map[Type][]Type{"Root": {"Child"}, "Child": {RawPointer(TypeI32, true)}}, false},
		{"array", map[Type][]Type{"Root": {"Child"}, "Child": {FixedArray(TypeI32, 2)}}, false},
		{"option", map[Type][]Type{"Root": {"Child"}, "Child": {Option(TypeI32)}}, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			calls := map[Type]int{}
			got := PlainViewStruct("Root", func(value Type) ([]Type, bool) { calls[value]++; fields, ok := item.graph[value]; return fields, ok })
			if got != item.want {
				t.Fatalf("got %v want %v", got, item.want)
			}
			for value, count := range calls {
				if count != 1 {
					t.Fatalf("revisited %s %d times", value, count)
				}
			}
		})
	}
	t.Run("deep", func(t *testing.T) {
		graph := map[Type][]Type{}
		const depth = 10000
		for i := 0; i < depth; i++ {
			graph[Type(fmt.Sprint(i))] = []Type{Type(fmt.Sprint(i + 1))}
		}
		graph[Type(fmt.Sprint(depth))] = []Type{TypeI64}
		if !PlainViewStruct("0", func(value Type) ([]Type, bool) { fields, ok := graph[value]; return fields, ok }) {
			t.Fatal("deep acyclic chain rejected")
		}
	})
}
