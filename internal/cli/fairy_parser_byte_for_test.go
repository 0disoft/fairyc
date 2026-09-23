package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestWindowsFairyParserByteFor(t *testing.T) {
	input := "module main;fn void f(ref u8[] view){for(let u8 byte in view){continue;}}"
	oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
		t.Fatalf("byte-for oracle: %v", err)
	}
	var probe strings.Builder
	probe.WriteString("fn i32 main(){\n")
	fmt.Fprintf(&probe, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));
 let usize block=syntax::declaration_at(ref parsed,1).body;
 let usize id=syntax::statement_at(ref parsed,syntax::statement_at(ref parsed,block).first).first;
 var syntax::Statement loop=syntax::statement_at(ref parsed,id);
 assert(loop.kind==18 && loop.start==%d && loop.end==%d);
 assert(syntax::expression_at(ref parsed,loop.target).kind==17);
 assert(syntax::expression_at(ref parsed,loop.expression).kind==2);
 assert(syntax::type_at(ref parsed,loop.second).kind==1);
 assert(syntax::statement_at(ref parsed,loop.first).kind==1);
 loop.second=0;assert(!integrity::valid_statement(ref parsed,loop,id));
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input), strings.Index(input, "for"), len(input)-1)
	for _, bad := range []string{
		"for(let u8 byte in){}",
		"for(let u8 byte in view {}",
	} {
		input := "module main;fn void f(){" + bad + "}fn void kept(){}"
		fmt.Fprintf(&probe, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
 assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && integrity::valid(ref parsed));
 0}case .err(_):{1}};assert(r==0);}
`, strconv.Quote(input))
	}
	probe.WriteString("return 0;}")
	runParserProbe(t, probe.String())
}
