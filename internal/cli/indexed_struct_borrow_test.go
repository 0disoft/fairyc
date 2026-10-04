package cli

import (
	"os"
	"strings"
	"testing"
)

func TestIndexedStructBorrowCheck(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/indexed_struct_borrow.fy")
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

func TestIndexedStructBorrowNative(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/indexed_struct_borrow.fy")
	if e != nil {
		t.Fatal(e)
	}
	base := `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=7});let ref mut Token[] items=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`
	cases := []struct{ name, source, trap string }{{"stored-borrows", string(s), ""}}
	for _, c := range []struct{ name, action, trap string }{
		{"shared-end", `let ref Token item=ref items[1];assert(item.value==7);`, "bounds"},
		{"mutable-huge", `let ref mut Token item=ref mut items[18446744073709551615];item.value=1;`, "bounds"},
		{"empty", `let ref mut Token[] empty=items[1..];let ref Token item=ref empty[0];assert(item.value==7);`, "bounds"},
		{"index-trap", `let ref mut Token item=ref mut items[{assert(false);0}];item.value=1;`, "assertion-failed"},
		{"checked-overflow", `let ref mut Token item=ref mut items[0];item.value=9223372036854775807;item.value+=1;`, "integer-overflow"},
	} {
		cases = append(cases, struct{ name, source, trap string }{c.name, strings.Replace(base, "ACTION", c.action, 1), c.trap})
	}
	runI64ViewNativeCases(t, "indexed-struct-borrow", cases)
}

func TestIndexedStructBorrowRejections(t *testing.T) {
	base := `module main;import std::vec as vec;struct Token{i64 value;}fn ref mut Token chosen(ref mut Token[] items){return ref mut items[0];}fn Option<ref mut Token> tagged(ref mut Token[] items){return .some(chosen(items));}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=7});let ref mut Token[] items=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`
	for name, action := range map[string]string{
		"owner-clear":     `let ref mut Token item=ref mut items[0];discard vec::clear<Token>(ref mut owner);assert(item.value>0);`,
		"owner-read":      `let ref mut Token item=ref mut items[0];assert(vec::len<Token>(ref owner)>0);assert(item.value>0);`,
		"returned":        `let ref mut Token item=chosen(items);items[0].value=1;assert(item.value>0);`,
		"tagged":          `let Option<ref mut Token> result=tagged(items);items[0].value=1;switch(result){case .some(item):{assert(item.value>0);}case .none:{}};`,
		"effectful-index": `let ref mut Token item=ref mut items[{discard vec::clear<Token>(ref mut owner);0}];assert(item.value>0);`,
		"index-sibling":   `let ref mut Token item=ref mut items[{let ref Token other=ref items[0];assert(other.value>0);0}];assert(item.value>0);`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, strings.Replace(base, "ACTION", action, 1))
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, "E2101") {
				t.Fatalf("expected E2101: %d %s", code, diag)
			}
		})
	}
	for name, decl := range map[string]string{
		"local-shared":  `fn ref Token bad(){var vec::Vec<Token> owner=vec::empty<Token>();let ref Token[] items=vec::view<Token>(ref owner);return ref items[0];}`,
		"local-mutable": `fn ref mut Token bad(){var vec::Vec<Token> owner=vec::empty<Token>();let ref mut Token[] items=vec::view_mut<Token>(ref mut owner);return ref mut items[0];}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, "module main;import std::vec as vec;struct Token{i64 value;}"+decl+"fn i32 main(){return 0;}")
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, "E2102") {
				t.Fatalf("expected E2102: %d %s", code, diag)
			}
		})
	}
}
func TestIndexedStructBorrowCache(t *testing.T) {
	s, e := os.ReadFile("../compiler/testdata/indexed_struct_borrow.fy")
	if e != nil {
		t.Fatal(e)
	}
	checkStructVecViewCache(t, string(s))
}

func TestIndexedStructBorrowSelectionNative(t *testing.T) {
	const source = `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> original=vec::empty<Token>();var vec::Vec<Token> other=vec::empty<Token>();discard vec::push<Token>(ref mut original,Token{.value=7});
 var ref mut Token[] items=vec::view_mut<Token>(ref mut original);let ref mut Token[] replacement=vec::view_mut<Token>(ref mut other);
 let ref mut Token item=ref mut items[{items=replacement;0}];item.value=42;assert(item.value==42);
 let ref Token[] before=vec::view<Token>(ref original);let ref Token[] after=vec::view<Token>(ref other);assert(before[0].value==42 && after.is_empty);return 0;}`
	runI64ViewNativeCases(t, "indexed-struct-borrow-selection", []struct{ name, source, trap string }{{"rebind-in-index", source, ""}})
}
