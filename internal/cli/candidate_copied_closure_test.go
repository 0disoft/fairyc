package cli

import (
	"encoding/json"
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
