module tests::module_cleanup_artifact;
import module_cleanup_artifact_encode as encoder;
import module_cleanup_artifact_decode as decoder;
import candidate_artifact_wire as wire;
import module_unit_cleanup_mir_model as ir;
import module_unit_cleanup_mir_verify as proof;
import module_types as types;
import candidate_limits as limits;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;
import module_unit_cleanup_mir_lower as lower;
import module_unit_moves as moves;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
fn ir::Program sample(bool project,usize history){
    var v::Vec<types::Type> types=v::empty<types::Type>();
    assert(v::push<types::Type>(ref mut types,types::Type{.kind=1,.scalar=1,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);
    assert(v::push<types::Type>(ref mut types,types::Type{.kind=1,.scalar=7,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);
    assert(v::push<types::Type>(ref mut types,types::Type{.kind=3,.scalar=0,.module_index=1,.declaration=3,.first=0,.second=0}).is_ok);
    assert(v::push<types::Type>(ref mut types,types::Type{.kind=5,.scalar=0,.module_index=0,.declaration=0,.first=3,.second=0}).is_ok);
    assert(v::push<types::Type>(ref mut types,types::Type{.kind=6,.scalar=0,.module_index=0,.declaration=0,.first=4,.second=1}).is_ok);
    assert(v::push<types::Type>(ref mut types,types::Type{.kind=3,.scalar=0,.module_index=0,.declaration=9,.first=0,.second=0}).is_ok);
    var v::Vec<ir::Glue> glues=v::empty<ir::Glue>();
    assert(v::push<ir::Glue>(ref mut glues,ir::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
    assert(v::push<ir::Glue>(ref mut glues,ir::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
    assert(v::push<ir::Glue>(ref mut glues,ir::Glue{.kind=1,.frame=2,.nonempty=true,.effects=1,.abilities=7,.field_start=0,.field_count=1}).is_ok);
    assert(v::push<ir::Glue>(ref mut glues,ir::Glue{.kind=2,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
    assert(v::push<ir::Glue>(ref mut glues,ir::Glue{.kind=3,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
    assert(v::push<ir::Glue>(ref mut glues,ir::Glue{.kind=1,.frame=0,.nonempty=false,.effects=0,.abilities=1,.field_start=1,.field_count=2}).is_ok);
    var v::Vec<ir::Frame> frames=v::empty<ir::Frame>();
    assert(v::push<ir::Frame>(ref mut frames,ir::Frame{.kind=1,.context=1,.module_index=0,.declaration=1,.result=1,.effects=1,.receiver=0,.receiver_type=0}).is_ok);
    assert(v::push<ir::Frame>(ref mut frames,ir::Frame{.kind=2,.context=2,.module_index=1,.declaration=3,.result=2,.effects=1,.receiver=1,.receiver_type=3}).is_ok);
    assert(v::push<ir::Frame>(ref mut frames,ir::Frame{.kind=3,.context=3,.module_index=1,.declaration=7,.result=1,.effects=1,.receiver=0,.receiver_type=0}).is_ok);
    var v::Vec<ir::Field> fields=v::empty<ir::Field>();
    assert(v::push<ir::Field>(ref mut fields,ir::Field{.owner=3,.ordinal=0,.type=1}).is_ok);
    assert(v::push<ir::Field>(ref mut fields,ir::Field{.owner=6,.ordinal=0,.type=5}).is_ok);
    assert(v::push<ir::Field>(ref mut fields,ir::Field{.owner=6,.ordinal=1,.type=3}).is_ok);
    var v::Vec<ir::Operand> operands=v::empty<ir::Operand>();
    assert(v::push<ir::Operand>(ref mut operands,ir::Operand{.frame=1,.kind=1,.id=2,.type=6,.start=100,.end=110}).is_ok);
    assert(v::push<ir::Operand>(ref mut operands,ir::Operand{.frame=1,.kind=2,.id=3,.type=3,.start=120,.end=130}).is_ok);
    var v::Vec<ir::Drop> drops=v::empty<ir::Drop>();
    assert(v::push<ir::Drop>(ref mut drops,ir::Drop{.operand=1,.exit=1,.start=110,.end=111}).is_ok);
    assert(v::push<ir::Drop>(ref mut drops,ir::Drop{.operand=2,.exit=6,.start=130,.end=131}).is_ok);
    return if(project){switch(ir::assemble_project(types,frames,glues,fields,operands,drops,history)){case .ok(p):{p}case .err(_):{assert(false);return sample(project,history);}}}else{switch(ir::assemble(types,frames,glues,fields,operands,drops,history)){case .ok(p):{p}case .err(_):{assert(false);return sample(project,history);}}};
}
fn wire::Payload encoded(ref ir::Program p,ref mut usize work){return switch(encoder::encode(p,work)){case .ok(p):{p}case .err(_):{assert(false);return encoded(p,work);}};}
fn ir::Program decoded(ref u8[] raw,ref mut usize work){return switch(decoder::decode(raw,work)){case .ok(p):{p}case .err(_):{assert(false);return decoded(raw,work);}};}
@effects(filesystem) fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(files::manifest(view[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
fn u32 rejected(ref u8[] raw){var usize work=0;return switch(decoder::decode(raw,ref mut work)){case .ok(_):{0}case .err(code):{code}};}
@test @effects(filesystem)
fn void exact_binary_roundtrip_preserves_provider_cleanup_and_order(){let expected=fixture();let p=sample(false,123);var usize work=0;let data=encoded(ref p,ref mut work);assert(bytes::equal(wire::content(ref data),bytes::view(ref expected)));work=0;let restored=decoded(wire::content(ref data),ref mut work);assert(!ir::project(ref restored) && ir::work(ref restored)==0 && proof::check(ref restored).code==0);work=0;let again=encoded(ref restored,ref mut work);assert(bytes::equal(wire::content(ref data),wire::content(ref again)));assert(ir::type_count(ref restored)==6 && ir::frame_count(ref restored)==3 && ir::drop_count(ref restored)==2);assert(ir::field_at(ref restored,1).ordinal==0 && ir::field_at(ref restored,2).ordinal==1);let provider=ir::frame_at(ref restored,ir::glue_at(ref restored,3).frame);assert(provider.kind==2 && provider.module_index==1 && provider.receiver_type==3 && provider.effects==1);}
@test
fn void project_mode_and_fresh_histories_have_exact_shared_work(){let p=sample(true,limits::project_work());let other=sample(true,0);var usize cost=0;let data=encoded(ref p,ref mut cost);var usize work=0;let equal=encoded(ref other,ref mut work);assert(bytes::equal(wire::content(ref data),wire::content(ref equal)));work=16777216-cost;let fit=encoded(ref p,ref mut work);assert(work==16777216 && bytes::equal(wire::content(ref fit),wire::content(ref data)));work=16777216-cost+1;assert(switch(encoder::encode(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);cost=0;let restored=decoded(wire::content(ref data),ref mut cost);assert(ir::project(ref restored) && ir::work(ref restored)==0 && cost>40960);work=16777216-cost;let exact=decoded(wire::content(ref data),ref mut work);assert(work==16777216 && ir::frame_count(ref exact)==3);work=16777216-cost+1;assert(switch(decoder::decode(wire::content(ref data),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);work=16777217;assert(switch(decoder::decode(wire::content(ref data),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==16777217);}
@test
fn void malformed_payloads_fail_before_publication_or_arithmetic_traps(){let p=sample(false,0);var usize work=0;let data=encoded(ref p,ref mut work);let raw=wire::content(ref data);assert(raw.len==1104);var usize i=0;while(i<raw.len){assert(rejected(raw[..i])!=0);i+=1;}var bytes::Buffer trailing=copied(raw);assert(bytes::append_byte(ref mut trailing,0).is_ok && rejected(bytes::view(ref trailing))==1001);var u32 mode=0;while(mode<12){var bytes::Buffer changed=copied(raw);let view=bytes::view_mut(ref mut changed);if(mode==0){view[7]=49;}else if(mode==1){view[15]=2;}else if(mode==2){view[16]=255;}else if(mode==3){view[24]=1;}else if(mode==4){view[144]=255;}else if(mode==5){view[207]=4;}else if(mode==6){view[463]=2;}else if(mode==7){view[703]=0;}else if(mode==8){view[943]=4;}else if(mode==9){view[967]=255;}else if(mode==10){view[1047]=3;}else{view[895]=1;}assert(rejected(bytes::view(ref changed))==if(mode==2){1002}else{1001});mode+=1;}var bytes::Buffer oversized=bytes::empty();assert(bytes::resize_zeroed(ref mut oversized,wire::byte_limit()+1).is_ok && rejected(bytes::view(ref oversized))==1002);}

fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn ir::Program retained(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,v::len<syntax::Parsed>(ref sources))){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(_):{assert(false);retained(root,provider)}};
}
fn ir::Program nested(){return retained(b"module main;import provider as p;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{Result<p::R,i32> value;R other;cleanup{}}fn R make(){return R{.n=1};}@effects(console)fn void f(){let Box box=Box{.value=.ok(p::make()),.other=make()};}",b"module provider;@effects(console)pub resource R{i32 n;cleanup{discard 1;}}pub fn R make(){return R{.n=1};}");}

@test
fn void source_lowered_ir_roundtrips_after_ast_and_flow_owners_leave_scope(){let p=nested();var usize work=0;let payload=encoded(ref p,ref mut work);work=0;let restored=decoded(wire::content(ref payload),ref mut work);assert(proof::check(ref restored).code==0 && ir::drop_count(ref restored)==ir::drop_count(ref p) && ir::operand_count(ref restored)==ir::operand_count(ref p));work=0;let again=encoded(ref restored,ref mut work);assert(bytes::equal(wire::content(ref payload),wire::content(ref again)));let o=ir::operand_at(ref restored,ir::drop_at(ref restored,0).operand);let glue=ir::glue_at(ref restored,o.type);assert(glue.field_count==2 && ir::field_at(ref restored,glue.field_start).ordinal==0 && ir::field_at(ref restored,glue.field_start+1).ordinal==1);}
