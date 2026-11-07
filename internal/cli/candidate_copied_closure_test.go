package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func candidateCopiedClosureSyntaxCases() []string {
	return []string{
		"module main;fn i32 main(){let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(2);}",
		"module main;fn i32 main(){let no_capture=fn [] i32(i32 x){x*2};return no_capture(21);}",
		"module main;fn i32 main(){let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;let full=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 w,i32 x,i32 y,i32 z){return a+b+c+d+w+x+y+z;};return full(5,6,7,14);}",
		"module main;\r\n// 캡처 원본을 보존한다.\r\nfn T keep<T>(T value){let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return -(offset-x)+82;};assert(add(0)==42);return value;}fn i32 main(){return keep<i32>(42);}",
	}
}
func TestCandidateCopiedClosureSyntax(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native replay; Linux deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "copied_closure_syntax.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::copied_closure_syntax;", "module main;", 1)
	var calls, expected strings.Builder
	for _, text := range candidateCopiedClosureSyntaxCases() {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("Go closure syntax: %v", err)
		}
		encoded, err := json.Marshal(parsed.AST.Functions[0])
		if err != nil {
			t.Fatal(err)
		}
		expected.Write(encoded)
		expected.WriteByte('\n')
		calls.WriteString("emit(b" + strconv.Quote(text) + ");")
	}
	program += "\n@effects(console) fn i32 main(){" + calls.String() + "return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "scanner", "syntax", "generic_source_function_json")
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || diagnostics != "" {
			t.Fatalf("%v: %d %s", args, code, diagnostics)
		}
		if out != expected.String() {
			t.Fatalf("%v: closure AST differs from Go\nactual: %s\nexpected: %s", args, out, expected.String())
		}
	}
}

func TestCandidateCopiedClosureCalls(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native replay; Linux deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "copied_closure_calls.fy"))
	if err != nil {
		t.Fatal(err)
	}
	view, err := os.ReadFile(filepath.Join("testdata", "copied_closure_for_in", "view.fy"))
	if err != nil {
		t.Fatal(err)
	}
	vector, err := os.ReadFile(filepath.Join("testdata", "copied_closure_for_in", "vector.fy"))
	if err != nil {
		t.Fatal(err)
	}
	sdk, err := os.ReadFile(filepath.Join("..", "sdkstdlib", "sources", "optional", "std", "vec.fy"))
	if err != nil {
		t.Fatal(err)
	}
	calls := candidateCopiedClosureAdmissionCalls() + "checked(b" + strconv.Quote(string(view)) + ",0);"
	vectorSource := strings.Replace(string(vector), "import std::bytes as bytes;", "", 1)
	// The unit value checker needs a contextual type for borrowing a Vec owner.
	// Exercise its existing named shared-descriptor form, not an untyped borrow.
	vectorSource = strings.Replace(vectorSource, "for(let ref i32 item in ref items){discard add(*item);assert(false);}", "{let ref v::Vec<i32> empty_view=ref items;for(let ref i32 item in empty_view){discard add(*item);assert(false);}}", 1)
	vectorSource = strings.Replace(vectorSource, "for(let ref i32 item in ref items){", "let ref v::Vec<i32> loop_view=ref items;for(let ref i32 item in loop_view){", 1)
	control := strings.Replace(vectorSource, "let add=fn [i32 offset] i32(i32 value){return offset+value;};", "", 1)
	control = strings.Replace(control, "let adjacent=fn [] i32(i32 value){return value;};", "", 1)
	control = strings.Replace(control, "fn i32 main()", "fn i32 add(i32 value){return 40+value;}fn i32 adjacent(i32 value){return value;}fn i32 main()", 1)
	calls += "assert(checked_vector(b" + strconv.Quote(vectorSource) + ",b" + strconv.Quote(string(sdk)) + ")==checked_vector(b" + strconv.Quote(control) + ",b" + strconv.Quote(string(sdk)) + "));"
	const loopPrefix = `module root;fn i32 main(){let i32[1] data=[1];let ref i32[] view=ref data[..];let add=fn [] i32(i32 n){return n;};`
	for _, body := range []string{
		`for(let i32 item in view){discard add(item);}return item;`,
		`for(let i32 add in view){discard add(1);}return 0;`,
	} {
		calls += "reject_typed(b" + strconv.Quote(loopPrefix+body+"}") + ");"
	}
	for _, body := range []string{
		`for(let i32 item in view){let next=fn [] i32(i32 n){return n;};discard next(item);}return 0;`,
		`for(let i32 item in view){let copy=add;discard copy(item);}return 0;`,
		`for(let i32 item in view){return add;}return 0;`,
	} {
		calls += "reject(b" + strconv.Quote(loopPrefix+body+"}") + ");"
	}
	program := strings.Replace(string(raw), "module tests::copied_closure_calls;", "module main;", 1) + "\n@effects(console) fn i32 main(){" + calls + "return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "scanner", "syntax", "module_signatures", "module_unit_bind", "module_unit_call_closure", "module_unit_values")
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
		}
	}
}

func candidateCopiedClosureProgram() string {
	return `module root;fn i32 main(){
var i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};offset=100;
assert(add(1)==41);assert(add(2)==42);if(true){let i32 add=7;assert(add==7);}
let second=fn [i32 offset] i32(i32 x){offset+x};assert(second(1)==101);
let no_capture=fn [] i32(i32 x){return x*2;};assert(no_capture(21)==42);
for(var i32 i=0;i<2;i+=1){assert(add(2)==42);}var i32 j=0;while(j<2){assert(add(j)==40+j);j+=1;}
let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;
let full=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 w,i32 x,i32 y,i32 z){return a+b+c+d+w+x+y+z;};assert(full(5,6,7,14)==42);
return add(2);}`
}
func candidateCopiedClosureAdmissionCalls() string {
	var out strings.Builder
	fmt.Fprintf(&out, "checked(b%s,5);", strconv.Quote(candidateCopiedClosureProgram()))
	fmt.Fprintf(&out, "checked(b%s,2);", strconv.Quote("module main;fn i32 main(){let c=fn [] i32(i32 x){return x;};if(true){let i32 c=7;let ref u8[] view=b\"xx\"[..];for(let u8 x in view){discard x;}assert(c==7);}return c(42);}"))
	fmt.Fprintf(&out, "checked(b%s,0);", strconv.Quote(candidateCopiedClosureSyntaxCases()[3]))
	fmt.Fprintf(&out, "checked(b%s,2);", strconv.Quote("module root;fn i32 main(){let c=fn [] i32(){42};for(let u8 x in b\"x\"){discard x;}return c();}"))
	base := "module root;fn i32 main(){let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(2);}"
	for _, replacement := range [][2]string{
		{"return add(2);", "return add;"}, {"return add(2);", "let other=add;return other(2);"},
		{"return offset+x;", "offset=1;return offset+x;"}, {"fn [i32 offset]", "fn []"}, {"fn [i32 offset]", "fn [ref i32 offset]"},
		{"fn [i32 offset]", "fn [i32 offset,i32 offset]"}, {"let add=fn", "let i32 add=fn"},
		{"let add=fn", "var add=fn"}, {"return add(2);", "return _fy_closure_main_1(40,2);"},
		{"return add(2);", "return add(.x=2);"}, {"let add=fn", "let _fy_closure_bad=fn"},
		{"return add(2);", "let add=42;return add;"}, {"return offset+x;", "return offset+missing;"},
		{"fn [i32 offset]", "fn [u32 offset]"}, {"i32(i32 x)", "i32(i64 x)"},
		{"i32(i32 x)", "i64(i32 x)"}, {"return offset+x;", "return add(x);"},
		{"i32(i32 x)", "i32(i32 a,i32 b,i32 c,i32 d,i32 e)"},
		{"fn [i32 offset]", "fn [i32 offset,i32 b,i32 c,i32 d,i32 e]"},
		{"return offset+x;", "return " + strings.Repeat("x+", 33) + "x;"},
	} {
		fmt.Fprintf(&out, "reject(b%s);", strconv.Quote(strings.Replace(base, replacement[0], replacement[1], 1)))
	}
	for _, raw := range []string{
		"module root;fn i32 main(){if(true){let c=fn [] i32(i32 x){return x;};return c(42);}return 0;}",
		"module root;fn i32 run(i32 c){let c=fn [] i32(i32 x){return x;};return c(42);}fn i32 main(){return run(1);}",
		"module root;fn i32 _fy_closure_hidden(){return 42;}fn i32 main(){let c=fn [] i32(){42};return c();}",
	} {
		fmt.Fprintf(&out, "reject(b%s);", strconv.Quote(raw))
	}
	for _, replacement := range [][2]string{{"let i32 offset=40;", "let bool offset=true;"}, {"add(2)", "add(true)"}, {"add(2)", "add()"}, {"return offset+x;", "return true;"}, {"return add(2);", "return add(2)+missing;"}} {
		fmt.Fprintf(&out, "reject_typed(b%s);", strconv.Quote(strings.Replace(base, replacement[0], replacement[1], 1)))
	}

	return out.String()
}
