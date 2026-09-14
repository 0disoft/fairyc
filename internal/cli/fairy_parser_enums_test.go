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
	input := "module main; @repr(C) pub enum Kind { First; /* gap */ Second { i32 code; ref mut pkg::Thing[] data; (ref u8)[] views; }; First; Blank {}; Again { u8 code; }; Complex { pkg::Box<pkg::Pair<u8,i32>> nested; ref mut pkg::Box<u8[4]>[] borrowed; (ref pkg::Box<pkg::Pair<u8,i32>>)[cfg::N][] grid; }; } enum Empty {}"
	oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || oracle.AST == nil || len(oracle.Tree.Diagnostics) != 0 {
		t.Fatalf("enum oracle: %v", err)
	}
	var p, typeChecks strings.Builder
	p.WriteString("fn i32 inspect(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0 && syntax::declaration_count(ref parsed) == 3 && syntax::field_count(ref parsed) == 13);\n")
	field, types := 0, 0
	for i, enum := range oracle.AST.Enums {
		total := len(enum.Variants)
		for _, member := range enum.Variants {
			total += len(member.Fields)
		}
		fmt.Fprintf(&p, "{ let d = syntax::declaration_at(ref parsed,%d); assert(d.kind == 11 && d.start == %d && d.end == %d && d.name_start == %d && d.name_end == %d && d.public == %t && d.field_start == %d && d.field_count == %d && d.body == 0 && d.result_type == 0 && d.generic_parameters == 0 && d.cleanup_body == 0); }\n", i+1, enum.Span.StartByte, enum.Span.EndByte, enum.NameSpan.StartByte, enum.NameSpan.EndByte, enum.Public, field, total)
		for _, member := range enum.Variants {
			first := field
			for _, f := range member.Fields {
				var checks strings.Builder
				root := emitCandidateType(t, &checks, f.Type, &types)
				fmt.Fprintf(&typeChecks, "fn i32 check_type_%d(ref syntax::Parsed parsed) { %s return 0; }\n", field, strings.ReplaceAll(checks.String(), "ref parsed", "parsed"))
				fmt.Fprintf(&p, "assert(check_type_%d(ref parsed) == 0);\n", field)
				fmt.Fprintf(&p, "{ let f = syntax::field_at(ref parsed,%d); assert(f.kind == 4 && f.start == %d && f.end == %d && f.name_start == %d && f.name_end == %d && f.type_root == %d && f.key_type == 0 && !f.public && f.payload_open == 0 && f.payload_end == 0 && f.payload_count == 0 && f.payload_start == 0); }\n", field, f.Span.StartByte, f.Span.EndByte, f.NameSpan.StartByte, f.NameSpan.EndByte, root)
				field++
			}
			fmt.Fprintf(&p, "{ let m = syntax::field_at(ref parsed,%d); assert(m.kind == 3 && m.start == %d && m.end == %d && m.name_start == %d && m.name_end == %d && m.type_root == 0 && m.key_type == 0 && !m.public && m.public_start == 0 && m.public_end == 0); }\n", field, member.Span.StartByte, member.Span.EndByte, member.NameSpan.StartByte, member.NameSpan.EndByte)
			text := input[member.NameSpan.EndByte:member.Span.EndByte]
			open, end := 0, 0
			if offset := strings.IndexByte(text, '{'); offset >= 0 {
				open = member.NameSpan.EndByte + offset
				end = member.NameSpan.EndByte + strings.LastIndexByte(text, '}') + 1
			} else {
				first = 0
			}
			fmt.Fprintf(&p, "{ let m = syntax::field_at(ref parsed,%d); assert(m.payload_start == %d && m.payload_count == %d && m.payload_open == %d && m.payload_end == %d); }\n", field, first, len(member.Fields), open, end)
			field++
		}
	}
	fmt.Fprintf(&p, "assert(syntax::type_count(ref parsed) == %d && syntax::declaration_at(ref parsed,1).attribute_count == 1 && integrity::valid(ref parsed)); return 0; }\n", types)
	p.WriteString(typeChecks.String())
	p.WriteString(`fn i32 rejected(syntax::Parsed parsed) {
 assert(syntax::problem_count(ref parsed) > 0 && syntax::declaration_count(ref parsed) == 2 && syntax::field_count(ref parsed) == 1 && syntax::type_count(ref parsed) == 0 && syntax::attribute_count(ref parsed) == 0 && syntax::declaration_at(ref parsed,1).kind == 11 && integrity::valid(ref parsed)); return 0;
}
`)
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
fn i32 payload_bound(usize count, i32 expected) {
 var bytes::Buffer input = bytes::empty();
 discard bytes::append(ref mut input,b"module main; enum E { V {");
 var usize i = 0; while(i < count) { discard bytes::append(ref mut input,b"u8 x;"); i += 1; }
 discard bytes::append(ref mut input,b"}; }");
 return switch(make(bytes::view(ref input))) {
  case .ok(parsed): { assert(expected == 0 && syntax::problem_count(ref parsed) == 0 && syntax::field_count(ref parsed) == count + 1 && syntax::type_count(ref parsed) == count && syntax::field_at(ref parsed,count).payload_count == count && integrity::valid(ref parsed)); 0 }
  case .err(status): { assert(expected != 0 && status == expected); 0 }
 };
}
fn i32 main() {
`)
	fmt.Fprintf(&p, "assert(switch(make(b%s)) { case .ok(parsed): { inspect(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input))
	badCases := []string{"enum E { A }", "pub enum E { A; B, }", "enum E { pub A; }", "enum E<T> {}", "enum E { A { i32 x }; }", "enum E { A { pub i32 x; }; }", "enum E { A { i32 x; } }", "enum E { A = 1; }", "@repr(C) pub enum E { A; B, }", "enum E { A; fn void nested() {} }"}
	for _, typ := range []string{"Box<Pair<u8,i32>", "Box<Pair<u8,i32>>>", "(ref Box<u8>", "Box<u8>[4"} {
		bad := "@repr(C) pub enum E { Good { u8 first; }; Bad { " + typ + " value; }; }"
		ref, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte("module main; "+bad)), source.DefaultLimits())
		if err != nil || ref.Tree == nil || len(ref.Tree.Diagnostics) == 0 {
			t.Fatalf("malformed payload type oracle %q: %v", typ, err)
		}
		badCases = append(badCases, bad)
	}
	for _, bad := range badCases {
		input := "module main; " + bad + " pub enum Kept { OK; }"
		fmt.Fprintf(&p, "assert(switch(make(b%s)) { case .ok(parsed): { rejected(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input))
	}
	p.WriteString(`assert(bound(1024,b"}",0) == 0); assert(bound(1025,b"}",4) == 0);
assert(bound(1024,b"} struct S { u8 x; }",4) == 0);
assert(bound(1024,b"} enum More { V; }",4) == 0);
assert(payload_bound(1023,0) == 0); assert(payload_bound(1024,4) == 0);
return 0; }`)
	runParserProbe(t, p.String())
}
