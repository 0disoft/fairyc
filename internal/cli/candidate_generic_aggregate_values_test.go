package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateGenericAggregateValues(t *testing.T) {
	const factory = `pub fn Holder<T> make<T>(T value){return Holder<T>{.payload=Cell<T>{.value=value}};}`
	const direct = `pub fn Cell<i32> direct(i32 value){return Cell<i32>{.value=value};}`
	wire := candidateAggregateLayoutOracle(t,
		candidateAggregateLayoutFixture{name: "direct construction", maker: aggregateLayoutMaker + direct},
		candidateAggregateLayoutFixture{name: "generic factory", maker: aggregateLayoutMaker + factory, root: `module root;import maker as m;fn i32 main(){let value=m::make<i32>(42);discard value;return 0;}`},
		candidateAggregateLayoutFixture{name: "imported struct constructor", root: `module root;import maker as m;fn i32 main(){let value=m::Cell<i32>{.value=42};let copied=value;return copied.value;}`},
		candidateAggregateLayoutFixture{name: "reordered fields", maker: aggregateLayoutMaker + `pub fn Pair<i32,u8> create(){return Pair<i32,u8>{.right=7,.left=42};}`},
		candidateAggregateLayoutFixture{name: "extended factory argument", maker: aggregateLayoutMaker + factory + `pub fn Holder<f64> extra(f64 value){return make<f64>(value);}`},
		candidateAggregateLayoutFixture{name: "constructor in cleanup", maker: strings.Replace(aggregateLayoutMaker+factory, "cleanup{discard h::finish<T>(this.payload.value);}", "cleanup{let copied=Cell<T>{.value=this.payload.value};discard h::finish<T>(copied.value);}", 1)},
		candidateAggregateLayoutFixture{name: "integer endpoints", maker: aggregateLayoutMaker + `pub fn Cell<i32> lower(){return Cell<i32>{.value=-2147483648};}pub fn Cell<i32> upper(){return Cell<i32>{.value=2147483647};}`},
		candidateAggregateLayoutFixture{name: "record array construction", maker: strings.Replace(aggregateLayoutMaker, "T left;U right;", "T left;U right;T[2] values;Cell<T>[2] records;", 1) + `pub fn Pair<i32,u8> create(){return Pair<i32,u8>{.right=7,.left=42,.values=[1,2],.records=[Cell<i32>{.value=3},Cell<i32>{.value=4}]};}`},
		candidateAggregateLayoutFixture{name: "resource shared borrow", maker: aggregateLayoutMaker + `pub fn i32 read(ref Holder<i32> value){return value.payload.value;}pub fn i32 use(){let value=Holder<i32>{.payload=Cell<i32>{.value=42}};return read(ref value);}`},
		candidateAggregateLayoutFixture{name: "imported mutable struct", maker: aggregateLayoutMaker + `pub fn void overwrite(ref mut Cell<i32> value){value.value=42;}`, root: `module root;import maker as m;fn i32 main(){var value=m::Cell<i32>{.value=1};m::overwrite(ref mut value);return value.value;}`},
		candidateAggregateLayoutFixture{name: "explicit resource burn", maker: aggregateLayoutMaker + `@abilities(store)pub resource Seal<T>{T n;cleanup{}}pub fn void consume(Seal<i32> value){burn value;}pub fn void use(){let value=Seal<i32>{.n=42};burn value;}`},
		candidateAggregateLayoutFixture{name: "missing field", maker: aggregateLayoutMaker + `pub fn Cell<i32> bad(){return Cell<i32>{};}`, reject: 2},
		candidateAggregateLayoutFixture{name: "wrong field type", maker: aggregateLayoutMaker + `pub fn Cell<i32> bad(){return Cell<i32>{.value=true};}`, reject: 1006},
		candidateAggregateLayoutFixture{name: "duplicate initializer", maker: aggregateLayoutMaker + `pub fn Cell<i32> bad(){return Cell<i32>{.value=1,.value=2};}`, reject: 2},
		candidateAggregateLayoutFixture{name: "unknown field", maker: aggregateLayoutMaker + `pub fn Cell<i32> bad(){return Cell<i32>{.unknown=1};}`, reject: 2},
		candidateAggregateLayoutFixture{name: "nested wrong field type", maker: aggregateLayoutMaker + `pub fn Holder<i32> bad(){return Holder<i32>{.payload=Cell<bool>{.value=true}};}`, reject: 1006},
		candidateAggregateLayoutFixture{name: "private resource constructor", root: `module root;import maker as m;fn i32 main(){let value=m::Holder<i32>{.payload=m::Cell<i32>{.value=42}};discard value;return 0;}`, reject: 103},
		candidateAggregateLayoutFixture{name: "integer above maximum", maker: aggregateLayoutMaker + `pub fn Cell<i32> bad(){return Cell<i32>{.value=2147483648};}`, reject: 1013},
		candidateAggregateLayoutFixture{name: "integer below minimum", maker: aggregateLayoutMaker + `pub fn Cell<i32> bad(){return Cell<i32>{.value=-2147483649};}`, reject: 1013},
		candidateAggregateLayoutFixture{name: "exclusive resource field borrow", maker: aggregateLayoutMaker + `pub fn void bad(){var value=Holder<i32>{.payload=Cell<i32>{.value=42}};let alias=ref mut value.payload;discard alias;}`, reject: 1008},
		candidateAggregateLayoutFixture{name: "drop resource cannot burn", maker: aggregateLayoutMaker + `pub fn void bad(){let value=Holder<i32>{.payload=Cell<i32>{.value=42}};burn value;}`, reject: 1109},
	)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_layouts.fy"))
	if err != nil {
		t.Fatal(err)
	}
	checks, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_values.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_layouts;", "module main;", 1)
	main = strings.Replace(main, "import module_unit_interfaces as interfaces;", "import module_unit_interfaces as interfaces;import module_unit_bind as binding;import module_unit_call_closure as closure;import module_unit_values as values;import module_unit_value_model as model;import module_unit_drops as drops;", 1)
	start := strings.Index(main, "fn u32 build_fields(")
	stop := strings.Index(main, "@effects(console) fn void verify_fields(")
	if start < 0 || stop < start {
		t.Fatal("missing layout probe field builder")
	}
	signature := main[start : start+strings.Index(main[start:], "{")+1]
	checkBody := string(checks[strings.Index(string(checks), "pub fn void verify("):])
	checkBody = strings.Replace(checkBody, "pub fn void verify(", "fn void verify_values(", 1)
	main = main[:start] + signature + aggregateValuesFieldBuilder + main[stop:] + checkBody + "\n@effects(console) fn i32 main(){return run();}\n"
	candidateAtlasProbeSize(t, "module_unit_values", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, wire); code != 0 || out != "" {
			t.Fatalf("aggregate construction values: %d %q", code, out)
		}
	}, "project_source_plan")
}

const aggregateValuesFieldBuilder = `
 let bound=switch(binding::bind(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(p):{return p.code;}};
 let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{return p.code;}};
 let checked=switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(p):{return p.code;}};verify_values(ref ready,ref checked,sources);
 let p=closure::imported(ref ready);var usize i=types::count(records)+1;while(i<=values::type_count(ref checked)){assert(v::push<types::Type>(records,values::type_at(ref checked,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::shape_count(p)){assert(v::push<interfaces::Shape>(shapes,interfaces::shape_at(p,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::field_count(p)){assert(v::push<interfaces::Field>(fields,interfaces::field_at(p,i)).is_ok);i+=1;}
 assert(bytes::append(names,bytes::view(interfaces::name_records(p))).is_ok);return 0;
}
`
