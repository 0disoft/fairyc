package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func emitCandidateType(t *testing.T, p *strings.Builder, typ ast.TypeReference, count *int) int {
	t.Helper()
	if typ.Kind == ast.TypeKindForeignFunction {
		if typ.Result == nil || typ.ABI != "C" {
			t.Fatal("foreign function type oracle")
		}
		result := emitCandidateType(t, p, *typ.Result, count)
		tail := 0
		for _, parameter := range typ.Parameters {
			value := emitCandidateType(t, p, parameter, count)
			*count++
			fmt.Fprintf(p, "{let n=syntax::type_at(ref parsed,%d); assert(n.kind==16 && n.start==%d && n.end==%d && n.left==%d && n.right==%d);}\n", *count, parameter.Span.StartByte, parameter.Span.EndByte, value, tail)
			tail = *count
		}
		*count++
		fmt.Fprintf(p, "{let n=syntax::type_at(ref parsed,%d); assert(n.kind==15 && n.start==%d && n.end==%d && n.left==%d && n.right==%d);}\n", *count, typ.Span.StartByte, typ.Span.EndByte, result, tail)
		return *count
	}
	child := 0
	if typ.Element != nil {
		child = emitCandidateType(t, p, *typ.Element, count)
	}
	kind := map[ast.TypeKind]int{ast.TypeKindNamed: 1, ast.TypeKindReference: 2, ast.TypeKindSlice: 4, ast.TypeKindGrouped: 5, ast.TypeKindRawPointer: 11}[typ.Kind]
	if typ.Mutable {
		kind = 3
		if typ.Kind == ast.TypeKindRawPointer {
			kind = 12
		}
	}
	if typ.Kind == ast.TypeKindArray {
		if typ.Length == nil {
			t.Fatal("array without written length")
		}
		lengthKind := 9
		if typ.Length.Kind == syntax.KindPath {
			lengthKind = 10
		}
		*count++
		length := *count
		fmt.Fprintf(p, "{ let n = syntax::type_at(ref parsed,%d); assert(n.kind == %d && n.start == %d && n.end == %d && n.left == 0 && n.right == 0); }\n", length, lengthKind, typ.Length.Span.StartByte, typ.Length.Span.EndByte)
		*count++
		fmt.Fprintf(p, "{ let n = syntax::type_at(ref parsed,%d); assert(n.kind == 8 && n.start == %d && n.end == %d && n.left == %d && n.right == %d); }\n", *count, typ.Span.StartByte, typ.Span.EndByte, child, length)
		return *count
	}
	if len(typ.Arguments) > 0 {
		if typ.Kind != ast.TypeKindNamed {
			t.Fatal("arguments on non-name")
		}
		baseEnd := typ.Span.StartByte + len(strings.TrimSpace(strings.SplitN(typ.Text, "<", 2)[0]))
		*count++
		base, tail := *count, 0
		fmt.Fprintf(p, "{ let n = syntax::type_at(ref parsed,%d); assert(n.kind == 1 && n.start == %d && n.end == %d && n.left == 0 && n.right == 0); }\n", base, typ.Span.StartByte, baseEnd)
		var arguments []int
		for _, arg := range typ.Arguments {
			value := emitCandidateType(t, p, arg, count)
			arguments = append(arguments, value)
			*count++
			fmt.Fprintf(p, "{ let n = syntax::type_at(ref parsed,%d); assert(n.kind == 7 && n.start == %d && n.end == %d && n.left == %d && n.right == %d); }\n", *count, arg.Span.StartByte, arg.Span.EndByte, value, tail)
			tail = *count
		}
		*count++
		fmt.Fprintf(p, "{ let n = syntax::type_at(ref parsed,%d); assert(n.kind == 6 && n.start == %d && n.end == %d && n.left == %d && n.right == %d); } assert(syntax::type_argument_count(ref parsed,%d) == %d);\n", *count, typ.Span.StartByte, typ.Span.EndByte, base, tail, *count, len(arguments))
		for i, arg := range arguments {
			fmt.Fprintf(p, "assert(syntax::type_argument(ref parsed,%d,%d) == %d);\n", *count, i, arg)
		}
		return *count
	}
	if kind == 0 {
		t.Fatalf("unsupported type oracle %+v", typ)
	}
	*count++
	fmt.Fprintf(p, "{ let node = syntax::type_at(ref parsed,%d); assert(node.kind == %d && node.start == %d && node.end == %d && node.left == %d && node.right == 0); let ref u8[] text = syntax::type_text(ref parsed,%d); assert(text.len == %d); }\n", *count, kind, typ.Span.StartByte, typ.Span.EndByte, child, *count, typ.Span.EndByte-typ.Span.StartByte)
	return *count
}

func TestWindowsFairyParserTypes(t *testing.T) {
	var p, main strings.Builder
	main.WriteString("fn i32 main() {\n")
	for i, typ := range []string{"i32", "pkg::Thing", "ref u8[]", "ref mut pkg::Thing[]", "(ref u8)[]", "ref ref mut T[][]", "pkg /*gap*/ :: Thing", "u8[0]", "ref u8[16]", "Box<u8[0x10]>[cfg::N][]", "(ref u8)[2][3]", "u8[ cfg /*gap*/ :: N ]", "unsafe extern C fn void()", "unsafe /* gap */ extern C fn i32(ref u8[],u32,)", "(unsafe extern C fn ptr u8(i32))[2]", "Box<unsafe extern C fn void(Box<Pair<u8,i32>>,unsafe extern C fn i32(u8))>"} {
		input := "module main; fn " + typ + " f(" + typ + " x) { let " + typ + " y = x; let z = y; return z; }"
		ref, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || ref.Tree == nil || ref.AST == nil || len(ref.Tree.Diagnostics) != 0 || len(ref.AST.Functions) != 1 {
			t.Fatalf("type oracle %s: %v", typ, err)
		}
		f := ref.AST.Functions[0]
		count := 0
		var roots []int
		for j, part := range []ast.TypeReference{f.ResultType, f.Parameters[0].Type, *f.Body.Statements[0].Binding.Type} {
			var checks strings.Builder
			roots = append(roots, emitCandidateType(t, &checks, part, &count))
			fmt.Fprintf(&p, "fn i32 inspect_%d_%d(ref syntax::Parsed parsed) { %s return 0; }\n", i, j, strings.ReplaceAll(checks.String(), "ref parsed", "parsed"))
		}
		result, param, local := roots[0], roots[1], roots[2]
		fmt.Fprintf(&p, "fn i32 inspect_%d(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0 && inspect_%d_0(ref parsed)==0 && inspect_%d_1(ref parsed)==0 && inspect_%d_2(ref parsed)==0);\n", i, i, i, i)
		fmt.Fprintf(&p, "assert(syntax::declaration_at(ref parsed,1).result_type == %d && syntax::entry_at(ref parsed,0).type_root == %d && syntax::entry_at(ref parsed,1).type_root == %d && syntax::entry_at(ref parsed,2).type_root == 0 && syntax::type_count(ref parsed) == %d && integrity::valid(ref parsed)); return 0; }\n", result, param, local, count)
		fmt.Fprintf(&main, "assert(switch(make(b%s)) { case .ok(parsed): { inspect_%d(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input), i)
	}
	t.Run("type trees", func(t *testing.T) { runParserProbe(t, p.String()+main.String()+"return 0;}") })
	p.Reset()
	main.Reset()
	main.WriteString("fn i32 main(){\n")
	for index, typ := range []string{"ptr", "u8[1+2]", "u8[-1]", "u8[1", "u8[N::]", "u8[1.0]", "pkg::", "ref", "(ref u8", "A<>", "unsafe fn void()", "unsafe extern X fn void()", "unsafe extern C fn void(i32 named)", "unsafe extern C fn void(i32,,)", "unsafe extern C fn void(i32"} {
		input := "module main; fn " + typ + " bad() {} fn void kept() {}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		// Go accepts an empty angle list; the normative type_arguments
		// production requires at least one type. Do not widen that contract.
		if err != nil || oracle.Tree == nil || (typ != "A<>" && len(oracle.Tree.Diagnostics) == 0) {
			t.Fatalf("invalid type oracle %q: %v", typ, err)
		}
		fmt.Fprintf(&main, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { if(syntax::problem_count(ref parsed) == 1 && syntax::declaration_count(ref parsed) == 2 && syntax::type_count(ref parsed) == 1){0}else{%d} } case .err(_): { 1 } }; if(r!=0){return r;} }\n", strconv.Quote(input), index+10)
	}
	for _, depth := range []int{31, 32} {
		input := "module main; fn " + strings.Repeat("unsafe extern C fn ", depth) + "void" + strings.Repeat("()", depth) + " f(){}"
		fmt.Fprintf(&main, "assert(switch(make(b%s)){case .ok(parsed):{assert(%d==31 && syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(status):{assert(%d==32 && status==4);0}}==0);\n", strconv.Quote(input), depth, depth)
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
assert(bound(b"module main; fn T",b"[1]",2047,b"[] f() {}",0)==0);
assert(bound(b"module main; fn T",b"[1]",2048,b" f() {}",4)==0);
assert(bound(b"module main; fn unsafe extern C fn void(",b"u8,",2047,b") f() {}",0)==0);
assert(bound(b"module main; fn unsafe extern C fn void(",b"u8,",2048,b") f() {}",4)==0);
return 0; }`)
	t.Run("recovery and limits", func(t *testing.T) { runParserProbe(t, p.String()+main.String()) })
}
