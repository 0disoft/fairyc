package cli

import (
	"os"
	"testing"
)

func TestStructArrayFieldBorrowNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/struct_array_field_borrows.fy")
	if err != nil {
		t.Fatal(err)
	}
	base := `module main;struct Pos{i64 x;}struct Point{Pos pos;}fn void edit(ref mut i64 x){*x=9;}fn i32 main(){var Point[2] a=[Point{.pos=Pos{.x=1}};2];`
	runI64ViewNativeCases(t, "struct-array-field-borrow", []struct{ name, source, trap string }{
		{"matrix", string(source), ""},
		{"shared-end", base + `let ref i64 x=ref a[2].pos.x;return 0;}`, "bounds"},
		{"exclusive-end", base + `edit(ref mut a[2].pos.x);return 0;}`, "bounds"},
		{"whole-field-end", base + `let ref mut Pos p=ref mut a[2].pos;p.x=9;return 0;}`, "bounds"},
		{"index-return", base + `edit(ref mut a[{return 0;0}].pos.x);assert(false);return 1;}`, ""},
	})
}

func TestStructArrayFieldBorrowCache(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/struct_array_field_borrows.fy")
	if err != nil {
		t.Fatal(err)
	}
	checkStructVecViewCache(t, string(source))
}
