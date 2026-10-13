module module_project_identified_ir;
import candidate_limits as limits;
import module_project_link as linking;
import module_project_identified_compile as units;
import module_signatures as signatures;
import std::vec as v;
import module_project_identities as identities;
import module_source_specializations as selection;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_callable_index as callables;
import module_unit_c_frame_index as targets;

pub resource Program{ir::Program program;identities::Table identities;targets::Index targets;cleanup{}}
pub fn ref ir::Program ir_program(ref Program p){return ref p.program;}
pub fn usize frame_count(ref Program p){return identities::frame_count(ref p.identities);}
pub fn selection::Row frame_at(ref Program p,usize id){return identities::frame_at(ref p.identities,id);}
pub fn selection::Row type_at(ref Program p,usize id){return identities::type_at(ref p.identities,id);}
pub fn ref u8[] identity(ref Program p,selection::Row row){return identities::identity(ref p.identities,row);}
fn clean::Problem fail(u32 code){return clean::Problem{.code,.module_index=0,.start=0,.end=0};}
// The checked program and its identity table are derived from the same owners;
// callers cannot supply an unrelated detached table to this factory.
pub fn Result<Program,clean::Problem> build(ref signatures::Program metadata,ref units::Plan input,ref mut usize work){
    if(!limits::project_charge(work,0)){return .err(fail(1002));}let ordinary=v::empty<ir::Program>();let ready=switch(linking::prepare_inputs(metadata,ref ordinary,input)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let program=switch(linking::program_inputs(metadata,ref ordinary,input,ref ready,work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let table=switch(identities::build(input,linking::type_plan(ref ready),linking::frame_plan(ref ready),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    if(identities::frame_count(ref table)!=ir::count(ref program) || identities::type_count(ref table)!=clean::type_count(ir::cleanup_ir(ref program))){return .err(fail(1001));}var usize measured=0;let contracts=switch(callables::build(ref program,ref mut measured)){case .ok(p):{p}case .err(code):{discard limits::project_charge(work,measured);return .err(fail(code));}};let index=switch(targets::build(ref program,ref contracts,ref mut measured)){case .ok(p):{p}case .err(code):{discard limits::project_charge(work,measured);return .err(fail(code));}};if(!limits::project_charge(work,measured)){return .err(fail(1002));}return .ok(Program{.program,.identities=table,.targets=index});
}
pub fn Result<usize,u32> call_frame(ref Program p,usize frame,usize at,ref mut usize work){assert(frame>0 && frame<=frame_count(p));let body=ir::at(ref p.program,frame);assert(at<ir::call_count(body));let c=ir::call_at(body,at);if(!c.checked){return .err(1001);}if(c.kind!=1 && c.kind!=2){return .ok(0);}var usize measured=0;var u32 code=0;let target=targets::find(ref p.program,ref p.targets,c.target_module,c.target_declaration,c.instance,c.kind==2,ref mut measured,ref mut code);if(!limits::project_charge(work,measured)){return .err(1002);}if(code!=0){return .err(code);}if(c.kind==2 && (target==0 || frame_at(p,target).node==0)){return .err(1001);}return .ok(target);}
