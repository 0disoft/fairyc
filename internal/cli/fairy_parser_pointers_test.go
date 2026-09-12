package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserPointers(t *testing.T) {
	var helpers, main strings.Builder
	main.WriteString("fn i32 main(){")
	for i, typ := range []string{"ptr void", "ptr mut u8", "ptr u8[16]", "(ptr u8)[16]", "ptr mut ptr u8", "ref mut ptr u8[]", "Box<ptr mut u8>", "ptr /*gap*/ mut pkg::T"} {
		input := "module main;fn " + typ + " f(" + typ + " value){}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.AST == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("pointer oracle %s: %v", typ, err)
		}
		f := oracle.AST.Functions[0]
		fmt.Fprintf(&helpers, "fn i32 pointer_%d(syntax::Parsed parsed){assert(syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed));", i)
		count := 0
		result := emitCandidateType(t, &helpers, f.ResultType, &count)
		parameter := emitCandidateType(t, &helpers, f.Parameters[0].Type, &count)
		fmt.Fprintf(&helpers, "assert(syntax::declaration_at(ref parsed,1).result_type==%d && syntax::entry_at(ref parsed,0).type_root==%d && syntax::type_count(ref parsed)==%d);return 0;}", result, parameter, count)
		fmt.Fprintf(&main, "assert(switch(make(b%s)){case .ok(parsed):{pointer_%d(parsed)}case .err(_):{1}}==0);", strconv.Quote(input), i)
	}
	for _, typ := range []string{"ptr", "ptr mut", "ptr mut mut T", "(ptr u8", "ptr u8["} {
		input := "module main;fn " + typ + " bad(){}fn void kept(){}"
		fmt.Fprintf(&main, `assert(switch(make(b%s)){case .ok(parsed):{syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::type_count(ref parsed)==1 && integrity::valid(ref parsed)}case .err(_):{false}});`, strconv.Quote(input))
	}
	for _, n := range []int{31, 32} {
		input := "module main;fn " + strings.Repeat("ptr ", n) + "u8 f(){}"
		fmt.Fprintf(&main, `assert(switch(make(b%s)){case .ok(parsed):{%d==31 && syntax::problem_count(ref parsed)==0 && integrity::valid(ref parsed)}case .err(status):{%d==32 && status==4}});`, strconv.Quote(input), n, n)
	}
	main.WriteString("return 0;}")
	runParserProbe(t, helpers.String()+main.String())
}
