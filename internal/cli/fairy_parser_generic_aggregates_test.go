package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserGenericAggregates(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){")
	for _, c := range []struct {
		decl string
		kind int
	}{
		{"struct Empty<T>{}", 4}, {"struct Pair<T,U>{T first;U second;}", 4},
		{"@repr(C) pub struct Box<T,>{pub ptr mut T value;}", 4},
		{"resource Owner<T>{T value;cleanup {}}", 6},
		{"pub resource Vec<T>{ptr mut T data;usize size;cleanup {let x=1;}}", 6},
		{"struct Duplicate<T,T>{T value;}", 4},
	} {
		p.WriteString(genericDeclarationProbe(t, "module main;"+c.decl, c.kind))
	}
	for _, decl := range []string{"struct Bad<>{}", "struct Bad<T,,U>{T value;}", "resource Bad<T::U>{}", "resource Bad<T>{T value}", "struct Bad<T<U>>{}", "state<T>{cell u8 value;}"} {
		input := "module main;" + decl + "fn void kept(){}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) == 0 {
			t.Fatalf("invalid aggregate oracle: %s", decl)
		}
		fmt.Fprintf(&p, `assert(switch(make(b%s)){case .ok(parsed):{syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::type_count(ref parsed)==1 && syntax::field_count(ref parsed)==0 && syntax::generic_parameter_count(ref parsed,1)==0 && integrity::valid(ref parsed)}case .err(_):{false}});`, strconv.Quote(input))
	}
	for _, n := range []int{2048, 2049} {
		input := "module main;struct Wide<" + strings.Repeat("T,", n-1) + "T>{}"
		fmt.Fprintf(&p, `assert(switch(make(b%s)){case .ok(parsed):{%d==2048 && syntax::problem_count(ref parsed)==0 && syntax::type_count(ref parsed)==4096 && syntax::generic_parameter_count(ref parsed,1)==2048 && integrity::valid(ref parsed)}case .err(status):{%d==2049 && status==4}});`, strconv.Quote(input), n, n)
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
