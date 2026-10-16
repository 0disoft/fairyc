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
import generic_signature_diagnostics as diagnostics;
import project_compiler_cached as cached;
import project_compiler_cache_path as cache_path;
import project_cache_disk as disk;
import std::vec as v;
import std::bytes as bytes;
import syntax as syntax;

pub struct Stats{usize instances;usize files;usize functions;u32 lock;}
// 1 load/lock, 2 compiler-input/coordinate projection, 3 semantic/IR compilation,
// 4 error publication allocation. Each failure retains its original owner.
pub resource Error{u32 phase;v::Vec<loader::Error> loads;v::Vec<loader::Loaded> projections;v::Vec<input::Input> typed;input::Problem projection;compiler::Problem compilation;v::Vec<diagnostics::Reports> reports;cleanup{}}
fn input::Problem no_projection(){return input::Problem{.code=0,.reason=0,.owner=0,.module_index=0,.declaration=0};}
fn compiler::Problem no_compilation(){return compiler::empty();}
pub fn Error load_failure(loader::Error e){var v::Vec<loader::Error> loads=v::empty<loader::Error>();let ok=v::push<loader::Error>(ref mut loads,e).is_ok;return Error{.phase=if(ok){1}else{4},.loads,.projections=v::empty<loader::Loaded>(),.typed=v::empty<input::Input>(),.projection=no_projection(),.compilation=no_compilation(),.reports=v::empty<diagnostics::Reports>()};}
pub fn Error projection_failure(loader::Loaded workspace,input::Problem problem){var v::Vec<loader::Loaded> projections=v::empty<loader::Loaded>();let ok=v::push<loader::Loaded>(ref mut projections,workspace).is_ok;return Error{.phase=if(ok){2}else{4},.loads=v::empty<loader::Error>(),.projections,.typed=v::empty<input::Input>(),.projection=problem,.compilation=no_compilation(),.reports=v::empty<diagnostics::Reports>()};}
pub fn Error compile_failure(loader::Loaded workspace,input::Input projected,compiler::Problem problem){return compile_failure_reports(workspace,projected,problem,v::empty<diagnostics::Reports>());}
pub fn Error compile_failure_reports(loader::Loaded workspace,input::Input projected,compiler::Problem problem,v::Vec<diagnostics::Reports> reports){var v::Vec<input::Input> typed=v::empty<input::Input>();var v::Vec<loader::Loaded> projections=v::empty<loader::Loaded>();let source_ok=v::push<input::Input>(ref mut typed,projected).is_ok;let workspace_ok=v::push<loader::Loaded>(ref mut projections,workspace).is_ok;return Error{.phase=if(source_ok && workspace_ok){3}else{4},.loads=v::empty<loader::Error>(),.projections,.typed,.projection=no_projection(),.compilation=problem,.reports};}
pub fn u32 phase(ref Error e){return e.phase;}
pub fn ref loader::Error load_error(ref Error e){assert(e.phase==1);return v::at<loader::Error>(ref e.loads,0);}
pub fn ref loader::Loaded projection_workspace(ref Error e){assert(e.phase==2);return v::at<loader::Loaded>(ref e.projections,0);}
pub fn input::Problem projection_error(ref Error e){assert(e.phase==2);return e.projection;}
pub fn ref input::Input compiler_input(ref Error e){assert(e.phase==3);return v::at<input::Input>(ref e.typed,0);}
pub fn ref loader::Loaded compiler_workspace(ref Error e){assert(e.phase==3);return v::at<loader::Loaded>(ref e.projections,0);}
pub fn usize report_count(ref Error e){return v::len<diagnostics::Reports>(ref e.reports);}
pub fn ref diagnostics::Reports signature_reports(ref Error e){assert(e.phase==3 && report_count(e)==1);return v::at<diagnostics::Reports>(ref e.reports,0);}
pub fn compiler::Problem compiler_error(ref Error e){assert(e.phase==3);return e.compilation;}
// Go retains the private SDK take_slot<T> function. Candidate cleanup MIR
// folds that leaf into storage_take, but an ordinary pop/clear path may also
// materialize the very same specialization. Count the missing function once.
fn bool folded_take(ref ir::Program program,usize frame){
    let glue=ir::cleanup_ir(program);let f=cleanup_ir::frame_at(glue,frame);
    if(f.kind!=2 || f.receiver_type==0){return false;}
    let receiver=cleanup_ir::type_at(glue,f.receiver_type);if(receiver.kind!=13){return false;}
    let body=ir::at(program,frame);var usize i=0;
    while(i<ir::node_count(body)){let e=ir::node_at(body,i).value;
        if(e.kind==6 && e.builtin.kind==103 && e.builtin.element==receiver.first){
            let generic=ir::node(body,e.left);if(generic!=0){let base=ir::node(body,ir::node_at(body,generic-1).value.left);
                if(base!=0 && bytes::equal(ir::node_text(body,base-1),b"take_slot")){return true;}}
        }i+=1;
    }return false;
}
fn bool materialized_take(ref ir::Program program,ref input::Input sources,usize frame){
    let glue=ir::cleanup_ir(program);let f=cleanup_ir::frame_at(glue,frame);
    let element=cleanup_ir::type_at(glue,f.receiver_type).first;let source=input::source(sources,f.module_index);
    var usize declaration=0;var usize i=0;
    while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);
        if(d.kind==3 && bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],b"take_slot")){declaration=i+1;break;}i+=1;
    }
    // The verified folded call belongs to the authenticated SDK Vec source.
    assert(declaration!=0);i=1;
    while(i<=ir::count(program)){let candidate=cleanup_ir::frame_at(glue,i);
        if(candidate.kind==1 && candidate.module_index==f.module_index && candidate.declaration==declaration && candidate.result==element){return true;}i+=1;
    }return false;
}
fn usize functions(ref ir::Program program,ref input::Input sources){
    var usize total=0;let glue=ir::cleanup_ir(program);var usize i=1;
    while(i<=ir::count(program)){let f=cleanup_ir::frame_at(glue,i);let b=ir::at(program,i);
        let root=ir::statement_at(b,ir::statement(b,ir::body(b))-1).value;
        // Go omits empty source cleanup functions; ownership MIR retains them.
        if(f.kind!=2 || root.second!=0 || root.expression!=0){total+=1;}
        if(folded_take(program,i) && !materialized_take(program,sources,i)){total+=1;}
        i+=1;
    }return total;
}
@effects(filesystem)
pub fn Result<Stats,Error> evaluate(ref u8[] root,bool locked){var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,locked,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(e):{return .err(load_failure(e));}};
    let instances=graph::node_count(visibility::workspace_graph(loader::workspace(ref loaded)));let lock=locks::state_code(loader::state(ref loaded));
    let table=switch(coordinates::collect(visibility::workspace_graph(loader::workspace(ref loaded)),ref mut meter)){case .ok(value):{value}case .err(e):{return .err(projection_failure(loaded,input::Problem{.code=e.code,.reason=2,.owner=e.owner,.module_index=0,.declaration=0}));}};
    let projected=switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(value):{value}case .err(e):{return .err(projection_failure(loaded,e));}};
    var usize source_work=0;var usize link_work=0;
    var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();let files=input::count(ref projected);let complete=switch(compiler::build_complete_reporting(ref projected,ref table,ref mut reports,ref mut source_work,ref mut link_work)){case .ok(value):{value}case .err(e):{return .err(compile_failure_reports(loaded,projected,e,reports));}};
    return .ok(Stats{.instances,.files,.functions=functions(identified::ir_program(compiler::checked_program(ref complete)),ref projected),.lock});
}

// Private opt-in path keeps loader/lock/projection error owners and command
// statistics identical. Only Complete selection changes after those gates.
@effects(filesystem,environment,clock)
pub fn Result<Stats,Error> evaluate_cached(ref u8[] root,bool locked,ref u8[] cache_root,disk::Limits policy,ref mut cached::Observation observed){
    *observed=cached::empty();var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,locked,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(load_failure(e));}};
    let instances=graph::node_count(visibility::workspace_graph(loader::workspace(ref loaded)));let lock=locks::state_code(loader::state(ref loaded));
    let table=switch(coordinates::collect(visibility::workspace_graph(loader::workspace(ref loaded)),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(projection_failure(loaded,input::Problem{.code=e.code,.reason=2,.owner=e.owner,.module_index=0,.declaration=0}));}};
    let projected=switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(projection_failure(loaded,e));}};
    var usize source_work=0;var usize link_work=0;var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();let files=input::count(ref projected);
    let complete=switch(cache_path::build(cache_root,policy,loader::workspace(ref loaded),ref projected,ref table,ref mut reports,ref mut source_work,ref mut link_work,observed)){case .ok(p):{p}case .err(e):{return .err(compile_failure_reports(loaded,projected,e,reports));}};
    return .ok(Stats{.instances,.files,.functions=functions(identified::ir_program(compiler::checked_program(ref complete)),ref projected),.lock});
}
