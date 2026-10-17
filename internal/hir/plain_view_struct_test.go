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
		{"scalar", map[Type][]Type{ParseType("Root"): {TypeBool, TypeU8, TypeI32, TypeI64, TypeU32, TypeUSize}}, true},
		{"diamond", map[Type][]Type{ParseType("Root"): {ParseType("Left"), ParseType("Right")}, ParseType("Left"): {ParseType("Leaf")}, ParseType("Right"): {ParseType("Leaf"), ParseType("Leaf")}, ParseType("Leaf"): {TypeI64}}, true},
		{"empty", map[Type][]Type{ParseType("Root"): {}}, false},
		{"nested-empty", map[Type][]Type{ParseType("Root"): {ParseType("Child"), ParseType("Child")}, ParseType("Child"): {}}, false},
		{"missing", map[Type][]Type{ParseType("Root"): {ParseType("Missing")}}, false},
		{"self-cycle", map[Type][]Type{ParseType("Root"): {ParseType("Root")}}, false},
		{"indirect-cycle", map[Type][]Type{ParseType("Root"): {ParseType("Child")}, ParseType("Child"): {ParseType("Root")}}, false},
		{"borrow", map[Type][]Type{ParseType("Root"): {ParseType("Child")}, ParseType("Child"): {SharedBorrow(TypeI32)}}, false},
		{"pointer", map[Type][]Type{ParseType("Root"): {ParseType("Child")}, ParseType("Child"): {RawPointer(TypeI32, true)}}, false},
		{"array", map[Type][]Type{ParseType("Root"): {ParseType("Child")}, ParseType("Child"): {FixedArray(TypeI32, 2)}}, true},
		{"zero-array", map[Type][]Type{ParseType("Root"): {FixedArray(TypeI32, 0)}}, false},
		{"nested-array", map[Type][]Type{ParseType("Root"): {FixedArray(FixedArray(TypeI32, 2), 2)}}, false},
		{"option", map[Type][]Type{ParseType("Root"): {ParseType("Child")}, ParseType("Child"): {Option(TypeI32)}}, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			calls := map[Type]int{}
			got := PlainViewStruct(ParseType("Root"), func(value Type) ([]Type, bool) { calls[value]++; fields, ok := item.graph[value]; return fields, ok })
			if got != item.want {
				t.Fatalf("got %v want %v", got, item.want)
			}
			stored := PlainStoredStruct(ParseType("Root"), func(value Type) ([]Type, bool) { fields, ok := item.graph[value]; return fields, ok })
			if want := item.want || item.name == "empty" || item.name == "nested-empty"; stored != want {
				t.Fatalf("stored graph got %v want %v", stored, want)
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
			graph[ParseType(fmt.Sprint(i))] = []Type{ParseType(fmt.Sprint(i + 1))}
		}
		graph[ParseType(fmt.Sprint(depth))] = []Type{TypeI64}
		if !PlainViewStruct(ParseType("0"), func(value Type) ([]Type, bool) { fields, ok := graph[value]; return fields, ok }) {
			t.Fatal("deep acyclic chain rejected")
		}
	})
}
