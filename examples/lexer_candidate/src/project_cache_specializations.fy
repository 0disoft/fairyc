module project_cache_specializations;
import project_compiler_core as core;
import module_project_interfaces as interfaces;
import module_project_identified_ir as checked;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import generic_package_bindings as source;
import module_source_specialization_set as sets;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

pub resource Selection{usize owner;v::Vec<bytes::Buffer> ids;cleanup{}}
pub fn usize owner(ref Selection p){return p.owner;}
pub fn usize count(ref Selection p){return v::len<bytes::Buffer>(ref p.ids);}
pub fn ref u8[] at(ref Selection p,usize index){return bytes::view(v::at<bytes::Buffer>(ref p.ids,index));}
fn u32 member(ref v::Vec<bytes::Buffer> ids,ref u8[] id,ref mut usize work){return switch(sets::contains(ids,id,work)){case .ok(found):{if(found){0}else{9}}case .err(code):{code}};}
// Complete publication retains original specialization membership, including
// declarations lowered into intrinsics. Check every surviving Vec/frame ID
// against that same set; neither a caller nor lowered numbering supplies IDs.
pub fn Result<Selection,u32> collect(ref core::Complete p,usize owner,ref mut usize work){
    if(owner>=512){return .err(9);}if(!budget::charge(work,0)){return .err(10);}let records=core::public_interfaces(p);var usize found=interfaces::count(records);var usize i=0;
    while(i<interfaces::count(records)){if(!budget::charge(work,1)){return .err(10);}if(interfaces::owner(interfaces::at(records,i))==owner){if(found!=interfaces::count(records)){return .err(9);}found=i;}i+=1;}if(found==interfaces::count(records)){return .err(9);}let ids=interfaces::source_specializations(interfaces::at(records,found));
    let program=core::checked_program(p);let provenance=checked::source_provenance(program);let plain=checked::ir_program(program);let contracts=ir::cleanup_ir(plain);i=1;
    while(i<=clean::type_count(contracts)){if(!budget::charge(work,1)){return .err(10);}let type=clean::type_at(contracts,i);let row=checked::type_at(program,i);if((row.node!=0)!=(type.kind==13)){return .err(9);}if(row.node!=0){if(type.module_index>=source::count(provenance)){return .err(9);}if(source::module_at(provenance,type.module_index).owner==owner){let code=member(ids,checked::identity(program,row),work);if(code!=0){return .err(code);}}}i+=1;}
    i=1;while(i<=checked::frame_count(program)){if(!budget::charge(work,1)){return .err(10);}let row=checked::frame_at(program,i);if(row.node!=0){let frame=clean::frame_at(contracts,i);if(frame.module_index>=source::count(provenance) || frame.kind!=1 && frame.kind!=2){return .err(9);}if(source::module_at(provenance,frame.module_index).owner==owner){let code=member(ids,checked::identity(program,row),work);if(code!=0){return .err(code);}}}i+=1;}
    var v::Vec<bytes::Buffer> output=v::empty<bytes::Buffer>();i=0;while(i<v::len<bytes::Buffer>(ids)){let id=bytes::view(v::at<bytes::Buffer>(ids,i));if(!budget::charge(work,id.len+1)){return .err(10);}let b=switch(bytes::copy(id)){case .ok(b):{b}case .err(_):{return .err(11);}};if(!v::push<bytes::Buffer>(ref mut output,b).is_ok){return .err(11);}i+=1;}return .ok(Selection{.owner,.ids=output});
}
pub fn Result<v::Vec<bytes::Buffer>,u32> copy_ids(ref Selection p,ref mut usize work){var v::Vec<bytes::Buffer> output=v::empty<bytes::Buffer>();var usize i=0;while(i<count(p)){let id=at(p,i);if(!budget::charge(work,id.len+1)){return .err(10);}let value=switch(bytes::copy(id)){case .ok(b):{b}case .err(_):{return .err(11);}};if(!v::push<bytes::Buffer>(ref mut output,value).is_ok){return .err(11);}i+=1;}return .ok(output);}
