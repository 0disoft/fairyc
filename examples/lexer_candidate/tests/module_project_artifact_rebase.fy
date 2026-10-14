module tests::module_project_artifact_rebase;
import tests::module_project_artifact_current_fixture as fixture;
import module_project_artifact_current as binding;
import module_project_artifact_rebase as rebase;
import module_project_artifact_model as model;
import module_source_artifact_model as sources;
import module_program_artifact_encode as encoder;
import module_program_artifact_wire as framing;
import candidate_artifact_wire as wire;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_sdk_sources as sdk;
import module_state_layout as state;
import module_unit_c_emit as emitter;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
@effects(console) fn binding::Rebound rebound(ref u8[] raw,ref fixture::Current current){let saved=fixture::restored(raw);var usize work=0;return switch(binding::build_sources(saved,fixture::sources(current),fixture::owners(current),fixture::imports(current),fixture::coordinate_table(current),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return rebound(raw,current);}};}
@effects(console) fn ir::Program rebased(ref binding::Rebound input,ref mut usize work){return switch(rebase::build(input,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return rebased(input,work);}};}
@effects(console) fn wire::Payload encoded(ref ir::Program p){var usize work=0;return switch(encoder::encode(p,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return encoded(p);}};}
fn usize entry(ref ir::Program p,ref fixture::Current original){var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);let parsed=v::at<syntax::Parsed>(fixture::sources(original),f.module_index);let d=syntax::declaration_at(parsed,f.declaration-1);if(d.kind==3 && bytes::equal(syntax::source_view(parsed)[d.name_start..d.name_end],b"main")){return i;}i+=1;}assert(false);return 0;}
fn void preserved(ref ir::Program old,ref ir::Program now,ref binding::Rebound mapping){assert(ir::work(now)==0 && clean::work(ir::cleanup_ir(now))==0 && sdk::work(ir::sdk_units(now))==0 && ir::count(old)==ir::count(now) && ir::field_count(old)==ir::field_count(now));var usize i=1;var usize field_reads=0;var usize calls=0;
    while(i<=clean::type_count(ir::cleanup_ir(old))){let a=clean::type_at(ir::cleanup_ir(old),i);let b=clean::type_at(ir::cleanup_ir(now),i);assert(a.kind==b.kind && a.scalar==b.scalar && a.declaration==b.declaration && a.first==b.first && a.second==b.second && b.module_index==if(a.kind==2 || a.kind==3 || a.kind==4 || a.kind==13){binding::module_at(mapping,a.module_index)}else{a.module_index});i+=1;}i=1;
    while(i<=ir::count(old)){let a=clean::frame_at(ir::cleanup_ir(old),i);let b=clean::frame_at(ir::cleanup_ir(now),i);assert(a.kind==b.kind && a.context==b.context && a.declaration==b.declaration && a.receiver_type==b.receiver_type && b.module_index==binding::module_at(mapping,a.module_index));let first=ir::at(old,i);let second=ir::at(now,i);assert(ir::source_bytes(first)==ir::source_bytes(second) && ir::start(first)==ir::start(second) && ir::end(first)==ir::end(second) && bytes::equal(ir::text_view(first),ir::text_view(second)));var usize j=0;
        while(j<ir::node_count(first)){let x=ir::node_at(first,j).value;let y=ir::node_at(second,j).value;if(x.field.id!=0){assert(x.field.id==y.field.id && x.field.owner==y.field.owner && y.field.module_index==binding::module_at(mapping,x.field.module_index));field_reads+=1;}j+=1;}j=0;while(j<ir::call_count(first)){let x=ir::call_at(first,j);let y=ir::call_at(second,j);assert(x.instance==y.instance && x.target_declaration==y.target_declaration && y.target_module==binding::module_at(mapping,x.target_module));calls+=1;j+=1;}i+=1;
    }assert(field_reads>0 && calls>0);let old_state=ir::state_layouts(old);let now_state=ir::state_layouts(now);assert(state::count(old_state)==1 && state::count(now_state)==1);let a=state::at(old_state,0);let b=state::at(now_state,0);assert(b.owner==binding::owner_at(mapping,a.owner) && b.module_index==binding::module_at(mapping,a.module_index) && state::equal(a.hash,b.hash) && bytes::equal(state::names(old_state),state::names(now_state)));
    i=0;var usize prior=0;while(i<sdk::count(ir::sdk_units(now))){let unit=sdk::at(ir::sdk_units(now),i);assert(i==0 || unit.module_index>prior);prior=unit.module_index;var usize j=0;var bool matched=false;while(j<sdk::count(ir::sdk_units(old))){let previous=sdk::at(ir::sdk_units(old),j);if(previous.sdk_id==unit.sdk_id){assert(unit.module_index==binding::module_at(mapping,previous.module_index) && bytes::equal(sdk::source_at(ir::sdk_units(old),j),sdk::source_at(ir::sdk_units(now),i)));matched=true;}j+=1;}assert(matched);i+=1;}
}
@effects(console) fn void text(ref u8[] raw){let result=io::write_bytes(raw);assert(result.is_ok);}
@effects(console) fn void member(ref u8[] raw){assert(io::write_usize(raw.len).is_ok);text(b"\n");text(raw);}
@effects(console) fn void packet(ref emitter::Bundle p){text(b"FYCPARTS1\n");member(emitter::project_header(p));member(emitter::project_literals(p));assert(io::write_usize(emitter::project_unit_count(p)).is_ok);text(b"\n");var usize i=0;while(i<emitter::project_unit_count(p)){member(emitter::project_source(p,i));i+=1;}}
@test @effects(filesystem,console)
fn void whole_ir_rebase_preserves_links_fields_state_sdk_and_work_then_emits_native_packet(){let data=fixture::packet();let raw=wire::content(ref data);let original=fixture::current(0);let reversed=fixture::current(1);let mapping=rebound(raw,ref reversed);let old=model::program(binding::restored(ref mapping));var usize cost=0;let p=rebased(ref mapping,ref mut cost);preserved(old,ref p,ref mapping);var usize fresh=0;assert(proof::check_metered(ref p,ref mut fresh).code==0 && fresh>0 && cost>fresh);var usize stamp_work=0;let stamps=switch(sources::extract(binding::provenance(ref mapping),ref mut stamp_work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;assert(model::check(ref p,ref stamps,binding::source_identities(ref mapping),ref mut work)==0);
    work=framing::work_limit()-cost;let exact=rebased(ref mapping,ref mut work);assert(work==framing::work_limit());let first=encoded(ref p);let second=encoded(ref exact);assert(bytes::equal(wire::content(ref first),wire::content(ref second)));work=framing::work_limit()-cost+1;assert(switch(rebase::build(ref mapping,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=framing::work_limit());work=framing::work_limit()+1;assert(switch(rebase::build(ref mapping,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==framing::work_limit()+1);
    let same=rebound(raw,ref original);work=0;let unchanged=rebased(ref same,ref mut work);let original_bytes=encoded(model::program(binding::restored(ref same)));let copied_bytes=encoded(ref unchanged);assert(bytes::equal(wire::content(ref original_bytes),wire::content(ref copied_bytes)));
    let selected=entry(old,ref original);let bundle=switch(emitter::partition_entry(p,selected,128)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return;}};assert(emitter::project_selected_frames(ref bundle)>1);packet(ref bundle);
}
