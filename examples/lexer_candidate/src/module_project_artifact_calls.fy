module module_project_artifact_calls;
import module_project_artifact_bodies as bodies;
import module_project_artifact_frames as frames;
import module_project_artifact_types as types;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_source_specializations as selection;
import module_project_body_equal as equal;
import module_unit_c_callable_index as contracts;
import module_unit_c_frame_index as targets;
import module_identity_artifact_model as identities;
import generic_source_plan as source;
import module_signatures as signatures;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Current source checking plus body/call comparison does not authorize a cache
// hit. Cleanup graphs, full metadata and complete project coverage follow.
fn Result<usize,u32> target(ref ir::Program program,ref targets::Index index,m::Invocation c,ref mut usize work){var usize measured=0;var u32 code=0;let id=targets::find(program,index,c.target_module,c.target_declaration,c.instance,c.kind==2,ref mut measured,ref mut code);if(!budget::charge(work,measured)){return .err(1002);}return if(code==0){.ok(id)}else{.err(code)};}
fn Result<usize,u32> ordinary(ref ir::Program program,ref contracts::Index index,ref signatures::Program metadata,m::Invocation c,ref types::Mapping map,ref mut usize work){
    // Charge the bounded binary signature lookup before entering its legacy API.
    var usize width=signatures::count(metadata);if(!budget::charge(work,1)){return .err(1002);}while(width!=0){if(!budget::charge(work,1)){return .err(1002);}width/=2;}
    let original=signatures::identity(metadata,c.target_module,c.target_declaration);if(original==0){return .err(1001);}let expected=signatures::at(metadata,original);var usize measured=0;var u32 code=0;let found=contracts::find(program,index,c.target_module,c.target_declaration,ref mut measured,ref mut code);if(!budget::charge(work,measured)){return .err(1002);}if(code!=0){return .err(code);}if(found==0){return .err(1001);}let saved=ir::callable_at(program,found-1);let name=signatures::linker_name(metadata,original);
    if(!budget::charge(work,name.len+1)){return .err(1002);}if(saved.template || saved.result!=types::mapped(map,expected.result) || saved.parameter_count!=expected.parameter_count || saved.parameter_count!=c.argument_count || saved.effects!=expected.effects || saved.must_use!=expected.must_use || saved.unsafe_call!=expected.unsafe_call || saved.foreign!=expected.foreign || !bytes::equal(name,ir::linker_name(program,found-1))){return .err(1001);}
    var usize i=0;while(i<expected.parameter_count){if(!budget::charge(work,1)){return .err(1002);}if(types::mapped(map,signatures::parameter_at(metadata,expected.parameter_start+i))!=ir::parameter_at(program,saved.parameter_start+i)){return .err(1001);}i+=1;}return .ok(original);
}
fn u32 invocation(ref ir::Program program,ref identities::Snapshot ids,ref contracts::Index callables,ref targets::Index indexed,ref signatures::Program metadata,ref interfaces::Interfaces imported,ref frames::Mapping map,ref ir::Body saved,ref m::Values fresh,usize at,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}var expected=m::call_at(fresh,at);let old=ir::call_at(saved,at);if(!expected.checked || !old.checked || expected.kind!=1 && expected.kind!=2){return 1001;}let selected=frames::selected(map);let type_map=frames::type_mapping(map);var usize result=0;var u32 effects=0;var bool foreign=false;
    if(expected.kind==2){if(expected.instance==0 || expected.instance>selection::function_count(selected) || expected.instance>generics::count(interfaces::specializations(imported))){return 1001;}let signature=generics::at(interfaces::specializations(imported),expected.instance);if(signature.module_index!=expected.target_module || signature.declaration!=expected.target_declaration || signature.parameter_count!=expected.argument_count){return 1001;}result=types::mapped(type_map,signature.result);effects=signature.effects;}
    else{if(expected.instance!=0 || old.instance!=0){return 1001;}let status=switch(ordinary(program,callables,metadata,expected,type_map,work)){case .ok(p):{p}case .err(code):{return code;}};let signature=signatures::at(metadata,status);foreign=signature.foreign;result=types::mapped(type_map,signature.result);effects=signature.effects;}
    // The producer's specialization number may differ. Authenticate that number
    // through its actual target frame and original-source g0 identity first.
    expected.instance=old.instance;if(!equal::call_same(expected,old)){return 1001;}let found=switch(target(program,indexed,old,work)){case .ok(id):{id}case .err(code):{return code;}};if(foreign){return if(found==0 && result==ir::node_at(saved,old.expression-1).value.type){0}else{1001};}if(found==0 || found>identities::frame_count(ids)){return 1001;}let actual=clean::frame_at(ir::cleanup_ir(program),found);if(actual.result!=result || actual.effects!=effects || actual.result!=ir::node_at(saved,old.expression-1).value.type){return 1001;}let retained=identities::frame_at(ids,found);
    if(old.kind==2){let row=selection::function_at(selected,m::call_at(fresh,at).instance);let id=selection::identity(selected,row);if(!budget::charge(work,id.len+1)){return 1002;}if((row.node!=0)!=(retained.node!=0) || !bytes::equal(id,identities::identity(ids,retained))){return 1001;}}
    else if(retained.node!=0 || identities::identity(ids,retained).len!=0){return 1001;}
    let callee=ir::at(program,found);var usize position=0;var usize i=1;while(i<=ir::local_count(callee)){if(!budget::charge(work,1)){return 1002;}let local=ir::local_at(callee,i);if(local.kind==1){if(position>=old.argument_count || local.type!=ir::argument_at(saved,old.argument_start+position).expected){return 1001;}position+=1;}i+=1;}return if(position==old.argument_count){0}else{1001};
}
pub fn Result<frames::Mapping,u32> match(ref ir::Program program,ref identities::Snapshot ids,ref checking::Plan checked,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    let map=switch(bodies::match(program,ids,checked,plan,sources,metadata,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var usize measured=0;let callables=switch(contracts::build(program,ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(1002);}return .err(code);}};let indexed=switch(targets::build(program,ref callables,ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(1002);}return .err(code);}};if(!budget::charge(work,measured)){return .err(1002);}let typed=checking::checked_values(checked);let imported=checking::checked_interfaces(checked);var usize i=0;
    while(i<values::count(typed)){let fresh=values::at(typed,i);let saved=ir::at(program,frames::mapped(ref map,i+1));var usize j=0;while(j<ir::argument_count(saved)){if(!budget::charge(work,1)){return .err(1002);}var a=m::argument_at(fresh,j);a.expected=types::mapped(frames::type_mapping(ref map),a.expected);a.actual=types::mapped(frames::type_mapping(ref map),a.actual);if(!a.checked || !equal::argument_same(a,ir::argument_at(saved,j))){return .err(1001);}j+=1;}
        j=0;while(j<ir::call_count(saved)){let code=invocation(program,ids,ref callables,ref indexed,metadata,imported,ref map,saved,fresh,j,work);if(code!=0){return .err(code);}j+=1;}i+=1;}return .ok(map);
}
