package cli

import (
	"os"
	"testing"
)

func TestStructArrayStorageNative(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/struct_array_storage.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;struct P{u8[2] bytes;}fn i32 main(){let P p=P{.bytes=[1,2]};`
	runI64ViewNativeCases(t, "struct-array-storage", []struct{ name, source, trap string }{
		{"matrix", string(source), ""}, {"past-end", base + `let u8 x=p.bytes[2];return 0;}`, "bounds"},
	})
}
func TestStructArrayStorageCache(t *testing.T) {
	source, e := os.ReadFile("../compiler/testdata/struct_array_storage.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(source))
}
