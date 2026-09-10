package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserGenericTypes(t *testing.T) {
	var p, main strings.Builder
	main.WriteString("fn i32 main() {\n")
	for i, typ := range []string{"Vec<T>", "Result<T,E>", "Box<Box<T>>", "Box<Box<Box<T>>>", "Box<Box<T> >", "Result<ref mut pkg::Thing[],Vec<T>,>", "pkg /*gap*/ :: Box< T >[]"} {
		input := "module main; fn " + typ + " f(" + typ + " x) { let " + typ + " y = x; return y; }"
		ref, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || ref.AST == nil {
			t.Fatalf("generic type oracle %q: %v", typ, err)
		}
		f := ref.AST.Functions[0]
		fmt.Fprintf(&p, "fn i32 inspect_%d(syntax::Parsed parsed) { assert(syntax::problem_count(ref parsed) == 0);\n", i)
		count := 0
		result := emitCandidateType(t, &p, f.ResultType, &count)
		param := emitCandidateType(t, &p, f.Parameters[0].Type, &count)
		local := emitCandidateType(t, &p, *f.Body.Statements[0].Binding.Type, &count)
		fmt.Fprintf(&p, "assert(syntax::declaration_at(ref parsed,1).result_type == %d && syntax::entry_at(ref parsed,0).type_root == %d && syntax::entry_at(ref parsed,1).type_root == %d && syntax::type_count(ref parsed) == %d); return 0; }\n", result, param, local, count)
		fmt.Fprintf(&main, "assert(switch(make(b%s)) { case .ok(parsed): { inspect_%d(parsed) } case .err(_): { 1 } } == 0);\n", strconv.Quote(input), i)
	}
	for _, typ := range []string{"A<>", "A<,T>", "A<T,,U>", "A<T", "A<T>>", "A<B<T>> >", "A<B<T>="} {
		input := "module main; fn " + typ + " bad() {} fn void kept() {}"
		fmt.Fprintf(&main, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 1 && syntax::declaration_count(ref parsed) == 2 && syntax::type_count(ref parsed) == 1); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	p.WriteString(`fn i32 bound(bool deep, usize count, i32 expected) {
 var bytes::Buffer input = bytes::empty();
 assert(switch(bytes::append(ref mut input,if(deep) { b"module main; fn " } else { b"module main; fn A<" })) { case .ok(_): { true } case .err(_): { false } });
 var usize i=0;
 while(i<count) { assert(switch(bytes::append(ref mut input,if(deep) { b"A<" } else { b"T," })) { case .ok(_): { true } case .err(_): { false } }); i+=1; }
 if(deep) {
  assert(switch(bytes::append(ref mut input,b"T")) { case .ok(_): { true } case .err(_): { false } });
  i=0; while(i<count) { assert(switch(bytes::append(ref mut input,b">")) { case .ok(_): { true } case .err(_): { false } }); i+=1; }
 } else { assert(switch(bytes::append(ref mut input,b">")) { case .ok(_): { true } case .err(_): { false } }); }
 assert(switch(bytes::append(ref mut input,b" f() {}")) { case .ok(_): { true } case .err(_): { false } });
 return switch(make(bytes::view(ref input))) { case .ok(parsed): { assert(expected == 0 && syntax::problem_count(ref parsed) == 0); 0 } case .err(status): { assert(status == expected && expected != 0); 0 } };
}
`)
	main.WriteString(`{ let i32 r = switch(make(b"module main; struct S { Vec<T> values; } fn i32 f() { return a >> b; }")) {
 case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0 && syntax::field_count(ref parsed) == 1); let field = syntax::field_at(ref parsed,0); assert(syntax::type_at(ref parsed,field.type_root).kind == 6); let expr = syntax::declaration_at(ref parsed,2).expression; let ref u8[] op = syntax::expression_text(ref parsed,expr); assert(op.len == 2 && op[0] == 62 && op[1] == 62); 0 }
 case .err(_): { 1 } }; assert(r == 0); }
assert(bound(true,31,0)==0); assert(bound(true,32,4)==0);
assert(bound(false,2047,0)==0); assert(bound(false,2048,4)==0);
return 0; }`)
	runParserProbe(t, p.String()+main.String())
}
