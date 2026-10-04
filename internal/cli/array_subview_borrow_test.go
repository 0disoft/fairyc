package cli

import (
	"os"
	"testing"
)

func TestArraySubviewBorrowNative(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/array_subview_borrow.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;fn void use(ref mut i64[] v,i32 next){assert(false);}fn i32 main(){var i64[2] a=[1,2];`
	runI64ViewNativeCases(t, "array-subview-borrow", []struct{ name, source, trap string }{
		{"matrix", string(source), ""},
		{"reversed", base + `let ref i64[] v=ref a[2..1];return 0;}`, "bounds"},
		{"past-end", base + `let ref i64[] v=ref a[..3];return 0;}`, "bounds"},
		{"start-past-end", base + `let ref i64[] v=ref a[3..];return 0;}`, "bounds"},
		{"exclusive-past-end", base + `let ref mut i64[] v=ref mut a[..3];return 0;}`, "bounds"},
		{"empty-index", base + `let ref i64[] v=ref a[2..];let i64 x=v[0];return 0;}`, "bounds"},
		{"start-return", base + `use(ref mut a[{return 0;0}..{assert(false);1}],{assert(false);1});assert(false);return 1;}`, ""},
		{"end-return", base + `use(ref mut a[0..{return 0;1}],{assert(false);1});assert(false);return 1;}`, ""},
	})
}

func TestArraySubviewBorrowCache(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/array_subview_borrow.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(source))
}
