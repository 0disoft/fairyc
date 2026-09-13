package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserEnums(t *testing.T) {
	input := "module main; @repr(C) pub enum Kind { First; /* gap */ Second; First; } enum Empty {}"
	oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || oracle.AST == nil || len(oracle.Tree.Diagnostics) != 0 {
		t.Fatalf("enum oracle: %v", err)
	}
	var p strings.Builder
	p.WriteString("fn i32 inspect(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0 && syntax::declaration_count(ref parsed) == 3 && syntax::field_count(ref parsed) == 3 && syntax::type_count(ref parsed) == 0);\n")
	field := 0
	for i, enum := range oracle.AST.Enums {
		fmt.Fprintf(&p, "{ let d = syntax::declaration_at(ref parsed,%d); assert(d.kind == 11 && d.start == %d && d.end == %d && d.name_start == %d && d.name_end == %d && d.public == %t && d.field_start == %d && d.field_count == %d && d.body == 0 && d.result_type == 0 && d.generic_parameters == 0 && d.cleanup_body == 0); }\n", i+1, enum.Span.StartByte, enum.Span.EndByte, enum.NameSpan.StartByte, enum.NameSpan.EndByte, enum.Public, field, len(enum.Variants))
		for _, member := range enum.Variants {
			fmt.Fprintf(&p, "{ let m = syntax::field_at(ref parsed,%d); assert(m.kind == 3 && m.start == %d && m.end == %d && m.name_start == %d && m.name_end == %d && m.type_root == 0 && m.key_type == 0 && !m.public && m.public_start == 0 && m.public_end == 0); }\n", field, member.Span.StartByte, member.Span.EndByte, member.NameSpan.StartByte, member.NameSpan.EndByte)
			field++
		}
	}
	p.WriteString("assert(syntax::declaration_at(ref parsed,1).attribute_count == 1 && integrity::valid(ref parsed)); return 0; }\n")
	p.WriteString(`fn i32 bound(usize count, ref u8[] suffix, i32 expected) {
 var bytes::Buffer input = bytes::empty();
 discard bytes::append(ref mut input,b"module main; enum E {");
 var usize i = 0; while(i < count) { discard bytes::append(ref mut input,b"V;"); i += 1; }
 discard bytes::append(ref mut input,suffix);
 return switch(make(bytes::view(ref input))) {
  case .ok(parsed): { assert(expected == 0 && syntax::problem_count(ref parsed) == 0 && syntax::field_count(ref parsed) == count && syntax::type_count(ref parsed) == 0 && integrity::valid(ref parsed)); 0 }
  case .err(status): { assert(expected != 0 && status == expected); 0 }
 };
}
fn i32 main() {
`)
	fmt.Fprintf(&p, "assert(switch(make(b%s)) { case .ok(parsed): { inspect(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input))
	for _, bad := range []string{"enum E { A }", "pub enum E { A; B, }", "enum E { pub A; }", "enum E<T> {}", "enum E { A { i32 x; }; }", "enum E { A = 1; }", "@repr(C) pub enum E { A; B, }", "enum E { A; fn void nested() {} }"} {
		input := "module main; " + bad + " pub enum Kept { OK; }"
		fmt.Fprintf(&p, "{ let r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) > 0 && syntax::declaration_count(ref parsed) == 2 && syntax::field_count(ref parsed) == 1 && syntax::type_count(ref parsed) == 0 && syntax::attribute_count(ref parsed) == 0 && syntax::declaration_at(ref parsed,1).kind == 11 && integrity::valid(ref parsed)); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	p.WriteString(`assert(bound(1024,b"}",0) == 0); assert(bound(1025,b"}",4) == 0);
assert(bound(1024,b"} struct S { u8 x; }",4) == 0);
assert(bound(1024,b"} enum More { V; }",4) == 0);
return 0; }`)
	runParserProbe(t, p.String())
}
