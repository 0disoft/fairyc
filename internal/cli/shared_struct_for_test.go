package cli

import (
	"os"
	"strings"
	"testing"
)

const sharedStructForBase = `module main;import std::vec as vec;struct Span{usize start;}struct Token{i64 value;Span position;}
fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();assert(vec::push<Token>(ref mut owner,Token{.value=7,.position=Span{.start=1}}).is_ok);let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`

func TestSharedStructForNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/shared_struct_for.fy")
	if err != nil {
		t.Fatal(err)
	}
	runI64ViewNativeCases(t, "shared-struct-for", []struct{ name, source, trap string }{
		{"stored-matrix", string(source), ""},
		{"singleton", strings.Replace(sharedStructForBase, "ACTION", `var usize count=0;for(let ref Token item in view){assert(item.value==7 && item.position.start==1);count+=1;}assert(count==1);view[0].value=9;`, 1), ""},
		{"body-trap", strings.Replace(sharedStructForBase, "ACTION", `for(let ref Token item in view){assert(item.value==8);}`, 1), "assertion-failed"},
	})
}

func TestSharedStructForCheckRejections(t *testing.T) {
	for _, tc := range []struct{ name, action, code string }{
		{"owner-clear-after-last-item", `for(let ref Token item in view){assert(item.value>0);discard vec::clear<Token>(ref mut owner);break;}`, "E2101"},
		{"owner-move", `for(let ref Token item in view){let vec::Vec<Token> moved=owner;assert(item.value>0);}`, "E2105"},
		{"parent-write-after-last-item", `for(let ref Token item in view){assert(item.value>0);view[0].value=9;}`, "E2101"},
		{"nested-field-store", `for(let ref Token item in view){item.position.start=9;}`, "E1008"},
		{"retained-return-write", `let Option<ref Token> found=first(view);view[0].value=9;switch(found){case .some(item):{assert(item.value>0);}case .none:{}};`, "E2101"},
		{"owner-after-return-break", `for(let ref Token item in view){break;}let Option<ref Token> found=first(view);discard vec::clear<Token>(ref mut owner);switch(found){case .some(item):{assert(item.value>0);}case .none:{}};`, "E2101"},
		{"local-escape", ``, "E2102"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(sharedStructForBase, "ACTION", tc.action, 1) + `fn Option<ref Token> first(ref mut Token[] items){for(let ref Token item in items){return .some(item);}return .none;}`
			if tc.name == "local-escape" {
				source = `module main;import std::vec as vec;struct Token{i64 value;}fn Option<ref Token> bad(){var vec::Vec<Token> owner=vec::empty<Token>();let ref Token[] view=vec::view<Token>(ref owner);for(let ref Token item in view){return .some(item);}return .none;}fn i32 main(){return 0;}`
			}
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, source)
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, tc.code) {
				t.Fatalf("expected %s: %d %s", tc.code, code, diag)
			}
		})
	}
}
