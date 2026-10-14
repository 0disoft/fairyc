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
import project_cache_disk as disk;
import project_cache_lookup as lookup;
import windows_rooted_file as rooted;

// A candidate is reusable only after fresh complete source admission and exact
// current v4/v0 key preimages. No cached hashes or partial result confer authority.
pub struct Problem{u32 phase;u32 code;core::Problem compilation;}
fn Problem fail(u32 phase,u32 code){return Problem{.phase,.code,.compilation=core::empty()};}

// Each cache operation keeps its own bounded histories; failed generations do
// not reset them. Normal cold compilation uses its separate existing budgets.
pub struct DiskWork{usize lookup;usize scan;usize io;usize projection;usize source;usize artifact;usize link;usize key;}
pub fn DiskWork empty_disk_work(){return DiskWork{.lookup=0,.scan=0,.io=0,.projection=0,.source=0,.artifact=0,.link=0,.key=0};}
// 1 routing, 2 disk lookup, 3 verified admission, 4 encoding, 5 publication.
pub struct DiskProblem{u32 phase;u32 code;Problem artifact;disk::Problem storage;}
fn DiskProblem disk_failure(u32 phase,u32 code){return DiskProblem{.phase,.code,.artifact=fail(0,0),.storage=disk::Problem{.kind=0,.stage=0,.os_code=0,.completed=0,.observed=empty_metadata()}};}
fn rooted::Metadata empty_metadata(){return rooted::Metadata{.identity=0,.modified=0,.size=0,.attributes=0};}
fn DiskProblem storage_failure(u32 phase,disk::Problem e){var p=disk_failure(phase,e.kind);p.storage=e;return p;}
fn DiskProblem artifact_failure(u32 phase,Problem e){var p=disk_failure(phase,e.code);p.artifact=e;return p;}
fn bool temporary_failure(Problem p){return p.code==10 || p.code==11 || p.code==1002 || p.code==1003;}
fn Result<core::Complete,Problem> restore_disk_candidate(ref u8[] raw,ref visibility::Resolved loaded,ref environment::Environment env,ref mut DiskWork work){
    var usize projection=work.projection;var usize source=work.source;var usize artifact=work.artifact;var usize link=work.link;var usize key=work.key;
    let result=restore(raw,loaded,env,ref mut projection,ref mut source,ref mut artifact,ref mut link,ref mut key);
    work.projection=projection;work.source=source;work.artifact=artifact;work.link=link;work.key=key;return result;
}
@effects(filesystem) pub fn Result<core::Complete,DiskProblem> from_disk(ref disk::Store store,ref visibility::Resolved loaded,ref environment::Environment env,ref mut DiskWork work){
    var usize lookup_work=work.lookup;let route_result=lookup::collect(loaded,env,ref mut lookup_work);work.lookup=lookup_work;
    let route=switch(route_result){case .ok(p):{p}case .err(code):{return .err(disk_failure(1,code));}};
    var usize scan=work.scan;let cursor_result=disk::candidates(store,lookup::digest(ref route),ref mut scan);work.scan=scan;
    let cursor=switch(cursor_result){case .ok(p):{p}case .err(e):{return .err(storage_failure(2,e));}};var usize index=disk::candidate_count(ref cursor);var Problem last=fail(0,0);
    while(index>0){var usize io=work.io;let selected=disk::next(store,ref cursor,ref mut index,ref mut io);work.io=io;
        let candidate=switch(selected){case .ok(p):{p}case .err(e):{if(e.kind==2 && last.phase!=0){return .err(artifact_failure(3,last));}return .err(storage_failure(2,e));}};
        switch(restore_disk_candidate(disk::candidate_bytes(ref candidate),loaded,env,work)){
            case .ok(p):{return .ok(p);}
            case .err(e):{if(temporary_failure(e)){return .err(artifact_failure(3,e));}last=e;if(work.io>=disk::work_limit()){return .err(disk_failure(2,9));}work.io+=1;discard disk::reject(store,ref candidate);}
        };
    }
    return if(last.phase!=0){.err(artifact_failure(3,last))}else{.err(disk_failure(2,2))};
}
@effects(filesystem,clock) pub fn Result<disk::Stats,DiskProblem> to_disk(ref disk::Store store,ref visibility::Resolved loaded,ref core::Complete complete,ref environment::Environment env,ref mut DiskWork work){
    var usize lookup_work=work.lookup;let route_result=lookup::collect(loaded,env,ref mut lookup_work);work.lookup=lookup_work;
    let route=switch(route_result){case .ok(p):{p}case .err(code):{return .err(disk_failure(1,code));}};
    var usize key=work.key;var usize artifact=work.artifact;let encoded=encode(loaded,complete,env,ref mut key,ref mut artifact);work.key=key;work.artifact=artifact;
    let data=switch(encoded){case .ok(p):{p}case .err(e):{return .err(artifact_failure(4,e));}};
    var usize io=work.io;var usize scan=work.scan;let result=disk::put(store,lookup::digest(ref route),payload::content(ref data),ref mut io,ref mut scan);work.io=io;work.scan=scan;
    return switch(result){case .ok(p):{.ok(p)}case .err(e):{.err(storage_failure(5,e))}};
}
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
