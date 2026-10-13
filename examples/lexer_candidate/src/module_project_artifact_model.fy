module module_project_artifact_model;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_sdk_sources as sdk;
import sdk_scopes as scopes;
import module_source_artifact_model as provenance;
import module_identity_artifact_model as identities;
import module_state_layout as state;
import module_program_artifact_wire as wire;
import sha256 as sha;
import std::bytes as bytes;

// Own every decoded component together. Source rebind and semantic admission
// still belong to the current-input owner factory, not this transport value.
pub resource Restored{ir::Program program;provenance::Snapshot provenance;identities::Snapshot identities;cleanup{}}
pub fn ref ir::Program program(ref Restored p){return ref p.program;}
pub fn ref provenance::Snapshot source_provenance(ref Restored p){return ref p.provenance;}
pub fn ref identities::Snapshot source_identities(ref Restored p){return ref p.identities;}
pub fn u32 check(ref ir::Program p,ref provenance::Snapshot stamps,ref identities::Snapshot ids,ref mut usize work){
    var usize measured=0;let sources=provenance::check(stamps,ref mut measured);if(!wire::charge(work,measured)){return 1002;}if(sources!=0){return sources;}measured=0;let named=identities::check(ids,ir::cleanup_ir(p),ref mut measured);if(!wire::charge(work,measured)){return 1002;}if(named!=0){return named;}
    let count=provenance::count(stamps);let owned=ir::sdk_units(p);if(count==0 || count!=sdk::module_count(owned)){return 1001;}var usize i=1;
    while(i<=ir::count(p)){if(!wire::charge(work,1)){return 1002;}let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.module_index>=count || ir::source_bytes(ir::at(p,i))!=provenance::module_at(stamps,f.module_index).source_size){return 1001;}i+=1;}
    i=1;while(i<=clean::type_count(ir::cleanup_ir(p))){if(!wire::charge(work,1)){return 1002;}let t=clean::type_at(ir::cleanup_ir(p),i);if((t.kind==2 || t.kind==3 || t.kind==4 || t.kind==13) && t.module_index>=count){return 1001;}i+=1;}
    i=0;while(i<ir::field_count(p)){if(!wire::charge(work,1)){return 1002;}let row=ir::field_at(p,i);if(row.module_index>=count || row.start>row.end || row.end>provenance::module_at(stamps,row.module_index).source_size){return 1001;}i+=1;}
    i=0;while(i<ir::callable_count(p)){if(!wire::charge(work,1)){return 1002;}if(ir::callable_at(p,i).module_index>=count){return 1001;}i+=1;}
    i=0;while(i<sdk::count(owned)){if(!wire::charge(work,1)){return 1002;}let unit=sdk::at(owned,i);if(unit.module_index>=count){return 1001;}let source=provenance::module_at(stamps,unit.module_index);let raw=sdk::source_at(owned,i);if(source.source_size!=raw.len || !bytes::equal(provenance::text(stamps,source.path),scopes::name(unit.sdk_id))){return 1001;}if(!wire::charge(work,raw.len+1)){return 1002;}let digest=sha::digest(raw);var usize j=0;while(j<8){if(source.digest[j]!=digest[j]){return 1001;}j+=1;}i+=1;}
    let states=ir::state_layouts(p);i=0;while(i<state::count(states)){if(!wire::charge(work,1)){return 1002;}let row=state::at(states,i);if(row.module_index>=count){return 1001;}let source=provenance::module_at(stamps,row.module_index);if(row.owner!=source.owner || row.end>source.source_size){return 1001;}i+=1;}return 0;
}
pub fn Result<Restored,u32> assemble(ir::Program program,provenance::Snapshot provenance,identities::Snapshot identities,ref mut usize work){let checked=check(ref program,ref provenance,ref identities,work);if(checked!=0){return .err(checked);}return .ok(Restored{.program,.provenance,.identities});}
