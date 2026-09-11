package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserCasts(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, expr := range []string{"x as T", "x as A as B", "-x as T * y", "a * b as T", "ref mut x as ref T", "(x as T).field", "f<T>() as Box<T[4]>", "(x as T) < y"} {
		input := "module main;fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("cast oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main;fn void f(){return -x as T * y;}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 let syntax::Expression binary=syntax::expression_at(ref parsed,5);
 assert(binary.kind==5 && binary.left==3 && binary.right==4);
 let syntax::Expression cast=syntax::expression_at(ref parsed,3);
 assert(cast.kind==23 && cast.left==2 && cast.right==2 && cast.start==%d && cast.end==%d);
 assert(syntax::expression_at(ref parsed,2).kind==4 && syntax::type_text(ref parsed,2)[0]==84 && syntax::expression_text(ref parsed,3).len==2);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "-x"), strings.Index(input, " *"))
	for _, expr := range []string{"x as", "x as 1", "x as T<", "x as T[-1]", "x as ptr T"} {
		input := "module main;fn void bad(){return " + expr + ";}fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::type_count(ref parsed)==1 && syntax::declaration_count(ref parsed)==2);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n=4095;
while(n<=4096) {
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){return x");
 var usize i=0;while(i<n){discard bytes::append(ref mut data,b" as T");i+=1;}
 discard bytes::append(ref mut data,b";}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==4095 && syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==4096 && syntax::type_count(ref parsed)==4096);0}case .err(status):{assert(n==4096 && status==4);0}};assert(r==0);n+=1;
}
return 0;}`)
	runParserProbe(t, p.String())
}
