package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserGenericCalls(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, expr := range []string{"f<T>()", "f<ptr T>()", "v::push<Box<Box<T>>>(x)", "f<ref mut u8[],T[4],>(x,y)", "f<T>()(1).x[0]", "a < b", "a >> b", "f < T > (x)"} {
		input := "module main; fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("generic call oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main; fn void f(){return f<T,U>(x);}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::expression_count(ref parsed)==5 && syntax::type_count(ref parsed)==5);
 let syntax::Expression call=syntax::expression_at(ref parsed,5);
 assert(call.kind==6 && call.left==2 && syntax::call_argument(ref parsed,5,0)==3);
 let syntax::Expression spec=syntax::expression_at(ref parsed,2);
 assert(spec.kind==20 && spec.left==1 && spec.right==5 && spec.start==%d && spec.end==%d);
 assert(syntax::specialization_argument_count(ref parsed,2)==2 && syntax::specialization_argument(ref parsed,2,0)==2 && syntax::specialization_argument(ref parsed,2,1)==4);
 assert(syntax::type_text(ref parsed,2)[0]==84 && syntax::type_text(ref parsed,4)[0]==85);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "f<T"), strings.Index(input, "(x)"))
	for _, expr := range []string{"f<>()", "f<,T>()", "f<T,,U>()", "f<T>>()", "f<ptr>()", "f<T>{}", "f<T>(;)", "a < b < c"} {
		input := "module main; fn void bad(){return " + expr + ";} fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::expression_count(ref parsed)==0 && syntax::type_count(ref parsed)==1);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	for _, n := range []int{126, 127} {
		comma := ""
		if n == 126 {
			comma = ","
		}
		input := "module main; fn void f(){return f<T" + strings.Repeat(" /*gap*/ []", n) + comma + ">();}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert((syntax::problem_count(ref parsed)==0)==%t);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input), n == 126)
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
