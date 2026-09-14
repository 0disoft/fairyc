package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
	"github.com/0disoft/fairyc/internal/target"
)

// This is a disposable comparison format, not a serialized compiler AST.
func referenceBodyTree(input string) (string, error) {
	if len(input) > 32768 {
		return "", fmt.Errorf("source budget")
	}
	r, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || r.Tree == nil || len(r.Tree.Diagnostics) != 0 || r.AST == nil || len(r.AST.Functions) != 1 {
		return "", fmt.Errorf("body input is not one error-free function")
	}
	root, ok := r.Tree.Node(r.Tree.Root)
	if !ok || len(root.Children) != 2 {
		return "", fmt.Errorf("expected module and one function only")
	}
	var out strings.Builder
	fmt.Fprintf(&out, "1 %d ", len(input))
	for _, b := range []byte(input) {
		fmt.Fprintf(&out, "%d ", b)
	}
	count := 0
	var failure error
	record := func(kind int, span diagnostic.Span, children int, name diagnostic.Span, depth int, operator string) bool {
		count++
		if depth >= 64 || count > 256 {
			failure = fmt.Errorf("body export budget")
			return false
		}
		fmt.Fprintf(&out, "%d %d %d %d %d %d ", kind, span.StartByte, span.EndByte, children, name.StartByte, name.EndByte)
		fmt.Fprintf(&out, "%d ", len(operator))
		for _, b := range []byte(operator) {
			fmt.Fprintf(&out, "%d ", b)
		}
		return true
	}
	var block func(*ast.Block, int) bool
	var expr func(*ast.Expression, int) bool
	var stmt func(ast.Statement, int) bool
	block = func(b *ast.Block, d int) bool {
		if b == nil {
			return false
		}
		n := len(b.Statements)
		if b.Tail != nil {
			n++
		}
		if !record(100, b.Span, n, diagnostic.Span{}, d, "") {
			return false
		}
		for _, s := range b.Statements {
			if !stmt(s, d+1) {
				return false
			}
		}
		return b.Tail == nil || expr(b.Tail, d+1)
	}
	expr = func(e *ast.Expression, d int) bool {
		if e == nil || e.Type != nil || len(e.TypeArguments) != 0 {
			return false
		}
		k := map[syntax.Kind]int{syntax.KindLiteral: 200, syntax.KindPath: 201, syntax.KindGroup: 202, syntax.KindPrefix: 203, syntax.KindBinary: 204, syntax.KindCall: 205, syntax.KindBlock: 206, syntax.KindIf: 207}[e.Kind]
		if k == 0 {
			failure = fmt.Errorf("unhandled expression %s at %d", e.Kind, e.Span.StartByte)
			return false
		}
		n := len(e.Children)
		if k == 206 {
			n = 1
		}
		operator := ""
		if k == 203 || k == 204 {
			operator = e.Text
		}
		if !record(k, e.Span, n, diagnostic.Span{}, d, operator) {
			return false
		}
		if k == 206 {
			return block(e.Block, d+1)
		}
		for i, c := range e.Children {
			if k == 207 && i > 0 && c.Kind == syntax.KindBlock {
				if !block(c.Block, d+1) {
					return false
				}
			} else if !expr(c, d+1) {
				return false
			}
		}
		return true
	}
	stmt = func(s ast.Statement, d int) bool {
		k := map[syntax.Kind]int{syntax.KindBinding: 101, syntax.KindReturn: 103, syntax.KindAssignment: 104, syntax.KindExprStmt: 105, syntax.KindWhile: 106, syntax.KindBreak: 107, syntax.KindContinue: 108}[s.Kind]
		if k == 0 || s.For != nil {
			failure = fmt.Errorf("unhandled statement %s at %d", s.Kind, s.Span.StartByte)
			return false
		}
		var children []*ast.Expression
		name := diagnostic.Span{}
		if k == 101 {
			if s.Binding == nil || s.Binding.Type != nil {
				return false
			}
			if s.Binding.Mutable {
				k = 102
			}
			name = s.Binding.NameSpan
			children = append(children, s.Binding.Expression)
		} else if k == 104 {
			children = append(children, s.Target, s.Expression)
		} else if k == 106 {
			children = s.Expression.Children
		} else if k != 107 && k != 108 && s.Expression != nil {
			children = append(children, s.Expression)
		}
		if !record(k, s.Span, len(children), name, d, s.Operator) {
			return false
		}
		for i, c := range children {
			if k == 106 && i == 1 {
				if !block(c.Block, d+1) {
					return false
				}
			} else if !expr(c, d+1) {
				return false
			}
		}
		return true
	}
	if !block(r.AST.Functions[0].Body, 0) {
		if failure == nil {
			failure = fmt.Errorf("unsupported body-tree node")
		}
		return "", failure
	}
	return out.String(), nil
}

func TestWindowsFairyParserBodyTree(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	inputs := []string{
		"module main; fn i32 f(i32 a){var x=1;{let y=2;call(x,y);}if(a>0){x+=a;}else if(false){}else{x=3;}while(x<a){if(false){continue;}x+=1;break;}return x;}",
		"module main; /* 한글 */\r\nfn void f(){let x={1};if(a){call((x),-2);}else{3}return;}",
		"module main;fn void f(){}",
		"module main;fn void f(){" + strings.Repeat("x;", 127) + "return;}",
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	for _, name := range []string{"syntax.fy", "integrity.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, "src/"+name, string(data))
	}
	program := parserCorpusProgram
	a := strings.Index(program, "@effects(console) fn i32 report(")
	b := strings.Index(program, "@effects(console) fn i32 main()")
	if a < 0 || b < a {
		t.Fatal("corpus driver boundary drift")
	}
	program = program[:a] + candidateBodyTreeWriter + program[b:]
	a = strings.Index(program, " return switch(parsed) {")
	if a < 0 {
		t.Fatal("corpus result boundary drift")
	}
	program = program[:a] + " return switch(parsed){case .ok(value):{report(value)}case .err(_):{4}};\n}\n"
	writeTestSource(t, root, "src/main.fy", program)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("compile: %d %s", code, stderr.String())
		}
		for _, input := range inputs {
			want, err := referenceBodyTree(input)
			if err != nil {
				t.Fatalf("reference for %q: %v", input, err)
			}
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewBufferString(input))
			if err != nil || out.ExitCode != 0 || out.Stderr != "" || out.TrapControl != "" || out.Stdout != want {
				t.Fatalf("body tree mismatch %v: err=%v output=%+v want=%s", profile, err, out, want)
			}
			t.Logf("profile=%v complete body bytes=%d canonical bytes=%d", profile, len(input), len(want))
		}
		for _, input := range []string{"module main;fn void f(){let i32 x=1;}", "module main;fn void f(){return [1];}", "module main;fn void f(){return 1}", "module main;fn void f(){" + strings.Repeat("x;", 128) + "}", "module main;const i32 X=1;fn void f(){}"} {
			if text, err := referenceBodyTree(input); err == nil || text != "" {
				t.Fatal("reference accepted unsupported/invalid/budget input")
			}
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewBufferString(input))
			if err != nil || out.ExitCode == 0 || out.Stdout != "" || out.Stderr != "" || out.TrapControl != "" {
				t.Fatalf("partial or successful rejected export: %+v %v", out, err)
			}
		}
	}
}
