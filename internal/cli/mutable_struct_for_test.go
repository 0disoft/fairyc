package cli

import (
	"os"
	"strings"
	"testing"
)

func TestMutableStructForCheck(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/mutable_struct_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	writeVecI64ViewWorkspace(t, root, string(s))
	code, _, diag := runForTest([]string{"check"}, root)
	if code != 0 {
		t.Fatalf("check: %d %s", code, diag)
	}
}
func TestMutableStructForNative(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/mutable_struct_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=9223372036854775807});let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`
	cases := []struct{ name, source, trap string }{{"stored-loop", string(s), ""},
		{"checked-overflow", strings.Replace(base, "ACTION", `for(let ref mut Token item in view){item.value+=1;}`, 1), "integer-overflow"},
		{"body-trap", strings.Replace(base, "ACTION", `for(let ref mut Token item in view){item.value={assert(false);1};}`, 1), "assertion-failed"},
		{"nested-independent", `module main;import std::vec as vec;struct Token{i64 value;}fn void nested(ref mut Token[] left,ref mut Token[] right){for(let ref mut Token a in left){for(let ref mut Token b in right){b.value+=a.value;}a.value+=1;}}fn i32 main(){var vec::Vec<Token> a=vec::empty<Token>();var vec::Vec<Token> b=vec::empty<Token>();discard vec::push<Token>(ref mut a,Token{.value=2});discard vec::push<Token>(ref mut b,Token{.value=7});let ref mut Token[] left=vec::view_mut<Token>(ref mut a);let ref mut Token[] right=vec::view_mut<Token>(ref mut b);nested(left,right);assert(left[0].value==3 && right[0].value==9);return 0;}`, ""},
	}
	runI64ViewNativeCases(t, "mutable-struct-for", cases)
}
func TestMutableStructForRejections(t *testing.T) {
	base := `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=7});let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`
	for name, action := range map[string]string{
		"owner-clear":           "for(let ref mut Token item in view){discard vec::clear<Token>(ref mut owner);}",
		"owner-after-last-item": "for(let ref mut Token item in view){item.value=9;discard vec::clear<Token>(ref mut owner);break;}",
		"parent-live":           "for(let ref mut Token item in view){view[0].value=1;assert(item.value>0);}",
		"retained-return":       "let Option<ref mut Token> saved=first(view);view[0].value=1;switch(saved){case .some(item):{assert(item.value>0);}case .none:{}};",
		"escape-next-iteration": "var Option<ref mut Token> saved=.none;for(let ref mut Token item in view){switch(saved){case .some(old):{assert(old.value>0);}case .none:{}};saved=.some(item);}switch(saved){case .some(item):{assert(item.value>0);}case .none:{}};",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			source := strings.Replace(base, "ACTION", action, 1) + `fn Option<ref mut Token> first(ref mut Token[] items){for(let ref mut Token item in items){return .some(item);}return .none;}`
			writeVecI64ViewWorkspace(t, root, source)
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, "E2101") {
				t.Fatalf("expected E2101: %d %s", code, diag)
			}
		})
	}
}
func TestMutableStructForCache(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/mutable_struct_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(s))
}
