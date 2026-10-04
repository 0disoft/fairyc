package cli

import (
	"os"
	"testing"
)

func TestStructArrayNative(t *testing.T) {
	src, err := os.ReadFile("../compiler/testdata/struct_arrays.fy")
	if err != nil {
		t.Fatal(err)
	}
	base := `module main;struct Token{i64 value;}fn i32 main(){var Token[2] data=[Token{.value=4000000000};2];`
	runI64ViewNativeCases(t, "struct-array", []struct{ name, source, trap string }{
		{"matrix", string(src), ""},
		{"index-end", base + `let Token copy=data[2];return 0;}`, "bounds"},
		{"store-end", base + `data[2]=Token{.value=7};return 0;}`, "bounds"},
		{"empty-view", base + `let ref Token[] view=ref data;let Token copy=view[2..][0];return 0;}`, "bounds"},
		{"reversed-view", base + `let ref Token[] view=ref data;let ref Token[] bad=view[2..1];return 0;}`, "bounds"},
	})
}
func TestStructArrayCache(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/struct_arrays.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(s))
}
