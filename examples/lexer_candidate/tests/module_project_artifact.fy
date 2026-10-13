module tests::module_project_artifact;
import tests::module_identity_artifact_fixture as fixture;
import module_project_artifact_model as model;
import module_project_artifact_encode as encoder;
import module_project_artifact_decode as decoder;
import module_program_artifact_wire as framing;
import candidate_artifact_wire as wire;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_sdk_sources as sdk;
import module_identity_artifact_model as identities;
import module_source_artifact_model as sources;
import module_source_artifact_rebind as rebind;
import project_compiler_input as input;
import std::bytes as bytes;
import std::io as io;

fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
@effects(filesystem,console) fn wire::Payload detached_packet(){let data=fixture::data();let loaded=fixture::loaded(fixture::workspace(ref data));let original=fixture::compiled(ref loaded);var usize work=0;return switch(encoder::encode(ref original,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return detached_packet();}};}
@effects(console) fn model::Restored decoded(ref u8[] raw,ref mut usize work){return switch(decoder::decode(raw,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return decoded(raw,work);}};}
@effects(console) fn wire::Payload encoded(ref model::Restored p,ref mut usize work){return switch(encoder::restored(p,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return encoded(p,work);}};}
fn u32 rejected(ref u8[] raw,ref mut usize work){return switch(decoder::decode(raw,work)){case .ok(_):{0}case .err(code):{code}};}
@test
fn void project_envelope_preflights_lengths_markers_and_trailing_bytes(){let raw=b"FYPRJ000\x00\x00\x00\x00\x00\x00\x00\x08FYPROG00\x00\x00\x00\x00\x00\x00\x00\x08FYPROV00\x00\x00\x00\x00\x00\x00\x00\x08FYIDS000";assert(raw.len==56);var usize work=0;assert(decoder::preflight(raw,ref mut work)==0);work=0;assert(rejected(raw,ref mut work)==1001);var usize i=0;while(i<raw.len){work=0;assert(decoder::preflight(raw[..i],ref mut work)!=0);i+=1;}
    var usize mode=0;while(mode<8){var bad=copied(raw);let view=bytes::view_mut(ref mut bad);if(mode==0){view[7]=49;}else if(mode==1){view[8]=255;}else if(mode==2){view[23]=49;}else if(mode==3){view[24]=255;}else if(mode==4){view[39]=49;}else if(mode==5){view[40]=255;}else if(mode==6){view[55]=49;}else{view[15]=7;}work=0;assert(decoder::preflight(bytes::view(ref bad),ref mut work)!=0);mode+=1;}
    var trailing=copied(raw);assert(bytes::append_byte(ref mut trailing,0).is_ok);work=0;assert(decoder::preflight(bytes::view(ref trailing),ref mut work)==1001);work=framing::work_limit()+1;assert(rejected(raw,ref mut work)==1002 && work==framing::work_limit()+1);
}
@test @effects(filesystem,console)
fn void actual_project_restores_detached_ir_sources_ids_and_rejects_sdk_contradictions(){let packet=detached_packet();let raw=wire::content(ref packet);var usize decode_cost=0;let restored=decoded(raw,ref mut decode_cost);let p=model::program(ref restored);let provenance=model::source_provenance(ref restored);let ids=model::source_identities(ref restored);assert(ir::work(p)==0 && ir::count(p)>100 && sources::count(provenance)==10 && sdk::count(ir::sdk_units(p))>0 && identities::frame_count(ids)==ir::count(p) && identities::names(ids).len>67);var usize fresh=0;assert(proof::check_metered(p,ref mut fresh).code==0 && fresh>0 && decode_cost>fresh);
    var usize encode_cost=0;let again=encoded(ref restored,ref mut encode_cost);assert(bytes::equal(raw,wire::content(ref again)));var usize work=framing::work_limit()-encode_cost;let fit=encoded(ref restored,ref mut work);assert(work==framing::work_limit() && bytes::equal(raw,wire::content(ref fit)));work=framing::work_limit()-encode_cost+1;assert(switch(encoder::restored(ref restored,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=framing::work_limit());
    work=framing::work_limit()-decode_cost;let exact=decoded(raw,ref mut work);assert(work==framing::work_limit() && ir::count(model::program(ref exact))==ir::count(p));work=framing::work_limit()-decode_cost+1;assert(rejected(raw,ref mut work)==1002 && work<=framing::work_limit());
    let data=fixture::data();let loaded=fixture::loaded(fixture::workspace(ref data));work=0;let current=switch(rebind::rebind(provenance,input::sources(fixture::inputs(ref loaded)),input::owners(fixture::inputs(ref loaded)),input::bindings(fixture::inputs(ref loaded)),fixture::coordinate_table(ref loaded),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};
    work=0;var c=switch(framing::cursor_marker(raw,b"FYPRJ000",ref mut work)){case .ok(c):{c}case .err(_):{assert(false);return;}};let program_span=framing::member_from(raw,ref mut c,framing::byte_limit(),b"FYPROG00",ref mut work);let source_span=framing::member_from(raw,ref mut c,8388608,b"FYPROV00",ref mut work);let identity_span=framing::member_from(raw,ref mut c,8388608,b"FYIDS000",ref mut work);assert(framing::end(raw,ref mut c,ref mut work)==0);
    var bad=copied(raw);bytes::view_mut(ref mut bad)[program_span.start+7]=49;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);bad=copied(raw);bytes::view_mut(ref mut bad)[identity_span.start+8]=255;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1002);
    let unit=sdk::at(ir::sdk_units(p),0);let digest_byte=source_span.start+16+unit.module_index*128+64+7;bad=copied(raw);let old=bytes::view(ref bad)[digest_byte];bytes::view_mut(ref mut bad)[digest_byte]=old^1;work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);
    bad=copied(raw);assert(bytes::append_byte(ref mut bad,0).is_ok);work=0;assert(rejected(bytes::view(ref bad),ref mut work)==1001);
}
