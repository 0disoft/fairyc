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
		{"recursive-leaf", []Type{ParseType("main::Token")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			array := FixedArray(ParseType("main::Token"), 2)
			lookups := 0
			fields := func(typ Type) ([]Type, bool) {
				if typ == ParseType("main::Root") {
					return []Type{array, array}, true
				}
				if typ == ParseType("main::Token") {
					lookups++
					return tc.fields, true
				}
				return nil, false
			}
			if got := PlainViewStruct(ParseType("main::Root"), fields); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			if lookups != 1 {
				t.Fatalf("repeated element graph validated %d times", lookups)
			}
		})
	}
}

func TestNominalArrayFieldViewBorrowShape(t *testing.T) {
	token := ParseType("main::Token")
	array := FixedArray(token, 2)
	slice := ParseType("[main::Token]")
	for _, tc := range []struct {
		name        string
		owner, view Type
		operator    string
		want        bool
	}{
		{"shared", SharedBorrow(array), SharedBorrow(slice), "&", true},
		{"mutable", MutableBorrow(array), MutableBorrow(slice), "&mut", true},
		{"mutable-from-shared", SharedBorrow(array), MutableBorrow(slice), "&mut", false},
		{"lost-mutability", MutableBorrow(array), SharedBorrow(slice), "&", false},
		{"nominal-mismatch", SharedBorrow(array), SharedBorrow(ParseType("[main::Other]")), "&", false},
		{"empty", SharedBorrow(FixedArray(token, 0)), SharedBorrow(slice), "&", false},
		{"past-limit", SharedBorrow(FixedArray(token, 4097)), SharedBorrow(slice), "&", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if ArrayViewBorrow(tc.owner, tc.view, tc.operator) != tc.want {
				t.Fatal("borrow shape mismatch")
			}
		})
	}
}
