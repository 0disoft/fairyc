package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserForeign(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, body := range []string{
		"unsafe fn void f() {}", "pub unsafe /* gap */ fn i32 f(i32 x){return x;}",
		"fn void Box::clear() {}", "@inline pub unsafe fn T Box /* owner */ :: /* method */ take<T>(T value){return value;}",
		"fn void plain() {}",
	} {
		input := "module main; /* 한글 */\r\n" + body
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || oracle.AST == nil || len(oracle.Tree.Diagnostics) != 0 || len(oracle.AST.Functions) != 1 {
			t.Fatalf("function header oracle %q: %v", body, err)
		}
		f := oracle.AST.Functions[0]
		unsafeStart, unsafeEnd := 0, 0
		if f.Unsafe {
			unsafeStart = strings.Index(input, "unsafe")
			unsafeEnd = unsafeStart + len("unsafe")
		}
		fmt.Fprintf(&p, `{let r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2 && integrity::valid(ref parsed));
 let d=syntax::declaration_at(ref parsed,1);
 assert(d.kind==3 && d.body>0 && d.start==%d && d.end==%d && d.name_start==%d && d.name_end==%d && d.public==%t);
 assert(d.unsafe_start==%d && d.unsafe_end==%d && d.owner_start==%d && d.owner_end==%d && syntax::generic_parameter_count(ref parsed,1)==%d);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), f.Span.StartByte, f.Span.EndByte, f.NameSpan.StartByte, f.NameSpan.EndByte, f.Public, unsafeStart, unsafeEnd, f.OwnerSpan.StartByte, f.OwnerSpan.EndByte, len(f.TypeParameters))
	}
	for _, body := range []string{"unsafe extern C fn void f();", "unsafe /* gap */ extern C fn i32 add(i32 a, i32 b);", "@link_name(\"foreign\") pub unsafe extern C fn i32 f(i32 value,);"} {
		input := "module main;" + body
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("foreign oracle %s: %v", body, err)
		}
		root, _ := oracle.Tree.Node(oracle.Tree.Root)
		d, _ := oracle.Tree.Node(root.Children[1])
		if d.Kind != syntax.KindForeign {
			t.Fatal("foreign kind")
		}
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
   assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2 && integrity::valid(ref parsed));
   let syntax::Declaration d=syntax::declaration_at(ref parsed,1);
   assert(d.kind==9 && d.start==%d && d.end==%d && d.name_start==%d && d.name_end==%d && d.result_type>0 && d.body==0 && d.expression==0);
   assert(syntax::statement_count(ref parsed)==0 && syntax::expression_count(ref parsed)==0);
   0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input), d.Span.StartByte, d.Span.EndByte, d.NameSpan.StartByte, d.NameSpan.EndByte)
	}
	for _, body := range []string{"unsafe extern X fn i32 bad();", "unsafe extern C i32 bad();", "unsafe extern C fn i32 bad(i32);", "unsafe extern C fn i32 bad() {}", "unsafe extern C fn i32 bad()"} {
		input := "module main;" + body + "fn void kept(){}"
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
   assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::entry_count(ref parsed)==0 && syntax::type_count(ref parsed)==1 && syntax::statement_count(ref parsed)==1 && integrity::valid(ref parsed));
   let syntax::Declaration d=syntax::declaration_at(ref parsed,1);assert(d.kind==3 && d.name_end-d.name_start==4);
   0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input))
	}
	for _, body := range []string{"unsafe fn void bad();", "fn void T::(){}", "fn void A::B::bad(){}", "unsafe fn void T::bad<T>(T){}", "unsafe extern C fn void T::bad();"} {
		input := "module main;" + body + "fn void kept(){}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) == 0 {
			t.Fatalf("invalid header oracle %q: %v", body, err)
		}
		fmt.Fprintf(&p, `{let r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && integrity::valid(ref parsed));
 let d=syntax::declaration_at(ref parsed,1); assert(d.kind==3 && d.name_end-d.name_start==4 && d.owner_start==0 && d.unsafe_start==0 && syntax::entry_count(ref parsed)==0 && syntax::type_count(ref parsed)==1);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input))
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
