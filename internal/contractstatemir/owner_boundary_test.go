package contractstatemir

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestHIRTypeContainsRawPointerTraversesOwnerTypeGraph(t *testing.T) {
	structType := hir.Type("ledger::PointerStruct")
	enumType := hir.Type("ledger::PointerEnum")
	resourceType := hir.Type("ledger::PointerResource")
	cycleType := hir.Type("ledger::Cycle")
	input := hir.Package{Modules: []hir.Module{{
		Structs: []hir.Struct{
			{Type: structType, Fields: []hir.StructField{{Type: hir.RawPointer(hir.TypeU8, false)}}},
			{Type: cycleType, Fields: []hir.StructField{{Type: cycleType}}},
		},
		Enums:     []hir.Enum{{Type: enumType, Variants: []hir.EnumVariant{{Fields: []hir.EnumField{{Type: hir.RawPointer(hir.TypeU32, true)}}}}}},
		Resources: []hir.Resource{{Type: resourceType, Fields: []hir.ResourceField{{Type: hir.RawPointer(hir.TypeU8, true)}}}},
	}}}
	tests := []struct {
		name      string
		typeValue hir.Type
		want      bool
	}{
		{name: "direct", typeValue: hir.RawPointer(hir.TypeU8, false), want: true},
		{name: "borrow element", typeValue: hir.SharedBorrow(hir.RawPointer(hir.TypeU8, false)), want: true},
		{name: "fixed array", typeValue: hir.FixedArray(hir.RawPointer(hir.TypeU8, false), 2), want: true},
		{name: "option", typeValue: hir.Option(hir.RawPointer(hir.TypeU8, false)), want: true},
		{name: "result", typeValue: hir.Result(hir.TypeU32, hir.RawPointer(hir.TypeU8, false)), want: true},
		{name: "struct field", typeValue: structType, want: true},
		{name: "enum field", typeValue: enumType, want: true},
		{name: "resource field", typeValue: resourceType, want: true},
		{name: "safe borrow", typeValue: hir.MutableBorrow(hir.TypeU32), want: false},
		{name: "pointer-free cycle", typeValue: cycleType, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hirTypeContainsRawPointer(input, test.typeValue, make(map[hir.Type]bool)); got != test.want {
				t.Fatalf("hirTypeContainsRawPointer(%q) = %t, want %t", test.typeValue, got, test.want)
			}
		})
	}
}
