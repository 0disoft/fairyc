module module_project_artifact_metadata;
import module_project_artifact_recipes as recipes;
import module_project_artifact_frames as frames;
import module_project_artifact_types as types;
import module_project_metadata as equality;
import module_unit_checked_bodies as checking;
import module_unit_interfaces as interfaces;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_callable_index as contracts;
import module_identity_artifact_model as identities;
import generic_source_plan as source;
import module_signatures as signatures;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// A unit match owns checked frame and metadata maps. Project-wide coverage,
// state/SDK equality and Complete/cache publication remain separate gates.
pub resource Matched {frames::Mapping frames;v::Vec<usize> layouts;v::Vec<usize> callables;cleanup{}}
pub fn ref frames::Mapping frame_mapping(ref Matched p){return ref p.frames;}
pub fn usize layout_count(ref Matched p){return v::len<usize>(ref p.layouts);}
pub fn usize layout_at(ref Matched p,usize index){return *v::at<usize>(ref p.layouts,index);}
pub fn usize callable_count(ref Matched p){return v::len<usize>(ref p.callables);}
pub fn usize callable_at(ref Matched p,usize index){return *v::at<usize>(ref p.callables,index);}
fn Result<v::Vec<usize>,u32> layout_index(ref ir::Program p,ref mut usize work){var v::Vec<usize> slots=v::empty<usize>();var usize i=0;
    while(i<=clean::type_count(ir::cleanup_ir(p))){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut slots,0).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::layout_count(p)){if(!budget::charge(work,1)){return .err(1002);}let type=ir::layout_at(p,i).value.type;if(type==0 || type>=v::len<usize>(ref slots) || *v::at<usize>(ref slots,type)!=0){return .err(1001);}let cell=v::at_mut<usize>(ref mut slots,type);*cell=i+1;i+=1;}return .ok(slots);
}
fn u32 layout(ref ir::Program p,ref interfaces::Interfaces imported,ref v::Vec<syntax::Parsed> sources,interfaces::Shape fresh,usize target,ref types::Mapping map,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(target==0 || target>ir::layout_count(p) || fresh.module_index>=v::len<syntax::Parsed>(sources)){return 1001;}let old=ir::layout_at(p,target-1).value;var expected=fresh;expected.type=types::mapped(map,fresh.type);
    if(expected.type==0 || !equality::shape(expected,old)){return 1001;}let parsed=v::at<syntax::Parsed>(sources,fresh.module_index);if(fresh.declaration==0 || fresh.declaration>syntax::declaration_count(parsed)){return 1001;}let d=syntax::declaration_at(parsed,fresh.declaration-1);let name=syntax::source_view(parsed)[d.name_start..d.name_end];if(!budget::charge(work,name.len+1)){return 1002;}if(!bytes::equal(name,ir::layout_name(p,target-1))){return 1001;}
    var usize i=0;while(i<fresh.field_count){if(!budget::charge(work,1)){return 1002;}var a=interfaces::field_at(imported,fresh.field_start+i);let b=ir::field_at(p,old.field_start+i);let type=types::mapped(map,a.type);if(a.type!=0 && type==0){return 1001;};a.type=type;
        if(a.payload_count!=0){if(a.payload_start<fresh.field_start || a.payload_start-fresh.field_start>=fresh.field_count){return 1001;};a.payload_start=old.field_start+a.payload_start-fresh.field_start;}
        let text=interfaces::field_name(imported,fresh.field_start+i);if(!budget::charge(work,text.len+1)){return 1002;}if(!equality::field(a,b) || !bytes::equal(text,ir::field_name(p,old.field_start+i))){return 1001;}i+=1;
    }return 0;
}
fn u32 callable(ref ir::Program p,ref interfaces::Interfaces imported,usize index,usize target,ref types::Mapping map,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(target==0 || target>ir::callable_count(p)){return 1001;}let a=interfaces::callable_at(imported,index);let b=ir::callable_at(p,target-1);let result=types::mapped(map,a.result);
    if(a.result!=0 && result==0 || a.module_index!=b.module_index || a.declaration!=b.declaration || a.template!=b.template || result!=b.result || a.parameter_count!=b.parameter_count || a.effects!=b.effects || a.must_use!=b.must_use || a.unsafe_call!=b.unsafe_call || a.foreign!=b.foreign){return 1001;}
    let text=interfaces::linker_name(imported,index);if(!budget::charge(work,text.len+1)){return 1002;}if(!bytes::equal(text,ir::linker_name(p,target-1))){return 1001;}var usize i=0;while(i<a.parameter_count){if(!budget::charge(work,1)){return 1002;}let type=types::mapped(map,interfaces::parameter_at(imported,a.parameter_start+i));if(type==0 || type!=ir::parameter_at(p,b.parameter_start+i)){return 1001;}i+=1;}return 0;
}
pub fn Result<Matched,u32> match(ref ir::Program program,ref identities::Snapshot ids,ref checking::Plan checked,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    let map=switch(recipes::match(program,ids,checked,plan,sources,metadata,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let imported=checking::checked_interfaces(checked);let indexed=switch(layout_index(program,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var usize measured=0;
    let contracts=switch(contracts::build(program,ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(1002);}return .err(code);}};if(!budget::charge(work,measured)){return .err(1002);}var v::Vec<usize> layouts=v::empty<usize>();var v::Vec<usize> callables=v::empty<usize>();var usize i=0;
    while(i<interfaces::shape_count(imported)){if(!budget::charge(work,1)){return .err(1002);}let s=interfaces::shape_at(imported,i);let type=types::mapped(frames::type_mapping(ref map),s.type);if(type==0 || type>=v::len<usize>(ref indexed)){return .err(1001);}let target=*v::at<usize>(ref indexed,type);let code=layout(program,imported,sources,s,target,frames::type_mapping(ref map),work);if(code!=0){return .err(code);}if(!v::push<usize>(ref mut layouts,target).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<interfaces::callable_count(imported)){let c=interfaces::callable_at(imported,i);measured=0;var u32 code=0;let target=contracts::find(program,ref contracts,c.module_index,c.declaration,ref mut measured,ref mut code);if(!budget::charge(work,measured)){return .err(1002);}if(code!=0){return .err(code);}let status=callable(program,imported,i,target,frames::type_mapping(ref map),work);if(status!=0){return .err(status);}if(!v::push<usize>(ref mut callables,target).is_ok){return .err(1003);}i+=1;}return .ok(Matched{.frames=map,.layouts,.callables});
}
