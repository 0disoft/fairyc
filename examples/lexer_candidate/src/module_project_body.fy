module module_project_body;
import module_project_types as types;
import module_project_frames as frames;
import module_project_metadata as metadata;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as m;
import module_unit_region_model as regions;
import std::bytes as bytes;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
pub fn Result<ir::Body,clean::Problem> copy(ref v::Vec<ir::Program> inputs,usize unit,usize frame,ref types::Plan graph,ref frames::Plan functions,ref metadata::Plan layouts,ref mut usize work){
    let source=v::at<ir::Program>(inputs,unit);let b=ir::at(source,frame);let f=clean::frame_at(ir::cleanup_ir(source),frame);
    var v::Vec<ir::Node> nodes=v::empty<ir::Node>();var v::Vec<ir::Statement> statements=v::empty<ir::Statement>();var v::Vec<m::Local> locals=v::empty<m::Local>();var v::Vec<m::Invocation> calls=v::empty<m::Invocation>();var v::Vec<m::Argument> arguments=v::empty<m::Argument>();var v::Vec<regions::Scope> scopes=v::empty<regions::Scope>();var v::Vec<regions::Placement> placements=v::empty<regions::Placement>();var v::Vec<regions::Exit> resets=v::empty<regions::Exit>();var usize i=0;
    while(i<ir::node_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,ir::start(b),ir::end(b)));}var n=ir::node_at(b,i);n.value.type=types::mapped(graph,unit,n.value.type);n.value.expected=types::mapped(graph,unit,n.value.expected);n.value.target=types::mapped(graph,unit,n.value.target);n.value.builtin.element=types::mapped(graph,unit,n.value.builtin.element);n.value.field.owner=types::mapped(graph,unit,n.value.field.owner);let field=n.value.field.id;n.value.field.id=metadata::mapped_field(layouts,unit,field);if(field!=0 && n.value.field.id==0){return .err(fail(1001,f.module_index,n.value.start,n.value.end));}if(!v::push<ir::Node>(ref mut nodes,n).is_ok){return .err(fail(1003,f.module_index,n.value.start,n.value.end));}i+=1;}
    i=0;while(i<ir::statement_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,0,0));}if(!v::push<ir::Statement>(ref mut statements,ir::statement_at(b,i)).is_ok){return .err(fail(1003,f.module_index,0,0));}i+=1;}
    i=1;while(i<=ir::local_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,0,0));}var n=ir::local_at(b,i);n.type=types::mapped(graph,unit,n.type);n.expected=types::mapped(graph,unit,n.expected);if(!v::push<m::Local>(ref mut locals,n).is_ok){return .err(fail(1003,f.module_index,n.start,n.end));}i+=1;}
    i=0;while(i<ir::call_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,0,0));}var c=ir::call_at(b,i);if(c.kind==2){c.instance=frames::specialized(functions,inputs,unit,c.target_module,c.target_declaration,c.instance);if(c.instance==0){return .err(fail(1001,f.module_index,c.start,c.end));}}if(!v::push<m::Invocation>(ref mut calls,c).is_ok){return .err(fail(1003,f.module_index,c.start,c.end));}i+=1;}
    i=0;while(i<ir::argument_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,0,0));}var a=ir::argument_at(b,i);a.expected=types::mapped(graph,unit,a.expected);a.actual=types::mapped(graph,unit,a.actual);if(!v::push<m::Argument>(ref mut arguments,a).is_ok){return .err(fail(1003,f.module_index,a.start,a.end));}i+=1;}
    i=0;while(i<ir::region_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,0,0));}var r=ir::region_at(b,i);r.arena_type=types::mapped(graph,unit,r.arena_type);r.source_type=types::mapped(graph,unit,r.source_type);if(!v::push<regions::Scope>(ref mut scopes,r).is_ok){return .err(fail(1003,f.module_index,r.start,r.end));}i+=1;}
    i=0;while(i<ir::placement_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,0,0));}var p=ir::placement_at(b,i);p.owner_type=types::mapped(graph,unit,p.owner_type);p.borrow_type=types::mapped(graph,unit,p.borrow_type);p.result_type=types::mapped(graph,unit,p.result_type);p.error_type=types::mapped(graph,unit,p.error_type);if(!v::push<regions::Placement>(ref mut placements,p).is_ok){return .err(fail(1003,f.module_index,p.start,p.end));}i+=1;}
    i=0;while(i<ir::reset_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,0,0));}if(!v::push<regions::Exit>(ref mut resets,ir::reset_at(b,i)).is_ok){return .err(fail(1003,f.module_index,0,0));}i+=1;}
    if(!m::charge(work,ir::text_bytes(b))){return .err(fail(1002,f.module_index,ir::start(b),ir::end(b)));}let text=switch(bytes::copy(ir::text_view(b))){case .ok(p):{p}case .err(_):{return .err(fail(1003,f.module_index,0,0));}};
    return .ok(ir::assemble_body_regions(ir::body(b),ir::source_bytes(b),ir::start(b),ir::end(b),nodes,statements,locals,calls,arguments,scopes,placements,resets,text));
}
