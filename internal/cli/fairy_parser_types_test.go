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

func emitCandidateType(t *testing.T, p *strings.Builder, typ ast.TypeReference, count *int) int {
	t.Helper()
	child := 0
	if typ.Element != nil {
		child = emitCandidateType(t, p, *typ.Element, count)
	}
	kind := map[ast.TypeKind]int{ast.TypeKindNamed: 1, ast.TypeKindReference: 2, ast.TypeKindSlice: 4, ast.TypeKindGrouped: 5}[typ.Kind]
	if typ.Mutable {
		kind = 3
	}
	if kind == 0 || len(typ.Arguments) > 0 {
		t.Fatalf("unsupported type oracle %+v", typ)
	}
	*count++
	fmt.Fprintf(p, "{ let node = syntax::type_at(ref parsed,%d); assert(node.kind == %d && node.start == %d && node.end == %d && node.left == %d && node.right == 0); let ref u8[] text = syntax::type_text(ref parsed,%d); assert(text.len == %d); }\n", *count, kind, typ.Span.StartByte, typ.Span.EndByte, child, *count, typ.Span.EndByte-typ.Span.StartByte)
	return *count
}

func TestWindowsFairyParserTypes(t *testing.T) {
	var p, main strings.Builder
	main.WriteString("fn i32 main() {\n")
	for i, typ := range []string{"i32", "pkg::Thing", "ref u8[]", "ref mut pkg::Thing[]", "(ref u8)[]", "ref ref mut T[][]", "pkg /*gap*/ :: Thing"} {
		input := "module main; fn " + typ + " f(" + typ + " x) { let " + typ + " y = x; let z = y; return z; }"
		ref, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || ref.AST == nil {
			t.Fatalf("type oracle %s: %v", typ, err)
		}
		f := ref.AST.Functions[0]
		fmt.Fprintf(&p, "fn i32 inspect_%d(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0);\n", i)
		count := 0
		result := emitCandidateType(t, &p, f.ResultType, &count)
		param := emitCandidateType(t, &p, f.Parameters[0].Type, &count)
		local := emitCandidateType(t, &p, *f.Body.Statements[0].Binding.Type, &count)
		fmt.Fprintf(&p, "assert(syntax::declaration_at(ref parsed,1).result_type == %d && syntax::entry_at(ref parsed,0).type_root == %d && syntax::entry_at(ref parsed,1).type_root == %d && syntax::entry_at(ref parsed,2).type_root == 0 && syntax::type_count(ref parsed) == %d); return 0; }\n", result, param, local, count)
		fmt.Fprintf(&main, "assert(switch(make(b%s)) { case .ok(parsed): { inspect_%d(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input), i)
	}
	for _, typ := range []string{"ptr u8", "u8[4]", "pkg::", "ref", "(ref u8", "A<B>"} {
		input := "module main; fn " + typ + " bad() {} fn void kept() {}"
		fmt.Fprintf(&main, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 1 && syntax::declaration_count(ref parsed) == 2 && syntax::type_count(ref parsed) == 1); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	p.WriteString(`fn i32 bound(ref u8[] prefix, ref u8[] unit, usize count, ref u8[] suffix, i32 expected) {
 var bytes::Buffer input = bytes::empty();
 assert(switch(bytes::append(ref mut input,prefix)) { case .ok(_): { true } case .err(_): { false } });
 var usize i=0;
 while(i<count) { assert(switch(bytes::append(ref mut input,unit)) { case .ok(_): { true } case .err(_): { false } }); i+=1; }
 assert(switch(bytes::append(ref mut input,suffix)) { case .ok(_): { true } case .err(_): { false } });
 return switch(make(bytes::view(ref input))) { case .ok(parsed): { assert(expected == 0 && syntax::problem_count(ref parsed) == 0); 0 } case .err(status): { assert(status == expected && expected != 0); 0 } };
}
`)
	main.WriteString(`assert(bound(b"module main; fn ",b"ref ",31,b"T f() {}",0)==0);
assert(bound(b"module main; fn ",b"ref ",32,b"T f() {}",4)==0);
assert(bound(b"module main; fn T",b"[]",4095,b" f() {}",0)==0);
assert(bound(b"module main; fn T",b"[]",4096,b" f() {}",4)==0);
return 0; }`)
	runParserProbe(t, p.String()+main.String())
}
