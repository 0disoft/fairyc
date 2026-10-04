package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const structVecViewBase = `module main; import std::vec as vec;
struct Token {u32 kind; usize offset; i64 value; bool active;}
fn i32 take(vec::Vec<Token> owned) {return 0;}
fn ref Token[] tail(ref vec::Vec<Token> owner) {return vec::view<Token>(owner)[..];}
fn Option<ref Token[]> tagged(ref vec::Vec<Token> owner) {return .some(tail(owner));}
fn i32 main() {
 var vec::Vec<Token> owned = vec::empty<Token>();
 assert(switch(vec::push<Token>(ref mut owned,Token {.kind=1,.offset=3,.value=4000000000,.active=true})) {case .ok(_): {true} case .err(_): {false}});
 let ref Token[] view = vec::view<Token>(ref owned);
 ACTION
 assert(view[0].kind == 1);
 return 0;
}`

func TestStructVecViewRejections(t *testing.T) {
	for _, item := range []struct{ name, action, code string }{
		{"push", "discard vec::push<Token>(ref mut owned,view[0]);", "E2101"},
		{"reserve-noop", "discard vec::reserve<Token>(ref mut owned,0);", "E2101"},
		{"clear", "discard vec::clear<Token>(ref mut owned);", "E2101"},
		{"truncate-noop", "discard vec::truncate<Token>(ref mut owned,100);", "E2101"},
		{"move", "let vec::Vec<Token> moved = owned;", "E2105"},
		{"replace", "owned = vec::empty<Token>();", "E2101"},
		{"derived", "let ref Token[] child=view[..]; discard vec::clear<Token>(ref mut owned); assert(child.len>0);", "E2101"},
		{"tagged", "let Option<ref Token[]> result=tagged(ref owned); discard vec::clear<Token>(ref mut owned); switch(result){case .some(child):{assert(child.len>0);} case .none:{}};", "E2101"},
		{"bounds-effect", "let ref Token[] child=view[..{discard vec::clear<Token>(ref mut owned);0}];", "E2101"},
		{"index-effect", "let Token copy=view[{discard vec::clear<Token>(ref mut owned);0}];", "E2101"},
		{"store", "view[0] = Token {.kind=2,.offset=3,.value=0,.active=false};", ""},
		{"field-store", "view[0].kind=2;", ""},
		{"mutable-descriptor-store", "var ref Token[] alias=view;alias[0].kind=2;", ""},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, strings.Replace(structVecViewBase, "ACTION", item.action, 1))
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" || item.code != "" && !strings.Contains(diag, item.code) {
				t.Fatalf("expected %s: %d %s", item.code, code, diag)
			}
		})
	}
	for name, definition := range map[string]string{
		"empty":    "struct Token {}",
		"pointer":  "struct Token {ptr mut i32 address;}",
		"borrow":   "struct Token {ref i32 borrowed;}",
		"array":    "struct Token {i32[2] values;}",
		"resource": "resource Token {cleanup {}}",
		"enum":     "enum Token {one, two}",
	} {
		t.Run(name, func(t *testing.T) {
			source := "module main; import std::vec as vec; " + definition + " fn i32 main(){let vec::Vec<Token> owned=vec::empty<Token>();let ref Token[] view=vec::view<Token>(ref owned);assert(view.is_empty);return 0;}"
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, source)
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" {
				t.Fatalf("unsafe shape: %d %s", code, diag)
			}
		})
	}
	for name, source := range map[string]string{
		"local-escape":      "struct Token {i32 value;} fn ref Token[] bad(){let vec::Vec<Token> owned=vec::empty<Token>();return vec::view<Token>(ref owned);}",
		"shared-to-mutable": "struct Token {i32 value;} fn void bad(ref Token[] view){let ref mut Token[] exclusive=view;}",
		"owned-slice":       "struct Token {i32 value;} fn void bad(Token[] view){}",
		"private-builtin":   "struct Token {i32 value;} fn void bad(){let ref Token[] view=unsafe{storage_borrow<Token[]>(null,null,0)};}",
		"wrong-nominal":     "struct Token {i32 value;} struct Other {i32 value;} fn void bad(ref vec::Vec<Token> owner){let ref Other[] view=vec::view<Token>(owner);assert(view.is_empty);}",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, "module main;import std::vec as vec;"+source+" fn i32 main(){return 0;}")
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || diag == "" {
				t.Fatalf("accepted: %d %s", code, diag)
			}
		})
	}
}

func TestStructVecViewNative(t *testing.T) {
	success := strings.Replace(structVecViewBase, "ACTION", "var Token copy=view[0];copy.kind=7;assert(copy.kind==7 && view[0].kind==1 && view[0].value==4000000000 && view[0].active);let ref Token[] middle=view[0..1];assert(middle.len==1 && middle[0].offset==3);assert(view[1..].is_empty);", 1)
	released := strings.Replace(structVecViewBase, "ACTION", "let Token copy=view[0];discard vec::clear<Token>(ref mut owned);assert(copy.kind==1 && copy.value==4000000000);", 1)
	released = strings.Replace(released, "assert(view[0].kind == 1);", "", 1)
	empty := "module main;import std::vec as vec;struct Token {i32 value;} fn i32 main(){var vec::Vec<Token> owned=vec::empty<Token>();let ref Token[] view=vec::view<Token>(ref owned);assert(view.is_empty && view[..].is_empty && view[0..0].is_empty);discard vec::reserve<Token>(ref mut owned,32);let ref Token[] fresh=vec::view<Token>(ref owned);assert(fresh.is_empty);return 0;}"
	cases := []struct{ name, source, trap string }{{"copy-and-subviews", success, ""}, {"copy-after-owner-clear", released, ""}, {"empty-and-growth", empty, ""}}
	for _, item := range []struct{ name, action string }{
		{"end-read", "let Token copy=view[1];"},
		{"empty-read", "let Token copy=view[1..][0];"},
		{"huge-read", "let Token copy=view[18446744073709551615];"},
		{"reversed", "let ref Token[] child=view[1..0];"},
		{"past-end", "let ref Token[] child=view[..2];"},
	} {
		cases = append(cases, struct{ name, source, trap string }{item.name, strings.Replace(structVecViewBase, "ACTION", item.action, 1), "bounds"})
	}
	runI64ViewNativeCases(t, "struct-vec-view", cases)
}

func TestStructVecViewCache(t *testing.T) {
	for _, item := range []struct{ name, source string }{
		{"scalar", strings.Replace(structVecViewBase, "ACTION", "for(let Token copy in view){assert(copy.value==4000000000);}", 1)},
		{"nested", nestedStructSource("for(let Token copy in view){assert(copy.position.span.start==3 && copy.value==4000000000);}")},
	} {
		t.Run(item.name, func(t *testing.T) { checkStructVecViewCache(t, item.source) })
	}
}

func checkStructVecViewCache(t *testing.T, source string) {
	t.Helper()
	root := t.TempDir()
	writeVecI64ViewWorkspace(t, root, source)
	configuration := workspacecompiler.Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
	session, err := workspacecompiler.NewSourceSession(configuration)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	load := func() workspaceloader.Workspace {
		w, err := workspaceloader.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	cold, err := session.Check(load())
	if err != nil || !cold.Compiler.Verified || !cold.Cache.Lowered {
		t.Fatalf("cold: %v %+v", err, cold.Cache)
	}
	warm, err := session.Check(load())
	if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != workspacecompiler.OutcomeAllHit || warm.Cache.Lowered {
		t.Fatalf("warm: %v %+v", err, warm.Cache)
	}
}

func TestStructTokenExampleCheck(t *testing.T) {
	root := t.TempDir()
	manifest, err := os.ReadFile(filepath.Join("..", "..", "examples", "struct_vec_view", "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "fairy.toml", string(manifest))
	for _, name := range []string{"main.fy", "tokens.fy"} {
		src, err := os.ReadFile(filepath.Join("..", "..", "examples", "struct_vec_view", "src", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, filepath.Join("src", name), string(src))
	}
	code, _, diag := runForTest([]string{"check"}, root)
	if code != 0 {
		t.Fatalf("example: %d %s", code, diag)
	}
}

func TestStructVecViewGenericIntegerAliases(t *testing.T) {
	source := "module main; import std::vec as vec; fn i32 main(){"
	for _, element := range []string{"u8", "i32", "i64", "u32", "usize"} {
		source += " {var vec::Vec<" + element + "> owned = vec::empty<" + element + ">();discard vec::push<" + element + ">(ref mut owned,7);let ref " + element + "[] values=vec::view<" + element + ">(ref owned);assert(values.len==1 && values[0]==7);}"
	}
	source += "return 0;}"
	root := t.TempDir()
	writeVecI64ViewWorkspace(t, root, source)
	code, _, diag := runForTest([]string{"check"}, root)
	if code != 0 {
		t.Fatalf("existing integer view alias: %d %s", code, diag)
	}
}
