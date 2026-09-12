package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserDefers(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, body := range []string{"defer x;", "defer f<T>(ref mut x);", "defer a[..];", "let x={defer f();1};", "defer {assert(true);1};", "defer 1;"} {
		input := "module main;fn void f(){" + body + "}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("defer oracle %q: %v", body, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main;fn void f(){defer x;}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 var syntax::Statement s=syntax::statement_at(ref parsed,1);
 assert(s.kind==13 && s.expression==1 && s.entry==0 && s.start==%d && s.end==%d);
 assert(syntax::entry_count(ref parsed)==0 && syntax::declaration_at(ref parsed,1).expression==0 && integrity::valid(ref parsed));
 s.expression=0;assert(!integrity::valid_statement(ref parsed,s,1));
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "defer"), strings.Index(input, "x;")+2)
	for _, body := range []string{"defer;", "defer x", "defer x,y;", "return defer x;", "let x={defer f()};"} {
		input := "module main;fn void bad(){" + body + "}fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::statement_count(ref parsed)==1 && syntax::declaration_count(ref parsed)==2);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n=2047;while(n<=2048){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){");
 var usize i=0;while(i<n){discard bytes::append(ref mut data,b"defer 1;");i+=1;}
 discard bytes::append(ref mut data,b"}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==2047 && syntax::problem_count(ref parsed)==0 && syntax::statement_count(ref parsed)==4095);0}case .err(status):{assert(n==2048 && status==4);0}};assert(r==0);n+=1;
}
return 0;}`)
	runParserProbe(t, p.String())
}
