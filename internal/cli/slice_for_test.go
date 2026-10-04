package cli

import (
	"os"
	"testing"
)

func TestSliceForNative(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/slice_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;fn i32 main(){var i64[2] data=[1,2];`
	runI64ViewNativeCases(t, "slice-for", []struct{ name, source, trap string }{
		{"matrix", string(source), ""},
		{"generic", `module main;fn void edit<T>(ref mut T[] view,T value){for(let ref mut T item in view[1..]){*item=value;}}fn i32 main(){var i64[2] data=[1,2];let ref mut i64[] view=ref mut data;edit<i64>(view,4000000000);assert(view[0]==1&&view[1]==4000000000);return 0;}`, ""},
		{"past-end", base + `for(let i64 item in data[..3]){assert(false);}return 0;}`, "bounds"},
		{"reversed", base + `for(let ref mut i64 item in data[2..1]){assert(false);}return 0;}`, "bounds"},
		{"start-past-end", base + `let ref i64[] view=ref data;for(let ref i64 item in view[3..]){assert(false);}return 0;}`, "bounds"},
		{"overflow", `module main;fn i32 main(){var i64[1] data=[9223372036854775807];for(let ref mut i64 item in data[..]){*item=*item+1;}return 0;}`, "integer-overflow"},
		{"bound-return", base + `for(let i64 item in data[{return 0;0}..{assert(false);1}]){assert(false);}assert(false);return 1;}`, ""},
		{"view-bound-return", base + `let ref i64[] view=ref data;for(let i64 item in view[{return 0;0}..{assert(false);1}]){assert(false);}assert(false);return 1;}`, ""},
	})
}
func TestSliceForCache(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/slice_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(source))
}
