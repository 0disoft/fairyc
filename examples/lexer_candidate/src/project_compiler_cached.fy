module project_compiler_cached;
import project_compiler_core as core;
import project_compiler_input as input;
import module_public_type_text as coordinates;
import project_module_visibility as visibility;
import project_cache_artifact as artifacts;
import project_cache_disk as disk;
import project_cache_environment as environment;
import generic_signature_diagnostics as diagnostics;
import std::vec as v;

// Private observations never change command output or compiler diagnostics.
// 0 unattempted, 1 cold compilation, 2 checked disk hit.
pub struct Observation{u32 path;bool stored;u32 lookup_phase;u32 lookup_code;u32 store_phase;u32 store_code;}
pub fn Observation empty(){return Observation{.path=0,.stored=false,.lookup_phase=0,.lookup_code=0,.store_phase=0,.store_code=0};}

// Cache errors are disposable misses. Cold compilation owns independent source
// and link histories and retains the exact existing error-reporting contract.
// A hit has passed full current-source admission, not just a checksum or key.
@effects(filesystem,clock)
pub fn Result<core::Complete,core::Problem> build(ref disk::Store store,ref environment::Environment env,ref visibility::Resolved loaded,ref input::Input sources,ref coordinates::Coordinates table,ref mut v::Vec<diagnostics::Reports> reports,ref mut artifacts::DiskWork cache_work,ref mut usize source_work,ref mut usize link_work,ref mut Observation observed){
    *observed=empty();
    if(v::len<diagnostics::Reports>(reports)!=0){return core::build_complete_reporting(sources,table,reports,source_work,link_work);}
    switch(artifacts::from_disk(store,loaded,env,cache_work)){
        case .ok(complete):{observed.path=2;return .ok(complete);}
        case .err(e):{observed.lookup_phase=e.phase;observed.lookup_code=e.code;}
    };
    observed.path=1;
    let complete=switch(core::build_complete_reporting(sources,table,reports,source_work,link_work)){
        case .ok(p):{p}case .err(e):{return .err(e);}
    };
    switch(artifacts::to_disk(store,loaded,ref complete,env,cache_work)){
        case .ok(_):{observed.stored=true;}
        case .err(e):{observed.store_phase=e.phase;observed.store_code=e.code;}
    };
    return .ok(complete);
}
