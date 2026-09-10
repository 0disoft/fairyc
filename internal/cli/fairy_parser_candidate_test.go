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

func TestWindowsFairyLexerStreamParser(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	const good = "module main; import std::io as io; fn void empty() {} fn i32 answer() { return 42; }"
	reference, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(good)), source.DefaultLimits())
	if err != nil || reference.AST == nil || len(reference.AST.Imports) != 1 || len(reference.AST.Functions) != 2 {
		t.Fatalf("reference syntax: %+v %v", reference, err)
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	parser, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "syntax.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/syntax.fy", string(parser))
	var program strings.Builder
	program.WriteString(`module main;
import scanner as scanner; import syntax as syntax; import std::bytes as bytes;
fn Result<syntax::Parsed, i32> make(ref u8[] input, usize limit) {
    var bool allocated = true;
    let bytes::Buffer owned = switch (bytes::copy(input)) { case .ok(value): { value } case .err(_): { allocated = false; bytes::empty() } };
    if (!allocated) { return .err(3); }
    return switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream): { syntax::parse(stream, limit) }
        case .err(status): { .err(status) }
    };
}
fn i32 inspect(syntax::Parsed parsed, usize declarations, usize problems, usize offset, i32 status) {
    assert(status == 0);
    assert(syntax::declaration_count(ref parsed) == declarations);
    assert(syntax::problem_count(ref parsed) == problems);
    let ref u8[] source = syntax::source_view(ref parsed);
    if (problems > 0) { assert(syntax::problem_at(ref parsed, 0).start == offset); assert(syntax::expression_count(ref parsed) == 0); assert(syntax::entry_count(ref parsed) == 0); }
    var usize index = 0; var usize end = 0; var usize functions = 0;
    while (index < declarations) {
        let syntax::Declaration value = syntax::declaration_at(ref parsed, index);
        assert(value.kind > 0 && value.kind < 4 && value.start >= end && value.end <= source.len);
        assert(value.name_start >= value.start && value.name_end > value.name_start && value.name_end <= value.end);
        if (value.kind == 3) { functions += 1; }
        end = value.end; index += 1;
    }
    if (problems > 0) { assert(syntax::statement_count(ref parsed) == functions); }
    return 0;
}
fn i32 exercise(ref u8[] input, usize limit, usize declarations, usize problems, usize offset, i32 expected) {
    return switch (make(input, limit)) {
        case .ok(parsed): { inspect(parsed, declarations, problems, offset, expected) }
        case .err(status): { assert(status == expected && expected != 0); 0 }
    };
}
fn i32 repeated(ref u8[] prefix, ref u8[] unit, usize count, ref u8[] suffix, usize declarations, i32 expected) {
    var bytes::Buffer input = bytes::empty();
    assert(switch (bytes::append(ref mut input, prefix)) { case .ok(_): { true } case .err(_): { false } });
    var usize index = 0;
    while (index < count) {
        assert(switch (bytes::append(ref mut input, unit)) { case .ok(_): { true } case .err(_): { false } });
        index += 1;
    }
    assert(switch (bytes::append(ref mut input, suffix)) { case .ok(_): { true } case .err(_): { false } });
    return exercise(bytes::view(ref input), 512, declarations, 0, 0, expected);
}
fn i32 nested_while(usize depth, usize declarations, i32 expected) {
    var bytes::Buffer input = bytes::empty();
    assert(switch (bytes::append(ref mut input, b"module main; fn void f() {")) { case .ok(_): { true } case .err(_): { false } });
    var usize index = 0;
    while (index < depth) {
        assert(switch (bytes::append(ref mut input, b"while(true){")) { case .ok(_): { true } case .err(_): { false } });
        index += 1;
    }
    assert(switch (bytes::append(ref mut input, b"break; continue;")) { case .ok(_): { true } case .err(_): { false } });
    index = 0;
    while (index <= depth) {
        assert(switch (bytes::append(ref mut input, b"}")) { case .ok(_): { true } case .err(_): { false } });
        index += 1;
    }
    return exercise(bytes::view(ref input), 512, declarations, 0, 0, expected);
}
fn i32 tree(syntax::Parsed parsed) {
`)
	imported := reference.AST.Imports[0]
	fmt.Fprintf(&program, "{ let syntax::Declaration value = syntax::declaration_at(ref parsed, 1); assert(value.kind == 2 && value.start == %d && value.end == %d && value.name_start == %d && value.name_end == %d); }\n", imported.Span.StartByte, imported.Span.EndByte, imported.AliasSpan.StartByte, imported.AliasSpan.EndByte)
	for index, function := range reference.AST.Functions {
		fmt.Fprintf(&program, "{ let syntax::Declaration value = syntax::declaration_at(ref parsed, %d); assert(value.kind == 3 && value.start == %d && value.end == %d && value.name_start == %d && value.name_end == %d); }\n", index+2, function.Span.StartByte, function.Span.EndByte, function.NameSpan.StartByte, function.NameSpan.EndByte)
	}
	program.WriteString("return 0; }\n")
	expressions := []string{"42", "name", "((42))", "-a + +b * ~c", "!a && b || c", "a - b - c", "a << b + c", "a | b ^ c & d", "a + b < c * d", "(a < b) == c", "a < (b == c)", "*a / b % c", "a >= b && c != d", "a <= b || c > d", "a >> b - c", "a /* gap */ + (b - c)"}
	expressions = append(expressions, "true", "false || !true", "a::b::c", "std /* gap */ :: io :: write", "f()", "f(a, b, c,)", "f(g(1), h(2, 3))", "-f(a) * g(b)", "f()(1)(2,3)", "std::io::write(a + 1, false)")
	for index, expr := range expressions {
		input := "module main; fn i32 f() { return " + expr + "; }"
		result, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || result.AST == nil {
			t.Fatalf("expression oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&program, "fn i32 expression_%d(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0);\n", index)
		count := 0
		var emit func(*ast.Expression) int
		emit = func(node *ast.Expression) int {
			left, right := 0, 0
			var arguments []int
			if len(node.Children) > 0 {
				left = emit(node.Children[0])
			}
			if node.Kind == syntax.KindCall {
				for _, argument := range node.Children[1:] {
					value := emit(argument)
					arguments = append(arguments, value)
					count++
					fmt.Fprintf(&program, "{ let syntax::Expression arg = syntax::expression_at(ref parsed, %d); assert(arg.kind == 7 && arg.left == %d && arg.right == %d && arg.start == %d && arg.end == %d); }\n", count, value, right, argument.Span.StartByte, argument.Span.EndByte)
					right = count
				}
			} else if len(node.Children) > 1 {
				right = emit(node.Children[1])
			}
			count++
			kind := map[syntax.Kind]int{syntax.KindLiteral: 1, syntax.KindPath: 2, syntax.KindGroup: 3, syntax.KindPrefix: 4, syntax.KindBinary: 5, syntax.KindCall: 6}[node.Kind]
			if kind == 0 {
				t.Fatalf("unsupported oracle node %s", node.Kind)
			}
			fmt.Fprintf(&program, "{ let syntax::Expression node = syntax::expression_at(ref parsed, %d); assert(node.kind == %d && node.start == %d && node.end == %d && node.left == %d && node.right == %d); }\n", count, kind, node.Span.StartByte, node.Span.EndByte, left, right)
			if node.Kind == syntax.KindCall {
				fmt.Fprintf(&program, "assert(syntax::call_argument_count(ref parsed, %d) == %d);\n", count, len(arguments))
				for index, argument := range arguments {
					fmt.Fprintf(&program, "assert(syntax::call_argument(ref parsed, %d, %d) == %d);\n", count, index, argument)
				}
			} else if node.Kind != syntax.KindGroup {
				text := node.Text
				if node.Kind == syntax.KindPath {
					text = input[node.Span.StartByte:node.Span.EndByte]
				}
				fmt.Fprintf(&program, "{ let ref u8[] text = syntax::expression_text(ref parsed, %d); assert(text.len == %d);", count, len(text))
				for offset, value := range []byte(text) {
					fmt.Fprintf(&program, " assert(text[%d] == %d);", offset, value)
				}
				program.WriteString(" }\n")
			}
			return count
		}
		rootID := emit(result.AST.Functions[0].Body.Statements[0].Expression)
		fmt.Fprintf(&program, "assert(syntax::expression_count(ref parsed) == %d && syntax::declaration_at(ref parsed, 1).expression == %d); return 0; }\n", count, rootID)
	}
	entryInput := emitParserEntryOracle(t, &program)
	blockInput := emitParserBlockOracle(t, &program)
	program.WriteString("fn i32 main() {\n")
	fmt.Fprintf(&program, "assert(switch (make(b%s, 512)) { case .ok(parsed): { block_tree(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(blockInput))
	fmt.Fprintf(&program, "assert(switch (make(b%s, 512)) { case .ok(parsed): { entry_tree(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(entryInput))
	for index, expr := range expressions {
		input := "module main; fn i32 f() { return " + expr + "; }"
		fmt.Fprintf(&program, "assert(switch (make(b%s, 512)) { case .ok(parsed): { expression_%d(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input), index)
	}
	cases := []struct {
		input                                         string
		limit, declarations, problems, offset, status int
	}{
		{good, 512, 4, 0, 0, 0},
		{"/*lead*/ module main;\r\nfn void f() {}", 512, 2, 0, 0, 0},
		{"module a::b; import c::d as e;", 512, 2, 0, 0, 0},
		{"", 512, 0, 1, 0, 0},
		{"fn void kept() {}", 512, 1, 1, 0, 0},
		{"module main; fn i32 broken( { } fn void kept() {}", 512, 2, 1, 27, 0},
		{"module main; fn i32 bad() { fn void nested() {} } fn void kept() {}", 512, 2, 1, 26, 0},
		{"module main; fn void ok() {} import a as b;", 512, 2, 1, 28, 0},
		{"module main; fn void f() {}", 1, 0, 0, 0, 4},
		{"module main;", 513, 0, 0, 0, 4},
		{"module main;\x00", 512, 0, 0, 0, 5},
		{"module main; " + strings.Repeat("fn ?; ", 33), 512, 0, 0, 0, 4},
	}
	cases[5].offset = strings.Index(cases[5].input, "{")
	cases[6].offset = strings.Index(cases[6].input, "fn void nested")
	cases[7].offset = strings.Index(cases[7].input, "import")
	for _, body := range []string{"if (true) { var x = 1; } else ;", "{ x = ; }", "if (true) return;", "x += ;", "while (true) ;", "while (;", "break ; continue ; while (true) { break } ;"} {
		input := "module main; fn void bad() { " + body + " } fn void kept() {}"
		offset := strings.LastIndex(input, ";")
		if strings.Contains(body, "return") {
			offset = strings.Index(input, "return")
		}
		if strings.Contains(body, "break }") {
			offset = strings.Index(input, "} ;")
		}
		cases = append(cases, struct {
			input                                         string
			limit, declarations, problems, offset, status int
		}{input, 512, 2, 1, offset, 0})
	}
	for _, depth := range []int{31, 32} {
		input := "module main; fn void f() {" + strings.Repeat("{", depth) + strings.Repeat("}", depth) + "}"
		decls, status := 2, 0
		if depth == 32 {
			decls, status = 0, 4
		}
		cases = append(cases, struct {
			input                                         string
			limit, declarations, problems, offset, status int
		}{input, 512, decls, 0, 0, status})
	}
	for _, depth := range []int{31, 32} {
		decls, status := 2, 0
		if depth == 32 {
			decls, status = 0, 4
		}
		fmt.Fprintf(&program, "assert(nested_while(%d, %d, %d) == 0);\n", depth, decls, status)
	}
	for _, extra := range []string{"", " fn void h() {}"} {
		decls, status := 3, 0
		if extra != "" {
			decls, status = 0, 4
		}
		fmt.Fprintf(&program, "assert(repeated(b\"module main; fn void f() {\", b\"1;\", 2047, b%s, %d, %d) == 0);\n", strconv.Quote("} fn void g() {}"+extra), decls, status)
	}
	for _, expr := range []string{"f(,)", "f(a,,b)", "f(a", "a::", "true::x"} {
		input := "module main; fn i32 bad() { return " + expr + "; } fn void kept() {}"
		offset := strings.Index(input, "return ") + len("return ")
		switch expr {
		case "f(,)":
			offset += 2
		case "f(a,,b)":
			offset += 4
		case "f(a":
			offset += 3
		case "a::":
			offset += 3
		case "true::x":
			offset += 4
		}
		cases = append(cases, struct {
			input                                         string
			limit, declarations, problems, offset, status int
		}{input, 512, 2, 1, offset, 0})
	}
	for _, body := range []string{"let i32 x = ;", "var x = 1; return +;"} {
		input := "module main; fn i32 bad(i32 a) { " + body + " } fn void kept() {}"
		cases = append(cases, struct {
			input                                         string
			limit, declarations, problems, offset, status int
		}{input, 512, 2, 1, strings.LastIndex(input, ";"), 0})
	}
	for _, count := range []int{1024, 1025} {
		decls, status := 2, 0
		if count == 1025 {
			decls, status = 0, 4
		}
		fmt.Fprintf(&program, "assert(repeated(b\"module main; fn void f() { \", b\"return;\", %d, b\" }\", %d, %d) == 0);\n", count, decls, status)
	}
	for _, expr := range []string{"a < b < c", "a + ;", "()", "(a", "a b"} {
		input := "module main; fn i32 bad() { return " + expr + "; } fn void kept() {}"
		// Exact fault locations are checked separately from the Go AST oracle,
		// because the candidate exposes private recovery categories.
		offset := strings.Index(input, "return ") + len("return ")
		switch expr {
		case "a < b < c":
			offset += 6
		case "a + ;":
			offset += 4
		case "()":
			offset++
		case "(a":
			offset += 2
		case "a b":
			offset += 2
		}
		cases = append(cases, struct {
			input                                         string
			limit, declarations, problems, offset, status int
		}{input, 512, 2, 1, offset, 0})
	}
	for _, count := range []int{63, 64} {
		input := "module main; fn i32 f() { return " + strings.Repeat("(", count) + "1" + strings.Repeat(")", count) + "; }"
		decls, status := 2, 0
		if count == 64 {
			decls, status = 0, 4
		}
		cases = append(cases, struct {
			input                                         string
			limit, declarations, problems, offset, status int
		}{input, 512, decls, 0, 0, status})
	}
	for _, prefix := range []string{"+", "+ +"} {
		decls, status := 2, 0
		if prefix == "+ +" {
			decls, status = 0, 4
		}
		fmt.Fprintf(&program, "assert(repeated(b%s, b\"1+\", 2047, b\"1; }\", %d, %d) == 0);\n", strconv.Quote("module main; fn i32 f() { return "+prefix), decls, status)
	}
	for _, input := range []string{"module", "module main", "module main; import x as", "module main; fn void f(", "module main; fn i32 f() { return"} {
		declarations := 0
		if strings.HasPrefix(input, "module main;") {
			declarations = 1
		}
		cases = append(cases, struct {
			input                                         string
			limit, declarations, problems, offset, status int
		}{input, 512, declarations, 1, len(input), 0})
	}
	literal := func(input string) {
		program.WriteString("b\"")
		for _, b := range []byte(input) {
			fmt.Fprintf(&program, "\\x%02X", b)
		}
		program.WriteString("\"")
	}
	for _, test := range cases {
		program.WriteString("assert(exercise(")
		literal(test.input)
		fmt.Fprintf(&program, ", %d, %d, %d, %d, %d) == 0);\n", test.limit, test.declarations, test.problems, test.offset, test.status)
	}
	program.WriteString("return switch (make(")
	literal(good)
	program.WriteString(", 512)) { case .ok(parsed): { tree(parsed) } case .err(_): { 1 } }; }")
	writeTestSource(t, root, "src/main.fy", program.String())
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		t.Logf("native parser profile %v", args)
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("%v: %d %q %s", args, code, stdout, stderr)
		}
	}
}
