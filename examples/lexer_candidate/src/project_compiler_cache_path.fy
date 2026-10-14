module project_compiler_cache_path;
import project_compiler_cached as cached;
import project_compiler_core as core;
import project_compiler_input as input;
import module_public_type_text as coordinates;
import project_module_visibility as visibility;
import project_cache_artifact as artifacts;
import project_cache_disk as disk;
import project_cache_environment as environment;
import generic_signature_diagnostics as diagnostics;
import std::vec as v;

// Only explicitly selected private commands use this bounded policy. Product
// default cache location, enablement and measured budgets remain separate.
pub fn disk::Limits command_limits(){return disk::Limits{.entries=64,.bytes=1073741824,.record=268435456,.temporary_age=6000000000};}
// Inputs and coordinates are derived by the command from this same loaded owner.
// Cache initialization errors are disposable misses, never source diagnostics.
@effects(filesystem,environment,clock)
pub fn Result<core::Complete,core::Problem> build(ref u8[] path,disk::Limits policy,ref visibility::Resolved loaded,ref input::Input sources,ref coordinates::Coordinates table,ref mut v::Vec<diagnostics::Reports> reports,ref mut usize source_work,ref mut usize link_work,ref mut cached::Observation observed){
    *observed=cached::empty();observed.path=1;
    if(path.len==0){return core::build_complete_reporting(sources,table,reports,source_work,link_work);}
    var usize environment_work=0;
    let env=switch(environment::current(ref mut environment_work)){
        case .ok(p):{p}
        case .err(e):{observed.lookup_phase=6;observed.lookup_code=e.code;return core::build_complete_reporting(sources,table,reports,source_work,link_work);}
    };
    let store=switch(disk::open(path,policy)){
        case .ok(p):{p}
        case .err(e):{observed.lookup_phase=7;observed.lookup_code=e.kind;return core::build_complete_reporting(sources,table,reports,source_work,link_work);}
    };
    var work=artifacts::empty_disk_work();
    return cached::build(ref store,ref env,loaded,sources,table,reports,ref mut work,source_work,link_work,observed);
}
