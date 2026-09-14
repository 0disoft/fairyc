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
	// Surface AST keeps the generic path spelling, but its precise path-only
	// span belongs to the CST. Read that origin instead of reparsing Type.Text.
	paths := map[[2]int]diagnostic.Span{}
	for _, node := range r.Tree.Nodes {
		if node.Kind == syntax.KindType && node.Cooked == string(syntax.TypeFormNamed) && len(node.Children) != 0 {
			if path, ok := r.Tree.Node(node.Children[0]); ok && path.Kind == syntax.KindPath {
				paths[[2]int{node.Span.StartByte, node.Span.EndByte}] = path.Span
			}
		}
	}
	var typeTree func(*ast.TypeReference, int) bool
	typeTree = func(typ *ast.TypeReference, d int) bool {
		if typ == nil {
			return false
		}
		k, n := 0, 0
		switch typ.Kind {
		case ast.TypeKindNamed:
			k = 301
			if len(typ.Arguments) != 0 {
				k, n = 306, len(typ.Arguments)+1
			}
		case ast.TypeKindReference:
			k, n = 302, 1
			if typ.Mutable {
				k = 303
			}
		case ast.TypeKindRawPointer:
			k, n = 311, 1
			if typ.Mutable {
				k = 312
			}
		case ast.TypeKindSlice:
			k, n = 304, 1
		case ast.TypeKindGrouped:
			k, n = 305, 1
		case ast.TypeKindArray:
			k, n = 308, 2
		case ast.TypeKindForeignFunction:
			if typ.ABI != "C" || typ.Result == nil {
				return false
			}
			k, n = 315, len(typ.Parameters)+1
		default:
			return false
		}
		if !record(k, typ.Span, n, diagnostic.Span{}, d, "") {
			return false
		}
		if k == 301 {
			return true
		}
		if k == 306 {
			span, ok := paths[[2]int{typ.Span.StartByte, typ.Span.EndByte}]
			if !ok || !record(301, span, 0, diagnostic.Span{}, d+1, "") {
				return false
			}
			for i := range typ.Arguments {
				if !typeTree(&typ.Arguments[i], d+1) {
					return false
				}
			}
			return true
		}
		if k == 315 {
			if !typeTree(typ.Result, d+1) {
				return false
			}
			for i := range typ.Parameters {
				if !typeTree(&typ.Parameters[i], d+1) {
					return false
				}
			}
			return true
		}
		if !typeTree(typ.Element, d+1) {
			return false
		}
		if k == 308 {
			if typ.Length == nil {
				return false
			}
			lengthKind := 309
			if typ.Length.Kind == syntax.KindPath {
				lengthKind = 310
			} else if typ.Length.Kind != syntax.KindLiteral {
				return false
			}
			return record(lengthKind, typ.Length.Span, 0, diagnostic.Span{}, d+1, "")
		}
		return true
	}
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
		if e == nil || (e.Type != nil && e.Kind != syntax.KindCast) || (len(e.TypeArguments) != 0 && e.Kind != syntax.KindSpecialize) {
			return false
		}
		k := map[syntax.Kind]int{syntax.KindLiteral: 200, syntax.KindPath: 201, syntax.KindGroup: 202, syntax.KindPrefix: 203, syntax.KindBinary: 204, syntax.KindCall: 205, syntax.KindBlock: 206, syntax.KindIf: 207, syntax.KindCast: 208, syntax.KindSpecialize: 209, syntax.KindFieldExpr: 210, syntax.KindIndex: 211, syntax.KindSlice: 212}[e.Kind]
		if e.Kind == syntax.KindOmitBound {
			if e.Text == "start" {
				k = 213
			} else if e.Text == "end" {
				k = 214
			}
		}
		if k == 0 {
			failure = fmt.Errorf("unhandled expression %s at %d", e.Kind, e.Span.StartByte)
			return false
		}
		n := len(e.Children)
		if k == 208 {
			if e.Type == nil || n != 1 {
				return false
			}
			n++
		}
		if k == 209 {
			if n != 1 || len(e.TypeArguments) == 0 {
				return false
			}
			n += len(e.TypeArguments)
		}
		if k == 206 {
			n = 1
		}
		operator := ""
		if k == 203 || k == 204 {
			operator = e.Text
		}
		name := diagnostic.Span{}
		if k == 210 {
			name = diagnostic.Span{StartByte: e.Span.EndByte - len(e.Text), EndByte: e.Span.EndByte}
		}
		if !record(k, e.Span, n, name, d, operator) {
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
		if k == 208 {
			return typeTree(e.Type, d+1)
		}
		for i := range e.TypeArguments {
			if !typeTree(&e.TypeArguments[i], d+1) {
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
		var writtenType *ast.TypeReference
		name := diagnostic.Span{}
		if k == 101 {
			if s.Binding == nil {
				return false
			}
			if s.Binding.Mutable {
				k = 102
			}
			name = s.Binding.NameSpan
			writtenType = s.Binding.Type
			if writtenType != nil {
				k += 8
			}
			children = append(children, s.Binding.Expression)
		} else if k == 104 {
			children = append(children, s.Target, s.Expression)
		} else if k == 106 {
			children = s.Expression.Children
		} else if k != 107 && k != 108 && s.Expression != nil {
			children = append(children, s.Expression)
		}
		n := len(children)
		if writtenType != nil {
			n++
		}
		if !record(k, s.Span, n, name, d, s.Operator) {
			return false
		}
		if writtenType != nil && !typeTree(writtenType, d+1) {
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
		"module main;fn void f(){let i32 x=1;var pkg::Thing y=x;{let ref mut ptr u8[] z=y;}}",
		"module main; /* 타입 */\r\nfn void f(){let Box /* gap */ <Pair<u8,i32>,ref T> x=seed;var (ref u8)[0x10][cfg /* gap */ :: N][] y=x;let ptr void p=x;}",
		"module main;fn void f(){let unsafe extern C fn void() x=callback;var (unsafe /* gap */ extern C fn ptr mut u8(i32,ref u8[],))[] y=x;let Box<unsafe extern C fn i32(Box<u8>,unsafe extern C fn void(u32))> z=y;}",
		"module main;fn void f(){" + strings.Repeat("let i32 x=1;", 85) + "}",
		"module main;fn void f(){let i32" + strings.Repeat("[]", 61) + " x=1;}",
		"module main; /* 변환 */\r\nfn void f(){let x=(a+b) as i32 as u8;var ptr mut u8 p=value as ptr mut u8;return callback as unsafe extern C fn i32(ref u8[]);}",
		"module main;fn void f(){let x=pkg::choose /* gap */ <Box<Pair<u8,i32>>,ref mut T>(first<i32>(1),second as u8);if(a<b){call(x);}return empty<unsafe extern C fn void(),u8[0x10]>();}",
		"module main;fn void f(){" + strings.Repeat("f<i32>();", 51) + "}",
		"module main;fn void f(){x as i32" + strings.Repeat("[]", 60) + ";}",
		"module main;fn void f(){f<i32" + strings.Repeat("[]", 59) + ">();}",
		"module main; /* 접근 */\r\nfn void f(){obj /* gap */ .field[at(1)].value+=2;return factory<i32>(x).items[0].run(y);}",
		"module main;fn void f(){a[..];a[0..];a[..0];a[0..0];a[start..end];a[ /* L */ .. /* R */ ];a[1.. /* R */ ];return a[..n][i].field;}",
		"module main;fn void f(){" + strings.Repeat("a[..];", 51) + "}",
		"module main;fn void f(){a" + strings.Repeat("[..]", 61) + ";}",
		"module main;fn void f(){a" + strings.Repeat(".field", 61) + ";}",
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
		for _, input := range []string{
			"module main;fn void f(){let ref mut =1;}",
			"module main;fn void f(){return [1];}",
			"module main;fn void f(){return 1}",
			"module main;fn void f(){" + strings.Repeat("x;", 128) + "}",
			"module main;const i32 X=1;fn void f(){}",
			"module main;fn void f(){" + strings.Repeat("let i32 x=1;", 86) + "}",
			"module main;fn void f(){let i32" + strings.Repeat("[]", 62) + " x=1;}",
			"module main;fn void f(){x as ;}",
			"module main;fn void f(){f<i32>(;}",
			"module main;fn void f(){" + strings.Repeat("f<i32>();", 52) + "}",
			"module main;fn void f(){x as i32" + strings.Repeat("[]", 61) + ";}",
			"module main;fn void f(){f<i32" + strings.Repeat("[]", 60) + ">();}",
			"module main;fn void f(){a.;}",
			"module main;fn void f(){a[];}",
			"module main;fn void f(){a[1..2;}",
			"module main;fn void f(){" + strings.Repeat("a[..];", 52) + "}",
			"module main;fn void f(){a" + strings.Repeat("[..]", 62) + ";}",
			"module main;fn void f(){a" + strings.Repeat(".field", 62) + ";}",
		} {
			if text, err := referenceBodyTree(input); err == nil || text != "" {
				t.Fatalf("reference accepted unsupported/invalid/budget input %q", input)
			}
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewBufferString(input))
			if err != nil || out.ExitCode == 0 || out.Stdout != "" || out.Stderr != "" || out.TrapControl != "" {
				t.Fatalf("partial or successful rejected export: %+v %v", out, err)
			}
		}
	}
}
