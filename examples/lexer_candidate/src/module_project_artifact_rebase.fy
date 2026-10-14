module module_project_artifact_rebase;
import module_project_artifact_current as current;
import module_project_artifact_model as artifact;
import module_project_artifact_rebase_cleanup as cleanup_builder;
import module_project_artifact_rebase_aux as auxiliary;
import module_project_artifact_rebase_body as body_builder;
import module_unit_body_ir_model as ir;
import module_unit_interfaces as interfaces;
import module_program_artifact_wire as work;
import std::bytes as bytes;
import std::vec as v;
// Keep type/frame/field IDs and byte spans; rebind only source/package indices.
// Fresh full verification still does not confer Complete/cache-hit authority.
pub fn Result<ir::Program,u32> build(ref current::Rebound input,ref mut usize used){if(!work::charge(used,0)){return .err(1002);}let p=artifact::program(current::restored(input));let cleaned=switch(cleanup_builder::build(ir::cleanup_ir(p),input,used)){case .ok(p):{p}case .err(code):{return .err(code);}};let states=switch(auxiliary::states(ir::state_layouts(p),input,used)){case .ok(p):{p}case .err(code):{return .err(code);}};let sdk=switch(auxiliary::sdk_sources(ir::sdk_units(p),input,used)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var v::Vec<ir::Body> bodies=v::empty<ir::Body>();var usize i=1;while(i<=ir::count(p)){let b=switch(body_builder::build(ir::at(p,i),input,used)){case .ok(p):{p}case .err(code):{return .err(code);}};if(!v::push<ir::Body>(ref mut bodies,b).is_ok){return .err(1003);}i+=1;}
    var v::Vec<ir::Layout> layouts=v::empty<ir::Layout>();var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();var v::Vec<usize> parameters=v::empty<usize>();i=0;
    while(i<ir::layout_count(p)){if(!work::charge(used,1)){return .err(1002);}var row=ir::layout_at(p,i);if(row.value.module_index>=current::module_count(input)){return .err(1001);}row.value.module_index=current::module_at(input,row.value.module_index);if(!v::push<ir::Layout>(ref mut layouts,row).is_ok){return .err(1003);}i+=1;}i=0;
    while(i<ir::field_count(p)){if(!work::charge(used,1)){return .err(1002);}var row=ir::field_at(p,i);if(row.module_index>=current::module_count(input)){return .err(1001);}row.module_index=current::module_at(input,row.module_index);if(!v::push<interfaces::Field>(ref mut fields,row).is_ok){return .err(1003);}i+=1;}i=0;
    while(i<ir::callable_count(p)){if(!work::charge(used,1)){return .err(1002);}var row=ir::callable_at(p,i);if(row.module_index>=current::module_count(input)){return .err(1001);}row.module_index=current::module_at(input,row.module_index);if(!v::push<interfaces::Callable>(ref mut callables,row).is_ok){return .err(1003);}i+=1;}i=0;while(i<ir::parameter_count(p)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<usize>(ref mut parameters,ir::parameter_at(p,i)).is_ok){return .err(1003);}i+=1;}
    if(!work::charge(used,ir::name_bytes(p)+1)){return .err(1002);}let names=switch(bytes::copy(ir::names_view(p))){case .ok(p):{p}case .err(_):{return .err(1003);}};var usize measured=0;let result=switch(ir::assemble_artifact(states,sdk,cleaned,bodies,layouts,fields,callables,parameters,names,ref mut measured)){case .ok(p):{p}case .err(e):{if(!work::charge(used,measured)){return .err(1002);}return .err(e.code);}};if(!work::charge(used,measured)){return .err(1002);}return .ok(result);
}
