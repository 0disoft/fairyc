package cli

import (
	"os"
	"strings"
	"testing"
)

func TestMutableFieldBorrowNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/mutable_field_borrow.fy")
	if err != nil {
		t.Fatal(err)
	}
	runI64ViewNativeCases(t, "mutable-field-borrow", []struct{ name, source, trap string }{{"fields-generations", strings.ReplaceAll(string(source), "return 42;", "return 0;"), ""}, {"checked-trap", `module main;struct Span{i64 start;}struct Token{Span position;}fn void edit(ref mut Span s){s.start+=1;}fn i32 main(){var Token t=Token{.position=Span{.start=9223372036854775807}};let ref mut Span s=ref mut t.position;edit(s);return 0;}`, "integer-overflow"}})
}
func TestMutableFieldBorrowCache(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/mutable_field_borrow.fy")
	if err != nil {
		t.Fatal(err)
	}
	checkStructVecViewCache(t, string(source))
}
