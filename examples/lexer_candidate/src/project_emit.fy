module project_emit;
import project_workspace_lock as loader;
import project_source_set as source_set;
import project_compiler_input as input;
import project_compiler_core as compiler;
import project_compiler_entry as entry;
import project_check as check;
import module_unit_c_emit as c;
import module_project_complete_c as complete_c;
import module_unit_cleanup_mir_model as clean;
import std::vec as v;
import std::bytes as bytes;
import project_lock_policy as locks;
import project_module_visibility as visibility;
import project_instance_graph as instances;
import project_workspace_input as spaces;
import project_workspace_catalog as catalog;
import project_manifest as manifest;
import project_coordinate_graph as coordinates;
import module_project_identified_ir as identified;
import generic_signature_diagnostics as diagnostics;
import project_compiler_cached as cached;
import project_compiler_cache_path as cache_path;
import project_cache_disk as disk;
import project_workspace_graph as graph;
import syntax as syntax;

// Successful emission keeps the publication metadata from this exact load.
// Runs retain only source bytes and display paths, not a second parser/graph owner.
pub resource TrapSource{bytes::Buffer key;bytes::Buffer path;bytes::Buffer source;cleanup{}}
pub resource Plan{c::Bundle bundle;bytes::Buffer name;locks::State lock_state;v::Vec<TrapSource> trap_sources;cleanup{}}
pub fn ref c::Bundle output_bundle(ref Plan plan){return ref plan.bundle;}
pub fn ref u8[] package_name(ref Plan plan){return bytes::view(ref plan.name);}
pub fn locks::State lock_state(ref Plan plan){return plan.lock_state;}
pub fn usize trap_count(ref Plan plan){return v::len<TrapSource>(ref plan.trap_sources);}
pub fn ref u8[] trap_key(ref Plan plan,usize index){return bytes::view(ref v::at<TrapSource>(ref plan.trap_sources,index).key);}
pub fn ref u8[] trap_path(ref Plan plan,usize index){return bytes::view(ref v::at<TrapSource>(ref plan.trap_sources,index).path);}
pub fn ref u8[] trap_bytes(ref Plan plan,usize index){return bytes::view(ref v::at<TrapSource>(ref plan.trap_sources,index).source);}
fn Result<v::Vec<TrapSource>,u32> snapshot(ref loader::Loaded loaded,ref input::Input projected,bool retain){
    var v::Vec<TrapSource> result=v::empty<TrapSource>();if(!retain){return .ok(result);}
    let g=visibility::workspace_graph(loader::workspace(loaded));let local=spaces::local_catalog(instances::input(graph::local_graph(g)));var usize i=0;
    while(i<input::count(projected)){
        let origin=input::origin(projected,i);var bytes::Buffer path=bytes::empty();var bytes::Buffer key=bytes::empty();
        if(!bytes::append_byte(ref mut key,112).is_ok || !bytes::append_usize(ref mut key,origin.owner+1).is_ok || !bytes::append_byte(ref mut key,47).is_ok || !bytes::append(ref mut key,input::file(projected,i)).is_ok){return .err(4);}
        if(graph::is_sdk(g,origin.owner)){if(!bytes::append(ref mut path,b"@sdk/stdlib/").is_ok){return .err(4);}}
        else{let index=graph::node(g,origin.owner).package_index;let package=catalog::path(local,index);if(package.len!=0 && !bytes::equal(package,b".") && (!bytes::append(ref mut path,package).is_ok || !bytes::append_byte(ref mut path,47).is_ok)){return .err(4);}
            let source_root=manifest::package_value(catalog::package(local,index),b"source");if(!bytes::append(ref mut path,source_root).is_ok || !bytes::append_byte(ref mut path,47).is_ok){return .err(4);}}
        if(!bytes::append(ref mut path,input::file(projected,i)).is_ok){return .err(4);}
        let source=switch(bytes::copy(syntax::source_view(input::source(projected,i)))){case .ok(p):{p}case .err(_):{return .err(4);}};
        if(!v::push<TrapSource>(ref mut result,TrapSource{.key,.path,.source}).is_ok){return .err(4);}i+=1;
    }return .ok(result);
}

// 1 frontend, 2 canonical entry, 3 checked C emission, 4 publication allocation.
// Failures retain their source/lock owners until diagnostics are published.
pub resource Error{u32 phase;v::Vec<check::Error> frontend;v::Vec<loader::Loaded> workspaces;v::Vec<input::Input> sources;entry::Problem selection;clean::Problem emission;cleanup{}}
fn Error frontend_failure(check::Error e){var v::Vec<check::Error> errors=v::empty<check::Error>();let ok=v::push<check::Error>(ref mut errors,e).is_ok;return Error{.phase=if(ok){1}else{4},.frontend=errors,.workspaces=v::empty<loader::Loaded>(),.sources=v::empty<input::Input>(),.selection=entry::Problem{.kind=0,.module_index=0,.start=0,.end=0},.emission=clean::Problem{.code=0,.module_index=0,.start=0,.end=0}};}
fn Error output_failure(loader::Loaded loaded,input::Input projected,u32 phase,entry::Problem selection,clean::Problem emission){var v::Vec<loader::Loaded> workspaces=v::empty<loader::Loaded>();var v::Vec<input::Input> sources=v::empty<input::Input>();let loaded_ok=v::push<loader::Loaded>(ref mut workspaces,loaded).is_ok;let source_ok=v::push<input::Input>(ref mut sources,projected).is_ok;return Error{.phase=if(loaded_ok && source_ok){phase}else{4},.frontend=v::empty<check::Error>(),.workspaces,.sources,.selection,.emission};}
pub fn u32 phase(ref Error e){return e.phase;}
pub fn ref check::Error frontend_error(ref Error e){assert(e.phase==1);return v::at<check::Error>(ref e.frontend,0);}
pub fn ref loader::Loaded workspace(ref Error e){assert(e.phase==2 || e.phase==3);return v::at<loader::Loaded>(ref e.workspaces,0);}
pub fn ref input::Input sources(ref Error e){assert(e.phase==2 || e.phase==3);return v::at<input::Input>(ref e.sources,0);}
pub fn entry::Problem entry_error(ref Error e){assert(e.phase==2);return e.selection;}
pub fn clean::Problem emission_error(ref Error e){assert(e.phase==3);return e.emission;}
@effects(filesystem)
pub fn Result<Plan,Error> evaluate(ref u8[] root,bool locked){return evaluate_run(root,locked,false);}
@effects(filesystem)
pub fn Result<Plan,Error> evaluate_run(ref u8[] root,bool locked,bool retain_locations){
    var source_set::Budget budget=source_set::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,locked,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::load_failure(e)));}};
    let table=switch(coordinates::collect(visibility::workspace_graph(loader::workspace(ref loaded)),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::projection_failure(loaded,input::Problem{.code=e.code,.reason=2,.owner=e.owner,.module_index=0,.declaration=0})));}};
    let projected=switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::projection_failure(loaded,e)));}};
    var usize source_work=0;var usize link_work=0;var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();
    let complete=switch(compiler::build_complete_reporting(ref projected,ref table,ref mut reports,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::compile_failure_reports(loaded,projected,e,reports)));}};
    return finish(loaded,projected,complete,retain_locations);
}

fn Result<Plan,Error> finish(loader::Loaded loaded,input::Input projected,compiler::Complete complete,bool retain_locations){
    var usize entry_work=0;let selected=switch(entry::select(ref projected,identified::ir_program(compiler::checked_program(ref complete)),ref mut entry_work)){case .ok(p):{p}case .err(e):{return .err(output_failure(loaded,projected,2,e,clean::Problem{.code=0,.module_index=0,.start=0,.end=0}));}};
    let bundle=switch(complete_c::partition_entry(complete,selected,128)){case .ok(p):{p}case .err(e):{return .err(output_failure(loaded,projected,3,entry::Problem{.kind=0,.module_index=0,.start=0,.end=0},e));}};
    let local=spaces::local_catalog(instances::input(visibility::instance_graph(loader::workspace(ref loaded))));
    let name=switch(bytes::copy(manifest::package_value(catalog::package(local,0),b"name"))){case .ok(p):{p}case .err(_):{return .err(output_failure(loaded,projected,4,entry::Problem{.kind=0,.module_index=0,.start=0,.end=0},clean::Problem{.code=0,.module_index=0,.start=0,.end=0}));}};
    let trap_sources=switch(snapshot(ref loaded,ref projected,retain_locations)){case .ok(p):{p}case .err(_):{return .err(output_failure(loaded,projected,4,entry::Problem{.kind=0,.module_index=0,.start=0,.end=0},clean::Problem{.code=0,.module_index=0,.start=0,.end=0}));}};
    return .ok(Plan{.bundle,.name,.lock_state=loader::state(ref loaded),.trap_sources});
}

// Private opt-in emission uses the same Complete owner and entry/output path.
// Loader/lock/projection failures keep their existing diagnostic owners.
@effects(filesystem,environment,clock)
pub fn Result<Plan,Error> evaluate_cached(ref u8[] root,bool locked,ref u8[] cache_root,disk::Limits policy,ref mut cached::Observation observed){return evaluate_cached_run(root,locked,cache_root,policy,observed,false);}
@effects(filesystem,environment,clock)
pub fn Result<Plan,Error> evaluate_cached_run(ref u8[] root,bool locked,ref u8[] cache_root,disk::Limits policy,ref mut cached::Observation observed,bool retain_locations){
    *observed=cached::empty();var source_set::Budget budget=source_set::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,locked,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::load_failure(e)));}};
    let table=switch(coordinates::collect(visibility::workspace_graph(loader::workspace(ref loaded)),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::projection_failure(loaded,input::Problem{.code=e.code,.reason=2,.owner=e.owner,.module_index=0,.declaration=0})));}};
    let projected=switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::projection_failure(loaded,e)));}};
    var usize source_work=0;var usize link_work=0;var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();
    let complete=switch(cache_path::build(cache_root,policy,loader::workspace(ref loaded),ref projected,ref table,ref mut reports,ref mut source_work,ref mut link_work,observed)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::compile_failure_reports(loaded,projected,e,reports)));}};
    return finish(loaded,projected,complete,retain_locations);
}
