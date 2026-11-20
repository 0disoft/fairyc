module tests::closure_program_artifact;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_verify as proof;
import module_unit_checked_bodies as checked;
import module_unit_cleanup_mir_model as clean;
import module_program_artifact_encode as encode;
import module_program_artifact_decode as decode;
import module_program_artifact_wire as framing;
import candidate_artifact_wire as wire;
import module_signatures as signatures;
import tests::module_body_artifact_sample as sample;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn Result<ir::Program,u32> retained(ref u8[] text){
    let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{return .err(11);}};
    let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return .err(1001);}};
    let source=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{return .err(1001);}};
    if(syntax::problem_count(ref source)!=0){return .err(syntax::problem_at(ref source,0).code);}var sources=v::empty<syntax::Parsed>();if(!v::push<syntax::Parsed>(ref mut sources,source).is_ok){return .err(11);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let owned=switch(checked::build(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    return switch(lower::lower_checked(ref sources,ref owned)){case .ok(p):{.ok(p)}case .err(e):{.err(e.code)}};
}
fn u32 rejected(ref u8[] raw){var usize work=0;return switch(decode::decode_closures(raw,ref mut work)){case .ok(_):{0}case .err(e):{e}};}
fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);copied(raw)}};}
fn void zero(ref mut bytes::Buffer raw,usize offset){var usize i=0;while(i<8){bytes::view_mut(raw)[offset+i]=0;i+=1;}}
// Detached Program roundtrip only. There is no source-bound Complete or cache
// publication here, and checking this source does not execute the assertions.
pub fn void verify(ref u8[] text){
    let p=switch(retained(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var usize cost=0;let saved=switch(encode::encode_closures(ref p,ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};let raw=wire::content(ref saved);assert(bytes::equal(raw[..8],b"FYPROG02"));
    var usize work=0;assert(switch(encode::encode(ref p,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});work=0;assert(switch(decode::decode(raw,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});work=0;assert(switch(decode::decode_aggregates(raw,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});
    var usize decoded_cost=0;let restored=switch(decode::decode_closures(raw,ref mut decoded_cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(clean::project(ir::cleanup_ir(ref restored))==clean::project(ir::cleanup_ir(ref p)) && clean::aggregates(ir::cleanup_ir(ref restored))==clean::aggregates(ir::cleanup_ir(ref p)));
    assert(ir::work(ref restored)==0 && clean::closures(ir::cleanup_ir(ref restored)) && ir::count(ref restored)==ir::count(ref p) && proof::check(ref restored).code==0);
    work=0;let again=switch(encode::encode_closures(ref restored,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(raw,wire::content(ref again)));
    work=framing::work_limit()-decoded_cost;assert(switch(decode::decode_closures(raw,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==framing::work_limit());work=framing::work_limit()-decoded_cost+1;assert(switch(decode::decode_closures(raw,ref mut work)){case .ok(_):{false}case .err(e):{e==1002}} && work<=framing::work_limit());
    var bad=copied(raw);bytes::view_mut(ref mut bad)[7]=48;assert(rejected(bytes::view(ref bad))==1001);bytes::view_mut(ref mut bad)[7]=49;assert(rejected(bytes::view(ref bad))==1001);assert(rejected(raw[..7])==1001 && rejected(raw[..raw.len-1])==1001);
    bad=copied(raw);assert(bytes::append_byte(ref mut bad,0).is_ok);assert(rejected(bytes::view(ref bad))==1001);
    work=0;var c=switch(framing::cursor_marker(raw,b"FYPROG02",ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let cleaned=framing::member_from(raw,ref mut c,8388608,b"FYCLEAN2",ref mut work);
    bad=copied(raw);bytes::view_mut(ref mut bad)[cleaned.start+15]=2;assert(rejected(bytes::view(ref bad))==1001);bad=copied(raw);bytes::view_mut(ref mut bad)[cleaned.start+23]=2;assert(rejected(bytes::view(ref bad))==1001);discard framing::member_from(raw,ref mut c,8388608,b"FYSDK000",ref mut work);discard framing::member_from(raw,ref mut c,8388608,b"FYMETA00",ref mut work);discard framing::member_from(raw,ref mut c,8388608,b"FYSTATE0",ref mut work);
    assert(framing::bodies_from(raw,ref mut c,ref mut work)==ir::count(ref p));bad=copied(raw);bytes::view_mut(ref mut bad)[cleaned.start+7]=48;assert(rejected(bytes::view(ref bad))==1001);var usize frame=1;
    while(frame<=ir::count(ref p)){let span=framing::member_from(raw,ref mut c,16777216,b"FYBODY01",ref mut work);assert(framing::status(ref c)==0);let b=ir::at(ref p,frame);
        bad=copied(raw);bytes::view_mut(ref mut bad)[span.start+7]=48;assert(rejected(bytes::view(ref bad))==1001);
        var usize i=0;while(i<ir::node_count(b)){if(ir::node_at(b,i).value.kind==36){bad=copied(raw);zero(ref mut bad,span.start+56+i*344+72);assert(rejected(bytes::view(ref bad))==1001);}i+=1;}
        // Structural staging cannot grant authority to an uninitialized callee.
        i=0;while(i<ir::call_count(b)){let call=ir::call_at(b,i);if(call.kind==4){let e=ir::node_at(b,call.expression-1).value;let callee=ir::node_at(b,ir::node(b,e.left)-1).value;let local=ir::local_at(b,callee.binding);if(local.kind==2){let first=span.start+56+ir::node_count(b)*344+8+ir::statement_count(b)*96+8;bad=copied(raw);zero(ref mut bad,first+(callee.binding-1)*144+32);assert(rejected(bytes::view(ref bad))==1001);}}i+=1;}frame+=1;
    }assert(framing::end(raw,ref mut c,ref mut work)==0);
}
// Full old-family byte equality is supplied by an independent Go record oracle.
pub fn void scalar_golden(ref u8[] expected){let p=sample::build();var usize work=0;let saved=switch(encode::encode(ref p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(wire::content(ref saved),expected));work=0;let restored=switch(decode::decode(expected,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ir::work(ref restored)==0 && proof::check(ref restored).code==0);work=0;assert(switch(encode::encode_closures(ref p,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});assert(rejected(expected)==1001);}
