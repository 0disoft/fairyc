package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserArrays(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, expr := range []string{"[]", "[0]", "[a,b,]", "[[a],[],[b,c]]", "f([a,b])", "[a,b][0]", "[f<T>(),ref mut x,y as T]"} {
		input := "module main;fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("array oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main;fn void f(){return [a, /*gap*/ b,];}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::expression_count(ref parsed)==5);
 let syntax::Expression array=syntax::expression_at(ref parsed,5);
 assert(array.kind==24 && array.left==0 && array.right==4 && array.start==%d && array.end==%d);
 assert(syntax::array_element_count(ref parsed,5)==2 && syntax::array_element(ref parsed,5,0)==1 && syntax::array_element(ref parsed,5,1)==3);
 let syntax::Expression link=syntax::expression_at(ref parsed,4);
 assert(link.kind==25 && link.left==3 && link.right==2 && link.start==%d && link.end==%d);
 assert(syntax::expression_text(ref parsed,5)[0]==91);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "["), strings.Index(input, "]")+1, strings.Index(input, "b,"), strings.Index(input, "b,")+1)
	p.WriteString(`{let i32 r=switch(make(b"module main;fn void f(){return [];}")){case .ok(parsed):{assert(syntax::expression_count(ref parsed)==1 && syntax::array_element_count(ref parsed,1)==0 && syntax::expression_at(ref parsed,1).right==0);0}case .err(_):{1}};assert(r==0);}`)
	for _, expr := range []string{"[,]", "[a,,b]", "[a b]", "[a,", "[x;]", "[[a],"} {
		input := "module main;fn void bad(){return " + expr + ";}fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize depth=63;
while(depth<=64) {
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){return ");
 var usize i=0;while(i<depth){discard bytes::append(ref mut data,b"[");i+=1;}
 discard bytes::append(ref mut data,b"0");i=0;while(i<depth){discard bytes::append(ref mut data,b"]");i+=1;}
 discard bytes::append(ref mut data,b";}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(depth==63 && syntax::problem_count(ref parsed)==0);0}case .err(status):{assert(depth==64 && status==4);0}};assert(r==0);depth+=1;
}
var usize count=2047;
while(count<=2048) {
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){return +[");
 var usize i=0;while(i<count){discard bytes::append(ref mut data,b"0,");i+=1;}
 discard bytes::append(ref mut data,b"];}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(count==2047 && syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==4096);0}case .err(status):{assert(count==2048 && status==4);0}};assert(r==0);count+=1;
}
return 0;}`)
	runParserProbe(t, p.String())
}
