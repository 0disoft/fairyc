package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserSlices(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, expr := range []string{"a[b..c]", "a[..c]", "a[b..]", "a[..]", "a[0]", "f<T>()[x+1..end()].field[0]", "a[..b[..c]]", "ref mut a[..]"} {
		input := "module main;fn void f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("slice oracle %q: %v", expr, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	for _, tc := range []struct {
		expr       string
		start, end int
	}{{"a[b..c]", 2, 3}, {"a[..c]", 0, 2}, {"a[b..]", 2, 0}, {"a[..]", 0, 0}} {
		input := "module main;fn void f(){return " + tc.expr + ";}"
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 let syntax::Expression slice=syntax::expression_at(ref parsed,syntax::expression_count(ref parsed));
 assert(slice.kind==26 && slice.left==1 && slice.start==%d && slice.end==%d);
 let syntax::Expression bounds=syntax::expression_at(ref parsed,slice.right);
 assert(bounds.kind==27 && bounds.left==%d && bounds.right==%d && bounds.start==%d && bounds.end==%d);
 assert(syntax::expression_text(ref parsed,slice.right).len==2);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "return ")+7, strings.Index(input, "]")+1, tc.start, tc.end, strings.Index(input, "[")+1, strings.Index(input, "]"))
	}
	for _, expr := range []string{"a[]", "a[....]", "a[x..y..z]", "a[..,]", "a[x..", "a[x,y]"} {
		input := "module main;fn void bad(){return " + expr + ";}fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n=2079;while(n<=2080){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){return +a");
 var usize i=0;while(i<n){discard bytes::append(ref mut data,b"[..]");i+=1;}
 discard bytes::append(ref mut data,b";}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==2079 && syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==4160);0}case .err(status):{assert(n==2080 && status==4);0}};assert(r==0);n+=1;
}
var usize depth=63;while(depth<=64){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){return ");
 var usize i=0;while(i<depth){discard bytes::append(ref mut data,b"a[..");i+=1;}
 discard bytes::append(ref mut data,b"a");i=0;while(i<depth){discard bytes::append(ref mut data,b"]");i+=1;}
 discard bytes::append(ref mut data,b";}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(depth==63 && syntax::problem_count(ref parsed)==0);0}case .err(status):{assert(depth==64 && status==4);0}};assert(r==0);depth+=1;
}
return 0;}`)
	runParserProbe(t, p.String())
}
