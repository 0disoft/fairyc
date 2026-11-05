package cli

import (
	"strings"
	"testing"
)

func TestCandidateGenericAggregateFlow(t *testing.T) {
	cases := append(candidateAggregateValueFixtures(),
		candidateAggregateLayoutFixture{name: "two cleanup obligations", maker: aggregateLayoutMaker + `pub fn void cleanup_pair(){let first=Holder<i32>{.payload=Cell<i32>{.value=1}};let second=Holder<i32>{.payload=Cell<i32>{.value=2}};}`},
		candidateAggregateLayoutFixture{name: "consuming call", maker: aggregateLayoutMaker + `pub fn void release_holder(Holder<i32> value){}pub fn void transfer(){let value=Holder<i32>{.payload=Cell<i32>{.value=42}};release_holder(value);}`},
		candidateAggregateLayoutFixture{name: "cleanup effects permitted", maker: aggregateLayoutMaker + `@effects(console)pub resource Echo<T>{T value;cleanup{discard this.value;}}@effects(console)pub fn void consume_echo(Echo<i32> value){}`},
		candidateAggregateLayoutFixture{name: "wrapped resource cleanup", maker: aggregateLayoutMaker + `pub fn void optional(Option<Holder<i32>> value){discard value;}`},
		candidateAggregateLayoutFixture{name: "nested resource cleanup", maker: aggregateLayoutMaker + `pub resource Envelope<T>{Holder<T> child;cleanup{}}pub fn void drop_envelope(Envelope<i32> value){}`},
		candidateAggregateLayoutFixture{name: "use after transfer", maker: aggregateLayoutMaker + `pub fn void bad(Holder<i32> value){let moved=value;discard value;discard moved;}`, reject: 1101},
		candidateAggregateLayoutFixture{name: "double consumption", maker: aggregateLayoutMaker + `pub fn void take(Holder<i32> value){}pub fn void bad(Holder<i32> value){take(value);take(value);}`, reject: 1101},
		candidateAggregateLayoutFixture{name: "moved owner field read", maker: aggregateLayoutMaker + `pub fn i32 bad(Holder<i32> value){let moved=value;return value.payload.value;}`, reject: 1101},
		candidateAggregateLayoutFixture{name: "duplicated resource initializer", maker: aggregateLayoutMaker + `pub resource Duo<T>{Holder<T> first;Holder<T> second;cleanup{}}pub fn void bad(Holder<i32> value){let pair=Duo<i32>{.first=value,.second=value};discard pair;}`, reject: 1101},
		candidateAggregateLayoutFixture{name: "partial resource transfer", maker: aggregateLayoutMaker + `pub resource Envelope<T>{Holder<T> child;cleanup{}}pub fn Holder<i32> bad(Envelope<i32> value){return value.child;}`, reject: 1103},
		candidateAggregateLayoutFixture{name: "implicit no-drop cleanup", maker: aggregateLayoutMaker + `@abilities(store)pub resource Seal<T>{T n;cleanup{}}pub fn void bad(Seal<i32> value){}`, reject: 1108},
		candidateAggregateLayoutFixture{name: "nested no-drop cleanup", maker: aggregateLayoutMaker + `@abilities(store)pub resource Seal<T>{T n;cleanup{}}pub resource Envelope<T>{Seal<T> child;cleanup{}}pub fn void bad(Envelope<i32> value){}`, reject: 1108},
		candidateAggregateLayoutFixture{name: "missing cleanup effects", maker: aggregateLayoutMaker + `@effects(console)pub resource Echo<T>{T value;cleanup{discard this.value;}}pub fn void bad(Echo<i32> value){}`, reject: 7102},
		candidateAggregateLayoutFixture{name: "escaping local borrow", maker: aggregateLayoutMaker + `pub fn ref Cell<i32> bad(){let value=Cell<i32>{.value=42};return ref value;}`, reject: 2102},
		candidateAggregateLayoutFixture{name: "inconsistent branch move", maker: aggregateLayoutMaker + `pub fn void take(Holder<i32> value){}pub fn void bad(Holder<i32> value,bool flag){if(flag){take(value);}discard value;}`, reject: 1104},
		candidateAggregateLayoutFixture{name: "wrapped use after move", maker: aggregateLayoutMaker + `pub fn void bad(Option<Holder<i32>> value){let moved=value;discard value;discard moved;}`, reject: 1101},
		candidateAggregateLayoutFixture{name: "consumed cleanup receiver", maker: strings.Replace(aggregateLayoutMaker, "cleanup{discard h::finish<T>(this.payload.value);}", "cleanup{let moved=this;discard moved;}", 1), reject: 1102},
	)
	wire := candidateAggregateLayoutOracle(t, cases...)
	candidateAggregateBodyProbe(t, "module_unit_checked_bodies", wire, aggregateFlowFieldBuilder, "generic_aggregate_flow.fy", "import module_unit_checked_bodies as checking;import module_unit_moves as moves;import module_unit_cleanup_policy as policy;")
}

const aggregateFlowFieldBuilder = `
 let owner=switch(checking::build(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(p):{return p.code;}};verify_values(ref owner,sources);
 let checked=checking::checked_values(ref owner);let p=checking::checked_interfaces(ref owner);var usize i=types::count(records)+1;while(i<=values::type_count(checked)){assert(v::push<types::Type>(records,values::type_at(checked,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::shape_count(p)){assert(v::push<interfaces::Shape>(shapes,interfaces::shape_at(p,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::field_count(p)){assert(v::push<interfaces::Field>(fields,interfaces::field_at(p,i)).is_ok);i+=1;}
 assert(bytes::append(names,bytes::view(interfaces::name_records(p))).is_ok);return 0;
}
`
