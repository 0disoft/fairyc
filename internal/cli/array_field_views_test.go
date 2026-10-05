package cli

import (
	"os"
	"testing"
)

func TestArrayFieldViewNative(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/array_field_views.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;struct P{u8[2] bytes;i64[2] totals;}fn void fill(ref mut u8[] v){}fn i32 main(){var P p=P{.bytes=[1,2],.totals=[1,2]};`
	runI64ViewNativeCases(t, "array-field-view", []struct{ name, source, trap string }{
		{"matrix", string(source), ""},
		{"store-end", base + `p.bytes[2]=3;return 0;}`, "bounds"},
		{"element-end", base + `let ref u8 x=ref p.bytes[2];return 0;}`, "bounds"},
		{"slice-reversed", base + `let ref u8[] v=ref p.bytes[2..1];return 0;}`, "bounds"},
		{"slice-end", base + `fill(ref mut p.bytes[..3]);return 0;}`, "bounds"},
		{"compound-overflow", base + `p.bytes[0]+=255;return 0;}`, "integer-overflow"},
		{"index-return", base + `p.bytes[{return 0;0}]={assert(false);3};assert(false);return 1;}`, ""},
		{"bound-return", base + `fill(ref mut p.bytes[{return 0;0}..{assert(false);1}]);assert(false);return 1;}`, ""},
	})
}
func TestArrayFieldViewCache(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/array_field_views.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(source))
}
