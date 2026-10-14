module tests::module_identity_artifact;
import tests::module_identity_artifact_fixture as fixture;
import tests::module_body_artifact_cleanup as ordinary;
import module_identity_artifact_model as model;
import module_identity_artifact_encode as encoder;
import module_identity_artifact_decode as decoder;
import module_program_artifact_encode as program_encoder;
import module_program_artifact_decode as program_decoder;
import module_source_artifact_model as source_model;
import module_source_artifact_encode as source_encoder;
import module_source_artifact_decode as source_decoder;
import module_source_artifact_rebind as source_rebind;
import project_compiler_input as input;
import module_project_identified_ir as project;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_sdk_sources as sdk;
import module_source_specializations as selection;
import module_public_interface_model as records;
import candidate_artifact_wire as wire;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
@effects(console) fn model::Snapshot decoded(ref u8[] raw,ref clean::Program context,ref mut usize work){return switch(decoder::decode(raw,context,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return decoded(raw,context,work);}};}
@effects(console) fn wire::Payload encoded(ref model::Snapshot p,ref clean::Program context,ref mut usize work){return switch(encoder::encode(p,context,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return encoded(p,context,work);}};}
@effects(filesystem,console) fn model::Snapshot detached(){let data=fixture::data();let context=fixture::context(0);var usize work=0;return decoded(fixture::golden(ref data),ref context,ref mut work);}
fn void same(ref model::Snapshot a,ref model::Snapshot b){assert(model::type_count(a)==model::type_count(b) && model::frame_count(a)==model::frame_count(b) && bytes::equal(model::names(a),model::names(b)));var usize i=1;while(i<=model::type_count(a)+model::frame_count(a)){let x=if(i<=model::type_count(a)){model::type_at(a,i)}else{model::frame_at(a,i-model::type_count(a))};let y=if(i<=model::type_count(b)){model::type_at(b,i)}else{model::frame_at(b,i-model::type_count(b))};assert(x.node==y.node && x.id.start==y.id.start && x.id.end==y.id.end);i+=1;}}
fn u32 rejected(ref u8[] raw,ref clean::Program context){var usize work=0;return switch(decoder::decode(raw,context,ref mut work)){case .ok(_):{0}case .err(code):{code}};}
fn model::Snapshot renumbered(ref model::Snapshot input,ref clean::Program context){var v::Vec<selection::Row> types=v::empty<selection::Row>();var v::Vec<selection::Row> frames=v::empty<selection::Row>();var usize i=1;
    while(i<=model::type_count(input)){var row=model::type_at(input,i);if(row.node!=0){row.node=4096;}assert(v::push<selection::Row>(ref mut types,row).is_ok);i+=1;}i=1;
    while(i<=model::frame_count(input)){var row=model::frame_at(input,i);if(row.node!=0){row.node=4096;}assert(v::push<selection::Row>(ref mut frames,row).is_ok);i+=1;}var usize work=0;
    return switch(model::assemble(copied(model::names(input)),types,frames,context,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);renumbered(input,context)}};
}
@test @effects(filesystem,console)
fn void identity_rows_match_independent_golden_and_detach(){let data=fixture::data();let context=fixture::context(0);let p=fixture::sample(ref context);var usize work=0;let data_bytes=encoded(ref p,ref context,ref mut work);assert(bytes::equal(wire::content(ref data_bytes),fixture::golden(ref data)));let copy=detached();same(ref p,ref copy);work=0;let again=encoded(ref copy,ref context,ref mut work);assert(bytes::equal(wire::content(ref again),fixture::golden(ref data)));
    let plain=ordinary::build();var v::Vec<selection::Row> types=v::empty<selection::Row>();var v::Vec<selection::Row> frames=v::empty<selection::Row>();let zero=selection::Row{.node=0,.id=records::Text{.start=0,.end=0}};assert(v::push<selection::Row>(ref mut types,zero).is_ok && v::push<selection::Row>(ref mut types,zero).is_ok && v::push<selection::Row>(ref mut frames,zero).is_ok);work=0;let ordinary=switch(model::assemble(bytes::empty(),types,frames,ref plain,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let empty=encoded(ref ordinary,ref plain,ref mut work);assert(wire::content(ref empty).len==104);work=0;let empty_copy=decoded(wire::content(ref empty),ref plain,ref mut work);assert(model::names(ref empty_copy).len==0 && model::frame_at(ref empty_copy,1).node==0);
}
@test @effects(filesystem,console)
fn void identity_codec_preserves_exact_work_and_context_bounds(){let p=detached();let context=fixture::context(0);var usize cost=0;let data=encoded(ref p,ref context,ref mut cost);var usize work=16777216-cost;let fit=encoded(ref p,ref context,ref mut work);assert(work==16777216 && bytes::equal(wire::content(ref fit),wire::content(ref data)));work=16777216-cost+1;assert(switch(encoder::encode(ref p,ref context,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);
    cost=0;let copy=decoded(wire::content(ref data),ref context,ref mut cost);work=16777216-cost;let exact=decoded(wire::content(ref data),ref context,ref mut work);assert(work==16777216);same(ref copy,ref exact);work=16777216-cost+1;assert(switch(decoder::decode(wire::content(ref data),ref context,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);work=16777217;assert(switch(decoder::decode(wire::content(ref data),ref context,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==16777217);
    let other=ordinary::build();assert(rejected(wire::content(ref data),ref other)==1001);let resource_context=fixture::context(1);assert(rejected(wire::content(ref data),ref resource_context)==1001);let defer_context=fixture::context(2);assert(rejected(wire::content(ref data),ref defer_context)==1001);
}
@test @effects(filesystem)
fn void identity_codec_rejects_truncation_spans_names_and_cleanup_mismatch(){let data=fixture::data();let raw=fixture::golden(ref data);let context=fixture::context(0);assert(raw.len==377);var usize i=0;while(i<raw.len){assert(rejected(raw[..i],ref context)!=0);i+=1;}var trailing=copied(raw);assert(bytes::append_byte(ref mut trailing,0).is_ok && rejected(bytes::view(ref trailing),ref context)==1001);
    var usize mode=0;while(mode<13){var bad=copied(raw);let view=bytes::view_mut(ref mut bad);if(mode==0){view[7]=49;}else if(mode==1){view[8]=255;}else if(mode==2){view[64]=1;}else if(mode==3){view[23]=1;}else if(mode==4){view[79]=1;}else if(mode==5){view[87]=66;}else if(mode==6){view[88]=255;}else if(mode==7){view[243]=71;}else if(mode==8){view[313]=98;}else if(mode==9){view[151]=1;}else if(mode==10){view[168]=255;}else if(mode==11){view[72]=1;}else{view[376]=65;}assert(rejected(bytes::view(ref bad),ref context)!=0);mode+=1;}
}
@test @effects(filesystem,console)
fn void actual_generic_vec_program_restores_ir_sdk_provenance_and_identity_rows(){let data=fixture::data();let loaded=fixture::loaded(fixture::workspace(ref data));let original=fixture::compiled(ref loaded);let clean=ir::cleanup_ir(project::ir_program(ref original));var usize work=0;
    let ids=switch(encoder::project_owner(ref original,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};work=0;let copied_ids=decoded(wire::content(ref ids),clean,ref mut work);work=0;assert(project::matches_snapshot(ref original,ref copied_ids,ref mut work)==0);var usize named=0;var usize vectors=0;var usize i=1;while(i<=model::frame_count(ref copied_ids)){if(model::frame_at(ref copied_ids,i).node!=0){named+=1;}i+=1;}i=1;while(i<=model::type_count(ref copied_ids)){if(model::type_at(ref copied_ids,i).node!=0){vectors+=1;}i+=1;}assert(named>1 && vectors>0 && sdk::count(ir::sdk_units(project::ir_program(ref original)))>0);
    let other_handles=renumbered(ref copied_ids,clean);work=0;assert(project::matches_snapshot(ref original,ref other_handles,ref mut work)==0);
    work=0;let program_data=switch(program_encoder::encode(project::ir_program(ref original),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};work=0;let restored=switch(program_decoder::decode(wire::content(ref program_data),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};assert(proof::check(ref restored).code==0 && ir::work(ref restored)==0 && ir::count(ref restored)==ir::count(project::ir_program(ref original)) && sdk::count(ir::sdk_units(ref restored))==sdk::count(ir::sdk_units(project::ir_program(ref original))) && bytes::equal(sdk::data(ir::sdk_units(ref restored)),sdk::data(ir::sdk_units(project::ir_program(ref original)))));
    work=0;let restored_ids=decoded(wire::content(ref ids),ir::cleanup_ir(ref restored),ref mut work);same(ref copied_ids,ref restored_ids);work=0;let again=switch(program_encoder::encode(ref restored,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(wire::content(ref again),wire::content(ref program_data)));
    work=0;let source_data=switch(source_encoder::provenance(project::source_provenance(ref original),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let stamps=switch(source_decoder::decode(wire::content(ref source_data),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let current=switch(source_rebind::rebind(ref stamps,input::sources(fixture::inputs(ref loaded)),input::owners(fixture::inputs(ref loaded)),input::bindings(fixture::inputs(ref loaded)),fixture::coordinate_table(ref loaded),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var bad=copied(wire::content(ref ids));let usize header=16+model::type_count(ref copied_ids)*24+8+model::frame_count(ref copied_ids)*24+8;var bool changed=false;i=1;while(i<=model::frame_count(ref copied_ids)){let row=model::frame_at(ref copied_ids,i);let frame=clean::frame_at(clean,i);if(row.node!=0 && frame.kind==1){let old=bytes::view(ref bad)[header+row.id.start+3];bytes::view_mut(ref mut bad)[header+row.id.start+3]=if(old==97){98}else{97};changed=true;break;}i+=1;}assert(changed);work=0;let valid_shape=decoded(bytes::view(ref bad),clean,ref mut work);work=0;assert(project::matches_snapshot(ref original,ref valid_shape,ref mut work)==1001);
}
