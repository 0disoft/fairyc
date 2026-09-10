package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
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
    if (problems > 0) { assert(syntax::problem_at(ref parsed, 0).start == offset); }
    var usize index = 0; var usize end = 0;
    while (index < declarations) {
        let syntax::Declaration value = syntax::declaration_at(ref parsed, index);
        assert(value.kind > 0 && value.kind < 4 && value.start >= end && value.end <= source.len);
        assert(value.name_start >= value.start && value.name_end > value.name_start && value.name_end <= value.end);
        end = value.end; index += 1;
    }
    return 0;
}
fn i32 exercise(ref u8[] input, usize limit, usize declarations, usize problems, usize offset, i32 expected) {
    return switch (make(input, limit)) {
        case .ok(parsed): { inspect(parsed, declarations, problems, offset, expected) }
        case .err(status): { assert(status == expected && expected != 0); 0 }
    };
}
fn i32 tree(syntax::Parsed parsed) {
`)
	imported := reference.AST.Imports[0]
	fmt.Fprintf(&program, "{ let syntax::Declaration value = syntax::declaration_at(ref parsed, 1); assert(value.kind == 2 && value.start == %d && value.end == %d && value.name_start == %d && value.name_end == %d); }\n", imported.Span.StartByte, imported.Span.EndByte, imported.AliasSpan.StartByte, imported.AliasSpan.EndByte)
	for index, function := range reference.AST.Functions {
		fmt.Fprintf(&program, "{ let syntax::Declaration value = syntax::declaration_at(ref parsed, %d); assert(value.kind == 3 && value.start == %d && value.end == %d && value.name_start == %d && value.name_end == %d); }\n", index+2, function.Span.StartByte, function.Span.EndByte, function.NameSpan.StartByte, function.NameSpan.EndByte)
	}
	program.WriteString("return 0; }\nfn i32 main() {\n")
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
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stdout != "" || stderr != "" {
			t.Fatalf("%v: %d %q %s", args, code, stdout, stderr)
		}
	}
}
