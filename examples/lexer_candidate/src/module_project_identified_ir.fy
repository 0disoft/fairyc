module module_project_identified_ir;
import candidate_limits as limits;
import module_project_artifact_admission as admitted;
import module_project_link as linking;
import module_project_identified_compile as units;
import module_signatures as signatures;
import std::vec as v;
import module_project_identities as identities;
import module_identity_artifact_model as snapshot;
import project_cache_encoding as snapshot_work;
import module_source_specializations as selection;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_callable_index as callables;
import module_unit_c_frame_index as targets;
import generic_package_bindings as bindings;
import module_source_provenance as provenance;
import module_public_type_text as coordinates;
import syntax as syntax;
import std::bytes as bytes;

pub resource Program{v::Vec<ir::Program> cold;v::Vec<admitted::Admitted> restored;identities::Table identities;targets::Index targets;bindings::Bindings provenance;cleanup{}}
pub fn ref bindings::Bindings source_provenance(ref Program p){return ref p.provenance;}
pub fn u32 current(ref Program p,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize work){return provenance::current(ref p.provenance,sources,metadata,table,work);}
pub fn ref ir::Program ir_program(ref Program p){return if(v::len<ir::Program>(ref p.cold)!=0){v::at<ir::Program>(ref p.cold,0)}else{admitted::program(v::at<admitted::Admitted>(ref p.restored,0))};}
pub fn ref identities::Table source_identities(ref Program p){return ref p.identities;}
pub fn usize frame_count(ref Program p){return identities::frame_count(ref p.identities);}
pub fn selection::Row frame_at(ref Program p,usize id){return identities::frame_at(ref p.identities,id);}
pub fn selection::Row type_at(ref Program p,usize id){return identities::type_at(ref p.identities,id);}
pub fn ref u8[] identity(ref Program p,selection::Row row){return identities::identity(ref p.identities,row);}
// The owner-facing comparison belongs above the detached snapshot model.
// Source-derived g0 bytes match independently of producer-local planner handles.
pub fn u32 matches_snapshot(ref Program current,ref snapshot::Snapshot saved,ref mut usize work){let context=ir::cleanup_ir(ir_program(current));let checked=snapshot::check(saved,context,work);if(checked!=0){return checked;}var usize i=1;
    while(i<=snapshot::type_count(saved)+snapshot::frame_count(saved)){if(!snapshot_work::charge(work,1)){return 1002;}let a=if(i<=snapshot::type_count(saved)){snapshot::type_at(saved,i)}else{snapshot::frame_at(saved,i-snapshot::type_count(saved))};let b=if(i<=snapshot::type_count(saved)){type_at(current,i)}else{frame_at(current,i-snapshot::type_count(saved))};let id=identity(current,b);if(!snapshot_work::charge(work,id.len+1)){return 1002;}if((a.node!=0)!=(b.node!=0) || !bytes::equal(snapshot::identity(saved,a),id)){return 1001;}i+=1;}return 0;
}
fn clean::Problem fail(u32 code){return clean::Problem{.code,.module_index=0,.start=0,.end=0};}
// The checked program and its identity table are derived from the same owners;
// callers cannot supply an unrelated detached table to this factory.
pub fn Result<Program,clean::Problem> build(ref signatures::Program metadata,ref units::Plan input,ref mut usize work){
    if(!limits::project_charge(work,0)){return .err(fail(1002));}let ordinary=v::empty<ir::Program>();let ready=switch(linking::prepare_inputs(metadata,ref ordinary,input)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let program=switch(linking::program_inputs(metadata,ref ordinary,input,ref ready,work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let table=switch(identities::build(input,linking::type_plan(ref ready),linking::frame_plan(ref ready),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    if(identities::frame_count(ref table)!=ir::count(ref program) || identities::type_count(ref table)!=clean::type_count(ir::cleanup_ir(ref program))){return .err(fail(1001));}var usize measured=0;let contracts=switch(callables::build(ref program,ref mut measured)){case .ok(p):{p}case .err(code):{discard limits::project_charge(work,measured);return .err(fail(code));}};let index=switch(targets::build(ref program,ref contracts,ref mut measured)){case .ok(p):{p}case .err(code):{discard limits::project_charge(work,measured);return .err(fail(code));}};if(!limits::project_charge(work,measured)){return .err(fail(1002));}var usize provenance_work=0;let stamps=switch(bindings::copy(units::provenance(input),ref mut provenance_work)){case .ok(p):{p}case .err(code):{discard limits::project_charge(work,provenance_work);return .err(fail(if(code==10){1002}else if(code==11){1003}else{1001}));}};if(!limits::project_charge(work,provenance_work)){return .err(fail(1002));}if(!limits::project_charge(work,1)){return .err(fail(1002));}var v::Vec<ir::Program> cold=v::empty<ir::Program>();if(!v::push<ir::Program>(ref mut cold,program).is_ok){return .err(fail(1003));}return .ok(Program{.cold,.restored=v::empty<admitted::Admitted>(),.identities=table,.targets=index,.provenance=stamps});
}
pub fn Result<usize,u32> call_frame(ref Program p,usize frame,usize at,ref mut usize work){assert(frame>0 && frame<=frame_count(p));let body=ir::at(ir_program(p),frame);assert(at<ir::call_count(body));let c=ir::call_at(body,at);if(!c.checked){return .err(1001);}if(c.kind!=1 && c.kind!=2){return .ok(0);}var usize measured=0;var u32 code=0;let target=targets::find(ir_program(p),ref p.targets,c.target_module,c.target_declaration,c.instance,c.kind==2,ref mut measured,ref mut code);if(!limits::project_charge(work,measured)){return .err(1002);}if(code!=0){return .err(code);}if(c.kind==2 && (target==0 || frame_at(p,target).node==0)){return .err(1001);}return .ok(target);}
// Retain the entire admitted owner; resource fields cannot be partially moved.
pub fn Result<Program,clean::Problem> from_admitted(admitted::Admitted restored,ref mut usize work){
    if(!limits::project_charge(work,0)){return .err(fail(1002));}let program=admitted::program(ref restored);let table=switch(identities::from_admitted(ref restored,work)){case .ok(p):{p}case .err(e):{return .err(e);}};var usize measured=0;
    let contracts=switch(callables::build(program,ref mut measured)){case .ok(p):{p}case .err(code):{discard limits::project_charge(work,measured);return .err(fail(code));}};let index=switch(targets::build(program,ref contracts,ref mut measured)){case .ok(p):{p}case .err(code):{discard limits::project_charge(work,measured);return .err(fail(code));}};if(!limits::project_charge(work,measured)){return .err(fail(1002));}
    measured=0;let stamps=switch(bindings::copy(admitted::source_provenance(ref restored),ref mut measured)){case .ok(p):{p}case .err(code):{if(!limits::project_charge(work,measured)){return .err(fail(1002));}return .err(fail(if(code==10){1002}else if(code==11){1003}else{1001}));}};if(!limits::project_charge(work,measured+1)){return .err(fail(1002));}
    var v::Vec<admitted::Admitted> owners=v::empty<admitted::Admitted>();if(!v::push<admitted::Admitted>(ref mut owners,restored).is_ok){return .err(fail(1003));}return .ok(Program{.cold=v::empty<ir::Program>(),.restored=owners,.identities=table,.targets=index,.provenance=stamps});
}
