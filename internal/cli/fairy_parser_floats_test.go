package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserFloats(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){\n")
	for _, raw := range []string{"0.0", "1e3", "1_024.5e-2", "1E+03", "1.25e-10", "999999999999999999999999.0", "1.2__3"} {
		input := "module main;fn f64 f(){return " + raw + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("float oracle %s: %v", raw, err)
		}
		var token lexer.Token
		found := false
		for _, candidate := range oracle.Tree.Tokens {
			if candidate.Kind == lexer.Float {
				token = candidate
				found = true
				break
			}
		}
		if !found {
			t.Fatal("missing float token")
		}
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
   assert(syntax::problem_count(ref parsed)==0 && syntax::expression_count(ref parsed)==1 && integrity::valid(ref parsed));
   let syntax::Expression e=syntax::expression_at(ref parsed,1);
   assert(e.kind==1 && e.left==0 && e.right==0 && e.start==%d && e.end==%d && syntax::literal_kind(ref parsed,1)==6);
`, strconv.Quote(input), token.Span.StartByte, token.Span.EndByte)
		for _, view := range []struct{ getter, want string }{{"expression_text", raw}, {"literal_cooked", token.Cooked}} {
			fmt.Fprintf(&p, "{let ref u8[] s=syntax::%s(ref parsed,1);assert(s.len==%d);", view.getter, len(view.want))
			for i, b := range []byte(view.want) {
				fmt.Fprintf(&p, "assert(s[%d]==%d);", i, b)
			}
			p.WriteString("}")
		}
		p.WriteString("0}case .err(_):{1}};assert(r==0);}\n")
	}
	// Keep rejection expectations tied to the current lexer, not guessed spelling rules.
	for _, raw := range []string{"1e", "1e+", "1.0suffix"} {
		input := "module main;fn f64 f(){return " + raw + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) == 0 {
			t.Fatalf("invalid float oracle %s: %v", raw, err)
		}
		fmt.Fprintf(&p, "{let i32 r=switch(make(b%s)){case .ok(_):{0}case .err(status):{status}};assert(r==5);}\n", strconv.Quote(input))
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
