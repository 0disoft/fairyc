module module_project_artifact_rebase_body;
import module_project_artifact_current as current;
import module_unit_body_ir_model as ir;
import module_unit_value_model as m;
import module_unit_region_model as regions;
import module_program_artifact_wire as work;
import std::bytes as bytes;
import std::vec as v;
pub fn Result<ir::Body,u32> build(ref ir::Body b,ref current::Rebound mapping,ref mut usize used){var v::Vec<ir::Node> nodes=v::empty<ir::Node>();var v::Vec<ir::Statement> statements=v::empty<ir::Statement>();var v::Vec<m::Local> locals=v::empty<m::Local>();var v::Vec<m::Invocation> calls=v::empty<m::Invocation>();var v::Vec<m::Argument> arguments=v::empty<m::Argument>();var v::Vec<regions::Scope> scopes=v::empty<regions::Scope>();var v::Vec<regions::Placement> placements=v::empty<regions::Placement>();var v::Vec<regions::Exit> resets=v::empty<regions::Exit>();var usize i=0;
    while(i<ir::node_count(b)){if(!work::charge(used,1)){return .err(1002);}var n=ir::node_at(b,i);if(n.value.field.id!=0){if(n.value.field.module_index>=current::module_count(mapping)){return .err(1001);}n.value.field.module_index=current::module_at(mapping,n.value.field.module_index);}if(!v::push<ir::Node>(ref mut nodes,n).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::statement_count(b)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<ir::Statement>(ref mut statements,ir::statement_at(b,i)).is_ok){return .err(1003);}i+=1;}i=1;while(i<=ir::local_count(b)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<m::Local>(ref mut locals,ir::local_at(b,i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::call_count(b)){if(!work::charge(used,1)){return .err(1002);}var c=ir::call_at(b,i);if(c.target_module>=current::module_count(mapping)){return .err(1001);}c.target_module=current::module_at(mapping,c.target_module);if(!v::push<m::Invocation>(ref mut calls,c).is_ok){return .err(1003);}i+=1;}i=0;while(i<ir::argument_count(b)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<m::Argument>(ref mut arguments,ir::argument_at(b,i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::region_count(b)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<regions::Scope>(ref mut scopes,ir::region_at(b,i)).is_ok){return .err(1003);}i+=1;}i=0;while(i<ir::placement_count(b)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<regions::Placement>(ref mut placements,ir::placement_at(b,i)).is_ok){return .err(1003);}i+=1;}i=0;while(i<ir::reset_count(b)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<regions::Exit>(ref mut resets,ir::reset_at(b,i)).is_ok){return .err(1003);}i+=1;}
    if(!work::charge(used,ir::text_bytes(b)+1)){return .err(1002);}let text=switch(bytes::copy(ir::text_view(b))){case .ok(p):{p}case .err(_):{return .err(1003);}};return .ok(ir::assemble_body_regions(ir::body(b),ir::source_bytes(b),ir::start(b),ir::end(b),nodes,statements,locals,calls,arguments,scopes,placements,resets,text));
}
