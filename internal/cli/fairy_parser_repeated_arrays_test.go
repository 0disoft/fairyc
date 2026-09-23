package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserRepeatedArrays(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, expr := range []string{"[0;4]", "[f();0]", "[x;N]", "[x;pkg::N]", "[[x;2];3]", "[.ok(x);2][0]", "[x;0xff]"} {
		input := "module main;fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("repeat oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main;fn void f(){return [f(); /*count*/ pkg::N];}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::expression_count(ref parsed)==4);
 let syntax::Expression repeated=syntax::expression_at(ref parsed,4);
 assert(repeated.kind==29 && repeated.left==2 && repeated.right==3 && repeated.start==%d && repeated.end==%d);
 assert(syntax::expression_at(ref parsed,2).kind==6 && syntax::expression_at(ref parsed,3).kind==2);
 assert(syntax::expression_text(ref parsed,3).len==6);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "["), strings.Index(input, "]")+1)
	for _, expr := range []string{"[;2]", "[x;]", "[x;2,]", "[x,y;2]", "[x;N+1]", "[x;-1]", "[x;f()]", "[x;(N)]", "[x;true]", "[x;pkg::]"} {
		input := "module main;fn void bad(){return " + expr + ";}fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n=1039;while(n<=1040){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){return +++[");
 var usize i=0;while(i<n){discard bytes::append(ref mut data,b"[0;1],");i+=1;}
 discard bytes::append(ref mut data,b"];}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==1039 && syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==4160);0}case .err(status):{assert(n==1040 && status==4);0}};assert(r==0);n+=1;
}
var usize depth=63;while(depth<=64){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){return ");
 var usize i=0;while(i<depth){discard bytes::append(ref mut data,b"[");i+=1;}
 discard bytes::append(ref mut data,b"0");i=0;while(i<depth){discard bytes::append(ref mut data,b";1]");i+=1;}
 discard bytes::append(ref mut data,b";}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(depth==63 && syntax::problem_count(ref parsed)==0);0}case .err(status):{assert(depth==64 && status==4);0}};assert(r==0);depth+=1;
}
return 0;}`)
	runParserProbe(t, p.String())
}
