package cli

import (
	"os"
	"strings"
	"testing"
)

func TestMutableIntegerForNative(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/mutable_integer_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	cases := []struct{ name, source, trap string }{{"stored-matrix", string(s), ""}}
	for _, tc := range []struct{ typ, max string }{{"u8", "255"}, {"i32", "2147483647"}, {"i64", "9223372036854775807"}, {"u32", "4294967295"}, {"usize", "18446744073709551615"}} {
		cases = append(cases, struct{ name, source, trap string }{tc.typ + "-overflow", "module main;fn i32 main(){var " + tc.typ + "[1] data=[" + tc.max + "];let ref mut " + tc.typ + "[] view=ref mut data;for(let ref mut " + tc.typ + " item in view){*item=*item+1;}return 0;}", "integer-overflow"})
	}
	cases = append(cases, struct{ name, source, trap string }{"body-trap", `module main;fn i32 main(){var i64[1] data=[7];let ref mut i64[] view=ref mut data;for(let ref mut i64 item in view){*item={assert(false);9};}return 0;}`, "assertion-failed"})
	cases = append(cases, struct{ name, source, trap string }{"vec-owner", `module main;import std::vec as vec;fn i32 main(){var vec::Vec<i64> owner=vec::empty<i64>();discard vec::push<i64>(ref mut owner,4000000000);discard vec::push<i64>(ref mut owner,7);let ref mut i64[] view=vec::view_mut_i64(ref mut owner);for(let ref mut i64 item in view){*item=*item+2;}assert(view[0]==4000000002 && view[1]==9);discard vec::clear<i64>(ref mut owner);assert(vec::len<i64>(ref owner)==0);return 0;}`, ""})
	runI64ViewNativeCases(t, "mutable-integer-for", cases)
}
func TestMutableIntegerForRejections(t *testing.T) {
	base := `module main;import std::vec as vec;fn i32 main(){var vec::Vec<i64> owner=vec::empty<i64>();discard vec::push<i64>(ref mut owner,7);let ref mut i64[] view=vec::view_mut_i64(ref mut owner);ACTION return 0;}`
	for name, action := range map[string]string{
		"owner-clear":           "for(let ref mut i64 item in view){discard vec::clear<i64>(ref mut owner);}",
		"owner-after-last-use":  "for(let ref mut i64 item in view){*item=9;discard vec::clear<i64>(ref mut owner);break;}",
		"owner-push":            "for(let ref mut i64 item in view){discard vec::push<i64>(ref mut owner,8);}",
		"retained-return":       "let Option<ref mut i64> saved=first(view);view[0]=1;switch(saved){case .some(item):{assert(*item>0);}case .none:{}};",
		"escape-next-iteration": "var Option<ref mut i64> saved=.none;for(let ref mut i64 item in view){switch(saved){case .some(old):{assert(*old>0);}case .none:{}};saved=.some(item);}switch(saved){case .some(item):{assert(*item>0);}case .none:{}};",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, strings.Replace(base, "ACTION", action, 1)+`fn Option<ref mut i64> first(ref mut i64[] items){for(let ref mut i64 item in items){return .some(item);}return .none;}`)
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, "E2101") {
				t.Fatalf("expected E2101: %d %s", code, diag)
			}
		})
	}
}
