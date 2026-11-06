module project_state_graph_disk_cache;
import project_state_graph_cache_lookup as lookup;
import project_state_graph_cache_artifact as artifact;
import project_cache_disk as disk;
import project_cache_environment as environment;
import project_cache_key as compatibility;
import project_compiler_core as core;
import module_public_type_text as coordinates;
import candidate_artifact_wire as wire;

// Routing, semantic verification and encoding each retain their 16MiB history;
// the existing store retains its 1GiB I/O and 16MiB directory histories.
pub struct Work{usize lookup;usize scan;usize io;usize verify;usize encode;}
pub fn Work empty_work(){return Work{.lookup=0,.scan=0,.io=0,.verify=0,.encode=0};}
// 1 routing, 2 disk lookup, 3 current-source/key admission, 4 encoding, 5 store.
pub struct Problem{u32 phase;u32 code;disk::Problem storage;}
fn Problem failure(u32 phase,u32 code){return Problem{.phase,.code,.storage=disk::empty_problem()};}
fn Problem storage_failure(u32 phase,disk::Problem error){return Problem{.phase,.code=error.kind,.storage=error};}
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
pub struct Observation{u32 path;bool stored;u32 lookup_phase;u32 lookup_code;u32 store_phase;u32 store_code;}
pub fn Observation empty(){return Observation{.path=0,.stored=false,.lookup_phase=0,.lookup_code=0,.store_phase=0,.store_code=0};}

// Candidate files remain untrusted. Invalid/incompatible semantic generations
// are skipped without deleting another owner's bytes. Work failure stops this
// attempt; it never restarts a ledger to try the next generation.
@effects(filesystem)
pub fn Result<core::Complete,Problem> from_disk_snapshot(ref disk::Store store,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref compatibility::Environment env,ref u8[] standard_library,ref mut Work work){
    var usize route_work=work.lookup;let selected=lookup::compute(actual,table,owner,env,standard_library,ref mut route_work);work.lookup=route_work;
    let route=switch(selected){case .ok(p):{p}case .err(code):{return .err(failure(1,code));}};
    if(work.verify>16777216){return .err(failure(3,1002));}if(work.io>disk::work_limit()){return .err(failure(2,9));}
    var usize scan=work.scan;let inventory=disk::candidates(store,lookup::value(ref route),ref mut scan);work.scan=scan;
    let cursor=switch(inventory){case .ok(p):{p}case .err(e):{return .err(storage_failure(2,e));}};
    var usize index=disk::candidate_count(ref cursor);var u32 last=0;
    while(index>0){
        var usize io=work.io;let next=disk::next(store,ref cursor,ref mut index,ref mut io);work.io=io;
        let candidate=switch(next){case .ok(p):{p}case .err(e):{if(e.kind==2 && last!=0){return .err(failure(3,last));}return .err(storage_failure(2,e));}};
        var usize verified=work.verify;let restored=artifact::restore_snapshot(disk::candidate_bytes(ref candidate),actual,table,owner,env,standard_library,ref mut verified);work.verify=verified;
        switch(restored){case .ok(p):{return .ok(p);}case .err(code):{if(code==1002 || code==1003){return .err(failure(3,code));}last=code;}};
    }
    return if(last==0){.err(failure(2,2))}else{.err(failure(3,last))};
}

// The publication route comes only from the same Complete being encoded.
// Callers cannot pair a valid Complete with another source/coordinate's route.
@effects(filesystem,clock)
pub fn Result<disk::Stats,Problem> to_disk_snapshot(ref disk::Store store,ref core::Complete complete,ref compatibility::Environment env,ref u8[] standard_library,ref mut Work work){
    // Publication's final sweep needs scan work. An already exhausted ledger
    // must fail before immutable bytes are published, not during that sweep.
    if(work.scan>=16777216 || work.io>=disk::work_limit()){return .err(failure(5,9));}
    var usize route_work=work.lookup;let selected=lookup::for_complete(complete,env,standard_library,ref mut route_work);work.lookup=route_work;
    let route=switch(selected){case .ok(p):{p}case .err(code):{return .err(failure(1,code));}};
    var usize encoded=work.encode;let saved=artifact::encode_snapshot(complete,env,standard_library,ref mut encoded);work.encode=encoded;
    let packet=switch(saved){case .ok(p):{p}case .err(code):{return .err(failure(4,code));}};
    var usize io=work.io;var usize scan=work.scan;let stored=disk::put(store,lookup::value(ref route),wire::content(ref packet),ref mut io,ref mut scan);work.io=io;work.scan=scan;
    return switch(stored){case .ok(p):{.ok(p)}case .err(e):{.err(storage_failure(5,e))}};
}

// Cache failures are disposable misses. Cold source checking uses its separate
// caller-owned budget and preserves the original Core Problem. Nothing is
// published before all source, graph, coordinate, body and interfaces succeed.
@effects(filesystem,clock)
pub fn Result<core::Complete,core::Problem> build_snapshot(ref disk::Store store,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref compatibility::Environment env,ref u8[] standard_library,ref mut Work work,ref mut usize source_work,ref mut Observation observed){
    *observed=empty();observed.path=1;
    switch(from_disk_snapshot(store,actual,table,owner,env,standard_library,work)){
        case .ok(p):{observed.path=2;return .ok(p);}
        case .err(e):{observed.lookup_phase=e.phase;observed.lookup_code=e.code;}
    };
    let complete=switch(core::build_complete_state_graph(actual,table,owner,source_work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    // Missing or semantically incompatible generations may be replaced by a
    // fresh result. Inventory/I/O/work failures disable writes for this attempt.
    if(!(observed.lookup_phase==2 && observed.lookup_code==2 || observed.lookup_phase==3 && observed.lookup_code==1001)){return .ok(complete);}
    switch(to_disk_snapshot(store,ref complete,env,standard_library,work)){
        case .ok(_):{observed.stored=true;}
        case .err(e):{observed.store_phase=e.phase;observed.store_code=e.code;}
    };
    return .ok(complete);
}

// Production adapters use the sealed current compiler/profile owner. Snapshot
// factories above are explicit compatibility inputs, not runtime authority.
@effects(filesystem)
pub fn Result<core::Complete,Problem> from_disk(ref disk::Store store,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref environment::Environment env,ref mut Work work){
    var usize copied=work.lookup;let selected=environment::key_environment(env,ref mut copied);work.lookup=copied;
    let snapshot=switch(selected){case .ok(p):{p}case .err(code):{return .err(failure(1,mapped(code)));}};
    return from_disk_snapshot(store,actual,table,owner,ref snapshot,environment::standard_library(env),work);
}
@effects(filesystem,clock)
pub fn Result<disk::Stats,Problem> to_disk(ref disk::Store store,ref core::Complete complete,ref environment::Environment env,ref mut Work work){
    var usize copied=work.lookup;let selected=environment::key_environment(env,ref mut copied);work.lookup=copied;
    let snapshot=switch(selected){case .ok(p):{p}case .err(code):{return .err(failure(1,mapped(code)));}};
    return to_disk_snapshot(store,complete,ref snapshot,environment::standard_library(env),work);
}
@effects(filesystem,clock)
pub fn Result<core::Complete,core::Problem> build(ref disk::Store store,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref environment::Environment env,ref mut Work work,ref mut usize source_work,ref mut Observation observed){
    var usize copied=work.lookup;let selected=environment::key_environment(env,ref mut copied);work.lookup=copied;
    let snapshot=switch(selected){case .ok(p):{p}case .err(code):{*observed=empty();observed.path=1;observed.lookup_phase=1;observed.lookup_code=mapped(code);return core::build_complete_state_graph(actual,table,owner,source_work);}};
    return build_snapshot(store,actual,table,owner,ref snapshot,environment::standard_library(env),work,source_work,observed);
}
