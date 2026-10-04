package cli

import (
	"os"
	"testing"
)

func TestStructArrayFieldNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/struct_array_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	base := `module main;struct Point{i64 x;}fn i32 main(){var Point[2] a=[Point{.x=1};2];`
	runI64ViewNativeCases(t, "struct-array-field", []struct{ name, source, trap string }{
		{"matrix", string(source), ""},
		{"store-end", base + `a[2].x=9;return 0;}`, "bounds"},
		{"bounds-before-rhs", base + `a[2].x={assert(false);9};return 0;}`, "bounds"},
		{"compound-end", base + `a[2].x+=9;return 0;}`, "bounds"},
		{"overflow", base + `a[0].x=9223372036854775807;a[0].x+=1;return 0;}`, "integer-overflow"},
		{"rhs-return", base + `a[0].x={return 0;9};assert(false);return 1;}`, ""},
	})
}

func TestStructArrayFieldCache(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/struct_array_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	checkStructVecViewCache(t, string(source))
}
