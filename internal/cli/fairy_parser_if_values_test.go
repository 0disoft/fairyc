package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserIfValues(t *testing.T) {
	t.Run("block expressions in statement position", testParserBlockExpressionStatements)
	var p strings.Builder
	p.WriteString("fn i32 main() {\n")
	for _, expr := range []string{
		"if (true) { 1 } else { 2 }",
		"if (a) {}",
		"if (a) { let x=1; x } else if (b) { 2 } else {3}",
		"1 + if(a){2}else{3} * 4",
		"(if(a){x}else{y}).field",
		"{ if(a){1}else{2} }",
		"{ if(a){} let x=1; x }",
		"if(if(a){b}else{c}){1}else{2}",
	} {
		input := "module main; fn void f() { return " + expr + "; }"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("if oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	input := "module main; fn void f(){ return if(a){1}else{2}; }"
	fmt.Fprintf(&p, `{ let i32 r = switch(make(b%s)) { case .ok(parsed): {
 let syntax::Expression e = syntax::expression_at(ref parsed,6);
 assert(e.kind == 13 && e.start == %d && e.end == %d && e.left == 1 && e.right == 5);
 let syntax::Expression branches = syntax::expression_at(ref parsed,e.right);
 assert(branches.kind == 14 && branches.left == 1 && branches.right == 4);
 assert(syntax::statement_at(ref parsed,branches.left).expression == 2);
 let syntax::Expression other = syntax::expression_at(ref parsed,branches.right);
 assert(other.kind == 12 && other.left == 2 && syntax::statement_at(ref parsed,other.left).expression == 3);
 assert(syntax::declaration_at(ref parsed,1).expression == 6); 0
 } case .err(_): { 1 } }; assert(r == 0); }
`, strconv.Quote(input), strings.Index(input, "if("), strings.Index(input, "; }"))
	for _, expr := range []string{"if(a) 1", "if(a){1}else 2", "if(){1}", "if(a){let x=;}else{2}", "if(a){1}else if(b)"} {
		input := "module main; fn void bad(){return " + expr + ";} fn void kept(){}"
		fmt.Fprintf(&p, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) > 0 && syntax::expression_count(ref parsed) == 0 && syntax::entry_count(ref parsed) == 0 && syntax::declaration_count(ref parsed) == 2); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n = 62;
while (n <= 63) {
 var bytes::Buffer data = bytes::empty(); discard bytes::append(ref mut data,b"module main;fn void f(){return ");
 var usize i=0; while(i < n) { discard bytes::append(ref mut data,b"if(a){}else "); i+=1; }
 discard bytes::append(ref mut data,b"if(a){};}");
 let i32 r = switch(make(bytes::view(ref data))) { case .ok(parsed): { assert(n == 62 && syntax::problem_count(ref parsed)==0); 0 } case .err(status): { assert(n == 63 && status == 4); 0 } }; assert(r==0); n+=1;
}

return 0; }`)
	runParserProbe(t, p.String())
}

func testParserBlockExpressionStatements(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, statement := range []string{
		"if(a){1}else{2}", "if(a){1}else if(b){2}else{3}",
		"if(a){1}else{2};", "if(a){1}else{2} + 3;",
		"if(a){x}else{y}.field;", "if(a){x}else{y}(arg);",
		"{ 1 };", "{ 1 } + 2;", "{ x }.field;", "{ x }[0];",
		"{ x } = y;", "if(a){x}else{y} += z;",
	} {
		input := "module main; /* 한글 */\r\nfn void f(){" + statement + "} fn void kept(){}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 || oracle.AST == nil {
			t.Fatalf("statement oracle %q: %v", statement, err)
		}
		s := oracle.AST.Functions[0].Body.Statements[0]
		fmt.Fprintf(&p, `{let r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==3 && integrity::valid(ref parsed));
 let body=syntax::declaration_at(ref parsed,1).body;
 assert(syntax::block_statement_count(ref parsed,body)==1);
 let s=syntax::statement_at(ref parsed,syntax::block_statement(ref parsed,body,0));
 assert(s.start==%d && s.end==%d);
 let e=syntax::expression_at(ref parsed,s.expression); assert(e.start==%d && e.end==%d);
 0 }case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), s.Span.StartByte, s.Span.EndByte, s.Expression.Span.StartByte, s.Expression.Span.EndByte)
	}
	for _, body := range []string{"if(a){let x=;}else{2}", "if(a){1}else{2} + ;", "{x}.;", "1", "while(a){1}", "region r in pool {1}"} {
		input := "module main;fn void bad(){" + body + "} fn void kept(){}"
		fmt.Fprintf(&p, `{let r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input))
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
