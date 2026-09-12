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

func genericDeclarationProbe(t *testing.T, input string, kind int) string {
	t.Helper()
	oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
		t.Fatalf("generic oracle %s: %v", input, err)
	}
	root, _ := oracle.Tree.Node(oracle.Tree.Root)
	d, _ := oracle.Tree.Node(root.Children[1])
	var parameters syntax.Node
	for _, id := range d.Children {
		n, _ := oracle.Tree.Node(id)
		if n.Kind == syntax.KindTypeParameters {
			parameters = n
		}
	}
	if parameters.Kind != syntax.KindTypeParameters {
		t.Fatal("missing generic oracle")
	}
	var p strings.Builder
	fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{
  assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==2 && integrity::valid(ref parsed));
  let syntax::Declaration d=syntax::declaration_at(ref parsed,1);
  assert(d.kind==%d && d.start==%d && d.end==%d && d.name_start==%d && d.name_end==%d && d.generic_start==%d && d.generic_end==%d && d.generic_parameters>0 && syntax::generic_parameter_count(ref parsed,1)==%d);
 `, strconv.Quote(input), kind, d.Span.StartByte, d.Span.EndByte, d.NameSpan.StartByte, d.NameSpan.EndByte, parameters.Span.StartByte, parameters.Span.EndByte, len(parameters.Children))
	for index, id := range parameters.Children {
		n, _ := oracle.Tree.Node(id)
		fmt.Fprintf(&p, `{let name=syntax::generic_parameter_at(ref parsed,1,%d);assert(name.kind==13 && name.start==%d && name.end==%d && name.left==0 && name.right==0);}`, index, n.Span.StartByte, n.Span.EndByte)
	}
	p.WriteString("{var link=syntax::type_at(ref parsed,d.generic_parameters);link.left=0;assert(!integrity::valid_type(ref parsed,link,d.generic_parameters));}0}case .err(_):{1}};assert(r==0);}")
	return p.String()
}

func TestWindowsFairyParserGenericFunctions(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){")
	for _, decl := range []string{"fn T identity<T>(T value){return value;}", "fn Vec<T> empty<T>(){}", "@tag pub fn Result<T,U> f< T /*gap*/, U, >(T a,U b){}", "fn void duplicate<T,T>(){}"} {
		p.WriteString(genericDeclarationProbe(t, "module main;"+decl, 3))
	}
	for _, decl := range []string{"fn void bad<>(){}", "fn void bad<T,,U>(){}", "fn void bad<T::U>(){}", "fn void bad<T=U>(){}", "fn void bad<T<U>>(){}", "fn void bad<T(i32 x){}", "unsafe extern C fn void bad<T>();"} {
		input := "module main;" + decl + "fn void kept(){}"
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) == 0 {
			t.Fatalf("invalid generic oracle: %s", decl)
		}
		fmt.Fprintf(&p, `assert(switch(make(b%s)){case .ok(parsed):{syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::type_count(ref parsed)==1 && syntax::entry_count(ref parsed)==0 && syntax::generic_parameter_count(ref parsed,1)==0 && integrity::valid(ref parsed)}case .err(_):{false}});`, strconv.Quote(input))
	}
	for _, n := range []int{2047, 2048} {
		input := "module main;fn void f<" + strings.Repeat("T,", n-1) + "T>(){}"
		fmt.Fprintf(&p, `assert(switch(make(b%s)){case .ok(parsed):{%d==2047 && syntax::problem_count(ref parsed)==0 && syntax::type_count(ref parsed)==4095 && syntax::generic_parameter_count(ref parsed,1)==2047 && integrity::valid(ref parsed)}case .err(status):{%d==2048 && status==4}});`, strconv.Quote(input), n, n)
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
