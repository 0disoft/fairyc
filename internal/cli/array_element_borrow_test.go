package cli

import (
	"os"
	"strings"
	"testing"
)

func TestArrayElementBorrowNative(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/array_element_borrow.fy")
	if e != nil {
		t.Fatal(e)
	}
	cases := []struct{ name, source, trap string }{{"stored-matrix", string(s), ""}}
	base := `module main;fn i32 main(){var i64[2] data=[7,9223372036854775807];ACTION return 0;}`
	for _, tc := range []struct{ name, action, trap string }{
		{"first-last", `let ref mut i64 first=ref mut data[0];*first=9;let ref mut i64 last=ref mut data[1];*last=10;assert(data[0]==9 && data[1]==10);`, ""},
		{"end", `let ref i64 item=ref data[2];assert(*item==0);`, "bounds"},
		{"huge", `let ref mut i64 item=ref mut data[18446744073709551615];*item=0;`, "bounds"},
		{"index-trap", `let ref mut i64 item=ref mut data[{assert(false);0}];*item=9;`, "assertion-failed"},
		{"overflow", `let ref mut i64 item=ref mut data[1];*item=*item+1;`, "integer-overflow"},
		{"rhs-trap", `let ref mut i64 item=ref mut data[0];*item={assert(false);9};`, "assertion-failed"},
	} {
		cases = append(cases, struct{ name, source, trap string }{tc.name, strings.Replace(base, "ACTION", tc.action, 1), tc.trap})
	}
	runI64ViewNativeCases(t, "array-element-borrow", cases)
}
func TestArrayElementBorrowCheckRejections(t *testing.T) {
	for name, src := range map[string]string{
		"parent-live":       `fn void bad(){var i64[2] data=[1,2];let ref mut i64 item=ref mut data[0];data[1]=9;assert(*item==1);}`,
		"index-replacement": `fn void bad(){var i64[2] data=[1,2];var i64[2] other=[7,8];let ref mut i64 item=ref mut data[{data=other;0}];*item=9;}`,
		"local-escape":      `fn ref mut i64 bad(){var i64[2] data=[1,2];return ref mut data[0];}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, "module main;"+src+"fn i32 main(){return 0;}")
			code, _, diag := runForTest([]string{"check"}, root)
			want := "E2101"
			if name == "local-escape" {
				want = "E2102"
			}
			if code != ExitSourceDiagnostics || !strings.Contains(diag, want) {
				t.Fatalf("expected %s: %d %s", want, code, diag)
			}
		})
	}
}
