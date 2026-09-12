package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserStaticAssertions(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, body := range []string{
		"static_assert(true);", "static_assert(false);", "static_assert(1+2==3);",
		"@tag(1) pub static_assert(true);",
		"const i32 A=2;static_assert(A==2);",
		"fn i32 static_assert(){return 1;}static_assert(static_assert()==1);",
		"fn i32 f(){let static_assert=1;return static_assert;}static_assert(true);",
	} {
		input := "module main;" + body
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("static assertion oracle %s: %v", body, err)
		}
		root, _ := oracle.Tree.Node(oracle.Tree.Root)
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==%d && integrity::valid(ref parsed));", strconv.Quote(input), len(root.Children))
		for index, id := range root.Children {
			d, _ := oracle.Tree.Node(id)
			if d.Kind != syntax.KindStaticAssert {
				continue
			}
			condition, _ := oracle.Tree.Node(d.Children[len(d.Children)-1])
			fmt.Fprintf(&p, `{let syntax::Declaration d=syntax::declaration_at(ref parsed,%d);
    assert(d.kind==8 && d.start==%d && d.end==%d && d.name_start==0 && d.name_end==0 && d.body==0 && d.result_type==0 && d.expression>0);
    let syntax::Expression e=syntax::expression_at(ref parsed,d.expression);
    assert(e.start==%d && e.end==%d);
   }`, index, d.Span.StartByte, d.Span.EndByte, condition.Span.StartByte, condition.Span.EndByte)
		}
		p.WriteString("0}case .err(_):{1}};assert(r==0);}\n")
	}
	for _, body := range []string{"static_assert();", "static_assert true;", "static_assert(true;", "static_assert(true)", "@tag static_assert();", "pub static_assert(1+);"} {
		input := "module main;" + body + "fn void kept(){}"
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::type_count(ref parsed)==1 && syntax::statement_count(ref parsed)==1 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}\n", strconv.Quote(input))
	}
	p.WriteString(`
 {let i32 r=switch(make(b"module main;static_assert(1+2==3);")){case .ok(parsed):{
  let syntax::Expression e=syntax::expression_at(ref parsed,syntax::declaration_at(ref parsed,1).expression);
  assert(e.kind==5 && e.left==3 && e.right==4 && syntax::expression_at(ref parsed,e.left).kind==5);
  0}case .err(_):{1}};assert(r==0);}
 var usize n=511;while(n<=512){
  var bytes::Buffer data=bytes::empty();discard bytes::append(ref mut data,b"module main;");
  var usize i=0;while(i<n){discard bytes::append(ref mut data,b"static_assert(true);");i+=1;}
  let i32 r=switch(make(bytes::view(ref data))){case .ok(parsed):{assert(n==511 && syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==512 && integrity::valid(ref parsed));0}case .err(status):{assert(n==512 && status==4);0}};assert(r==0);n+=1;
 }return 0;}`)
	runParserProbe(t, p.String())
}
