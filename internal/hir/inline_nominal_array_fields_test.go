package hir

import "testing"

func TestInlineNominalArrayFieldGraph(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fields []Type
		want   bool
	}{
		{"scalars", []Type{TypeI64, TypeBool, TypeU8}, true},
		{"empty", nil, false},
		{"array-leaf", []Type{FixedArray(TypeI64, 2)}, false},
		{"borrow-leaf", []Type{SharedBorrow(TypeI64)}, false},
		{"pointer-leaf", []Type{RawPointer(TypeU8, false)}, false},
		{"tagged-leaf", []Type{Option(TypeI32)}, false},
		{"recursive-leaf", []Type{Type("main::Token")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			array := FixedArray(Type("main::Token"), 2)
			lookups := 0
			fields := func(typ Type) ([]Type, bool) {
				if typ == Type("main::Root") {
					return []Type{array, array}, true
				}
				if typ == Type("main::Token") {
					lookups++
					return tc.fields, true
				}
				return nil, false
			}
			if got := PlainViewStruct(Type("main::Root"), fields); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			if lookups != 1 {
				t.Fatalf("repeated element graph validated %d times", lookups)
			}
		})
	}
}
