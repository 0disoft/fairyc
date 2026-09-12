package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserStructs(t *testing.T) {
	input := "module main; pub struct Pair { pub i32 left; ref mut pkg::Thing[] right; (ref u8)[] views; } struct Empty {} fn void kept() {}"
	ref, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || ref.AST == nil {
		t.Fatalf("struct oracle: %v", err)
	}
	var p strings.Builder
	p.WriteString("fn i32 inspect(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0 && syntax::declaration_count(ref parsed) == 4 && syntax::field_count(ref parsed) == 3);\n")
	count, fieldIndex := 0, 0
	for i, s := range ref.AST.Structs {
		fmt.Fprintf(&p, "{ let d = syntax::declaration_at(ref parsed,%d); assert(d.kind == 4 && d.start == %d && d.end == %d && d.name_start == %d && d.name_end == %d && d.public == %t && d.field_start == %d && d.field_count == %d && d.result_type == 0 && d.body == 0); }\n", i+1, s.Span.StartByte, s.Span.EndByte, s.NameSpan.StartByte, s.NameSpan.EndByte, s.Public, fieldIndex, len(s.Fields))
		for _, f := range s.Fields {
			root := emitCandidateType(t, &p, f.Type, &count)
			public := strings.HasPrefix(input[f.Span.StartByte:], "pub")
			pubStart, pubEnd := 0, 0
			if public {
				pubStart, pubEnd = f.Span.StartByte, f.Span.StartByte+3
			}
			fmt.Fprintf(&p, "{ let f = syntax::field_at(ref parsed,%d); assert(f.start == %d && f.end == %d && f.name_start == %d && f.name_end == %d && f.type_root == %d && f.public == %t && f.public_start == %d && f.public_end == %d); }\n", fieldIndex, f.Span.StartByte, f.Span.EndByte, f.NameSpan.StartByte, f.NameSpan.EndByte, root, public, pubStart, pubEnd)
			fieldIndex++
		}
	}
	p.WriteString("return 0; }\n")
	p.WriteString(`fn i32 bound(usize fields, ref u8[] suffix, i32 expected) {
 var bytes::Buffer input = bytes::empty();
 assert(switch(bytes::append(ref mut input,b"module main; struct S {")) { case .ok(_): { true } case .err(_): { false } });
 var usize i=0;
 while(i<fields) { assert(switch(bytes::append(ref mut input,b"u8 x;")) { case .ok(_): { true } case .err(_): { false } }); i+=1; }
 assert(switch(bytes::append(ref mut input,suffix)) { case .ok(_): { true } case .err(_): { false } });
 return switch(make(bytes::view(ref input))) { case .ok(parsed): { assert(expected == 0 && syntax::problem_count(ref parsed) == 0 && syntax::field_count(ref parsed) == fields); 0 } case .err(status): { assert(status == expected && expected != 0); 0 } };
}
fn i32 main() {
`)
	fmt.Fprintf(&p, "assert(switch(make(b%s)) { case .ok(parsed): { inspect(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input))
	for _, bad := range []string{"struct X { i32 x }", "pub struct X { i32 x; ref ; }", "struct X<T,,> {}", "struct X { cleanup {} }", "struct X { fn void nested() {} }", "resource X<T,,> { cleanup {} }"} {
		text := "module main; " + bad + " pub fn void kept() {}"
		fmt.Fprintf(&p, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) > 0 && syntax::declaration_count(ref parsed) == 2 && syntax::field_count(ref parsed) == 0 && syntax::type_count(ref parsed) == 1 && syntax::declaration_at(ref parsed,1).public); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(text))
	}
	p.WriteString(`assert(bound(1024,b"}",0)==0); assert(bound(1025,b"}",4)==0); assert(bound(1024,b"} struct T { u8 y; }",4)==0); return 0; }`)
	runParserProbe(t, p.String())
}
