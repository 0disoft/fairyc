package hir

import "testing"

func TestContainsTaggedTypeVisitsEachIdentityOnce(t *testing.T) {
	root := TypeI32
	for range 80 {
		child := Option(root)
		root = Result(child, child)
	}
	calls := make(map[Type]int)
	if ContainsTaggedType(root, func(value Type) bool { calls[value]++; return false }) {
		t.Fatal("negative predicate matched")
	}
	if len(calls) != 161 {
		t.Fatalf("visited %d types, want 161", len(calls))
	}
	for _, count := range calls {
		if count != 1 {
			t.Fatal("shared identity was tested more than once")
		}
	}
}

func TestContainsTaggedTypeKeepsTraversalBoundaryAndOrder(t *testing.T) {
	for _, value := range []Type{SharedBorrow(TypeI32), RawPointer(TypeI32, true), FixedArray(TypeI32, 1)} {
		if ContainsTaggedType(value, func(value Type) bool { return value == TypeI32 }) {
			t.Fatal("traversed a non-tagged ownership boundary")
		}
	}
	root := Result(Option(TypeI32), TypeU8)
	var last Type
	if !ContainsTaggedType(root, func(value Type) bool { last = value; return value == TypeI32 || value == TypeU8 }) || last != TypeI32 {
		t.Fatal("success-first short circuit changed")
	}
}
