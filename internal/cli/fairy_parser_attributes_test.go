package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserAttributes(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main() {\n")
	for _, input := range []string{
		"module main; @effects(console) pub fn void f() {}",
		"module main; @marker @ns::tag(a::b, 0xFF, \"한글\",) fn void f() {}",
		"module main; @empty() @same @same struct R { u8 x; }",
		`module main; @meta(b"bytes", c"cstring", "text") fn void f() {}`,
		"module main; @bad resource R<T> { cleanup {} }",
	} {
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.AST == nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("attribute oracle %q: %v", input, err)
		}
		var start, end int
		// Both ordinary item owners use the same attribute contract.
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0 && syntax::declaration_count(ref parsed) == 2);", strconv.Quote(input))
		if len(oracle.AST.Functions) > 0 {
			f := oracle.AST.Functions[0]
			start, end = f.Span.StartByte, f.Span.EndByte
			emitAttributeChecks(&p, input, f.Attributes)
		} else if len(oracle.AST.Resources) > 0 {
			r := oracle.AST.Resources[0]
			start, end = r.Span.StartByte, r.Span.EndByte
			emitAttributeChecks(&p, input, r.Attributes)
		} else {
			s := oracle.AST.Structs[0]
			start, end = s.Span.StartByte, s.Span.EndByte
			emitAttributeChecks(&p, input, s.Attributes)
		}
		fmt.Fprintf(&p, "let syntax::Declaration d = syntax::declaration_at(ref parsed,1); assert(d.start == %d && d.end == %d && d.attribute_start == 0 && d.attribute_count == syntax::attribute_count(ref parsed));", start, end)
		if at := strings.Index(input, "pub"); at >= 0 {
			fmt.Fprintf(&p, "assert(d.public && d.public_start == %d && d.public_end == %d);", at, at+3)
		}
		p.WriteString("0 } case .err(_): { 1 } }; assert(result == 0); }\n")
	}
	for _, bad := range []string{
		"@bad(,1) fn void bad() {}",
		"@bad(true) fn void bad() {}",
		"@bad(x x) fn void bad() {}",
		"@ns:: fn void bad() {}",
		"@bad fn void bad() { let x = ; }",
		"@bad resource R<T,,> { cleanup {} }",
		"@bad import x as x;",
	} {
		input := "module main; " + bad + " @kept fn void kept() {}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		// Boolean expressions remain outside the attribute argument grammar.
		normativeOnly := strings.HasPrefix(bad, "@bad(true)")
		if err != nil || oracle.Tree == nil || (!normativeOnly && len(oracle.Tree.Diagnostics) == 0) {
			t.Fatalf("invalid attribute oracle %q: %v", bad, err)
		}
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 1 && syntax::declaration_count(ref parsed) == 2 && syntax::attribute_count(ref parsed) == 1); let syntax::Declaration d = syntax::declaration_at(ref parsed,1); assert(d.attribute_start == 0 && d.attribute_count == 1 && d.name_start == %d); assert(syntax::attribute_at(ref parsed,0).argument_start == 0 && syntax::attribute_at(ref parsed,0).argument_count == 0); 0 } case .err(_): { 1 } }; assert(result == 0); }\n", strconv.Quote(input), strings.LastIndex(input, "kept()"))
	}
	p.WriteString(`
var usize mode = 0;
while (mode < 2) {
 let usize limit = if (mode == 0) { 256 } else { 1024 };
 var usize n = limit;
 while (n <= limit + 1) {
  var bytes::Buffer data = switch(bytes::copy(b"module main; ")) { case .ok(value): { value } case .err(_): { bytes::empty() } };
  if (mode == 1) { discard bytes::append(ref mut data,b"@a("); }
  var usize i = 0;
  while (i < n) { discard bytes::append(ref mut data,if (mode == 0) { b"@a " } else { b"x," }); i += 1; }
  if (mode == 1) { discard bytes::append(ref mut data,b")"); }
  discard bytes::append(ref mut data,b"fn void f() {}");
  let i32 result = switch(make(bytes::view(ref data))) {
   case .ok(parsed): { assert(n == limit && syntax::problem_count(ref parsed) == 0); assert(syntax::attribute_count(ref parsed) == if (mode == 0) { 256 } else { 1 }); if (mode == 1) { assert(syntax::attribute_at(ref parsed,0).argument_count == 1024); } 0 }
   case .err(status): { assert(n == limit + 1 && status == 4); 0 }
  }; assert(result == 0); n += 1;
 }
 mode += 1;
}
return 0; }`)
	runParserProbe(t, p.String())
}

func emitAttributeChecks(p *strings.Builder, input string, attributes []ast.Attribute) {
	fmt.Fprintf(p, "assert(syntax::attribute_count(ref parsed) == %d);", len(attributes))
	argIndex := 0
	for i, a := range attributes {
		nameStart := a.Span.StartByte + 1
		nameEnd := nameStart + len(a.Name)
		fmt.Fprintf(p, "{ let a = syntax::attribute_at(ref parsed,%d); assert(a.start == %d && a.end == %d && a.name_start == %d && a.name_end == %d && a.argument_start == %d && a.argument_count == %d); }", i, a.Span.StartByte, a.Span.EndByte, nameStart, nameEnd, argIndex, len(a.Arguments))
		for _, arg := range a.Arguments {
			raw := input[arg.Span.StartByte:arg.Span.EndByte]
			kind := 1
			if strings.HasPrefix(raw, "\"") || strings.HasPrefix(raw, "b\"") || strings.HasPrefix(raw, "c\"") {
				kind = 3
			} else if raw[0] >= '0' && raw[0] <= '9' {
				kind = 2
			}
			fmt.Fprintf(p, "{ let a = syntax::attribute_argument_at(ref parsed,%d); assert(a.kind == %d && a.start == %d && a.end == %d); let ref u8[] raw = syntax::attribute_argument_text(ref parsed,%d); assert(raw.len == %d);", argIndex, kind, arg.Span.StartByte, arg.Span.EndByte, argIndex, len(raw))
			for n, b := range []byte(raw) {
				fmt.Fprintf(p, "assert(raw[%d] == %d);", n, b)
			}
			p.WriteString("}")
			argIndex++
		}
	}
}
