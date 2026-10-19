module module_unit_interpreter_values;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_constant_arithmetic as math;
import std::bytes as bytes;
import std::vec as v;

// Immutable aggregate cells and frozen place descriptors share a per-test heap.
// Mutable binding root cells carry updates across checked reference arguments.
pub struct Outcome{u32 error;u32 trap;usize bits;}
pub fn Outcome result(usize bits){return Outcome{.error=0,.trap=0,.bits};}
pub fn Outcome failure(u32 error){return Outcome{.error,.trap=0,.bits=0};}
pub fn Outcome trapped(u32 trap){return Outcome{.error=0,.trap,.bits=0};}
pub fn u32 scalar(ref ir::Program p,usize type){let t=clean::type_at(ir::cleanup_ir(p),type);return if(t.kind==1){t.scalar}else{0};}
pub fn bool admitted(ref ir::Program p,usize type){if(rx_reference(p,type) || rx_view(p,type)){return true;}let t=clean::type_at(ir::cleanup_ir(p),type);return t.kind==1 && t.scalar<=7 || t.kind>=2 && t.kind<=6 || t.kind==16 || t.kind==19;}
pub fn Outcome allocate(ref mut v::Vec<usize> heap,usize count,ref mut usize steps){
    if(count>999999 || *steps<count+1 || v::len<usize>(heap)>1000000-count-1){return failure(2);};
    *steps=*steps-count-1;let id=v::len<usize>(heap)+1;
    if(!v::push<usize>(heap,count).is_ok){return failure(4);}var usize i=0;while(i<count){if(!v::push<usize>(heap,0).is_ok){return failure(4);}i+=1;}return result(id);
}
pub fn Outcome item(ref v::Vec<usize> heap,usize id,usize index){
    if(id==0 || id>v::len<usize>(heap)){return failure(4);}let count=*v::at<usize>(heap,id-1);if(index>=count){return trapped(1);}if(count>v::len<usize>(heap)-id){return failure(4);}return result(*v::at<usize>(heap,id+index));
}
pub fn Outcome field_index(ref ir::Program p,usize type,usize field,ref mut usize steps){var usize i=0;while(i<ir::layout_count(p)){if(*steps==0){return failure(2);};*steps=*steps-1;let s=ir::layout_at(p,i).value;if(s.type==type){return if(field>s.field_start && field<=s.field_start+s.field_count){result(field-s.field_start-1)}else{failure(4)};}i+=1;}return failure(4);}
pub fn Outcome read_place(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(id==0 || depth>=64){return failure(4);}let e=ir::node_at(b,id-1).value;if(e.kind==2 && e.binding!=0){return item(heap,*v::at<usize>(locals,e.binding),0);}
    let place=rx_place(p,b,id,locals,values,heap,steps,depth);if(place.error!=0 || place.trap!=0){return place;}return rx_get(heap,place.bits,steps);
}
pub fn Outcome write_place(ref ir::Program p,ref ir::Body b,usize id,usize bits,ref mut v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(id==0 || depth>=64){return failure(4);}let e=ir::node_at(b,id-1).value;if(e.kind==2 && e.binding!=0){return rx_set_cell(heap,*v::at<usize>(locals,e.binding),bits);}
    let place=rx_place(p,b,id,locals,values,heap,steps,depth);if(place.error!=0 || place.trap!=0){return place;}return rx_put(heap,place.bits,bits,steps);
}
pub fn Outcome aggregate(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    let e=ir::node_at(b,id-1).value;let t=clean::type_at(ir::cleanup_ir(p),e.type);var usize count=if(t.kind==16){t.second}else{0};
    if(t.kind==2 || t.kind==3){var usize i=0;var bool found=false;while(i<ir::layout_count(p)){if(*steps==0){return failure(2);};*steps=*steps-1;let s=ir::layout_at(p,i).value;if(s.type==e.type){count=s.field_count;found=true;break;}i+=1;}if(!found){return failure(4);}}
    let a=allocate(heap,count,steps);if(a.error!=0){return a;}
    if(e.kind==29){let bits=*v::at<usize>(values,ir::node(b,e.left));var usize i=0;while(i<count){let slot=v::at_mut<usize>(heap,a.bits+i);*slot=bits;i+=1;}return a;}
    var usize link=e.right;var usize position=count;while(link!=0){let row=ir::node_at(b,ir::node(b,link)-1).value;var usize child=ir::node(b,row.left);if(child==0 || position==0){return failure(4);}position-=1;var usize index=position;if(t.kind==2 || t.kind==3){let resolved=field_index(p,e.type,row.field.id,steps);if(resolved.error!=0){return resolved;}index=resolved.bits;}let bits=*v::at<usize>(values,child);let slot=v::at_mut<usize>(heap,a.bits+index);*slot=bits;link=row.right;}return if(position==0){a}else{failure(4)};
}
pub fn bool slots(ref mut v::Vec<usize> values,usize count){if(!v::reserve<usize>(values,count).is_ok){return false;}while(v::len<usize>(values)<count){if(!v::push<usize>(values,0).is_ok){return false;}}return true;}
pub fn Outcome computed(math::RuntimeValue r){return if(!r.supported){failure(1)}else if(r.trap!=0){trapped(r.trap)}else{result(r.bits)};}
pub fn Outcome literal(ref ir::Body b,usize id,u32 type){
    if(type==6){return result(if(bytes::equal(ir::node_text(b,id-1),b"true")){1}else{0});}
    let text=ir::node_cooked(b,id-1);var usize bits=0;var usize i=0;if(text.len==0){return failure(4);}while(i<text.len){let digit=text[i] as usize;if(digit<48 || digit>57 || bits>(18446744073709551615-(digit-48))/10){return failure(4);}bits=bits*10+(digit-48);i+=1;}return result(bits);
}
pub fn bool rx_view(ref ir::Program p,usize type){let t=clean::type_at(ir::cleanup_ir(p),type);return t.kind==9 || t.kind==10 || t.kind==11 || t.kind==17 || t.kind==18 || t.kind==15 && t.scalar>=18 && t.scalar<=29 && t.scalar!=22 && t.scalar!=23;}
pub fn bool rx_reference(ref ir::Program p,usize type){let t=clean::type_at(ir::cleanup_ir(p),type);return t.kind==7 || t.kind==8;}
pub fn Outcome rx_cell(ref mut v::Vec<usize> heap,usize bits,ref mut usize steps){let a=allocate(heap,1,steps);if(a.error!=0){return a;}{let slot=v::at_mut<usize>(heap,a.bits);*slot=bits;}return a;}
pub fn Outcome rx_set_cell(ref mut v::Vec<usize> heap,usize cell,usize bits){let old=item(heap,cell,0);if(old.error!=0 || old.trap!=0){return old;}{let slot=v::at_mut<usize>(heap,cell);*slot=bits;}return result(bits);}
pub fn Outcome rx_descriptor(ref mut v::Vec<usize> heap,usize root,usize path,usize offset,usize length,ref mut usize steps){
    let a=allocate(heap,4,steps);if(a.error!=0){return a;}
    {let slot=v::at_mut<usize>(heap,a.bits);*slot=root;}{let slot=v::at_mut<usize>(heap,a.bits+1);*slot=path;}{let slot=v::at_mut<usize>(heap,a.bits+2);*slot=offset;}{let slot=v::at_mut<usize>(heap,a.bits+3);*slot=length;}return a;
}
pub fn Outcome rx_path_read(ref v::Vec<usize> heap,usize root,usize path,ref mut usize steps,usize depth){
    if(depth>=64){return failure(4);}if(*steps==0){return failure(2);};*steps=*steps-1;if(path==0){return result(root);}
    let parent=item(heap,path,0);let index=item(heap,path,1);if(parent.error!=0 || index.error!=0 || parent.trap!=0 || index.trap!=0){return failure(4);}
    let value=rx_path_read(heap,root,parent.bits,steps,depth+1);if(value.error!=0 || value.trap!=0){return value;}return item(heap,value.bits,index.bits);
}
pub fn Outcome rx_get(ref v::Vec<usize> heap,usize descriptor,ref mut usize steps){
    let cell=item(heap,descriptor,0);let path=item(heap,descriptor,1);if(cell.error!=0 || path.error!=0 || cell.trap!=0 || path.trap!=0){return failure(4);}
    let root=item(heap,cell.bits,0);if(root.error!=0 || root.trap!=0){return root;}return rx_path_read(heap,root.bits,path.bits,steps,0);
}
pub fn Outcome rx_extend(ref mut v::Vec<usize> heap,usize descriptor,usize index,ref mut usize steps){
    let base=rx_get(heap,descriptor,steps);if(base.error!=0 || base.trap!=0){return base;}let checked=item(heap,base.bits,index);if(checked.error!=0 || checked.trap!=0){return checked;}
    let root=item(heap,descriptor,0);let old=item(heap,descriptor,1);let path=allocate(heap,2,steps);if(path.error!=0){return path;}
    {let slot=v::at_mut<usize>(heap,path.bits);*slot=old.bits;}{let slot=v::at_mut<usize>(heap,path.bits+1);*slot=index;}return rx_descriptor(heap,root.bits,path.bits,0,0,steps);
}
pub fn Outcome rx_path_write(ref mut v::Vec<usize> heap,usize root,usize path,usize bits,ref mut usize steps,usize depth){
    if(depth>=64){return failure(4);}if(*steps==0){return failure(2);};*steps=*steps-1;if(path==0){return result(bits);}
    let parent=item(heap,path,0);let index=item(heap,path,1);if(parent.error!=0 || index.error!=0 || parent.trap!=0 || index.trap!=0){return failure(4);}
    let base=rx_path_read(heap,root,parent.bits,steps,0);if(base.error!=0 || base.trap!=0){return base;}let checked=item(heap,base.bits,index.bits);if(checked.error!=0 || checked.trap!=0){return checked;}
    let count=*v::at<usize>(heap,base.bits-1);let copy=allocate(heap,count,steps);if(copy.error!=0){return copy;}var usize i=0;
    while(i<count){let value=if(i==index.bits){bits}else{*v::at<usize>(heap,base.bits+i)};{let slot=v::at_mut<usize>(heap,copy.bits+i);*slot=value;}i+=1;}
    return rx_path_write(heap,root,parent.bits,copy.bits,steps,depth+1);
}
pub fn Outcome rx_put(ref mut v::Vec<usize> heap,usize descriptor,usize bits,ref mut usize steps){
    let cell=item(heap,descriptor,0);let path=item(heap,descriptor,1);if(cell.error!=0 || path.error!=0 || cell.trap!=0 || path.trap!=0){return failure(4);}
    let root=item(heap,cell.bits,0);if(root.error!=0 || root.trap!=0){return root;}let updated=rx_path_write(heap,root.bits,path.bits,bits,steps,0);if(updated.error!=0 || updated.trap!=0){return updated;}return rx_set_cell(heap,cell.bits,updated.bits);
}
pub fn Outcome rx_view_index(ref mut v::Vec<usize> heap,usize descriptor,usize index,ref mut usize steps){
    let offset=item(heap,descriptor,2);let length=item(heap,descriptor,3);if(offset.error!=0 || length.error!=0 || offset.trap!=0 || length.trap!=0){return failure(4);}if(index>=length.bits){return trapped(1);}return rx_extend(heap,descriptor,offset.bits+index,steps);
}
pub fn Outcome rx_place(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(id==0 || depth>=64){return failure(4);}if(*steps==0){return failure(2);};*steps=*steps-1;
    let e=ir::node_at(b,id-1).value;let left=ir::node(b,e.left);
    if(e.kind==2 && e.binding!=0){return rx_descriptor(heap,*v::at<usize>(locals,e.binding),0,0,0,steps);}
    if(e.kind==3){return rx_place(p,b,left,locals,values,heap,steps,depth+1);}
    if(e.kind==4 && bytes::equal(ir::node_text(b,id-1),b"*")){return result(*v::at<usize>(values,left));}
    if(e.kind!=8 && e.kind!=9 || left==0){return failure(1);}
    let rt=ir::node_at(b,left-1).value.type;
    if(e.kind==9 && rx_view(p,rt)){return rx_view_index(heap,*v::at<usize>(values,left),*v::at<usize>(values,ir::node(b,e.right)),steps);}
    let base=if(rx_reference(p,rt)){result(*v::at<usize>(values,left))}else{rx_place(p,b,left,locals,values,heap,steps,depth+1)};if(base.error!=0 || base.trap!=0){return base;}
    let index=if(e.kind==8){field_index(p,e.field.owner,e.field.id,steps)}else{result(*v::at<usize>(values,ir::node(b,e.right)))};if(index.error!=0 || index.trap!=0){return index;}return rx_extend(heap,base.bits,index.bits,steps);
}
pub fn Outcome rx_array_view(ref ir::Program p,ref ir::Body b,usize left,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);if(rx_view(p,ir::node_at(b,left-1).value.type)){return result(*v::at<usize>(values,left));}if(rt.kind!=16){return failure(1);}
    let place=rx_place(p,b,left,locals,values,heap,steps,0);if(place.error!=0 || place.trap!=0){return place;}
    let root=item(heap,place.bits,0);let path=item(heap,place.bits,1);return rx_descriptor(heap,root.bits,path.bits,0,rt.second,steps);
}
pub fn Outcome rx_bytes(ref u8[] data,ref mut v::Vec<usize> heap,ref mut usize steps){
    let a=allocate(heap,data.len,steps);if(a.error!=0){return a;}var usize i=0;while(i<data.len){{let slot=v::at_mut<usize>(heap,a.bits+i);*slot=data[i] as usize;}i+=1;}
    let root=rx_cell(heap,a.bits,steps);if(root.error!=0){return root;}return rx_descriptor(heap,root.bits,0,0,data.len,steps);
}
pub fn Outcome rx_literal(ref ir::Body b,usize id,ref mut v::Vec<usize> heap,ref mut usize steps){return rx_bytes(ir::node_cooked(b,id-1),heap,steps);}
pub fn Outcome rx_slice(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    let e=ir::node_at(b,id-1).value;let left=ir::node(b,e.left);let pair=ir::node_at(b,ir::node(b,e.right)-1).value;
    let base=rx_array_view(p,b,left,locals,values,heap,steps);if(base.error!=0 || base.trap!=0){return base;}let length=item(heap,base.bits,3);let offset=item(heap,base.bits,2);
    let usize start=if(pair.left==0){0}else{*v::at<usize>(values,ir::node(b,pair.left))};let end=if(pair.right==0){length.bits}else{*v::at<usize>(values,ir::node(b,pair.right))};
    if(start>end || end>length.bits){return trapped(1);}let root=item(heap,base.bits,0);let path=item(heap,base.bits,1);return rx_descriptor(heap,root.bits,path.bits,offset.bits+start,end-start,steps);
}

// Abstract lifetime oracle, not a host allocator. A fresh activation owns
// [active, Arena descriptor]. Reset invalidates capabilities passed to callees;
// heap cells stay address-stable until the enclosing test finishes.
// Four-cell Arenas carry a private provenance/liveness marker after the three
// SDK fields. Only the authenticated constructor can publish such a resource.
pub fn bool arena_modeled(ref v::Vec<usize> heap,usize bits){return bits!=0 && bits<=v::len<usize>(heap) && *v::at<usize>(heap,bits-1)==4 && v::len<usize>(heap)-bits>=4;}
pub fn Outcome arena_create(ref mut v::Vec<usize> heap,ref mut usize steps){
    let arena=allocate(heap,4,steps);if(arena.error!=0){return arena;}
    {let slot=v::at_mut<usize>(heap,arena.bits+1);*slot=1;}{let slot=v::at_mut<usize>(heap,arena.bits+3);*slot=1;}
    let carrier=allocate(heap,2,steps);if(carrier.error!=0){return carrier;}
    {let slot=v::at_mut<usize>(heap,carrier.bits);*slot=1;}{let slot=v::at_mut<usize>(heap,carrier.bits+1);*slot=arena.bits;}return carrier;
}
pub fn Outcome arena_drop(usize bits,ref mut v::Vec<usize> heap){
    if(!arena_modeled(heap,bits) || *v::at<usize>(heap,bits+3)!=1 || *v::at<usize>(heap,bits+2)!=0){return failure(4);}
    {let slot=v::at_mut<usize>(heap,bits+3);*slot=0;}return result(0);
}
pub fn Outcome region_begin(usize owner,ref mut v::Vec<usize> heap,ref mut usize steps){
    let arena=rx_get(heap,owner,steps);if(arena.error!=0 || arena.trap!=0){return arena;}
    if(arena_modeled(heap,arena.bits)){if(*v::at<usize>(heap,arena.bits+3)!=1 || *v::at<usize>(heap,arena.bits+2)!=0 || *v::at<usize>(heap,arena.bits+1)==0){return failure(4);}{let slot=v::at_mut<usize>(heap,arena.bits+2);*slot=1;}}
    let cell=allocate(heap,2,steps);if(cell.error!=0){return cell;}
    {let slot=v::at_mut<usize>(heap,cell.bits);*slot=1;}
    {let slot=v::at_mut<usize>(heap,cell.bits+1);*slot=owner;}return cell;
}
pub fn Outcome region_reset(usize activation,ref mut v::Vec<usize> heap,ref mut usize steps){
    let active=item(heap,activation,0);if(active.error!=0 || active.trap!=0 || active.bits!=1){return failure(4);}
    let owner=item(heap,activation,1);if(owner.error!=0 || owner.trap!=0){return failure(4);}let arena=rx_get(heap,owner.bits,steps);if(arena.error!=0 || arena.trap!=0){return arena;}
    if(arena_modeled(heap,arena.bits)){if(*v::at<usize>(heap,arena.bits+3)!=1 || *v::at<usize>(heap,arena.bits+2)!=1){return failure(4);}{let slot=v::at_mut<usize>(heap,arena.bits+2);*slot=0;}{let slot=v::at_mut<usize>(heap,arena.bits+1);*slot=if(*slot==18446744073709551615){1}else{*slot+1};}}
    {let slot=v::at_mut<usize>(heap,activation);*slot=0;}return result(0);
}
pub fn Outcome region_place(ref ir::Body b,usize id,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    var usize i=0;while(i<ir::placement_count(b)){if(*steps==0){return failure(2);};*steps=*steps-1;let x=ir::placement_at(b,i);
        if(ir::node(b,x.expression)==id){
            let capability=item(heap,*v::at<usize>(locals,x.capability),0);if(capability.error!=0 || capability.trap!=0){return capability;}
            let active=item(heap,capability.bits,0);if(active.error!=0 || active.trap!=0 || active.bits!=1){return failure(4);}
            let argument=ir::node(b,x.argument);if(argument==0){return failure(4);}
            let cell=rx_cell(heap,*v::at<usize>(values,argument),steps);if(cell.error!=0){return cell;}
            let borrow=rx_descriptor(heap,cell.bits,0,0,0,steps);if(borrow.error!=0){return borrow;}
            let result=allocate(heap,2,steps);if(result.error!=0){return result;}
            {let slot=v::at_mut<usize>(heap,result.bits);*slot=1;}
            {let slot=v::at_mut<usize>(heap,result.bits+1);*slot=borrow.bits;}return result;
        }i+=1;
    }return failure(4);
}
