module project_emit;
import project_workspace_lock as loader;
import project_source_set as source_set;
import project_compiler_input as input;
import project_compiler_core as compiler;
import project_compiler_entry as entry;
import project_check as check;
import module_unit_c_emit as c;
import module_unit_cleanup_mir_model as clean;
import std::vec as v;

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
pub fn Result<c::Bundle,Error> evaluate(ref u8[] root,bool locked){
    var source_set::Budget budget=source_set::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,locked,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::load_failure(e)));}};
    let projected=switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::projection_failure(loaded,e)));}};
    let program=switch(compiler::build(ref projected)){case .ok(p):{p}case .err(e):{return .err(frontend_failure(check::compile_failure(loaded,projected,e)));}};
    var usize entry_work=0;let selected=switch(entry::select(ref projected,ref program,ref mut entry_work)){case .ok(p):{p}case .err(e):{return .err(output_failure(loaded,projected,2,e,clean::Problem{.code=0,.module_index=0,.start=0,.end=0}));}};
    let bundle=switch(c::partition_entry(program,selected,128)){case .ok(p):{p}case .err(e):{return .err(output_failure(loaded,projected,3,entry::Problem{.kind=0,.module_index=0,.start=0,.end=0},e));}};return .ok(bundle);
}
