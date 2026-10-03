package cli

import (
	"strings"
	"testing"
)

func nestedStructSource(action string) string {
	source := strings.Replace(structVecViewBase, "struct Token {u32 kind; usize offset; i64 value; bool active;}", "struct Span {usize start; usize end;} struct Position {Span span; u32 line;} struct Token {u32 kind; Position position; i64 value; bool active;}", 1)
	source = strings.Replace(source, ".offset=3", ".position=Position{.span=Span{.start=3,.end=7},.line=2}", 1)
	return strings.Replace(source, "ACTION", action, 1)
}

func TestStructNestedRejections(t *testing.T) {
	for name, leaf := range map[string]string{
		"empty": "struct Span {}", "pointer": "struct Span {ptr mut i32 address;}",
		"borrow": "struct Span {ref i32 value;}", "array": "struct Span {i32[2] values;}",
		"resource": "resource Span {cleanup {}}", "enum": "enum Span {one,two}",
		"option": "struct Span {Option<i32> value;}", "result": "struct Span {Result<i32,u32> value;}",
		"self-cycle": "struct Span {Span value;}", "indirect-cycle": "struct Span {Child value;} struct Child {Span value;}",
	} {
		t.Run(name, func(t *testing.T) {
			source := "module main;import std::vec as vec;" + leaf + " struct Token {Span position;} fn i32 main(){let vec::Vec<Token> owner=vec::empty<Token>();let ref Token[] view=vec::view<Token>(ref owner);for(let Token item in view){}return 0;}"
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, source)
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" {
				t.Fatalf("accepted: %d %s", code, diag)
			}
		})
	}
	for _, item := range []struct{ name, action, code string }{
		{"loop-mutation", "for(let Token item in view){discard vec::clear<Token>(ref mut owned);}", "E2101"},
		{"loop-move", "for(let Token item in view){return take(owned);}", "E2105"},
		{"nested-field-store", "view[0].position.span.start=9;", ""},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, nestedStructSource(item.action))
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" || item.code != "" && !strings.Contains(diag, item.code) {
				t.Fatalf("expected %s: %d %s", item.code, code, diag)
			}
		})
	}
}

func TestStructNestedNative(t *testing.T) {
	copy := nestedStructSource("var Token copy=view[0];var Position position=copy.position;var Span span=position.span;span.start=9;copy=Token{.kind=copy.kind,.position=Position{.span=span,.line=position.line},.value=copy.value,.active=copy.active};assert(copy.position.span.start==9 && view[0].position.span.start==3);assert(view[0].position.span.end==7 && view[0].position.line==2);let ref Token[] part=tail(ref owned)[0..1];var i64 sum=0;for(let Token item in part){assert(item.position.span.start==3);sum+=item.value;}assert(sum==4000000000);let Token saved=view[0];discard vec::clear<Token>(ref mut owned);assert(saved.position.span.start==3 && saved.position.span.end==7 && saved.value==4000000000 && saved.active);")
	copy = strings.Replace(copy, "assert(view[0].kind == 1);", "", 1)
	cases := []struct{ name, source, trap string }{
		{"deep-copy-loop-release", copy, ""},
		{"empty", `module main;import std::vec as vec;struct Span {i64 start; i64 end;} struct Token {Span span;} fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();let ref Token[] view=vec::view<Token>(ref owner);for(let Token item in view){assert(false);}assert(view[0..0].is_empty);discard vec::reserve<Token>(ref mut owner,8);return 0;}`, ""},
		{"shared-child-types", `module main;import std::vec as vec;struct Span {i64 start; i64 end;} struct Token {Span left;Span right;u8 kind;} fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.left=Span{.start=-4000000000,.end=0},.right=Span{.start=0,.end=9223372036854775807},.kind=7});let ref Token[] view=vec::view<Token>(ref owner);for(let Token item in view){assert(item.left.start==-4000000000 && item.right.end==9223372036854775807 && item.kind==7);}return 0;}`, ""},
		{"end-read", nestedStructSource("let Token copy=view[1];"), "bounds"},
		{"reversed", nestedStructSource("let ref Token[] part=view[1..0];"), "bounds"},
	}
	runI64ViewNativeCases(t, "struct-nested", cases)
}
