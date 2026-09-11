package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserSwitchValues(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main() {\n")
	for _, expr := range []string{
		"switch(x){}",
		"switch(x){case .ok(value):{value} case .err(_):{0}}",
		"switch(x){case .none:{} case .none:{1}}",
		"switch(f()){case .ok(x):{let y=x; if(y){1}else{2}}}",
		"1 + switch(x){case .ok(y):{y}} * 2",
		"switch(x){case .ok(y):{switch(y){case .done:{1}}}}",
	} {
		input := "module main; fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("switch oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 0); 0 } case .err(_): { 1 } }; assert(r == 0); }\n", strconv.Quote(input))
	}
	input := "module main; fn void f(){return switch(x){case .ok(v):{v} case .err(_):{0}};}"
	fmt.Fprintf(&p, `{ let i32 r = switch(make(b%s)) { case .ok(parsed): {
 let syntax::Expression e = syntax::expression_at(ref parsed,12);
 assert(e.kind == 15 && e.start == %d && e.end == %d && e.left == 1 && e.right == 11);
 assert(syntax::switch_case_count(ref parsed,12) == 2 && syntax::switch_case(ref parsed,12,0) == 5 && syntax::switch_case(ref parsed,12,1) == 10);
 let syntax::Expression arm = syntax::expression_at(ref parsed,5);
 assert(arm.kind == 18 && arm.left == 3 && arm.right == 1 && arm.start == %d && arm.end == %d);
 let syntax::Expression pat = syntax::expression_at(ref parsed,arm.left);
 assert(pat.kind == 16 && pat.left == 2 && pat.right == 0 && pat.start == %d && pat.end == %d);
 assert(syntax::expression_text(ref parsed,3).len == 2 && syntax::expression_text(ref parsed,2)[0] == 118);
 assert(syntax::expression_at(ref parsed,7).kind == 17 && syntax::expression_text(ref parsed,7)[0] == 95);
 assert(syntax::statement_at(ref parsed,1).expression == 4 && syntax::statement_at(ref parsed,2).expression == 9);
 assert(syntax::declaration_at(ref parsed,1).expression == 12); 0
 } case .err(_): { 1 } }; assert(r == 0); }
`, strconv.Quote(input), strings.Index(input, "switch("), strings.LastIndex(input, ";"), strings.Index(input, "case"), strings.Index(input, " case .err"), strings.Index(input, ".ok"), strings.Index(input, ":{v}"))
	for _, expr := range []string{"switch(x){case _: {1}}", "switch(x){case .ok():{1}}", "switch(x){case .ok(a,b):{1}}", "switch(x){case .ok(x) {1}}", "switch(x){case .ok(x):1}", "switch(x){.ok:{1}}", "switch(x){case .ok{.x=x}:{1}}", "switch(){case .ok:{1}}"} {
		input := "module main; fn void bad(){return " + expr + ";} fn void kept(){}"
		fmt.Fprintf(&p, "{ let i32 r = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) > 0 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2); 0 } case .err(_): { 1 } }; assert(r==0); }\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n=818;
while(n<=819) {
 var bytes::Buffer data=bytes::empty(); discard bytes::append(ref mut data,b"module main;fn void f(){return switch(a+a+a){");
 var usize i=0; while(i<n) { discard bytes::append(ref mut data,b"case .ok(x):{1}"); i+=1; }
 discard bytes::append(ref mut data,b"};}");
 let i32 r=switch(make(bytes::view(ref data))) { case .ok(parsed): { assert(n==818 && syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==4096 && syntax::switch_case_count(ref parsed,4096)==818); 0 } case .err(status): { assert(n==819 && status==4); 0 } }; assert(r==0); n+=1;
}
n=31;
while(n<=32) {
 var bytes::Buffer data=bytes::empty(); discard bytes::append(ref mut data,b"module main;fn void f(){return ");
 var usize i=0; while(i<n) { discard bytes::append(ref mut data,b"switch(x){case .ok:{"); i+=1; }
 discard bytes::append(ref mut data,b"1");
 i=0; while(i<n) { discard bytes::append(ref mut data,b"}}"); i+=1; }
 discard bytes::append(ref mut data,b";}");
 let i32 r=switch(make(bytes::view(ref data))) { case .ok(parsed): { assert(n==31 && syntax::problem_count(ref parsed)==0); 0 } case .err(status): { assert(n==32 && status==4); 0 } }; assert(r==0); n+=1;
}
return 0; }`)
	runParserProbe(t, p.String())
}
