module module_project_link;
import module_project_types as types;
import module_project_frames as frames;
import module_project_metadata as metadata;
import module_project_body as bodies;
import module_project_body_equal as equality;
import module_project_cleanup as cleaning;
import module_project_calls as calls;
import module_signatures as signatures;
import module_types as descriptors;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_sdk_sources as sdk;
import module_unit_interfaces as interfaces;
import module_unit_value_model as budget;
import std::bytes as bytes;
import std::vec as v;

fn clean::Problem fail(u32 code){return clean::Problem{.code,.module_index=0,.start=0,.end=0};}
fn Result<sdk::Sources,clean::Problem> shared_sdk(ref signatures::Program metadata,ref v::Vec<ir::Program> inputs,ref mut usize work){let first=ir::sdk_units(v::at<ir::Program>(inputs,0));var usize unit=0;while(unit<v::len<ir::Program>(inputs)){if(!budget::charge(work,1)){return .err(fail(1002));}let current=ir::sdk_units(v::at<ir::Program>(inputs,unit));if(sdk::module_count(current)!=signatures::module_count(metadata) || sdk::count(first)!=sdk::count(current) || !bytes::equal(sdk::data(first),sdk::data(current))){return .err(fail(1001));}var usize i=0;while(i<sdk::count(first)){if(!budget::charge(work,1)){return .err(fail(1002));}let a=sdk::at(first,i);let b=sdk::at(current,i);if(a.module_index!=b.module_index || a.sdk_id!=b.sdk_id || a.start!=b.start || a.end!=b.end){return .err(fail(1001));}i+=1;}unit+=1;}
    var v::Vec<sdk::Unit> units=v::empty<sdk::Unit>();var usize i=0;while(i<sdk::count(first)){if(!v::push<sdk::Unit>(ref mut units,sdk::at(first,i)).is_ok){return .err(fail(1003));}i+=1;}if(!budget::charge(work,sdk::data(first).len)){return .err(fail(1002));}let source=switch(bytes::copy(sdk::data(first))){case .ok(p):{p}case .err(_):{return .err(fail(1003));}};return sdk::assemble(sdk::module_count(first),units,source,sdk::work(first));
}
// Link independently checked units without joining source or inventing extern
// bodies. Original lexical IDs stay local; type/field/frame IDs become global.
pub fn Result<ir::Program,clean::Problem> build(ref signatures::Program catalog,ref v::Vec<ir::Program> inputs){
    if(v::len<ir::Program>(inputs)==0 || v::len<ir::Program>(inputs)>256){return .err(fail(if(v::len<ir::Program>(inputs)>256){1002}else{1001}));}var v::Vec<descriptors::Graph> graphs=v::empty<descriptors::Graph>();var usize unit=0;
    while(unit<v::len<ir::Program>(inputs)){let graph=switch(types::from_ir(catalog,v::at<ir::Program>(inputs,unit))){case .ok(p):{p}case .err(e):{return .err(e);}};if(!v::push<descriptors::Graph>(ref mut graphs,graph).is_ok){return .err(fail(1003));}unit+=1;}
    let graph=switch(types::build(catalog,ref graphs)){case .ok(p):{p}case .err(e):{return .err(e);}};let functions=switch(frames::build(inputs,ref graph)){case .ok(p):{p}case .err(e):{return .err(e);}};let layouts=switch(metadata::build(inputs,ref graph)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var usize work=0;if(!budget::charge(ref mut work,types::work(ref graph)+frames::work(ref functions)+metadata::work(ref layouts))){return .err(fail(1002));}let sdk_sources=switch(shared_sdk(catalog,inputs,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var v::Vec<ir::Body> retained=v::empty<ir::Body>();var usize i=1;while(i<=frames::count(ref functions)){let origin=frames::origin(ref functions,i);let b=switch(bodies::copy(inputs,origin.unit,origin.frame,ref graph,ref functions,ref layouts,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!v::push<ir::Body>(ref mut retained,b).is_ok){return .err(fail(1003));}i+=1;}
    unit=0;while(unit<v::len<ir::Program>(inputs)){let input=v::at<ir::Program>(inputs,unit);i=1;while(i<=ir::count(input)){let id=frames::mapped(ref functions,unit,i);let origin=frames::origin(ref functions,id);if(origin.unit!=unit || origin.frame!=i){let duplicate=switch(bodies::copy(inputs,unit,i,ref graph,ref functions,ref layouts,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!equality::same(v::at<ir::Body>(ref retained,id-1),ref duplicate,ref mut work)){return .err(fail(if(work>=1048576){1002}else{1001}));}}i+=1;}unit+=1;}
    let cleaned=switch(cleaning::build(inputs,ref graph,ref functions,ref layouts,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var v::Vec<ir::Layout> shapes=v::empty<ir::Layout>();var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();var v::Vec<usize> parameters=v::empty<usize>();i=0;
    while(i<metadata::layout_count(ref layouts)){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}if(!v::push<ir::Layout>(ref mut shapes,metadata::layout_at(ref layouts,i)).is_ok){return .err(fail(1003));}i+=1;}i=0;while(i<metadata::field_count(ref layouts)){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}if(!v::push<interfaces::Field>(ref mut fields,metadata::field_at(ref layouts,i)).is_ok){return .err(fail(1003));}i+=1;}i=0;while(i<metadata::callable_count(ref layouts)){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}if(!v::push<interfaces::Callable>(ref mut callables,metadata::callable_at(ref layouts,i)).is_ok){return .err(fail(1003));}i+=1;}i=0;while(i<metadata::parameter_count(ref layouts)){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}if(!v::push<usize>(ref mut parameters,metadata::parameter_at(ref layouts,i)).is_ok){return .err(fail(1003));}i+=1;}
    if(!budget::charge(ref mut work,metadata::names(ref layouts).len)){return .err(fail(1002));}let names=switch(bytes::copy(metadata::names(ref layouts))){case .ok(p):{p}case .err(_):{return .err(fail(1003));}};
    let result=switch(ir::assemble_sdk(sdk_sources,cleaned,retained,shapes,fields,callables,parameters,names,work)){case .ok(p):{p}case .err(e):{return .err(e);}};let linked=calls::check(ref result,ref mut work);if(linked.code!=0){return .err(linked);}return .ok(result);
}
