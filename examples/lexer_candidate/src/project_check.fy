module project_check;
import project_workspace_lock as loader;
import project_lock_policy as locks;
import project_module_visibility as visibility;
import project_workspace_graph as graph;
import project_compiler_input as input;
import project_compiler_core as compiler;
import project_source_set as sources;
import module_unit_body_ir_model as ir;
import module_project_identified_ir as identified;
import project_coordinate_graph as coordinates;
import module_unit_cleanup_mir_model as cleanup_ir;
import std::vec as v;

pub struct Stats{usize instances;usize files;usize functions;u32 lock;}
// 1 load/lock, 2 compiler-input/coordinate projection, 3 semantic/IR compilation,
// 4 error publication allocation. Each failure retains its original owner.
pub resource Error{u32 phase;v::Vec<loader::Error> loads;v::Vec<loader::Loaded> projections;v::Vec<input::Input> typed;input::Problem projection;compiler::Problem compilation;cleanup{}}
fn input::Problem no_projection(){return input::Problem{.code=0,.reason=0,.owner=0,.module_index=0,.declaration=0};}
fn compiler::Problem no_compilation(){return compiler::empty();}
pub fn Error load_failure(loader::Error e){var v::Vec<loader::Error> loads=v::empty<loader::Error>();let ok=v::push<loader::Error>(ref mut loads,e).is_ok;return Error{.phase=if(ok){1}else{4},.loads,.projections=v::empty<loader::Loaded>(),.typed=v::empty<input::Input>(),.projection=no_projection(),.compilation=no_compilation()};}
pub fn Error projection_failure(loader::Loaded workspace,input::Problem problem){var v::Vec<loader::Loaded> projections=v::empty<loader::Loaded>();let ok=v::push<loader::Loaded>(ref mut projections,workspace).is_ok;return Error{.phase=if(ok){2}else{4},.loads=v::empty<loader::Error>(),.projections,.typed=v::empty<input::Input>(),.projection=problem,.compilation=no_compilation()};}
pub fn Error compile_failure(loader::Loaded workspace,input::Input projected,compiler::Problem problem){var v::Vec<input::Input> typed=v::empty<input::Input>();var v::Vec<loader::Loaded> projections=v::empty<loader::Loaded>();let source_ok=v::push<input::Input>(ref mut typed,projected).is_ok;let workspace_ok=v::push<loader::Loaded>(ref mut projections,workspace).is_ok;return Error{.phase=if(source_ok && workspace_ok){3}else{4},.loads=v::empty<loader::Error>(),.projections,.typed,.projection=no_projection(),.compilation=problem};}
pub fn u32 phase(ref Error e){return e.phase;}
pub fn ref loader::Error load_error(ref Error e){assert(e.phase==1);return v::at<loader::Error>(ref e.loads,0);}
pub fn ref loader::Loaded projection_workspace(ref Error e){assert(e.phase==2);return v::at<loader::Loaded>(ref e.projections,0);}
pub fn input::Problem projection_error(ref Error e){assert(e.phase==2);return e.projection;}
pub fn ref input::Input compiler_input(ref Error e){assert(e.phase==3);return v::at<input::Input>(ref e.typed,0);}
pub fn ref loader::Loaded compiler_workspace(ref Error e){assert(e.phase==3);return v::at<loader::Loaded>(ref e.projections,0);}
pub fn compiler::Problem compiler_error(ref Error e){assert(e.phase==3);return e.compilation;}
// SDK Vec destructors live in verified glue rather than ordinary Body records.
// Include each concrete destructor in the same MIR-function statistic as Go.
fn usize functions(ref ir::Program program){var usize total=ir::count(program);let glue=ir::cleanup_ir(program);var usize i=1;while(i<=cleanup_ir::type_count(glue)){if(cleanup_ir::type_at(glue,i).kind==13 && cleanup_ir::glue_at(glue,i).kind==1){total+=1;}i+=1;}return total;}
@effects(filesystem)
pub fn Result<Stats,Error> evaluate(ref u8[] root,bool locked){var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,locked,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(e):{return .err(load_failure(e));}};
    let instances=graph::node_count(visibility::workspace_graph(loader::workspace(ref loaded)));let lock=locks::state_code(loader::state(ref loaded));
    let table=switch(coordinates::collect(visibility::workspace_graph(loader::workspace(ref loaded)),ref mut meter)){case .ok(value):{value}case .err(e):{return .err(projection_failure(loaded,input::Problem{.code=e.code,.reason=2,.owner=e.owner,.module_index=0,.declaration=0}));}};
    let projected=switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(value):{value}case .err(e):{return .err(projection_failure(loaded,e));}};
    var usize source_work=0;var usize link_work=0;
    let files=input::count(ref projected);let complete=switch(compiler::build_complete(ref projected,ref table,ref mut source_work,ref mut link_work)){case .ok(value):{value}case .err(e):{return .err(compile_failure(loaded,projected,e));}};
    return .ok(Stats{.instances,.files,.functions=functions(identified::ir_program(compiler::checked_program(ref complete))),.lock});
}
