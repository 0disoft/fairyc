module project_cache_artifact;
import project_cache_artifact_wire as transport;
import project_cache_keys as keys;
import project_cache_environment as environment;
import project_compiler_core as core;
import project_compiler_input as input;
import project_coordinate_graph as coordinates;
import project_module_visibility as visibility;
import module_project_artifact_encode as encoder;
import module_project_artifact_decode as decoder;
import module_program_artifact_wire as wire;
import candidate_artifact_wire as payload;

// A candidate is reusable only after fresh complete source admission and exact
// current v4/v0 key preimages. No cached hashes or partial result confer authority.
pub struct Problem{u32 phase;u32 code;core::Problem compilation;}
fn Problem fail(u32 phase,u32 code){return Problem{.phase,.code,.compilation=core::empty()};}
// Key derivation retains its 16MiB history; transport/reverification uses the
// independent 1GiB aggregate artifact history. Return no encoded prefix.
pub fn Result<payload::Payload,Problem> encode(ref visibility::Resolved loaded,ref core::Complete complete,ref environment::Environment env,ref mut usize key_work,ref mut usize artifact_work){
    if(!wire::charge(artifact_work,0)){return .err(fail(1,1002));}let current=switch(keys::collect(loaded,complete,env,key_work)){case .ok(p):{p}case .err(code):{return .err(fail(5,code));}};
    let program=switch(encoder::encode(core::checked_program(complete),artifact_work)){case .ok(p):{p}case .err(code):{return .err(fail(3,code));}};
    return switch(transport::encode(ref current,payload::content(ref program),artifact_work)){case .ok(p):{.ok(p)}case .err(code):{.err(fail(1,code))}};
}
// Parse all framing before allocating a decoded owner. Restore uses current
// loader-owned sources; it neither cold-lowers replacement MIR nor trusts keys
// from the envelope. A late key mismatch drops both newly rebuilt members.
pub fn Result<core::Complete,Problem> restore(ref u8[] raw,ref visibility::Resolved loaded,ref environment::Environment env,ref mut usize projection_work,ref mut usize source_work,ref mut usize artifact_work,ref mut usize link_work,ref mut usize key_work){
    let parts=switch(transport::preflight(raw,artifact_work)){case .ok(p):{p}case .err(code):{return .err(fail(1,code));}};
    let table=switch(coordinates::collect(visibility::workspace_graph(loaded),projection_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,if(e.code==6){10}else if(e.code==5){11}else{9}));}};
    let sources=switch(input::project(loaded,projection_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,if(e.code==1){10}else if(e.code==2){11}else{9}));}};
    let saved=switch(decoder::decode(raw[parts.program.start..parts.program.end],artifact_work)){case .ok(p):{p}case .err(code):{return .err(fail(3,code));}};
    let complete=switch(core::restore_complete_sources(saved,input::sources(ref sources),input::owners(ref sources),input::bindings(ref sources),ref table,source_work,artifact_work,link_work)){case .ok(p):{p}case .err(e):{return .err(Problem{.phase=4,.code=e.code,.compilation=e});}};
    let current=switch(keys::collect(loaded,ref complete,env,key_work)){case .ok(p):{p}case .err(code):{return .err(fail(5,code));}};
    let checked=transport::matches(raw[parts.keys.start..parts.keys.end],ref current,artifact_work);if(checked!=0){return .err(fail(6,checked));}
    return .ok(complete);
}
