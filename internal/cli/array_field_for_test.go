package cli

import (
	"os"
	"testing"
)

func TestArrayFieldForNative(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/array_field_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;struct P{i64[2] data;}fn i32 main(){var P p=P{.data=[1,2]};`
	runI64ViewNativeCases(t, "array-field-for", []struct{ name, source, trap string }{
		{"matrix", string(source), ""},
		{"past-end", base + `for(let i64 x in p.data[..3]){assert(false);}return 0;}`, "bounds"},
		{"reversed", base + `for(let ref mut i64 x in p.data[2..1]){assert(false);}return 0;}`, "bounds"},
		{"start-past-end", base + `for(let ref i64 x in p.data[3..]){assert(false);}return 0;}`, "bounds"},
		{"overflow", `module main;struct P{i64[1] data;}fn i32 main(){var P p=P{.data=[9223372036854775807]};for(let ref mut i64 x in p.data){*x+=1;}return 0;}`, "integer-overflow"},
		{"bound-return", base + `for(let i64 x in p.data[{return 0;0}..{assert(false);1}]){assert(false);}assert(false);return 1;}`, ""},
		{"end-return", base + `for(let i64 x in p.data[0..{return 0;1}]){assert(false);}assert(false);return 1;}`, ""},
	})
}

func TestArrayFieldForCache(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/array_field_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(source))
}
