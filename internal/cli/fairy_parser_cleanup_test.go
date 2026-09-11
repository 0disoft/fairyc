package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserCleanupBodies(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, body := range []string{"", "let x=1;assert(x==1);discard x;", "while(true){break;}", "if(true){discard f();}else{assert(false);}", "let x={1};", "return;"} {
		input := "module main;resource R {cleanup {" + body + "}}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("cleanup oracle %q: %v", body, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_at(ref parsed,1).cleanup_body!=0);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	input := "module main;resource R {u8 x;cleanup {let y=1;assert(y==1);discard y;}}fn void kept(){}"
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 let syntax::Declaration d=syntax::declaration_at(ref parsed,1);
 assert(d.kind==6 && d.body==0 && d.cleanup_body==7 && d.entry_start==0 && d.entry_count==1);
 assert(d.cleanup_start==%d && d.cleanup_end==%d && syntax::statement_count(ref parsed)==8);
 assert(syntax::block_statement_count(ref parsed,d.cleanup_body)==3 && syntax::statement_at(ref parsed,5).kind==12);
 assert(syntax::entry_at(ref parsed,0).expression==1 && syntax::declaration_at(ref parsed,2).cleanup_body==0);
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "cleanup"), strings.Index(input, "}}")+1)
	for _, body := range []string{"assert();", "let x=;", "discard x", "1", "fn void nested(){}"} {
		input := "module main;@bad resource R {u8 x;cleanup {" + body + "}}fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::field_count(ref parsed)==0 && syntax::attribute_count(ref parsed)==0 && syntax::type_count(ref parsed)==1 && syntax::statement_count(ref parsed)==1);0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
var usize n=2047;while(n<=2048){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;fn void f(){} resource R {cleanup {");
 var usize i=0;while(i<n){discard bytes::append(ref mut data,b"discard 1;");i+=1;}
 discard bytes::append(ref mut data,b"}}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==2047 && syntax::problem_count(ref parsed)==0 && syntax::statement_count(ref parsed)==4096);0}case .err(status):{assert(n==2048 && status==4);0}};assert(r==0);n+=1;
}
var usize depth=31;while(depth<=32){
 var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;resource R {cleanup {");
 var usize i=0;while(i<depth){discard bytes::append(ref mut data,b"{");i+=1;}
 i=0;while(i<depth){discard bytes::append(ref mut data,b"}");i+=1;}
 discard bytes::append(ref mut data,b"}}");
 let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(depth==31 && syntax::problem_count(ref parsed)==0);0}case .err(status):{assert(depth==32 && status==4);0}};assert(r==0);depth+=1;
}
return 0;}`)
	runParserProbe(t, p.String())
}
