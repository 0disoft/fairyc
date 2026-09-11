package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserVisibility(t *testing.T) {
	input := "module main; pub /*gap*/ fn void exposed() {} fn i32 private() { return 1; } pub fn bool second() { return true; }"
	reference, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || reference.AST == nil {
		t.Fatalf("visibility oracle: %v", err)
	}
	var p strings.Builder
	p.WriteString("fn i32 inspect(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0 && syntax::declaration_count(ref parsed) == 4);\n")
	for i, f := range reference.AST.Functions {
		public := i != 1
		start, end := 0, 0
		if public {
			start, end = f.Span.StartByte, f.Span.StartByte+3
		}
		fmt.Fprintf(&p, "{ let syntax::Declaration d = syntax::declaration_at(ref parsed,%d); assert(d.public == %t && d.start == %d && d.end == %d && d.public_start == %d && d.public_end == %d && d.name_start == %d && d.name_end == %d); }\n", i+1, public, f.Span.StartByte, f.Span.EndByte, start, end, f.NameSpan.StartByte, f.NameSpan.EndByte)
	}
	p.WriteString("return 0; } fn i32 main() {\n")
	fmt.Fprintf(&p, "assert(switch(make(b%s)) { case .ok(parsed): { inspect(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input))
	for _, bad := range []string{"pub struct X<T> {}", "pub resource X<T> { cleanup {} }", "pub import a as b;", "pub"} {
		text := "module main; " + bad + " pub fn void kept() {}"
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) > 0 && syntax::declaration_count(ref parsed) == 2); let syntax::Declaration d = syntax::declaration_at(ref parsed,1); assert(d.public && d.start == %d); 0 } case .err(_): { 1 } }; assert(result == 0); }\n", strconv.Quote(text), strings.LastIndex(text, "pub fn"))
	}
	p.WriteString("return 0; }")
	runParserProbe(t, p.String())
}
