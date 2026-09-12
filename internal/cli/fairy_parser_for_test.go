package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserFor(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for mask := 0; mask < 8; mask++ {
		initial, condition, post := "", "", ""
		if mask&1 != 0 {
			initial = "var i32 i=0"
		}
		if mask&2 != 0 {
			condition = "i<3"
		}
		if mask&4 != 0 {
			post = "i+=1"
		}
		input := "module main;fn void f(){for(" + initial + ";" + condition + ";" + post + "){}}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("for oracle %s: %v", input, err)
		}
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
   assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));
   let usize block=syntax::declaration_at(ref parsed,1).body;
   let usize id=syntax::statement_at(ref parsed,syntax::statement_at(ref parsed,block).first).first;
   var syntax::Statement loop=syntax::statement_at(ref parsed,id);
   assert(loop.kind==14 && loop.start==%d && loop.end==%d);
   assert((loop.expression!=0)==%t && syntax::statement_at(ref parsed,loop.first).kind==1);
   let usize header_id=loop.second;
   var syntax::Statement header=syntax::statement_at(ref parsed,header_id);
   assert(header.kind==15 && (header.first!=0)==%t && (header.second!=0)==%t);
   assert(header.start==%d && header.end==%d);
   if(header.first!=0){assert(syntax::statement_at(ref parsed,header.first).kind==2);}
   if(header.second!=0){assert(syntax::statement_at(ref parsed,header.second).kind==4);}
   loop.second=loop.first;assert(!integrity::valid_statement(ref parsed,loop,id));
   header.second=header_id;assert(!integrity::valid_statement(ref parsed,header,header_id));
   0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "for"), len(input)-1, mask&2 != 0, mask&1 != 0, mask&4 != 0, strings.Index(input, "for")+3, strings.LastIndex(input, "){")+1)
	}
	for _, body := range []string{"for(i=0;i<3;i+=1){continue;}", "for(let i=0;;){}", "for(;;){for(;;){break;}}", "for(;;){defer f();break;}"} {
		input := "module main;fn void f(){" + body + "}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("for oracle %s: %v", body, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	for _, body := range []string{"for(;){}", "for(;;{}", "for(;;);", "for(return 1;;){}", "for(f();;){}", "for(;;f()){}", "for(;;let i=1){}", "for(var i=;true;i+=1){}", "for(;;i+=){}", "for(;;){1}"} {
		input := "module main;fn void bad(){" + body + "}fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::statement_count(ref parsed)==1 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
 var usize n=1023;while(n<=1024){
  var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){");
  var usize i=0;while(i<n){discard bytes::append(ref mut data,b"for(;;){}");i+=1;}
  discard bytes::append(ref mut data,b"}");
  let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==1023 && syntax::problem_count(ref parsed)==0 && syntax::statement_count(ref parsed)==4093 && integrity::valid(ref parsed));0}case .err(status):{assert(n==1024 && status==4);0}};assert(r==0);n+=1;
 }
 n=31;while(n<=32){
  var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){");
  var usize i=0;while(i<n){discard bytes::append(ref mut data,b"for(;;){");i+=1;}
  i=0;while(i<=n){discard bytes::append(ref mut data,b"}");i+=1;}
  let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==31 && syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(status):{assert(n==32 && status==4);0}};assert(r==0);n+=1;
 }return 0;}`)
	runParserProbe(t, p.String())
}
