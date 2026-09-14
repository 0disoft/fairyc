package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	gosyntax "github.com/0disoft/fairyc/internal/syntax"
)

func TestWindowsFairyParserRegions(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, arena := range []string{"pool", "ref mut pool", "pool + other", "get<T>()", "(Pool{.x=1})", "get(Pool{.x=1})", "slots[Pool{.x=1}.x]", "if(true){pool}else{other}"} {
		input := "module main; /* 한글 */\r\nfn void f(){region /* name */ scratch in " + arena + " {burn token;}}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("region oracle %q: %v", arena, err)
		}
		var region gosyntax.Node
		var visit func(gosyntax.NodeID)
		visit = func(id gosyntax.NodeID) {
			n, ok := oracle.Tree.Node(id)
			if !ok {
				t.Fatal("missing CST node")
			}
			if n.Kind == gosyntax.KindRegion {
				region = n
			}
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(oracle.Tree.Root)
		if region.Kind != gosyntax.KindRegion || len(region.Children) != 2 || region.Text != "scratch" {
			t.Fatal("region CST shape")
		}
		a, _ := oracle.Tree.Node(region.Children[0])
		b, _ := oracle.Tree.Node(region.Children[1])
		nameStart := strings.Index(input, "scratch")
		fmt.Fprintf(&p, `{let r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));
 let outer=syntax::statement_at(ref parsed,syntax::declaration_at(ref parsed,1).body);
 let id=syntax::statement_at(ref parsed,outer.first).first; var s=syntax::statement_at(ref parsed,id);
 assert(outer.second==1 && s.kind==17 && s.start==%d && s.end==%d && s.second==0 && s.entry==0 && s.operator==0);
 let a=syntax::expression_at(ref parsed,s.expression); let b=syntax::statement_at(ref parsed,s.first); let name=syntax::expression_at(ref parsed,s.target);
 assert(a.start==%d && a.end==%d && b.kind==1 && b.start==%d && b.end==%d && b.second==1);
 assert(name.kind==17 && name.start==%d && name.end==%d && syntax::expression_text(ref parsed,s.target).len==7);
 let original=s; s.target=0; assert(!integrity::valid_statement(ref parsed,s,id));
 s=original;s.expression=0;assert(!integrity::valid_statement(ref parsed,s,id));
 s=original;s.first=id;assert(!integrity::valid_statement(ref parsed,s,id));
 assert(integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), region.Span.StartByte, region.Span.EndByte, a.Span.StartByte, a.Span.EndByte, b.Span.StartByte, b.Span.EndByte, nameStart, nameStart+len("scratch"))
	}
	for _, body := range []string{"region in pool {}", "region scratch pool {}", "region scratch in {}", "region scratch in pool;", "region scratch in pool {1}", "return region scratch in pool {};"} {
		input := "module main;fn void bad(){" + body + "}fn void kept(){}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) == 0 {
			t.Fatalf("invalid region oracle %q: %v", body, err)
		}
		fmt.Fprintf(&p, "assert(switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::expression_count(ref parsed)==0 && syntax::entry_count(ref parsed)==0 && syntax::statement_count(ref parsed)==1 && syntax::declaration_count(ref parsed)==2 && integrity::valid(ref parsed));0}case .err(_):{1}}==0);\n", strconv.Quote(input))
	}
	for _, n := range []int{31, 32} {
		input := "module main;fn void f(){" + strings.Repeat("region r in pool {", n) + strings.Repeat("}", n) + "}"
		fmt.Fprintf(&p, "assert(switch(make(b%s)){case .ok(parsed):{assert(%d==31 && syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(status):{assert(%d==32 && status==4);0}}==0);\n", strconv.Quote(input), n, n)
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
