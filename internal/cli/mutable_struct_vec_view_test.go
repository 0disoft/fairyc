package cli

import (
	"os"
	"strings"
	"testing"
)

const mutableStructViewBase = `module main;import std::vec as vec;
struct Span {usize start;usize end;}
struct Token {u32 kind;Span position;i64 value;bool active;}
fn i64 read(ref Token[] values){return values[0].value;}
fn ref Token[] retain(ref Token[] values){return values[..];}
fn Token token(i64 value) {return Token{.kind=2,.position=Span{.start=3,.end=8},.value=value,.active=true};}
fn ref mut Token[] tail(ref mut vec::Vec<Token> owner,usize start){return vec::view_mut<Token>(owner)[start..];}
fn Option<ref mut Token[]> tagged(ref mut vec::Vec<Token> owner){return .some(tail(owner,0));}
fn i32 main(){
 var vec::Vec<Token> owner=vec::empty<Token>();
 assert(switch(vec::push<Token>(ref mut owner,token(4000000000))){case .ok(_):{true}case .err(_):{false}});
 assert(switch(vec::push<Token>(ref mut owner,token(7))){case .ok(_):{true}case .err(_):{false}});
 let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);
 ACTION
 return 0;
}`

func mutableStructSource(action string) string {
	return strings.Replace(mutableStructViewBase, "ACTION", action, 1)
}

func TestMutableStructVecViewNative(t *testing.T) {
	fixture, err := os.ReadFile("../compiler/testdata/mutable_struct_vec_view.fy")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{
		{"rhs-trap", mutableStructSource(`view[0]={assert(false);token(42)};`), "assertion-failed"},
		{"whole-replacement", string(fixture), ""},
		{"child-last-use", mutableStructSource(`let ref mut Token[] child=view[1..];child[0]=token(6000000000);assert(child[0].value==6000000000);view[0]=token(42);assert(view[1].value==6000000000 && view[0].value==42);assert(view[2..].is_empty);discard vec::clear<Token>(ref mut owner);assert(vec::len<Token>(ref owner)==0);`), ""},
		{"returned-tagged", strings.Replace(mutableStructSource(`assert(view[0].value==43);`), "let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);", `let Option<ref mut Token[]> result=tagged(ref mut owner);switch(result){case .some(child):{child[0]=token(43);assert(child[0].value==43);}case .none:{assert(false);}};let ref Token[] view=vec::view<Token>(ref owner);`, 1), ""},
		{"empty-growth", `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();let ref mut Token[] empty=vec::view_mut<Token>(ref mut owner);assert(empty.is_empty && empty[..].is_empty && empty[0..0].len==0);discard vec::reserve<Token>(ref mut owner,4);discard vec::push<Token>(ref mut owner,Token{.value=42});let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);view[0]=Token{.value=4000000000};assert(view[0].value==4000000000);return 0;}`, ""},
	}
	for _, item := range []struct{ name, action string }{
		{"end-store", `view[2]=token(42);`},
		{"bounds-before-rhs", `view[2]={assert(false);token(42)};`},
		{"empty-store", `let ref mut Token[] child=view[2..];child[0]=token(42);`},
		{"huge-store", `view[18446744073709551615]=token(42);`},
		{"end-read", `let Token copy=view[2];`},
		{"reversed", `let ref mut Token[] child=view[1..0];`},
		{"past-end", `let ref mut Token[] child=view[..3];`},
	} {
		cases = append(cases, struct{ name, source, trap string }{item.name, mutableStructSource(item.action), "bounds"})
	}
	integers := "module main;import std::vec as vec;fn i32 main(){"
	for _, element := range []string{"u8", "i32", "i64", "u32", "usize"} {
		integers += "{var vec::Vec<" + element + "> owner=vec::empty<" + element + ">();discard vec::push<" + element + ">(ref mut owner,7);let ref mut " + element + "[] view=vec::view_mut<" + element + ">(ref mut owner);view[0]=42;assert(view[0]==42);let ref " + element + "[] shared=vec::view<" + element + ">(ref owner);assert(shared[0]==42);}"
	}
	integers += "return 0;}"
	cases = append(cases, struct{ name, source, trap string }{"generic-integer-aliases", integers, ""})
	runI64ViewNativeCases(t, "mutable-struct-vec-view", cases)
}

func TestMutableStructVecViewCache(t *testing.T) {
	checkStructVecViewCache(t, mutableStructSource(`let ref mut Token[] child=view[1..];child[0]=token(42);assert(child[0].value==42);assert(view[1].value==42);`))
}

func TestMutableStructVecViewRejections(t *testing.T) {
	for _, item := range []struct{ name, action, code string }{
		{"owner-read", `assert(vec::len<Token>(ref owner)==2);assert(view.len==2);`, "E2101"},
		{"push", `discard vec::push<Token>(ref mut owner,token(1));assert(view.len==2);`, "E2101"},
		{"reserve-noop", `discard vec::reserve<Token>(ref mut owner,0);assert(view.len==2);`, "E2101"},
		{"truncate-noop", `discard vec::truncate<Token>(ref mut owner,9);assert(view.len==2);`, "E2101"},
		{"clear", `discard vec::clear<Token>(ref mut owner);assert(view.len==2);`, "E2101"},
		{"replace-owner", `owner=vec::empty<Token>();assert(view.len==2);`, "E2101"},
		{"move-owner", `let vec::Vec<Token> moved=owner;assert(view.len==2);`, "E2105"},
		{"parent-child", `let ref mut Token[] child=view[1..];view[0]=token(1);assert(child.len==1);`, "E2101"},
		{"mutable-siblings", `let ref mut Token[] first=view[..1];let ref mut Token[] second=view[1..];assert(first.len==1 && second.len==1);`, "E2101"},
		{"shared-reborrow", `let ref Token[] shared=retain(view);view[0]=token(1);assert(shared.len==2);`, "E2101"},
		{"bounds-effect", `let ref mut Token[] child=view[..{discard vec::clear<Token>(ref mut owner);0}];assert(child.len==0);`, "E2101"},
		{"index-effect", `view[{discard vec::clear<Token>(ref mut owner);0}]=token(1);`, "E2101"},
		{"rhs-effect", `view[0]={discard vec::clear<Token>(ref mut owner);token(1)};`, "E2101"},
		{"shared-field-store", `let ref Token[] shared=vec::view<Token>(ref owner);shared[0].value=1;`, "E1008"},
		{"element-borrow-live-parent", `let ref mut Token item=ref mut view[0];view[0]=token(9);assert(item.value==1);`, "E2101"},
		{"for-in-owner-mutation", `for(let Token item in view){discard vec::clear<Token>(ref mut owner);assert(item.active);}`, "E2101"},
		{"wrong-value", `view[0]=1;`, "E1006"},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, mutableStructSource(item.action))
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" || item.code != "" && !strings.Contains(diag, item.code) {
				t.Fatalf("expected %s: %d %s", item.code, code, diag)
			}
		})
	}
	// Reference parameters have no local root loan. Both direct and
	// returned storage descriptors must still freeze the parent until last use.
	for _, item := range []struct{ name, element, producer string }{
		{"parameter-direct", "Token", "vec::view_mut<Token>(owner)"},
		{"parameter-returned", "Token", "returned(owner)"},
		{"parameter-integer", "i64", "vec::view_mut<i64>(owner)"},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			source := "module main;import std::vec as vec;struct Token{i64 value;}" +
				"fn ref mut Token[] returned(ref mut vec::Vec<Token> owner){return vec::view_mut<Token>(owner);}" +
				"fn void bad(ref mut vec::Vec<" + item.element + "> owner){let ref mut " + item.element + "[] child=" + item.producer +
				";discard vec::clear<" + item.element + ">(owner);assert(child.len==0);}fn i32 main(){return 0;}"
			writeVecI64ViewWorkspace(t, root, source)
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, "E2101") {
				t.Fatalf("expected parent conflict: %d %s", code, diag)
			}
		})
	}
	for name, source := range map[string]string{
		"multiple-origins": `struct Token{i64 value;}fn ref mut Token[] bad(ref mut vec::Vec<Token> a,ref mut vec::Vec<Token> b,bool choose){return if(choose){vec::view_mut<Token>(a)}else{vec::view_mut<Token>(b)};}`,
		"tagged-owner":     `struct Token{i64 value;}fn Option<ref mut Token[]> tagged(ref mut vec::Vec<Token> owner){return .some(vec::view_mut<Token>(owner));}fn void bad(ref mut vec::Vec<Token> owner){let Option<ref mut Token[]> child=tagged(owner);discard vec::clear<Token>(owner);switch(child){case .some(view):{assert(view.len==0);}case .none:{}};}`,
		"local-escape":     `struct Token{i64 value;}fn ref mut Token[] bad(){var vec::Vec<Token> owner=vec::empty<Token>();return vec::view_mut<Token>(ref mut owner);}`,
		"wrong-nominal":    `struct Token{i64 value;}struct Other{i64 value;}fn void bad(ref mut vec::Vec<Token> owner){let ref mut Other[] view=vec::view_mut<Token>(owner);assert(view.len==0);}`,
		"shared-input":     `struct Token{i64 value;}fn void bad(ref vec::Vec<Token> owner){let ref mut Token[] view=vec::view_mut<Token>(owner);assert(view.len==0);}`,
		"nested-pointer":   `struct Span{ptr i64 address;}struct Token{Span position;}fn void bad(ref mut vec::Vec<Token> owner){let ref mut Token[] view=vec::view_mut<Token>(owner);assert(view.len==0);}`,
		"resource":         `resource Token{cleanup{}}fn void bad(ref mut vec::Vec<Token> owner){let ref mut Token[] view=vec::view_mut<Token>(owner);assert(view.len==0);}`,
		"foreign-view":     `struct Token{i64 value;}unsafe extern C fn void external(ref mut Token[] view);fn void bad(ref mut Token[] view){unsafe{external(view)};}`,
		"empty":            `struct Token{}fn void bad(ref mut vec::Vec<Token> owner){let ref mut Token[] view=vec::view_mut<Token>(owner);assert(view.len==0);}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, "module main;import std::vec as vec;"+source+"fn i32 main(){return 0;}")
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" || name == "multiple-origins" && !strings.Contains(diag, "E2103") || name == "tagged-owner" && !strings.Contains(diag, "E2101") {
				t.Fatalf("accepted: %d %s", code, diag)
			}
		})
	}
}

func TestMutableStructVecViewParameterNative(t *testing.T) {
	source := `module main;import std::vec as vec;struct Token{i64 value;}
 fn Option<ref mut Token[]> tagged(ref mut vec::Vec<Token> owner){return .some(vec::view_mut<Token>(owner));}
 fn void edit(ref mut vec::Vec<Token> owner){
  let Option<ref mut Token[]> result=tagged(owner);
  switch(result){case .some(view):{view[0]=Token{.value=5000000000};assert(view[0].value==5000000000);}case .none:{assert(false);}};
  discard vec::clear<Token>(owner);assert(vec::len<Token>(owner)==0);
 }
 fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();discard vec::push<Token>(ref mut owner,Token{.value=7});edit(ref mut owner);return 0;}`
	runI64ViewNativeCases(t, "mutable-struct-parameter", []struct{ name, source, trap string }{{"tagged-last-use", source, ""}})
}
