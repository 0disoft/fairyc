package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

const parserProbePrelude = `module main;
import scanner as scanner; import syntax as syntax; import std::bytes as bytes;
fn Result<syntax::Parsed, i32> make(ref u8[] input) {
    var bool allocated = true;
    let bytes::Buffer owned = switch (bytes::copy(input)) { case .ok(value): { value } case .err(_): { allocated = false; bytes::empty() } };
    if (!allocated) { return .err(3); }
    return switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream): { syntax::parse(stream, 512) } case .err(status): { .err(status) }
    };
}
`

func runParserProbe(t *testing.T, program string) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "syntax.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/syntax.fy", string(data))
	writeTestSource(t, root, "src/main.fy", parserProbePrelude+program)
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		t.Logf("native parser probe %v", args)
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("%v: %d %q %s", args, code, stdout, stderr)
		}
	}
}

func TestWindowsFairyParserPostfix(t *testing.T) {
	var p, main strings.Builder
	main.WriteString("fn i32 main() {\n")
	for n, expr := range []string{"a.b", "a[1]", "a.b[i].c", "f().items[i + 1].value", "-a.b[i] * c", "a[b[c]]", "a /*gap*/ . b [ i ]", "a.b()"} {
		input := "module main; fn i32 f() { return " + expr + "; }"
		result, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || result.AST == nil || result.Tree == nil || len(result.Tree.Diagnostics) != 0 {
			t.Fatalf("oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "fn i32 check_%d(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0);\n", n)
		count := 0
		var emit func(*ast.Expression) int
		emit = func(e *ast.Expression) int {
			left, right := 0, 0
			if len(e.Children) > 0 {
				left = emit(e.Children[0])
			}
			if len(e.Children) > 1 {
				right = emit(e.Children[1])
			}
			kind := map[syntax.Kind]int{syntax.KindLiteral: 1, syntax.KindPath: 2, syntax.KindGroup: 3, syntax.KindPrefix: 4, syntax.KindBinary: 5, syntax.KindCall: 6, syntax.KindFieldExpr: 8, syntax.KindIndex: 9}[e.Kind]
			if kind == 0 {
				t.Fatalf("unknown oracle kind %s", e.Kind)
			}
			count++
			fmt.Fprintf(&p, "{ let syntax::Expression e = syntax::expression_at(ref parsed,%d); assert(e.kind == %d && e.start == %d && e.end == %d && e.left == %d && e.right == %d); }\n", count, kind, e.Span.StartByte, e.Span.EndByte, left, right)
			if kind == 8 {
				fmt.Fprintf(&p, "{ let ref u8[] name = syntax::expression_text(ref parsed,%d); assert(name.len == %d);", count, len(e.Text))
				for i, b := range []byte(e.Text) {
					fmt.Fprintf(&p, "assert(name[%d] == %d);", i, b)
				}
				p.WriteString("}\n")
			}
			return count
		}
		root := emit(result.AST.Functions[0].Body.Statements[0].Expression)
		fmt.Fprintf(&p, "assert(syntax::expression_count(ref parsed) == %d && syntax::declaration_at(ref parsed,1).expression == %d); return 0; }\n", count, root)
		fmt.Fprintf(&main, "assert(switch(make(b%s)) { case .ok(parsed): { check_%d(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input), n)
	}
	p.WriteString("fn i32 rejected(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 1 && syntax::declaration_count(ref parsed) == 2 && syntax::expression_count(ref parsed) == 0 && syntax::statement_count(ref parsed) == 1); return 0; }\n")
	for _, expr := range []string{"a.", "a.true", "a[]", "a[1", "a[1..2]", "a[..]"} {
		input := "module main; fn i32 bad() { return " + expr + "; } fn void kept() {}"
		fmt.Fprintf(&main, "assert(switch(make(b%s)) { case .ok(parsed): { rejected(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input))
	}
	main.WriteString("return 0; }")
	runParserProbe(t, p.String()+main.String())
}
