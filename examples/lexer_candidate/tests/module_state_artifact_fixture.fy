module tests::module_state_artifact_fixture;
import module_state_layout as state;
import module_signatures as signatures;
import module_catalog as catalog;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
pub fn state::Plan sample(){var v::Vec<state::Layout> layouts=v::empty<state::Layout>();var v::Vec<state::Field> fields=v::empty<state::Field>();
assert(v::push<state::Layout>(ref mut layouts,state::Layout{.owner=7,.module_index=2,.declaration=3,.start=10,.end=150,.uid_start=0,.uid_end=13,.field_start=0,.field_count=5,.hash=[2228573728,2676884447,2887346710,2989097650,72141878,1087492126,1089733372,2723509058]}).is_ok);
assert(v::push<state::Field>(ref mut fields,state::Field{.kind=1,.key=0,.value=3,.name_start=13,.name_end=25,.start=20,.end=30,.prefix=[506277646,883593563,1609707817,2481775483,2757222962,1456554660,1461222578,2550245238]}).is_ok);
assert(v::push<state::Field>(ref mut fields,state::Field{.kind=2,.key=4,.value=3,.name_start=25,.name_end=33,.start=40,.end=50,.prefix=[3089451091,3058147140,1445608855,3058200239,3040188848,1974353460,1828231896,1061597317]}).is_ok);
assert(v::push<state::Field>(ref mut fields,state::Field{.kind=1,.key=0,.value=1,.name_start=33,.name_end=39,.start=60,.end=70,.prefix=[107627531,1408091346,2307935832,1734039910,2051719330,2702277835,854158243,1565747435]}).is_ok);
assert(v::push<state::Field>(ref mut fields,state::Field{.kind=1,.key=0,.value=2,.name_start=39,.name_end=44,.start=80,.end=90,.prefix=[1789612539,1451449907,3781494778,2756169587,648033904,217120459,1798569327,1796992695]}).is_ok);
assert(v::push<state::Field>(ref mut fields,state::Field{.kind=1,.key=0,.value=5,.name_start=44,.name_end=48,.start=100,.end=110,.prefix=[2975261854,3744990163,2065963572,3361067548,1339804074,2306037198,1681351006,3529104001]}).is_ok);
let names=switch(bytes::copy(b"example.tokentotal_supplybalancespausedcountroot")){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};return state::assemble(names,layouts,fields);}
pub fn ir::Program program(){var v::Vec<syntax::Parsed> input=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut input,parsed(b"module main;@contract_uid(\"example.\\u{74}oken\")state{cell u64 total_supply;map bytes20 u64 balances;cell bool paused;cell u32 count;cell bytes32 root;}fn i32 main(){return 42;}")).is_ok);
let metadata=switch(signatures::build(ref input)){case .ok(p):{p}case .err(_):{assert(false);return program();}};
let unit=switch(compile::range(ref input,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return program();}};var v::Vec<ir::Program> units=v::empty<ir::Program>();assert(v::push<ir::Program>(ref mut units,unit).is_ok);return switch(link::build(ref metadata,ref units)){case .ok(p):{p}case .err(_):{assert(false);return program();}};
}
pub fn state::Plan scoped(){var v::Vec<syntax::Parsed> input=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut input,parsed(b"module one;@contract_uid(\"x\")state{cell u64 supply;}fn i32 f(){return 1;}")).is_ok && v::push<syntax::Parsed>(ref mut input,parsed(b"module two;@contract_uid(\"y\")state{cell bool flag;}fn i32 f(){return 2;}")).is_ok);
var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok && v::push<usize>(ref mut owners,1).is_ok);let bindings=v::empty<catalog::Binding>();let c=switch(catalog::build_scoped(ref input,ref owners,ref bindings)){case .ok(p):{p}case .err(_):{assert(false);return scoped();}};let metadata=switch(signatures::build_catalog(ref input,c)){case .ok(p):{p}case .err(_):{assert(false);return scoped();}};var usize work=0;return switch(state::copy(signatures::state_layouts(ref metadata),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return scoped();}};
}
