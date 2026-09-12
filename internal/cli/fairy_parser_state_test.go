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

func TestWindowsFairyParserState(t *testing.T) {
	var p strings.Builder
	p.WriteString("fn i32 main(){")
	for _, body := range []string{"state {}", "state {cell u64 total;map bytes20 u64 balances;}", "@contract_uid(\"token\") pub state {map ref u8[] Box<i32> items;cell i32[4] four;}", "state {cell u8 a;}state {map u8 u32 b;}"} {
		input := "module main;" + body
		oracle, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
		if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
			t.Fatalf("state oracle %s: %v", body, err)
		}
		root, _ := oracle.Tree.Node(oracle.Tree.Root)
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==%d && integrity::valid(ref parsed));`, strconv.Quote(input), len(root.Children))
		fieldIndex := 0
		for index, id := range root.Children {
			d, _ := oracle.Tree.Node(id)
			if d.Kind != syntax.KindState {
				continue
			}
			fmt.Fprintf(&p, `{let syntax::Declaration d=syntax::declaration_at(ref parsed,%d);assert(d.kind==10 && d.start==%d && d.end==%d && d.name_start==0 && d.name_end==0 && d.body==0 && d.result_type==0);}`, index, d.Span.StartByte, d.Span.EndByte)
			for _, child := range d.Children {
				field, _ := oracle.Tree.Node(child)
				if field.Kind != syntax.KindStateCell && field.Kind != syntax.KindStateMap {
					continue
				}
				kind := 1
				keyStart, keyEnd := 0, 0
				value, _ := oracle.Tree.Node(field.Children[0])
				if field.Kind == syntax.KindStateMap {
					kind = 2
					keyStart, keyEnd = value.Span.StartByte, value.Span.EndByte
					value, _ = oracle.Tree.Node(field.Children[1])
				}
				fmt.Fprintf(&p, `assert(syntax::field_at(ref parsed,%d).kind==%d && syntax::field_at(ref parsed,%d).start==%d && syntax::field_at(ref parsed,%d).end==%d && syntax::field_at(ref parsed,%d).name_start==%d && syntax::field_at(ref parsed,%d).name_end==%d);
    {let usize value=syntax::field_at(ref parsed,%d).type_root;assert(syntax::type_at(ref parsed,value).start==%d && syntax::type_at(ref parsed,value).end==%d);}`, fieldIndex, kind, fieldIndex, field.Span.StartByte, fieldIndex, field.Span.EndByte, fieldIndex, field.NameSpan.StartByte, fieldIndex, field.NameSpan.EndByte, fieldIndex, value.Span.StartByte, value.Span.EndByte)
				if kind == 2 {
					fmt.Fprintf(&p, `{let usize key=syntax::field_at(ref parsed,%d).key_type;assert(syntax::type_at(ref parsed,key).start==%d && syntax::type_at(ref parsed,key).end==%d);}`, fieldIndex, keyStart, keyEnd)
				}
				fieldIndex++
			}
		}
		fmt.Fprintf(&p, `assert(syntax::field_count(ref parsed)==%d);0}case .err(_):{1}};assert(r==0);}`, fieldIndex)
	}
	for _, body := range []string{"state {cell u8 good;map u8 missing;}", "state {cell u8 a}", "state {u8 a;}", "state {cell pub u8 a;}", "state name {}"} {
		input := "module main;" + body + "fn void kept(){}"
		fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)>0 && syntax::declaration_count(ref parsed)==2 && syntax::field_count(ref parsed)==0 && syntax::type_count(ref parsed)==1 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input))
	}
	for _, count := range []int{1024, 1025} {
		input := "module main;state {" + strings.Repeat("cell u8 a;", count) + "}"
		if count == 1024 {
			fmt.Fprintf(&p, `{let i32 r=switch(make(b%s)){case .ok(parsed):{assert(syntax::problem_count(ref parsed)==0 && syntax::field_count(ref parsed)==1024 && integrity::valid(ref parsed));0}case .err(_):{1}};assert(r==0);}`, strconv.Quote(input))
		} else {
			fmt.Fprintf(&p, `assert(switch(make(b%s)){case .ok(_):{false}case .err(status):{status==4}});`, strconv.Quote(input))
		}
	}
	p.WriteString("return 0;}")
	runParserProbe(t, p.String())
}
