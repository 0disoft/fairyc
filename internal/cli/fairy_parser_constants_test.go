package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserConstants(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, declaration := range []string{"const i32 A = 40 + 2;", "pub const bool READY = true;", "const i32[2] ITEMS = [1,2];", "const i32 A = other::VALUE;", "@tag(1) pub const i32 A = (1 + 2) * 3;"} {
		input := "module main;" + declaration
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("const oracle %s: %v", declaration, err)
		}
		tree := oracle.Tree
		root, _ := tree.Node(tree.Root)
		d, _ := tree.Node(root.Children[1])
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
   assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2 && integrity::valid(ref parsed));
   let syntax::Declaration d=syntax::declaration_at(ref parsed,1);
   assert(d.kind==7 && d.body==0 && d.result_type>0 && d.expression>0);
   assert(d.start==%d && d.end==%d && d.name_start==%d && d.name_end==%d);
   assert(d.public==%t);
   0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), d.Span.StartByte, d.Span.EndByte, d.NameSpan.StartByte, d.NameSpan.EndByte, strings.Contains(declaration, "pub "))
	}
	for _, declaration := range []string{"const A=1;", "const i32 =1;", "const i32 A=;", "const i32 A=1", "pub const i32 A=(1+);", "@tag const i32 A=;"} {
		input := "module main;" + declaration + " fn void kept(){}"
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
   assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2);
   assert(syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::statement_count(ref parsed)==1 && syntax::type_count(ref parsed)==1);
   assert(integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input))
	}
	p.WriteString(`
 {let i32 r=switch(make(b"module main;const i32 A=40+2;")){case .ok(parsed):{
  let syntax::Declaration d=syntax::declaration_at(ref parsed,1);
  let syntax::Expression e=syntax::expression_at(ref parsed,d.expression);
  assert(e.kind==5 && e.left==1 && e.right==2 && e.start==24 && e.end==28);
  assert(syntax::type_at(ref parsed,d.result_type).start==18 && syntax::type_at(ref parsed,d.result_type).end==21);
  0}case .err(_):{1}};assert(r==0);}
 var usize n=511;while(n<=512){
  var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;");
  var usize i=0;while(i<n){discard bytes::append(ref mut data,b"const i32 A=1;");i+=1;}
  let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==511 && syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==512 && integrity::valid(ref parsed));0}case .err(status):{assert(n==512 && status==4);0}};assert(r==0);n+=1;
 }return 0;}`)
	runParserProbe(t, p.String())
}
