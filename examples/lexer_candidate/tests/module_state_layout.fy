module tests::module_state_layout;
import module_state_layout as state;
import module_signatures as signatures;
import module_catalog as catalog;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import project_diagnostic_codes as diagnostics;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn v::Vec<syntax::Parsed> sources(ref u8[] source){var v::Vec<syntax::Parsed> result=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut result,parsed(source)).is_ok);return result;}
fn signatures::Program metadata(ref u8[] source){let input=sources(source);return switch(signatures::build(ref input)){case .ok(p):{p}case .err(_):{assert(false);return metadata(source);}};}
fn ref u8[] canonical_source(){return b"module main;@contract_uid(\"example.\\u{74}oken\")state{cell u64 total_supply;map bytes20 u64 balances;cell bool paused;cell u32 count;cell bytes32 root;}fn i32 main(){return 42;}";}
fn void failure(ref u8[] source,u32 code,ref u8[] span){let input=sources(source);let error=switch(signatures::build(ref input)){case .ok(_):{assert(false);return;}case .err(p):{p}};assert(error.code==code && error.module_index==0);assert(diagnostics::code(3,2,error.code)==code);if(span.len!=0){assert(bytes::equal(source[error.start..error.end],span));}}
@test
fn void state_layout_owns_order_codecs_and_reference_canonical_hashes(){let metadata=metadata(canonical_source());let p=signatures::state_layouts(ref metadata);assert(state::count(p)==1 && state::field_count(p)==5 && bytes::equal(state::uid(p,0),b"example.token"));assert(signatures::declaration_lookup(ref metadata,0,b"balances").code==5);assert(signatures::type_count(ref metadata)==1);var usize work=0;assert(state::validate(p,ref mut work)==0);assert(state::equal(state::at(p,0).hash,[2228573728,2676884447,2887346710,2989097650,72141878,1087492126,1089733372,2723509058]));assert(bytes::equal(state::name(p,0),b"total_supply") && state::equal(state::field_at(p,0).prefix,[506277646,883593563,1609707817,2481775483,2757222962,1456554660,1461222578,2550245238]));assert(bytes::equal(state::name(p,1),b"balances") && state::equal(state::field_at(p,1).prefix,[3089451091,3058147140,1445608855,3058200239,3040188848,1974353460,1828231896,1061597317]));assert(bytes::equal(state::name(p,2),b"paused") && state::equal(state::field_at(p,2).prefix,[107627531,1408091346,2307935832,1734039910,2051719330,2702277835,854158243,1565747435]));assert(bytes::equal(state::name(p,3),b"count") && state::equal(state::field_at(p,3).prefix,[1789612539,1451449907,3781494778,2756169587,648033904,217120459,1798569327,1796992695]));assert(bytes::equal(state::name(p,4),b"root") && state::equal(state::field_at(p,4).prefix,[2975261854,3744990163,2065963572,3361067548,1339804074,2306037198,1681351006,3529104001]));let copied=switch(state::copy(p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(state::equal(state::at(ref copied,0).hash,state::at(p,0).hash));assert(state::field_at(ref copied,1).kind==2 && state::field_at(ref copied,1).key==4 && state::field_at(ref copied,1).value==3);}
@test
fn void state_declarations_reject_bad_authority_duplicates_and_nonstorage_codecs(){
    failure(b"module main;state{cell u64 supply;}",7201,b"");
    failure(b"module main;@contract_uid(\"x\")pub state{cell bool flag;}",7201,b"");
    failure(b"module main;@contract_uid(\"\")state{}",7201,b"@contract_uid(\"\")");
    failure(b"module main;@contract_uid(\"x/y\")state{}",7201,b"");
    failure(b"module main;@contract_uid(1)state{}",7201,b"@contract_uid(1)");
    failure(b"module main;@contract_uid(\"x\")@contract_uid(\"y\")state{}",7201,b"@contract_uid(\"y\")");
    failure(b"module main;@other(\"x\")state{}",7201,b"@other(\"x\")");
    failure(b"module main;@contract_uid(\"x\")state{cell u32 supply;cell u64 supply;}",7202,b"supply");
    failure(b"module main;@contract_uid(\"x\")state{cell String value;}",7203,b"String");
    failure(b"module main;@contract_uid(\"x\")state{map i32 u64 value;}",7203,b"i32");
    failure(b"module main;@contract_uid(\"x\")state{cell Option<u32> value;}",7203,b"Option<u32>");
    failure(b"module main;@contract_uid(\"x\")state{cell codec::u64 value;}",7203,b"codec::u64");
}
@test
fn void state_uniqueness_uses_package_owners_and_precedes_codec_validation(){
    var v::Vec<syntax::Parsed> input=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut input,parsed(b"module one;@contract_uid(\"x\")state{cell u64 supply;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut input,parsed(b"module two;@contract_uid(\"y\")state{cell bool flag;}")).is_ok);
    assert(switch(signatures::build(ref input)){case .ok(_):{false}case .err(e):{e.code==7201 && e.module_index==1}});
    var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok && v::push<usize>(ref mut owners,1).is_ok);let bindings=v::empty<catalog::Binding>();let c=switch(catalog::build_scoped(ref input,ref owners,ref bindings)){case .ok(c):{c}case .err(_):{assert(false);return;}};let p=switch(signatures::build_catalog(ref input,c)){case .ok(p):{p}case .err(_):{assert(false);return;}};let states=signatures::state_layouts(ref p);assert(state::count(states)==2 && state::at(states,0).owner==0 && state::at(states,1).owner==1);var usize work=0;assert(state::validate(states,ref mut work)==0);
    failure(b"module main;@contract_uid(\"x\")state{cell String value;}@contract_uid(\"y\")state{}",7201,b"@contract_uid(\"y\")state{}");
}
@test
fn void linked_ir_retains_state_and_rejects_changed_layout_identity(){
    let input=sources(canonical_source());let metadata=switch(signatures::build(ref input)){case .ok(p):{p}case .err(_):{assert(false);return;}};let unit=switch(compile::range(ref input,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};var v::Vec<ir::Program> units=v::empty<ir::Program>();assert(v::push<ir::Program>(ref mut units,unit).is_ok);let program=switch(link::build(ref metadata,ref units)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(proof::check(ref program).code==0);let p=ir::state_layouts(ref program);assert(state::count(p)==1 && state::equal(state::at(p,0).hash,state::at(signatures::state_layouts(ref metadata),0).hash));
    var usize mode=0;while(mode<3){var bytes::Buffer names=bytes::empty();assert(bytes::append(ref mut names,state::uid(p,0)).is_ok);var v::Vec<state::Layout> layouts=v::empty<state::Layout>();var row=state::at(p,0);if(mode==0){row.hash[0]=0;}if(mode==2){row.field_count=4;}assert(v::push<state::Layout>(ref mut layouts,row).is_ok);var v::Vec<state::Field> fields=v::empty<state::Field>();var usize i=0;while(i<5){assert(bytes::append(ref mut names,state::name(p,i)).is_ok);var field=state::field_at(p,i);if(mode==1 && i==0){field.prefix[0]=0;}assert(v::push<state::Field>(ref mut fields,field).is_ok);i+=1;}let changed=state::assemble(names,layouts,fields);var usize work=0;assert(state::validate(ref changed,ref mut work)==9);mode+=1;}
}
