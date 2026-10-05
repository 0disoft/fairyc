package cli

import (
	"os"
	"testing"
)

func TestNominalArrayFieldViewNative(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/nominal_array_field_views.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;struct T{i64 x;}struct P{T[2] data;}fn void fill(ref mut T[] v){}fn i32 main(){var P p=P{.data=[T{.x=1};2]};`
	runI64ViewNativeCases(t, "nominal-array-field-view", []struct{ name, source, trap string }{
		{"matrix", string(source), ""},
		{"element-end", base + `let ref T x=ref p.data[2];return 0;}`, "bounds"},
		{"store-end", base + `p.data[2]=T{.x=3};return 0;}`, "bounds"},
		{"leaf-end", base + `p.data[2].x=3;return 0;}`, "bounds"},
		{"slice-reversed", base + `let ref T[] v=ref p.data[2..1];return 0;}`, "bounds"},
		{"slice-end", base + `fill(ref mut p.data[..3]);return 0;}`, "bounds"},
		{"for-end", base + `for(let T t in p.data[..3]){assert(false);}return 0;}`, "bounds"},
		{"for-reversed", base + `for(let ref mut T t in p.data[2..1]){assert(false);}return 0;}`, "bounds"},
		{"overflow", `module main;struct T{i64 x;}struct P{T[1] data;}fn i32 main(){var P p=P{.data=[T{.x=9223372036854775807}]};p.data[0].x+=1;return 0;}`, "integer-overflow"},
		{"index-return", base + `p.data[{return 0;0}].x={assert(false);3};assert(false);return 1;}`, ""},
		{"bound-return", base + `fill(ref mut p.data[{return 0;0}..{assert(false);1}]);assert(false);return 1;}`, ""},
		{"for-return", base + `for(let T t in p.data[0..{return 0;1}]){assert(false);}assert(false);return 1;}`, ""},
	})
}

func TestNominalArrayFieldViewCache(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/nominal_array_field_views.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(source))
}
