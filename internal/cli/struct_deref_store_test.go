package cli

import (
	"os"
	"strings"
	"testing"
)

func TestStructDerefStoreCheckAndCache(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/struct_deref_store.fy")
	if e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	writeVecI64ViewWorkspace(t, root, string(s))
	code, _, diag := runForTest([]string{"check"}, root)
	if code != 0 {
		t.Fatalf("check: %d %s", code, diag)
	}
	checkStructVecViewCache(t, string(s))
}
func TestStructDerefStoreNative(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/struct_deref_store.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=7});let ref mut Token[] items=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`
	cases := []struct{ name, source, trap string }{{"stored-copy", string(s), ""}}
	for _, tc := range []struct{ name, action, trap string }{
		{"bounds-before-store", `let ref mut Token item=ref mut items[1];*item=Token{.value=9};`, "bounds"},
		{"rhs-assertion", `let ref mut Token item=ref mut items[0];*item={assert(false);Token{.value=9}};`, "assertion-failed"},
		{"rhs-overflow", `let ref mut Token item=ref mut items[0];*item=Token{.value=9223372036854775807+item.value};`, "integer-overflow"},
	} {
		cases = append(cases, struct{ name, source, trap string }{tc.name, strings.Replace(base, "ACTION", tc.action, 1), tc.trap})
	}
	cases = append(cases, struct{ name, source, trap string }{"selected-before-rebind", `module main;struct T{i64 value;}fn i32 main(){var T a=T{.value=1};var T b=T{.value=2};var ref mut T target=ref mut a;let ref mut T other=ref mut b;*target={target=other;T{.value=4000000000}};assert(a.value==4000000000 && b.value==2);return 0;}`, ""})
	runI64ViewNativeCases(t, "struct-deref-store", cases)
}
