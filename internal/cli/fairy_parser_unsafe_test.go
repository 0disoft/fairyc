package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserUnsafe(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){")
	for _, expr := range []string{"unsafe {41}", "unsafe /*gap*/ {add(41)}", "unsafe {unsafe {1}}", "unsafe {let a=1;a+2}"} {
		input := "module main;fn i32 f(){return " + expr + ";}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("unsafe oracle %s: %v", expr, err)
		}
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
   assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));
   let syntax::Expression e=syntax::expression_at(ref parsed,syntax::declaration_at(ref parsed,1).expression);
   assert(e.kind==31 && e.start==%d && e.end==%d && e.right==0 && syntax::statement_at(ref parsed,e.left).kind==1);
   0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input), strings.Index(input, "unsafe"), strings.Index(input, "unsafe")+len(expr))
	}
	for _, body := range []string{"unsafe {} return 1;", "unsafe {};return 1;", "return unsafe {1}+2;", "return unsafe {f()}?;"} {
		input := "module main;fn i32 f(){" + body + "}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("unsafe combination oracle %s", body)
		}
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input))
	}
	for _, body := range []string{"return unsafe 1;", "return unsafe {1+};", "unsafe;"} {
		input := "module main;fn i32 bad(){" + body + "}fn void kept(){}"
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::expression_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input))
	}
	for _, depth := range []int{31, 33} {
		input := "module main;fn i32 f(){return " + strings.Repeat("unsafe {", depth) + "1" + strings.Repeat("}", depth) + ";}"
		if depth == 31 {
			fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input))
		} else {
			fmt.Fprintf(&p, `assert(switch(make(b%s)){case .ok(_):{false}case .err(status):{status==4}});`, strconv.Quote(input))
		}
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
