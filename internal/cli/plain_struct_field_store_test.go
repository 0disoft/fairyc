package cli

import (
	"os"
	"testing"
)

func TestPlainStructFieldStoreNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/plain_struct_field_store.fy")
	if err != nil {
		t.Fatal(err)
	}
	runI64ViewNativeCases(t, "plain-struct-field-store", []struct{ name, source, trap string }{
		{"copy-order-selection", string(source), ""},
		{"rhs-trap", `module main;struct Span{i64 start;i64 end;}struct Owner{Span position;}fn Span bad(i64 seed){var i64 n=seed;n+=1;return Span{.start=n,.end=0};}fn i32 main(){var Owner s=Owner{.position=Span{.start=3,.end=4}};s.position=bad(9223372036854775807);return 0;}`, "integer-overflow"},
	})
}
func TestPlainStructFieldStoreCache(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/plain_struct_field_store.fy")
	if err != nil {
		t.Fatal(err)
	}
	checkStructVecViewCache(t, string(source))
}
