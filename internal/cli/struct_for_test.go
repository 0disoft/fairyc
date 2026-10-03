package cli

import (
	"strings"
	"testing"
)

func structForSource(action string) string {
	source := strings.Replace(structVecViewBase, "ACTION", action, 1)
	return strings.Replace(source, "assert(view[0].kind == 1);", "", 1)
}

func TestStructForRejections(t *testing.T) {
	for _, item := range []struct{ name, action, code string }{
		{"clear", "for(let Token item in view){discard vec::clear<Token>(ref mut owned);}", "E2101"},
		{"clear-break", "for(let Token item in view){discard vec::clear<Token>(ref mut owned);break;}", "E2101"},
		{"clear-continue", "for(let Token item in view){discard vec::clear<Token>(ref mut owned);continue;}", "E2101"},
		{"reserve-noop", "for(let Token item in view){discard vec::reserve<Token>(ref mut owned,0);}", "E2101"},
		{"push-copy", "for(let Token item in view){discard vec::push<Token>(ref mut owned,item);}", "E2101"},
		{"replace", "for(let Token item in view){owned=vec::empty<Token>();}", "E2101"},
		{"move", "for(let Token item in view){discard take(owned);}", "E2105"},
		{"move-early-return", "for(let Token item in view){return take(owned);}", "E2105"},
		{"clear-return", "for(let Token item in view){discard vec::clear<Token>(ref mut owned);return 0;}", "E2101"},
		{"nested-move", "for(let Token item in view){for(let Token nested in view){discard take(owned);}}", "E2105"},
		{"item-store", "for(let Token item in view){item.kind=2;}", ""},
		{"scalar-item", "for(let i64 item in view){}", ""},
		{"wrong-nominal", "for(let Other item in view){}", ""},
		{"mutable-descriptor", "var ref Token[] alias=view;for(let Token item in alias){}", ""},
		{"slice-expression", "for(let Token item in view[..]){}", ""},
		{"call-expression", "for(let Token item in tail(ref owned)){}", ""},
		{"direct-vector", "for(let Token item in owned){}", ""},
		{"move-propagation", "for(let Token item in view){let i32 value=take_result(owned)?;}", "E2105"},
		{"reference-item", "for(let ref Token item in view){}", ""},
		{"mutable-reference-item", "for(let ref mut Token item in view){}", ""},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			source := strings.Replace(structForSource(item.action), "struct Token", "struct Other {u32 kind;usize offset;i64 value;bool active;} struct Token", 1)
			if item.name == "move-propagation" {
				source = strings.Replace(source, "fn i32 main() {", "fn Result<i32,u32> probe() {", 1)
				source = strings.Replace(source, "return 0;\n}", "return .ok(0);\n}", 1)
				source += "fn Result<i32,u32> take_result(vec::Vec<Token> owned){return .err(7);} fn i32 main(){return 0;}"
			}
			writeVecI64ViewWorkspace(t, root, source)
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" || item.code != "" && !strings.Contains(diag, item.code) {
				t.Fatalf("expected %s: %d %s", item.code, code, diag)
			}
		})
	}
}

func TestStructForNative(t *testing.T) {
	cases := []struct{ name, source, trap string }{
		{"singleton-copy", structForSource("var usize count=0;for(let Token item in view){assert(item.value==4000000000 && item.active);var Token copy=item;copy.kind=9;assert(copy.kind==9 && view[0].kind==1);count+=1;}assert(count==1);"), ""},
		{"empty", `module main;import std::vec as vec;struct Token {i32 value;} fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();let ref Token[] view=vec::view<Token>(ref owner);var usize count=0;for(let Token item in view){count+=1;}assert(count==0);discard vec::reserve<Token>(ref mut owner,8);return 0;}`, ""},
		{"parameter-nested-mixed", `module main;import std::vec as vec;pub struct Token {i64 value;} fn i64 total(ref Token[] values){var i64 sum=0;let ref u8[] bytes=b"ab";for(let Token token in values){if(token.value==1){continue;}for(let u8 byte in bytes){assert(byte==97 || byte==98);sum+=token.value;}if(token.value==3){break;}}return sum;} fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=1});discard vec::push<Token>(ref mut owner,Token{.value=2});discard vec::push<Token>(ref mut owner,Token{.value=3});discard vec::push<Token>(ref mut owner,Token{.value=4});let ref Token[] view=vec::view<Token>(ref owner);assert(total(view)==10);var usize count=0;for(let Token outer in view){for(let Token inner in view){assert(inner.value>0 && outer.value>0);count+=1;}}assert(count==16);discard vec::clear<Token>(ref mut owner);return 0;}`, ""},
		{"early-return-and-release", `module main;import std::vec as vec;struct Token {i64 value;} fn Token first(ref Token[] values){for(let Token token in values){return token;}return Token{.value=0};} fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=4000000000});let ref Token[] view=vec::view<Token>(ref owner);let Token copy=first(view);for(let Token token in view){assert(token.value==copy.value);break;}discard vec::clear<Token>(ref mut owner);assert(copy.value==4000000000);let vec::Vec<Token> moved=owner;return 0;}`, ""},
		{"returned-tagged-subviews", structForSource("let ref Token[] returned=tail(ref owned);let Option<ref Token[]> payload=tagged(ref owned);var usize count=0;for(let Token item in returned){count+=1;assert(item.offset==3);}switch(payload){case .some(alias):{let ref Token[] partial=alias[0..1];for(let Token item in partial){count+=1;assert(item.active);}let ref Token[] empty=alias[1..];for(let Token item in empty){assert(false);}}case .none:{assert(false);}};assert(count==2);discard vec::clear<Token>(ref mut owned);"), ""},
		{"error-propagation", `module main;import std::vec as vec;struct Token {i64 value;} fn Result<i32,u32> failure(){return .err(7);} fn Result<usize,u32> visit(ref Token[] values){for(let Token token in values){let i32 result=failure()?;assert(token.value>0);}return .ok(0);} fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=1});let ref Token[] view=vec::view<Token>(ref owner);assert(switch(visit(view)){case .ok(_):{false}case .err(code):{code==7}});discard vec::clear<Token>(ref mut owner);assert(vec::len<Token>(ref owner)==0);return 0;}`, ""},
		{"checked-sum-overflow", `module main;import std::vec as vec;struct Token {i64 value;} fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=9223372036854775807});discard vec::push<Token>(ref mut owner,Token{.value=1});let ref Token[] view=vec::view<Token>(ref owner);var i64 sum=0;for(let Token item in view){sum+=item.value;}return 0;}`, "integer-overflow"},
	}
	runI64ViewNativeCases(t, "struct-for", cases)
}
