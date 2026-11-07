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
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

type closureSourcePrepareCase struct {
	name, raw      string
	values, direct int
}

func closureSourcePrepareCases() []closureSourcePrepareCase {
	var cases []closureSourcePrepareCase
	for count := 0; count <= 4; count++ {
		var declarations, captures, sum strings.Builder
		sum.WriteString("x")
		for i := 0; i < count; i++ {
			fmt.Fprintf(&declarations, "var i32 c%d=%d;", i, i+1)
			if i != 0 {
				captures.WriteByte(',')
			}
			fmt.Fprintf(&captures, "i32 c%d", i)
			fmt.Fprintf(&sum, "+c%d", i)
		}
		raw := "module root; // 원본 바이트\r\nfn fn /* signature */ i32(i32) make(){" + declarations.String() + "let fn i32(i32) action=fn [" + captures.String() + "] i32(i32 x){let i32 result=" + sum.String() + ";return result;};let copy=action;return copy;}"
		cases = append(cases, closureSourcePrepareCase{fmt.Sprintf("captures-%d", count), raw, 1, 0})
	}
	cases = append(cases,
		closureSourcePrepareCase{"same-type-distinct-targets", "module root;fn fn i32(i32) first(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){offset+x};return action;}fn fn i32(i32) second(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){offset-x};return action;}fn i32 apply(fn i32(i32) action,i32 x){return action(x);}", 2, 0},
		closureSourcePrepareCase{"mixed-old-and-value", "module root;fn i32 main(){let i32 offset=40;let old=fn [i32 offset] i32(i32 x){return offset+x;};let fn i32(i32) action=fn [i32 offset] i32(i32 x){let i32 local=offset+x;return local;};let alias=action;return old(alias(2));}", 1, 1},
		closureSourcePrepareCase{"untyped-zero-and-four-params", "module root;fn i32 main(){let none=fn [] i32(){42};let full=fn [] i32(i32 a,i32 b,i32 c,i32 d){return a+b+c+d;};return full(none(),0,0,0);}", 0, 2},
		closureSourcePrepareCase{"deferred-trapping-body", "module root;fn fn i32(i32) make(i32 zero){let fn i32(i32) action=fn [i32 zero] i32(i32 x){let unused=x/zero;return x;};return action;}", 1, 0},
		closureSourcePrepareCase{"locals16", "module root;fn fn i32(i32) make(){let fn i32(i32) action=fn [] i32(i32 x){" + copiedClosureLocalsCountBody(16) + "};return action;}", 1, 0},
		closureSourcePrepareCase{"nodes64", "module root;fn fn i32(i32) make(){let fn i32(i32) action=fn [] i32(i32 x){" + copiedClosureLocalsBudgetBody(false) + "};return action;}", 1, 0},
	)
	return cases
}

func closureSourcePrepareRefusals() []string {
	const base = "module root;fn fn i32(i32) make(){let fn i32(i32) action=fn [] i32(i32 x){return x;};return action;}"
	var bad []string
	for _, edit := range [][2]string{
		{"let fn i32(i32)", "var fn i32(i32)"},
		{"let fn i32(i32)", "let fn i64(i32)"},
		{"let fn i32(i32)", "let fn i32(i64)"},
		{"let fn i32(i32)", "let fn i32()"},
		{"let fn i32(i32)", "let fn i32(i32,i32)"},
		{"let fn i32(i32)", "let unsafe extern C fn i32(i32)"},
		{"let fn i32(i32)", "let Option<fn i32(i32)>"},
		{"i32(i32 x)", "i32()"},
		{"i32(i32 x)", "i32(i32 x,i32 y)"},
		{"i32(i32 x)", "i32(i64 x)"},
		{"fn [] i32", "fn [] i64"},
		{"return x;", "let i32 a=x;let i32 a=x;return a;"},
		{"return x;", "var i32 a=x;return a;"},
		{"return x;", "let i32 a=missing;return a;"},
		{"return x;", copiedClosureLocalsCountBody(17)},
		{"return x;", copiedClosureLocalsBudgetBody(true)},
	} {
		bad = append(bad, strings.Replace(base, edit[0], edit[1], 1))
	}
	bad = append(bad,
		"module root;fn i32 main(){if(true){let fn i32(i32) action=fn [] i32(i32 x){return x;};discard action;}return 0;}",
		"module root;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};let fn i32(i32) other=fn [i32 action] i32(i32 x){return x;};return other(1);}",
		"module root;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};let alias=action;let fn i32(i32) other=fn [i32 alias] i32(i32 x){return x;};return other(1);}",
		"module root;fn i32 main(fn i32(i32) action){let fn i32(i32) other=fn [i32 action] i32(i32 x){return x;};return other(1);}",
		"module root;fn i32 main(){let old=fn [] i32(i32 x){return x;};let fn i32(i32) other=fn [i32 old] i32(i32 x){return x;};return other(1);}",
		"module root;fn i32 main(){let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;let i32 e=5;let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d,i32 e] i32(i32 x){return x;};return action(1);}",
	)
	return bad
}

func closureSourcePrepareProgram(t *testing.T) (string, string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_source_prepare.fy"))
	if err != nil {
		t.Fatal(err)
	}
	var calls, expected strings.Builder
	for _, tc := range closureSourcePrepareCases() {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("root.fy", []byte(tc.raw)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("%s: %v", tc.name, err)
		}
		for _, fn := range parsed.AST.Functions {
			encoded, err := json.Marshal(fn)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&expected, "%s", genericplan.DigestText("source-function-ast-v0", string(encoded)))
			expected.WriteByte('\n')
		}
		fmt.Fprintf(&calls, "prepared(b%s,%d,%d);\n", strconv.Quote(tc.raw), tc.values, tc.direct)
	}
	for _, raw := range closureSourcePrepareRefusals() {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("root.fy", []byte(raw)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("refusal must reach source preparation: %s: %v", raw, err)
		}
		fmt.Fprintf(&calls, "refused(b%s);\n", strconv.Quote(raw))
	}
	program := strings.Replace(string(raw), "module tests::closure_value_source_prepare;", "module main;", 1)
	program += "\n@effects(console) fn i32 main(){" + calls.String() + "return 0;}\n"
	return program, expected.String()
}

func TestCandidateClosureValueSourcePrepareSyntax(t *testing.T) {
	program, _ := closureSourcePrepareProgram(t)
	for _, name := range []string{"copied_closure_lower", "copied_closure_tables", "syntax", "name_resolution"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(name+".fy", raw), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("%s: %v %+v", name, err, parsed.Tree.Diagnostics)
		}
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(program)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("focused harness: %v %+v", err, parsed.Tree.Diagnostics)
	}
	t.Logf("%d source cases; %d source-preparation refusal controls; owner and probe syntax", len(closureSourcePrepareCases()), len(closureSourcePrepareRefusals()))
}

func closureSourcePrepareWorkspace(t *testing.T, program string) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "scanner", "syntax", "copied_closure_tables", "closure_value_type_source", "generic_source_function_json", "name_resolution")
	return root
}
func TestCandidateClosureValueSourcePrepareCheck(t *testing.T) {
	program, _ := closureSourcePrepareProgram(t)
	root := closureSourcePrepareWorkspace(t, program)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("focused source-preparation harness check: %d %s %s", code, out, diagnostic)
	}
}
func TestCandidateClosureValueSourcePrepareNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("bounded source-preparation native probe requires Windows AMD64")
	}
	program, expected := closureSourcePrepareProgram(t)
	root := closureSourcePrepareWorkspace(t, program)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || diagnostic != "" || out != expected {
			t.Fatalf("focused source preparation %v: %d diagnostics=%s\nactual=%s\nexpected=%s", args, code, diagnostic, out, expected)
		}
		t.Logf("%v: %d source cases, %d refusal controls; original JSON definition hashes, copied constructor metadata, ordered snapshots, name resolution", args, len(closureSourcePrepareCases()), len(closureSourcePrepareRefusals()))
	}
}
