package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserReferences(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, expr := range []string{"ref x", "ref mut x", "ref /*gap*/ mut x", "*ref mut a", "ref a.b[0] + c", "(ref a).b", "f(ref a,ref mut b)", "ref ref mut a", "ref 1"} {
		input := "module main; fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("reference oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main; fn void f(){return ref /*gap*/ mut a.b[0] + c;}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 let syntax::Expression e=syntax::expression_at(ref parsed,7);
 assert(e.kind==5 && e.left==5 && e.right==6);
 let syntax::Expression r=syntax::expression_at(ref parsed,5);
 assert(r.kind==22 && r.left==4 && r.right==0 && r.start==%d && r.end==%d);
 assert(syntax::expression_at(ref parsed,4).kind==9 && syntax::expression_text(ref parsed,5).len==3);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "ref "), strings.Index(input, " + c"))
	for _, expr := range []string{"ref", "ref mut", "ref mut +", "ref mut mut x"} {
		input := "module main; fn void bad(){return " + expr + ";} fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	for _, n := range []int{63, 64} {
		input := "module main; fn void f(){return " + strings.Repeat("ref ", n) + "x;}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(%t && syntax::problem_count(ref parsed)==0);0}case .err(status):{assert(%t && status==4);0}};assert(r==0);}\n", strconv.Quote(input), n == 63, n == 64)
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
