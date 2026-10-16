module module_unit_body_ir_enums;
import module_unit_body_ir_model as ir;
import candidate_limits as limits;
import module_unit_body_ir_lookup as lookup;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_interfaces as interfaces;
import std::bytes as bytes;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut usize work,usize amount,ref mut u32 code){if(!budget::charge(work,amount)){*code=1002;return false;}return true;}
// IDs and declaration coordinates survive release of all parsed source owners.
pub fn usize layout(ref ir::Program p,usize type,ref mut usize work,ref mut u32 code){var usize found=0;var usize i=0;while(i<ir::layout_count(p)){if(!charge(work,1,code)){return 0;}if(ir::layout_at(p,i).value.type==type){if(found!=0){return 0;}found=i+1;}i+=1;}return found;}
pub fn bool member(ref ir::Program p,budget::Field field,usize owner,u32 kind){
    if(field.id==0 || field.id>ir::field_count(p) || field.owner!=owner || owner==0 || owner>clean::type_count(ir::cleanup_ir(p))){return false;}let f=ir::field_at(p,field.id-1);let t=clean::type_at(ir::cleanup_ir(p),owner);
    return f.kind==kind && f.module_index==t.module_index && f.declaration==t.declaration && field.module_index==f.module_index && field.declaration==f.declaration && field.source_index==f.source_index;
}
fn bool coordinates(interfaces::Field f,interfaces::Shape shape){return f.module_index==shape.module_index && f.declaration==shape.declaration && f.name_start<f.name_end && f.start>=shape.start && f.end<=shape.end && f.start<=f.end;}
fn bool names(ref ir::Program p,usize begin,usize end,usize id,u32 kind,ref mut usize work,ref mut u32 code){var usize i=begin;while(i<end){if(!charge(work,ir::field_name(p,id).len+1,code)){return false;}if(ir::field_at(p,i).kind==kind && bytes::equal(ir::field_name(p,i),ir::field_name(p,id))){return false;}i+=1;}return true;}
fn bool shape(ref ir::Program p,usize index,ref mut usize work,ref mut u32 code){
    let s=ir::layout_at(p,index).value;let end=s.field_start+s.field_count;var usize i=s.field_start;var usize previous_end=s.start;
    // records.fy stores payload fields before their owning variant marker.
    while(i<end){var usize selected=i;while(selected<end && ir::field_at(p,selected).kind==4){if(!charge(work,1,code)){return false;}selected+=1;}if(selected==end || !charge(work,1,code)){return false;}let f=ir::field_at(p,selected);
        if(f.kind!=3 || f.type!=0 || !coordinates(f,s) || f.start<previous_end || f.payload_count!=selected-i || f.payload_count==0 && f.payload_start!=0 || f.payload_count!=0 && f.payload_start!=i || !names(p,s.field_start,selected,selected,3,work,code)){return false;}
        var usize j=i;while(j<=selected){if(!charge(work,1,code)){return false;}let child=ir::field_at(p,j);if(j!=s.field_start && child.source_index!=ir::field_at(p,j-1).source_index+1){return false;}if(j<selected && (child.type==0 || child.payload_start!=0 || child.payload_count!=0 || !coordinates(child,s) || child.start<f.start || child.end>f.end || !names(p,i,j,j,4,work,code))){return false;}j+=1;}previous_end=f.end;i=selected+1;
    }return !s.repr_c && s.abilities==0 && !s.cleanup_nonempty && s.cleanup_effects==0;
}
// Recheck acyclic inert payloads independently of the released source graph.
// Safe borrows, owned resources, Vec, Region and void are not enum payloads.
fn bool inert(ref ir::Program p,usize id,ref mut v::Vec<u8> marks,ref mut usize work,usize depth,ref mut u32 code){
    if(depth>=64){*code=1002;return false;}if(id==0 || id>v::len<u8>(marks) || !charge(work,1,code)){return false;}let mark=*v::at<u8>(marks,id-1);if(mark==2){return true;}if(mark==1){return false;}{let slot=v::at_mut<u8>(marks,id-1);*slot=1;}
    let t=clean::type_at(ir::cleanup_ir(p),id);if(t.kind!=5 && t.kind!=6 && clean::glue_at(ir::cleanup_ir(p),id).kind!=0){return false;}
    if(t.kind==1){if(t.scalar==7){return false;}}else if(t.kind==12){}
    else if(t.kind==5 || t.kind==6){if(!inert(p,t.first,marks,work,depth+1,code) || t.kind==6 && !inert(p,t.second,marks,work,depth+1,code)){return false;}}
    else if(t.kind==2 || t.kind==4){let found=layout(p,id,work,code);if(found==0){return false;}let s=ir::layout_at(p,found-1).value;var usize i=0;
        while(i<s.field_count){let f=ir::field_at(p,s.field_start+i);if(!charge(work,1,code)){return false;}if(t.kind==2 && (f.kind!=0 || !coordinates(f,s))){return false;}if((f.kind==0 || f.kind==4) && !inert(p,f.type,marks,work,depth+1,code)){return false;}i+=1;}
    }else{return false;}{let slot=v::at_mut<u8>(marks,id-1);*slot=2;}return true;
}
fn bool construction(ref ir::Program p,ref ir::Body b,usize id,ref mut usize work,ref mut u32 code){
    let e=ir::node_at(b,id-1).value;let found=layout(p,e.type,work,code);if(found==0 || e.target!=e.type || !member(p,e.field,e.type,3)){return false;}let s=ir::layout_at(p,found-1).value;let selected=e.field.id-1;let f=ir::field_at(p,selected);
    if(selected<s.field_start || selected>=s.field_start+s.field_count){return false;}let callee=if(e.kind==28){id}else{lookup::node(b,e.left)};if(callee==0 || !charge(work,ir::field_name(p,selected).len+1,code) || !bytes::equal(ir::node_text(b,callee-1),ir::field_name(p,selected))){return false;}
    if(e.kind==28){return f.payload_count==0 && e.left==0 && e.right==0;}if(ir::node_at(b,callee-1).value.kind!=28 || ir::node_at(b,callee-1).value.state!=2){return false;}
    var v::Vec<u8> seen=v::empty<u8>();var usize i=0;while(i<f.payload_count){if(!charge(work,1,code)){return false;}if(!v::push<u8>(ref mut seen,0).is_ok){*code=1003;return false;}i+=1;}var usize link=e.right;var usize count=0;
    while(link!=0){if(count>=f.payload_count || !charge(work,1,code)){return false;}let pair=lookup::node(b,link);if(pair==0){return false;}let row=ir::node_at(b,pair-1).value;let child=lookup::node(b,row.left);
        if(row.kind!=11 || child==0 || !member(p,row.field,e.type,4) || row.field.id<=f.payload_start || row.field.id>f.payload_start+f.payload_count){return false;}let position=row.field.id-1-f.payload_start;let payload=ir::field_at(p,row.field.id-1);let value=ir::node_at(b,child-1).value;
        if(*v::at<u8>(ref seen,position)!=0 || value.state!=1 || value.type!=payload.type || row.expected!=payload.type || !charge(work,ir::field_name(p,row.field.id-1).len+1,code) || !bytes::equal(ir::node_text(b,pair-1),ir::field_name(p,row.field.id-1))){return false;}{let slot=v::at_mut<u8>(ref mut seen,position);*slot=1;}count+=1;link=row.right;
    }return count==f.payload_count;
}
pub fn clean::Problem check(ref ir::Program p,ref mut usize work){var usize total=0;return check_scoped(p,work,ref mut total,false);}
// Project scans retain the same 1 MiB prefix/body allowances and charge every
// completed chunk once to the caller's bounded aggregate verification ledger.
pub fn clean::Problem check_metered(ref ir::Program p,ref mut usize work,ref mut usize total){return check_scoped(p,work,total,clean::project(ir::cleanup_ir(p)));}
fn bool checkpoint(bool project,ref mut usize work,ref mut usize total){if(!project){return true;}if(!limits::project_charge(total,*work)){return false;};*work=0;return true;}
fn clean::Problem check_scoped(ref ir::Program p,ref mut usize work,ref mut usize total,bool project){
    var u32 problem=0;let code=ref mut problem;
    var v::Vec<u8> marks=v::empty<u8>();var usize i=0;while(i<clean::type_count(ir::cleanup_ir(p))){if(!charge(work,1,code)){return fail(1002,0,0,0);}if(!v::push<u8>(ref mut marks,0).is_ok){return fail(1003,0,0,0);}i+=1;}
    i=0;while(i<ir::layout_count(p)){let s=ir::layout_at(p,i).value;if(clean::type_at(ir::cleanup_ir(p),s.type).kind==4){if(layout(p,s.type,work,code)!=i+1 || !shape(p,i,work,code) || !inert(p,s.type,ref mut marks,work,0,code)){return fail(if(*code!=0){*code}else{1001},s.module_index,s.start,s.end);}}i+=1;}
    if(!checkpoint(project,work,total)){return fail(1002,0,0,0);}
    var usize frame=1;while(frame<=ir::count(p)){let b=ir::at(p,frame);i=0;while(i<ir::node_count(b)){if(!charge(work,1,code)){return fail(1002,0,0,0);}let e=ir::node_at(b,i).value;
        if(e.state==1 && (e.kind==28 || e.kind==10) && clean::type_at(ir::cleanup_ir(p),e.type).kind==4 && !construction(p,b,i+1,work,code)){return fail(if(*code!=0){*code}else{1001},clean::frame_at(ir::cleanup_ir(p),frame).module_index,e.start,e.end);}i+=1;
    }if(!checkpoint(project,work,total)){return fail(1002,0,0,0);}frame+=1;}return fail(0,0,0,0);
}
