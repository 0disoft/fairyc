module tests::closure_source_recipes;
import module_project_artifact_calls as calls;
import module_project_artifact_recipes as recipes;
import module_project_artifact_bodies as bodies;
import module_project_artifact_frames as frames;
import module_project_artifact_coverage as coverage;
import module_unit_artifact_identities as identities;
import module_identity_artifact_model as ids;
import module_project_compile as compile;
import module_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_program_artifact_encode as encode;
import module_program_artifact_decode as decode;
import module_program_artifact_wire as budget;
import candidate_artifact_wire as wire;
import module_signatures as signatures;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import project_source_plan as planning;
import generic_source_plan as plan;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn void require(bool ok,ref u8[] phase,u32 point){if(!ok){discard io::write_stderr("closure-source-recipes phase=");discard io::write_stderr_bytes(phase);discard io::write_stderr(" checkpoint=");discard io::write_stderr_u32(point);discard io::write_stderr("\n");assert(false);}}
@effects(console) fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{require(false,b"copied",1);copied(raw)}};}
@effects(console) fn syntax::Parsed parsed(ref u8[] raw){let scanned=switch(scanner::collect(copied(raw),false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{require(false,b"parsed",2);return parsed(b"module empty;");}};return switch(syntax::parse(scanned,512)){case .ok(p):{require(syntax::problem_count(ref p)==0,b"parsed",3);p}case .err(_):{require(false,b"parsed",4);parsed(b"module empty;")}};}
pub resource Fixture{v::Vec<syntax::Parsed> sources;catalog::Catalog scope;signatures::Program metadata;plan::Plan plan;cleanup{}}
@effects(console) fn Fixture fixture(ref u8[] text){
    var sources=v::empty<syntax::Parsed>();require(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok,b"fixture",5);var owners=v::empty<usize>();require(v::push<usize>(ref mut owners,0).is_ok,b"fixture",6);let imports=v::empty<catalog::Binding>();
    var rows=v::empty<coordinates::Coordinate>();require(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=77}).is_ok,b"fixture",7);let table=coordinates::assemble(copied(b"fypc1-sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),rows);
    let scope=switch(catalog::build_scoped(ref sources,ref owners,ref imports)){case .ok(p):{p}case .err(_):{require(false,b"fixture",8);return fixture(text);}};let signature_scope=switch(catalog::build_scoped(ref sources,ref owners,ref imports)){case .ok(p):{p}case .err(_):{require(false,b"fixture",9);return fixture(text);}};
    let metadata=switch(signatures::build_catalog(ref sources,signature_scope)){case .ok(p):{p}case .err(_):{require(false,b"fixture",10);return fixture(text);}};var usize work=0;let original=switch(planning::build_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(p):{p}case .err(_):{require(false,b"fixture",11);return fixture(text);}};return Fixture{.sources,.scope,.metadata,.plan=original};
}
@effects(console) fn identified::Program unit(ref Fixture p){var usize work=0;return switch(compile::range_identified(ref p.sources,ref p.metadata,ref p.plan,0,1,ref mut work)){case .ok(p):{p}case .err(_):{require(false,b"unit",12);unit(p)}};}
@effects(console) fn wire::Payload encoded(ref identified::Program p){var usize work=0;return switch(encode::encode_closures(identified::ir_program(p),ref mut work)){case .ok(p):{p}case .err(_):{require(false,b"encoded",13);encoded(p)}};}
@effects(console) fn ir::Program restored(ref u8[] raw){var usize work=0;return switch(decode::decode_closures(raw,ref mut work)){case .ok(p):{require(ir::work(ref p)==0 && proof::check(ref p).code==0,b"restored",14);p}case .err(_):{require(false,b"restored",15);restored(raw)}};}
@effects(console) fn ids::Snapshot snapshot(ref identified::Program p){var usize work=0;return switch(identities::extract(p,ref mut work)){case .ok(p):{p}case .err(_):{require(false,b"snapshot",16);snapshot(p)}};}
@effects(console) fn u32 match(ref ir::Program p,ref ids::Snapshot snapshot,ref Fixture f,ref compile::CheckedBodies checked,u32 mode,ref mut usize work){
    if(mode==3){return switch(coverage::checked_range(p,snapshot,ref f.plan,checked,ref f.sources,ref f.metadata,work)){case .ok(c):{require(coverage::stats(ref c).frames==ir::count(p),b"match",17);0}case .err(e):{e.code}};}
    if(mode==4){return switch(coverage::build(p,snapshot,ref f.plan,ref f.sources,ref f.metadata,work)){case .ok(c):{require(coverage::stats(ref c).files==1 && coverage::stats(ref c).frames==ir::count(p),b"match",18);0}case .err(e):{e.code}};}
    return switch(if(mode==0){calls::match(p,snapshot,compile::checked_state(checked),ref f.plan,ref f.sources,ref f.metadata,work)}else if(mode==1){recipes::match(p,snapshot,compile::checked_state(checked),ref f.plan,ref f.sources,ref f.metadata,work)}else{recipes::match_unit(p,snapshot,compile::checked_state(checked),ref f.plan,ref f.sources,ref f.metadata,work)}){case .ok(map):{require(frames::unit_count(ref map)==ir::count(p),b"match",19);0}case .err(code):{code}};
}
@effects(console) fn compile::CheckedBodies checked(ref Fixture f){return switch(compile::check_range(ref f.sources,ref f.metadata,0,1)){case .ok(p):{p}case .err(_):{require(false,b"checked",20);checked(f)}};}
// Real original sources produce the detached Program. No provider or replacement
// lowering stands in for the saved recipe at the comparison/coverage boundary.
@effects(console) pub fn void verify(ref u8[] name,ref u8[] text,bool boundary){
    let f=fixture(text);let owned=unit(ref f);let data=encoded(ref owned);let p=restored(wire::content(ref data));let snapshot=snapshot(ref owned);let checked=checked(ref f);var u32 mode=0;
    while(mode<5){var usize work=0;let code=match(ref p,ref snapshot,ref f,ref checked,mode,ref mut work);if(code!=0){discard io::write_stderr_bytes(name);discard io::write_stderr(" mode=");discard io::write_stderr_u32(mode);discard io::write_stderr(" code=");discard io::write_stderr_u32(code);require(false,b"verify",21);}require(work>0,b"verify",22);
        if(boundary){let cost=work;require(cost<budget::work_limit(),b"verify",23);work=budget::work_limit()-cost;require(match(ref p,ref snapshot,ref f,ref checked,mode,ref mut work)==0 && work==budget::work_limit(),b"verify",24);work=budget::work_limit()-cost+1;require(match(ref p,ref snapshot,ref f,ref checked,mode,ref mut work)==1002 && work<=budget::work_limit(),b"verify",25);work=budget::work_limit()+1;require(match(ref p,ref snapshot,ref f,ref checked,mode,ref mut work)==1002 && work==budget::work_limit()+1,b"verify",26);}mode+=1;
    }
}
@effects(console) pub fn void mismatch(ref u8[] first,ref u8[] second){
    let a=fixture(first);let b=fixture(second);let owner=unit(ref a);let other=unit(ref b);let raw=encoded(ref other);let p=restored(wire::content(ref raw));let snapshot=snapshot(ref owner);let checked=checked(ref a);var u32 mode=0;
    while(mode<5){var usize work=0;require(match(ref p,ref snapshot,ref a,ref checked,mode,ref mut work)==1001 && work>0,b"mismatch",27);mode+=1;}
}
@effects(console) pub fn void scalar_control(){let f=fixture(b"module main;fn i32 plus(i32 x){return x+1;}fn i32 main(){return plus(41);}");let owned=unit(ref f);var usize work=0;let data=switch(encode::encode(identified::ir_program(ref owned),ref mut work)){case .ok(p):{p}case .err(_):{require(false,b"scalar_control",28);return;}};work=0;let p=switch(decode::decode(wire::content(ref data),ref mut work)){case .ok(p):{p}case .err(_):{require(false,b"scalar_control",29);return;}};let snapshot=snapshot(ref owned);let checked=checked(ref f);var u32 mode=0;while(mode<5){work=0;require(match(ref p,ref snapshot,ref f,ref checked,mode,ref mut work)==0,b"scalar_control",30);mode+=1;}work=0;require(switch(encode::encode_closures(ref p,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}},b"scalar_control",31);}
@effects(console) fn void number(ref mut bytes::Buffer raw,usize at,usize input){var bytes::Buffer out=bytes::empty();var budget::Status status=budget::Status{.code=0};var usize work=0;budget::number(ref mut out,input,ref mut status,ref mut work);require(status.code==0 && at+8<=bytes::len(raw),b"number",32);var usize i=0;while(i<8){bytes::view_mut(raw)[at+i]=bytes::view(ref out)[i];i+=1;}}
@effects(console) fn budget::Span body_span(ref u8[] raw,usize frame){var usize work=0;var c=switch(budget::cursor_marker(raw,b"FYPROG02",ref mut work)){case .ok(p):{p}case .err(_):{require(false,b"body_span",33);return body_span(raw,frame);}};discard budget::member_from(raw,ref mut c,8388608,b"FYCLEAN2",ref mut work);discard budget::member_from(raw,ref mut c,8388608,b"FYSDK000",ref mut work);discard budget::member_from(raw,ref mut c,8388608,b"FYMETA00",ref mut work);discard budget::member_from(raw,ref mut c,8388608,b"FYSTATE0",ref mut work);let count=budget::bodies_from(raw,ref mut c,ref mut work);var usize i=1;while(i<=count){let span=budget::member_from(raw,ref mut c,16777216,b"FYBODY01",ref mut work);require(c.code==0,b"body_span",34);if(i==frame){return span;}i+=1;}require(false,b"body_span",35);return budget::Span{.start=0,.end=0};}
// All three variants retain a valid detached Program. The argument span passes
// even full body matching; only the source call/argument recipe detects it.
@effects(console) pub fn void detached_forgery(){
    let f=fixture(b"module main;fn i32 main(){let fn i32(i32) f=fn [] i32(i32 x){return x+1;};let fn i32(i32) g=fn [] i32(i32 x){return x*2;};discard g;return f(20);}");let owned=unit(ref f);let original=identified::ir_program(ref owned);let data=encoded(ref owned);let raw=wire::content(ref data);let snapshot=snapshot(ref owned);let checked=checked(ref f);var usize frame=1;
    while(frame<=ir::count(original)){let b=ir::at(original,frame);var usize i=0;while(i<ir::call_count(b)){let call=ir::call_at(b,i);if(call.kind==4){let span=body_span(raw,frame);let node=ir::node(b,ir::node_at(b,call.expression-1).value.left);let callable=ir::node_at(b,node-1).value;var usize alternative=0;var usize j=1;while(j<=ir::local_count(b)){let l=ir::local_at(b,j);if(l.type==callable.type && j!=callable.binding){alternative=j;}j+=1;}require(alternative!=0,b"detached_forgery",36);var u32 mode=0;
            while(mode<3){var changed=copied(raw);if(mode==0){let arguments=span.start+56+ir::node_count(b)*344+8+ir::statement_count(b)*96+8+ir::local_count(b)*144+8+ir::call_count(b)*88+8;let a=ir::argument_at(b,call.argument_start);number(ref mut changed,arguments+call.argument_start*96+48,a.start+1);}
                else if(mode==1){let at=span.start+56+(node-1)*344;number(ref mut changed,at+24,alternative);number(ref mut changed,at+128,alternative);}
                else{let at=span.start+56+(node-1)*344;number(ref mut changed,at+280,ir::node_at(b,node-1).syntax.token+1);}
                var usize decoded=0;let p=switch(decode::decode_closures(bytes::view(ref changed),ref mut decoded)){case .ok(p):{require(proof::check(ref p).code==0,b"detached-proof",100+mode);p}case .err(code):{discard io::write_stderr("mutation=");discard io::write_stderr_u32(mode);discard io::write_stderr(" decode=");discard io::write_stderr_u32(code);require(false,b"detached-decode",100+mode);return;}};var usize work=0;if(mode==0){require(bodies::match(ref p,ref snapshot,compile::checked_state(ref checked),ref f.plan,ref f.sources,ref f.metadata,ref mut work).is_ok,b"detached_forgery",37);}work=0;require(match(ref p,ref snapshot,ref f,ref checked,0,ref mut work)==1001 && work>0,b"detached_forgery",38);work=0;require(match(ref p,ref snapshot,ref f,ref checked,3,ref mut work)==1001,b"detached_forgery",39);mode+=1;
            }
            // A nonzero static target and an uninitialized named callable do
            // not survive the independent fresh Program proof at decode.
            var bad=copied(raw);let calls=span.start+56+ir::node_count(b)*344+8+ir::statement_count(b)*96+8+ir::local_count(b)*144+8;number(ref mut bad,calls+i*88+24,1);var usize work=0;require(switch(decode::decode_closures(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(e):{e==1001}},b"detached_forgery",40);bad=copied(raw);let locals=span.start+56+ir::node_count(b)*344+8+ir::statement_count(b)*96+8;number(ref mut bad,locals+(callable.binding-1)*144+32,0);work=0;require(switch(decode::decode_closures(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(e):{e==1001}},b"detached_forgery",41);return;
        }i+=1;}frame+=1;
    }require(false,b"detached_forgery",42);
}
