package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserVariants(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, expr := range []string{".none", ".ok(1)", ".err(status)", ".ok(.err(x))", ".ok(x,y,).field[0]", ". /*gap*/ ok()"} {
		input := "module main;fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("variant oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main;fn void f(){return . /*gap*/ ok(x);}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 let syntax::Expression name=syntax::expression_at(ref parsed,1);
 assert(name.kind==28 && name.left==0 && name.right==0 && name.start==%d && name.end==%d);
 assert(syntax::expression_text(ref parsed,1).len==2);
 let syntax::Expression call=syntax::expression_at(ref parsed,4);
 assert(call.kind==6 && call.left==1 && syntax::call_argument(ref parsed,4,0)==2);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, ". /*"), strings.Index(input, "ok(")+2)
	for _, expr := range []string{".", ".true", ".ok(,)", ".ok(x", ".R {.x=1}"} {
		input := "module main;fn void bad(){return " + expr + ";}fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n=63;while(n<=64){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){return ");
 var usize i=0;while(i<n){discard bytes::append(ref mut data,b".ok(");i+=1;}
 discard bytes::append(ref mut data,b"0");i=0;while(i<n){discard bytes::append(ref mut data,b")");i+=1;}
 discard bytes::append(ref mut data,b";}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==63 && syntax::problem_count(ref parsed)==0);0}case .err(status):{assert(n==64 && status==4);0}};assert(r==0);n+=1;
}
return 0;}`)
	runParserProbe(t, p.String())
}
