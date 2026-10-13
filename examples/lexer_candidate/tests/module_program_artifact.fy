module tests::module_program_artifact;
import tests::module_artifact_source_fixture as fixture;
import tests::module_state_artifact_fixture as state_fixture;
import tests::module_body_artifact_sample as sample;
import module_program_artifact_encode as encoder;
import module_program_artifact_decode as decoder;
import module_program_artifact_wire as framing;
import candidate_artifact_wire as wire;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_c_emit as emit;
import module_unit_cleanup_mir_model as clean;
import module_state_layout as state;
import std::bytes as bytes;
import std::io as io;

fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
@effects(console) fn wire::Payload encoded(ref ir::Program p,ref mut usize work){return switch(encoder::encode(p,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return encoded(p,work);}};}
@effects(console) fn ir::Program decoded(ref u8[] raw,ref mut usize work){return switch(decoder::decode(raw,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return decoded(raw,work);}};}
fn u32 rejected(ref u8[] raw,ref mut usize work){return switch(decoder::decode(raw,work)){case .ok(_):{0}case .err(code):{code}};}
@effects(console) fn ir::Program rich(){return fixture::retained(b"module main;import provider as p;enum E{Empty;Data{i32 a;bool b;};}fn E choose(){return .Data{.a=1,.b=true};}@repr(C)struct S{i32 n;bool yes;}@effects(console)resource R{i32 n;cleanup{discard 1;}}@must_use @effects(console)fn i32 f(i32 n){let E e=choose();discard e;let S x=S{.n=n,.yes=true};let R r=R{.n=n};let p::S y=p::make(n);return p::number(n);}",b"module provider;pub struct S{i32 n;}pub fn S make(i32 n){return S{.n=n};}@link_name(\"foreign_number\")unsafe extern C fn i32 foreign(i32 n);@must_use pub fn i32 number(i32 n){return unsafe{foreign(n)};}");}
@effects(console) fn ir::Program detached(bool states){let p=if(states){state_fixture::program()}else{rich()};var usize work=0;let raw=encoded(ref p,ref mut work);work=0;return decoded(wire::content(ref raw),ref mut work);}
@test @effects(console)
fn void whole_program_roundtrip_preserves_source_metadata_and_state(){let original=rich();var usize work=0;let data=encoded(ref original,ref mut work);let p=detached(false);assert(ir::work(ref p)==0 && proof::check(ref p).code==0 && ir::count(ref p)==ir::count(ref original) && clean::frame_count(ir::cleanup_ir(ref p))==ir::count(ref p));work=0;let again=encoded(ref p,ref mut work);assert(bytes::equal(wire::content(ref data),wire::content(ref again)) && ir::layout_count(ref p)>0 && ir::field_count(ref p)>0 && ir::parameter_count(ref p)>0);
    let with_state=state_fixture::program();work=0;let state_data=encoded(ref with_state,ref mut work);let restored=detached(true);work=0;let state_again=encoded(ref restored,ref mut work);assert(bytes::equal(wire::content(ref state_data),wire::content(ref state_again)) && state::count(ir::state_layouts(ref restored))==1 && state::field_count(ir::state_layouts(ref restored))==5 && bytes::equal(state::uid(ir::state_layouts(ref restored),0),b"example.token"));
    var bytes::Buffer first=bytes::empty();var bytes::Buffer second=bytes::empty();assert(emit::render(ref with_state,1,ref mut first).code==0 && emit::render(ref restored,1,ref mut second).code==0 && bytes::equal(bytes::view(ref first),bytes::view(ref second)));
}
@test @effects(console)
fn void whole_program_accounts_for_nested_verification_and_exact_work(){let p=sample::build();var usize cost=0;let data=encoded(ref p,ref mut cost);assert(cost>wire::content(ref data).len);var usize work=framing::work_limit()-cost;let exact=encoded(ref p,ref mut work);assert(work==framing::work_limit() && bytes::equal(wire::content(ref exact),wire::content(ref data)));work=framing::work_limit()-cost+1;assert(switch(encoder::encode(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=framing::work_limit());
    cost=0;let restored=decoded(wire::content(ref data),ref mut cost);var usize fresh=0;assert(proof::check_metered(ref restored,ref mut fresh).code==0 && cost>fresh && fresh>0);work=framing::work_limit()-cost;let fit=decoded(wire::content(ref data),ref mut work);assert(work==framing::work_limit() && ir::work(ref fit)==0);work=framing::work_limit()-cost+1;assert(rejected(wire::content(ref data),ref mut work)==1002 && work<=framing::work_limit());work=framing::work_limit()+1;assert(rejected(wire::content(ref data),ref mut work)==1002 && work==framing::work_limit()+1);
}
fn framing::Cursor parts(ref u8[] raw){var usize work=0;var c=switch(framing::cursor(raw,ref mut work)){case .ok(c):{c}case .err(_):{assert(false);return parts(raw);}};discard framing::member_from(raw,ref mut c,8388608,b"FYCLEAN0",ref mut work);discard framing::member_from(raw,ref mut c,8388608,b"FYSDK000",ref mut work);discard framing::member_from(raw,ref mut c,8388608,b"FYMETA00",ref mut work);discard framing::member_from(raw,ref mut c,8388608,b"FYSTATE0",ref mut work);assert(framing::status(ref c)==0);return c;}
@test @effects(console)
fn void whole_program_preflights_all_truncations_and_component_boundaries(){let p=sample::build();var usize work=0;let data=encoded(ref p,ref mut work);let raw=wire::content(ref data);var usize i=0;while(i<raw.len){work=0;assert(framing::preflight(raw[..i],ref mut work)!=0);i+=1;}
    var trailing=copied(raw);assert(bytes::append_byte(ref mut trailing,0).is_ok);work=0;assert(rejected(bytes::view(ref trailing),ref mut work)==1001);var bad=copied(raw);bytes::view_mut(ref mut bad)[7]=49;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);bad=copied(raw);bytes::view_mut(ref mut bad)[8]=255;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1002);bad=copied(raw);bytes::view_mut(ref mut bad)[23]=49;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);
    let c=parts(raw);bad=copied(raw);bytes::view_mut(ref mut bad)[c.offset+7]=0;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);bad=copied(raw);bytes::view_mut(ref mut bad)[c.offset]=255;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1002);
}
@test @effects(console)
fn void whole_program_rejects_nested_hash_context_and_frame_corruption(){let p=state_fixture::program();var usize work=0;let data=encoded(ref p,ref mut work);let raw=wire::content(ref data);var c=switch(framing::cursor(raw,ref mut work)){case .ok(c):{c}case .err(_):{assert(false);return;}};
    let clean_span=framing::member_from(raw,ref mut c,8388608,b"FYCLEAN0",ref mut work);let sdk_span=framing::member_from(raw,ref mut c,8388608,b"FYSDK000",ref mut work);let meta_span=framing::member_from(raw,ref mut c,8388608,b"FYMETA00",ref mut work);let state_span=framing::member_from(raw,ref mut c,8388608,b"FYSTATE0",ref mut work);assert(framing::bodies_from(raw,ref mut c,ref mut work)==1);let body_span=framing::member_from(raw,ref mut c,16777216,b"FYBODY00",ref mut work);assert(framing::end(raw,ref mut c,ref mut work)==0);
    var bad=copied(raw);bytes::view_mut(ref mut bad)[state_span.start+95]=0;work=13;assert(rejected(bytes::view(ref bad),ref mut work)==1001 && work>13);bad=copied(raw);bytes::view_mut(ref mut bad)[body_span.start+15]=2;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);bad=copied(raw);bytes::view_mut(ref mut bad)[meta_span.start+23]=0;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);
    bad=copied(raw);bytes::view_mut(ref mut bad)[sdk_span.start+15]=255;work=0;assert(rejected(bytes::view(ref bad),ref mut work)!=0);bad=copied(raw);bytes::view_mut(ref mut bad)[clean_span.start+15]=2;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);
}
