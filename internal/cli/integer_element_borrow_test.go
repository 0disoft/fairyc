package cli

import (
	"os"
	"strings"
	"testing"
)

func TestIntegerElementBorrowNative(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/integer_element_borrow.fy")
	if e != nil {
		t.Fatal(e)
	}
	cases := []struct{ name, source, trap string }{{"integer-matrix", string(s), ""}}
	base := `module main;fn i32 main(){var i64[2] data=[7,9223372036854775807];let ref mut i64[] view=ref mut data;ACTION return 0;}`
	for _, tc := range []struct{ name, action, trap string }{
		{"end", `let ref i64 item=ref view[2];assert(*item==0);`, "bounds"},
		{"huge", `let ref mut i64 item=ref mut view[18446744073709551615];*item=0;`, "bounds"},
		{"empty", `let ref mut i64[] empty=view[2..];let ref mut i64 item=ref mut empty[0];*item=0;`, "bounds"},
		{"index-trap", `let ref mut i64 item=ref mut view[{assert(false);0}];*item=0;`, "assertion-failed"},
		{"overflow", `let ref mut i64 item=ref mut view[1];*item=*item+1;`, "integer-overflow"},
	} {
		cases = append(cases, struct{ name, source, trap string }{tc.name, strings.Replace(base, "ACTION", tc.action, 1), tc.trap})
	}
	cases = append(cases, struct{ name, source, trap string }{"selected-before-index", `module main;fn i32 main(){var i64[1] before=[7];var i64[1] after=[8];var ref mut i64[] view=ref mut before;let ref mut i64[] other=ref mut after;let ref mut i64 item=ref mut view[{view=other;0}];*item=42;assert(before[0]==42 && after[0]==8);return 0;}`, ""})
	cases = append(cases, struct{ name, source, trap string }{"selected-before-rhs", `module main;fn i32 main(){var i64[1] before=[7];var i64[1] after=[8];let ref mut i64[] a=ref mut before;let ref mut i64[] b=ref mut after;var ref mut i64 item=ref mut a[0];let ref mut i64 other=ref mut b[0];*item={item=other;42};assert(before[0]==42 && after[0]==8);return 0;}`, ""})
	runI64ViewNativeCases(t, "integer-element-borrow", cases)
}
