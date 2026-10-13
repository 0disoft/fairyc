module module_metadata_artifact_model;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_interfaces as interfaces;
import module_unit_sdk_sources as sdk;
import module_state_layout as state;
import candidate_limits as limits;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// Bounded component ownership, not a checked executable or Complete owner.
pub resource Metadata{v::Vec<ir::Layout> layouts;v::Vec<interfaces::Field> fields;v::Vec<interfaces::Callable> callables;v::Vec<usize> parameters;bytes::Buffer names;cleanup{}}
pub fn usize layout_count(ref Metadata p){return v::len<ir::Layout>(ref p.layouts);}
pub fn ir::Layout layout_at(ref Metadata p,usize i){return *v::at<ir::Layout>(ref p.layouts,i);}
pub fn usize field_count(ref Metadata p){return v::len<interfaces::Field>(ref p.fields);}
pub fn interfaces::Field field_at(ref Metadata p,usize i){return *v::at<interfaces::Field>(ref p.fields,i);}
pub fn usize callable_count(ref Metadata p){return v::len<interfaces::Callable>(ref p.callables);}
pub fn interfaces::Callable callable_at(ref Metadata p,usize i){return *v::at<interfaces::Callable>(ref p.callables,i);}
pub fn usize parameter_count(ref Metadata p){return v::len<usize>(ref p.parameters);}
pub fn usize parameter_at(ref Metadata p,usize i){return *v::at<usize>(ref p.parameters,i);}
pub fn ref u8[] names(ref Metadata p){return bytes::view(ref p.names);}
fn bool span(usize start,usize end,usize size){return start<=end && end<=size;}
fn bool bounded(usize layouts,usize fields,usize callables,usize parameters,usize names){return layouts<=1024 && fields<=limits::project_fields() && callables<=limits::project_callables() && parameters<=32768 && names<=1048576;}
pub fn u32 check(ref Metadata p,ref clean::Program cleaned,usize modules,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(modules>limits::source_files() || !bounded(layout_count(p),field_count(p),callable_count(p),parameter_count(p),names(p).len)){return 1002;}
    let nt=clean::type_count(cleaned);let owners=if(modules==0){limits::source_files()}else{modules};var usize i=0;
    while(i<layout_count(p)){if(!budget::charge(work,1)){return 1002;}let row=layout_at(p,i);let s=row.value;
        if(s.type==0 || s.type>nt || s.module_index>=owners || s.field_start>field_count(p) || s.field_count>field_count(p)-s.field_start || !span(row.name_start,row.name_end,names(p).len) || s.abilities>7 || s.cleanup_effects>4095){return 1001;}
        let t=clean::type_at(cleaned,s.type);if(t.module_index!=s.module_index || t.declaration!=s.declaration){return 1001;}i+=1;
    }
    i=0;while(i<field_count(p)){if(!budget::charge(work,1)){return 1002;}let f=field_at(p,i);
        if(f.type>nt || f.module_index>=owners || f.declaration>1048576 || f.source_index>1048576 || f.payload_start>field_count(p) || f.payload_count>field_count(p)-f.payload_start || !span(f.name_start,f.name_end,names(p).len)){return 1001;}i+=1;
    }
    i=0;while(i<callable_count(p)){if(!budget::charge(work,1)){return 1002;}let c=callable_at(p,i);
        if(c.module_index>=owners || c.declaration==0 || c.declaration>1048576 || c.result>nt || c.parameter_start>parameter_count(p) || c.parameter_count>parameter_count(p)-c.parameter_start || !span(c.link_start,c.link_end,names(p).len) || c.effects>4095){return 1001;}i+=1;
    }
    i=0;while(i<parameter_count(p)){if(!budget::charge(work,1)){return 1002;}if(parameter_at(p,i)>nt){return 1001;}i+=1;}return 0;
}
pub fn Result<Metadata,u32> assemble(v::Vec<ir::Layout> layouts,v::Vec<interfaces::Field> fields,v::Vec<interfaces::Callable> callables,v::Vec<usize> parameters,bytes::Buffer names,ref clean::Program cleaned,usize modules,ref mut usize work){let p=Metadata{.layouts,.fields,.callables,.parameters,.names};let code=check(ref p,cleaned,modules,work);if(code!=0){return .err(code);}return .ok(p);}
pub fn Result<Metadata,u32> extract(ref ir::Program p,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}if(!bounded(ir::layout_count(p),ir::field_count(p),ir::callable_count(p),ir::parameter_count(p),ir::name_bytes(p))){return .err(1002);}
    var v::Vec<ir::Layout> layouts=v::empty<ir::Layout>();var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();var v::Vec<usize> parameters=v::empty<usize>();var usize i=0;
    while(i<ir::layout_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<ir::Layout>(ref mut layouts,ir::layout_at(p,i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::field_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<interfaces::Field>(ref mut fields,ir::field_at(p,i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<interfaces::Callable>(ref mut callables,ir::callable_at(p,i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::parameter_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut parameters,ir::parameter_at(p,i)).is_ok){return .err(1003);}i+=1;}
    if(!budget::charge(work,ir::name_bytes(p)+1)){return .err(1002);}let names=switch(bytes::copy(ir::names_view(p))){case .ok(p):{p}case .err(_):{return .err(1003);}};
    return assemble(layouts,fields,callables,parameters,names,ir::cleanup_ir(p),sdk::module_count(ir::sdk_units(p)),work);
}
pub fn Result<ir::Program,u32> restore(ref Metadata p,state::Plan states,sdk::Sources sdk_sources,clean::Program cleaned,v::Vec<ir::Body> bodies,ref mut usize work){
    let code=check(p,ref cleaned,sdk::module_count(ref sdk_sources),work);if(code!=0){return .err(code);}
    // Resource fields remain owned by p. Copy them explicitly into the fresh
    // Program; no partial resource move or retained borrow crosses this API.
    var v::Vec<ir::Layout> layouts=v::empty<ir::Layout>();var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();var v::Vec<usize> parameters=v::empty<usize>();var usize i=0;
    while(i<layout_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<ir::Layout>(ref mut layouts,layout_at(p,i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<field_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<interfaces::Field>(ref mut fields,field_at(p,i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<callable_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<interfaces::Callable>(ref mut callables,callable_at(p,i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<parameter_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut parameters,parameter_at(p,i)).is_ok){return .err(1003);}i+=1;}
    if(!budget::charge(work,names(p).len+1)){return .err(1002);}let names=switch(bytes::copy(names(p))){case .ok(p):{p}case .err(_):{return .err(1003);}};
    var usize measured=0;let restored=switch(ir::assemble_artifact(states,sdk_sources,cleaned,bodies,layouts,fields,callables,parameters,names,ref mut measured)){case .ok(p):{p}case .err(e):{if(!budget::charge(work,measured)){return .err(1002);}return .err(e.code);}};
    if(!budget::charge(work,measured)){return .err(1002);}return .ok(restored);
}
