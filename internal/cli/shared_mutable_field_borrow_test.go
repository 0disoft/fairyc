package cli

import (
	"os"
	"strings"
	"testing"
)

func TestSharedMutableFieldBorrowNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/shared_mutable_field_borrow.fy")
	if err != nil {
		t.Fatal(err)
	}
	runI64ViewNativeCases(t, "shared-mutable-field-borrow", []struct{ name, source, trap string }{
		{"shared-fields-origins", strings.ReplaceAll(string(source), "return 42;", "return 0;"), ""},
		{"post-release-trap", `module main;import std::process as process;struct Span{i64 start;}struct Token{Span position;}fn void edit(ref mut Token t,i64 delta){let ref i64 before=ref t.position.start;assert(*before==9223372036854775807);t.position.start+=delta;}fn i32 main(){var Token t=Token{.position=Span{.start=9223372036854775807}};let i64 delta=if(process::arg_count()>0){1}else{0};assert(delta>0);edit(ref mut t,delta);return 0;}`, "integer-overflow"},
	})
}
func TestSharedMutableFieldBorrowCache(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/shared_mutable_field_borrow.fy")
	if err != nil {
		t.Fatal(err)
	}
	checkStructVecViewCache(t, string(source))
}
