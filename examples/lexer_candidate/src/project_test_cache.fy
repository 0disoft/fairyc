module project_test_cache;
import project_compiler_cached as cached;
import project_compiler_cache_path as policy;
import project_compiler_core as core;
import project_compiler_input as input;
import module_public_type_text as coordinates;
import project_cache_disk as disk;
import project_cache_environment as environment;
import project_cache_unit_store as units;
import generic_signature_diagnostics as diagnostics;
import std::vec as v;

// Tests use project_tests/copy_tests, not the ordinary whole-project projection.
// Reuse only units admitted against these current test sources; rebuild the
// whole linked owner and interfaces, then execute tests anew on every call.
@effects(filesystem,environment,clock)
pub fn Result<core::Complete,core::Problem> build(ref u8[] path,ref input::Input sources,ref coordinates::Coordinates table,ref mut v::Vec<diagnostics::Reports> reports,ref mut usize source_work,ref mut usize link_work,ref mut cached::Observation observed){
    *observed=cached::empty();observed.path=1;var usize environment_work=0;
    let env=switch(environment::current(ref mut environment_work)){
        case .ok(p):{p}
        case .err(e):{observed.lookup_phase=6;observed.lookup_code=e.code;return core::build_complete_reporting(sources,table,reports,source_work,link_work);}
    };
    let store=switch(disk::open(path,policy::command_limits())){
        case .ok(p):{p}
        case .err(e):{observed.lookup_phase=7;observed.lookup_code=e.kind;return core::build_complete_reporting(sources,table,reports,source_work,link_work);}
    };
    var work=units::empty();let result=core::build_complete_mixed(ref store,ref env,sources,table,reports,ref mut work,source_work,link_work);
    observed.reused_units=work.hits;observed.rebuilt_units=work.misses;observed.stored_units=work.stored;observed.stored=work.stored>0;
    if(work.hits>0){observed.path=if(work.misses>0){3}else{4};}
    return result;
}
pub fn ref str status(u32 path,bool stored){
    if(path==4){return "cache: units hit\n";}if(path==3){return "cache: mixed\n";}
    if(path==1 && stored){return "cache: cold stored\n";}if(path==1){return "cache: cold uncached\n";}return "cache: skipped\n";
}
