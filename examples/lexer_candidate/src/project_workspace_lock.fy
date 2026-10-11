module project_workspace_lock;
import project_module_visibility as visibility;
import project_lock_policy as locks;
import project_source_set as sources;
import std::vec as v;

pub resource Loaded{visibility::Resolved workspace;locks::State state;cleanup{}}
// 1 workspace/SDK/import load, 2 lock policy, 3 error allocation. Full source
// type checking follows this load, as in the stable Go CLI.
pub resource Error{u32 phase;v::Vec<visibility::LoadError> workspace;v::Vec<locks::Problem> locks;cleanup{}}
fn Error input_failure(visibility::LoadError error){var v::Vec<visibility::LoadError> errors=v::empty<visibility::LoadError>();let ok=v::push<visibility::LoadError>(ref mut errors,error).is_ok;return Error{.phase=if(ok){1}else{3},.workspace=errors,.locks=v::empty<locks::Problem>()};}
fn Error lock_failure(locks::Problem error){var v::Vec<locks::Problem> errors=v::empty<locks::Problem>();let ok=v::push<locks::Problem>(ref mut errors,error).is_ok;return Error{.phase=if(ok){2}else{3},.workspace=v::empty<visibility::LoadError>(),.locks=errors};}
pub fn u32 phase(ref Error error){return error.phase;}
pub fn ref visibility::Resolved workspace(ref Loaded loaded){return ref loaded.workspace;}
pub fn locks::State state(ref Loaded loaded){return loaded.state;}
pub fn ref visibility::LoadError input_error(ref Error error){assert(error.phase==1 && v::len<visibility::LoadError>(ref error.workspace)==1);return v::at<visibility::LoadError>(ref error.workspace,0);}
pub fn locks::Problem policy_error(ref Error error){assert(error.phase==2 && v::len<locks::Problem>(ref error.locks)==1);return *v::at<locks::Problem>(ref error.locks,0);}
@effects(filesystem)
pub fn Result<Loaded,Error> load(ref u8[] root,bool locked,ref mut sources::Budget budget,ref mut usize meter){return load_embedded(root,false,locked,budget,meter);}
@effects(filesystem)
pub fn Result<Loaded,Error> load_embedded(ref u8[] root,bool candidate,bool locked,ref mut sources::Budget budget,ref mut usize meter){
    var sources::Budget staged=*budget;
    let workspace=switch(visibility::load_embedded(root,candidate,ref mut staged,meter)){case .ok(value):{value}case .err(e):{return .err(input_failure(e));}};
    let state=switch(locks::apply(root,ref workspace,locked,meter)){case .ok(value):{value}case .err(e):{return .err(lock_failure(e));}};
    *budget=staged;return .ok(Loaded{.workspace,.state});
}
