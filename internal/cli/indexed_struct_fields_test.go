package cli

import (
	"os"
	"strings"
	"testing"
)

func TestIndexedStructFieldsNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/indexed_struct_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	base := `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=9223372036854775807});let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`
	cases := []struct{ name, source, trap string }{{"fields", string(source), ""}}
	for _, item := range []struct{ name, action, trap string }{
		{"bounds-before-rhs", `view[1].value={assert(false);1};`, "bounds"},
		{"rhs-trap", `view[0].value={assert(false);1};`, "assertion-failed"},
		{"overflow", `view[0].value+=1;`, "integer-overflow"},
		{"borrow-bounds", `let ref i64 value=ref view[1].value;assert(*value==1);`, "bounds"},
		{"borrow-huge", `let ref mut i64 value=ref mut view[18446744073709551615].value;*value=1;`, "bounds"},
		{"nested-replacement", `view[0].value=-7;let ref i64 shared=ref view[0].value;assert(*shared==-7);`, ""},
	} {
		cases = append(cases, struct{ name, source, trap string }{item.name, strings.Replace(base, "ACTION", item.action, 1), item.trap})
	}
	runI64ViewNativeCases(t, "indexed-struct-fields", cases)
}
func TestIndexedStructFieldsCheck(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/indexed_struct_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeVecI64ViewWorkspace(t, root, string(source))
	code, _, diag := runForTest([]string{"check"}, root)
	if code != 0 {
		t.Fatalf("check: %d %s", code, diag)
	}
}

func TestIndexedStructFieldsRejections(t *testing.T) {
	base := `module main;import std::vec as vec;struct Span{usize start;usize end;}struct Token{i64 value;Span position;}fn ref mut i64 selected(ref mut Token[] view){return ref mut view[0].value;}fn Option<ref mut i64> tagged(ref mut Token[] view){return .some(selected(view));}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=7,.position=Span{.start=1,.end=2}});let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`
	for name, action := range map[string]string{
		"parent":       `let ref mut i64 child=ref mut view[0].value;view[0].value=2;assert(*child==7);`,
		"other-field":  `let ref mut i64 child=ref mut view[0].value;view[0].position.start=2;assert(*child==7);`,
		"other-index":  `let ref mut i64 child=ref mut view[0].value;let ref mut usize other=ref mut view[1].position.start;assert(*child==7);`,
		"shared-child": `let ref i64 child=ref view[0].value;view[0].value=2;assert(*child==7);`,
		"owner":        `let ref mut i64 child=ref mut view[0].value;discard vec::clear<Token>(ref mut owner);assert(*child==7);`,
		"returned":     `let ref mut i64 child=selected(view);view[0].value=2;assert(*child==7);`,
		"tagged":       `let Option<ref mut i64> child=tagged(view);view[0].value=2;switch(child){case .some(value):{assert(*value==7);}case .none:{}};`,
		"index-effect": `view[{discard vec::clear<Token>(ref mut owner);0}].value=2;`,
		"rhs-effect":   `view[0].value={discard vec::clear<Token>(ref mut owner);2};`,
		"alias":        `let ref mut Token[] alias=view;let ref mut i64 child=ref mut view[0].value;alias[0].value=2;assert(*child==7);`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, strings.Replace(base, "ACTION", action, 1))
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, "E2101") {
				t.Fatalf("conflict: %d %s", code, diag)
			}
		})
	}
	for name, source := range map[string]string{
		"shared-store":       `fn void bad(ref Token[] view){view[0].value=1;}`,
		"shared-exclusive":   `fn void bad(ref Token[] view){let ref mut i64 child=ref mut view[0].value;}`,
		"temporary":          `fn void bad(ref mut vec::Vec<Token> owner){let ref i64 child=ref vec::view_mut<Token>(owner)[0].value;}`,
		"temporary-argument": `fn void consume(ref i64 value){} fn void bad(ref mut vec::Vec<Token> owner){consume(ref vec::view_mut<Token>(owner)[0].value);}`,
		"local-escape":       `fn ref i64 bad(){var vec::Vec<Token> owner=vec::empty<Token>();let ref Token[] view=vec::view<Token>(ref owner);return ref view[0].value;}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, "module main;import std::vec as vec;struct Token{i64 value;}"+source+"fn i32 main(){return 0;}")
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" {
				t.Fatalf("accepted: %d %s", code, diag)
			}
		})
	}
}
func TestIndexedStructFieldsCache(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/indexed_struct_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	checkStructVecViewCache(t, string(source))
}

func TestIndexedStructFieldsSelectionNative(t *testing.T) {
	const base = `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){
 var vec::Vec<Token> first=vec::empty<Token>();var vec::Vec<Token> second=vec::empty<Token>();
 discard vec::push<Token>(ref mut first,Token{.value=7});discard vec::push<Token>(ref mut second,Token{.value=9});
 var ref mut Token[] view=vec::view_mut<Token>(ref mut first);
 let ref mut Token[] next=vec::view_mut<Token>(ref mut second);
 ACTION
 let ref Token[] original=vec::view<Token>(ref first);let ref Token[] other=vec::view<Token>(ref second);
 assert(original[0].value==42 && other[0].value==9);return 0;}`
	runI64ViewNativeCases(t, "indexed-fields-selection", []struct{ name, source, trap string }{
		{"index-rebind", strings.Replace(base, "ACTION", `view[{view=next;0}].value=42;`, 1), ""},
		{"rhs-rebind", strings.Replace(base, "ACTION", `view[0].value={view=next;42};`, 1), ""},
	})
}

func TestIndexedStructFieldsIndexTrapNative(t *testing.T) {
	const base = `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=7});let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`
	runI64ViewNativeCases(t, "indexed-fields-index-trap", []struct{ name, source, trap string }{
		{"store", strings.Replace(base, "ACTION", `view[{assert(false);0}].value=42;`, 1), "assertion-failed"},
		{"borrow", strings.Replace(base, "ACTION", `let ref mut i64 value=ref mut view[{assert(false);0}].value;*value=42;`, 1), "assertion-failed"},
	})
}
