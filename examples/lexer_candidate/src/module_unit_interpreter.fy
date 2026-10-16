module module_unit_interpreter;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_cfg_iteration as iteration;
import module_project_identified_ir as identified;
import module_constant_arithmetic as math;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import project_compiler_core as compiler;
import project_test_options as test_options;
import project_compiler_input as input;
import syntax as syntax;

// An interpreter consumes checked owned IR and verified CFGs only. It cannot
// emit C, launch a process, or invoke a host adapter. Aggregate handles refer to
// immutable per-test cells. Assignment rebuilds only the projected path.
// Errors: 1 unsupported, 2 step limit, 3 call depth, 4 invariant/allocation.
resource Location{usize frame;usize node;bytes::Buffer text;cleanup{}}
pub resource Plan{v::Vec<cfg::Graph> graphs;v::Vec<Location> locations;cleanup{}}
pub struct Outcome{u32 error;u32 trap;usize bits;}
fn Outcome result(usize bits){return Outcome{.error=0,.trap=0,.bits};}
fn Outcome failure(u32 error){return Outcome{.error,.trap=0,.bits=0};}
fn Outcome trapped(u32 trap){return Outcome{.error=0,.trap,.bits=0};}
pub fn ref u8[] detail(u32 error){let text=b"interpreter step limit exceededinterpreter call-depth limit exceededcandidate interpreter reached an unsupported value or operation";return if(error==2){text[..31]}else if(error==3){text[31..68]}else{text[68..]};}
pub fn Result<Plan,u32> prepare(ref ir::Program p){let graphs=switch(flow::lower_all(p)){case .ok(g):{g}case .err(e):{return .err(if(e.code==1002){1}else{4});}};return .ok(Plan{.graphs,.locations=v::empty<Location>()});}

// The stable test input retains canonical file names and exact UTF-8 geometry.
// No absolute paths or reconstructed source spellings enter this intrinsic.
fn Result<Plan,u32> prepare_sources(ref ir::Program p,ref input::Input sources){
    let graphs=switch(flow::lower_all(p)){case .ok(g):{g}case .err(e):{return .err(if(e.code==1002){1}else{4});}};var v::Vec<Location> locations=v::empty<Location>();
    var usize frame=1;while(frame<=ir::count(p)){let b=ir::at(p,frame);let f=clean::frame_at(ir::cleanup_ir(p),frame);var usize i=0;
        while(i<ir::node_count(b)){let e=ir::node_at(b,i).value;if(e.kind==6 && e.builtin.kind==200){
            if(f.module_index>=input::count(sources)){return .err(4);}let raw=syntax::source_view(input::source(sources,f.module_index));let file=input::file(sources,f.module_index);if(e.start>raw.len || file.len==0){return .err(4);}
            var usize line=1;var usize start=0;var usize at=0;while(at<e.start){if(raw[at]==10){line+=1;start=at+1;}at+=1;}
            var bytes::Buffer text=bytes::empty();if(!bytes::append(ref mut text,file).is_ok || !bytes::append(ref mut text,b":").is_ok || !bytes::append_usize(ref mut text,line).is_ok || !bytes::append(ref mut text,b":").is_ok || !bytes::append_usize(ref mut text,e.start-start+1).is_ok || !bytes::append(ref mut text,b"@").is_ok || !bytes::append_usize(ref mut text,e.start).is_ok){return .err(4);}
            if(!v::push<Location>(ref mut locations,Location{.frame,.node=i+1,.text}).is_ok){return .err(4);}
        }i+=1;}frame+=1;
    }return .ok(Plan{.graphs,.locations});
}
fn Outcome location_value(ref Plan plan,usize frame,usize node,ref mut v::Vec<usize> heap,ref mut usize steps){
    var usize i=0;while(i<v::len<Location>(ref plan.locations)){if(*steps==0){return failure(2);};*steps=*steps-1;let row=v::at<Location>(ref plan.locations,i);if(row.frame==frame && row.node==node){
        let data=bytes::view(ref row.text);let a=allocate(heap,data.len,steps);if(a.error!=0){return a;}var usize j=0;while(j<data.len){{let slot=v::at_mut<usize>(heap,a.bits+j);*slot=data[j] as usize;}j+=1;}let root=rx_cell(heap,a.bits,steps);if(root.error!=0){return root;}return rx_descriptor(heap,root.bits,0,0,data.len,steps);
    }i+=1;}return failure(1);
}

fn u32 scalar(ref ir::Program p,usize type){let t=clean::type_at(ir::cleanup_ir(p),type);return if(t.kind==1){t.scalar}else{0};}
fn bool admitted(ref ir::Program p,usize type){if(rx_reference(p,type) || rx_view(p,type)){return true;}let t=clean::type_at(ir::cleanup_ir(p),type);return t.kind==1 && t.scalar<=7 || t.kind>=2 && t.kind<=6 || t.kind==16;}
fn Outcome allocate(ref mut v::Vec<usize> heap,usize count,ref mut usize steps){
    if(count>999999 || *steps<count+1 || v::len<usize>(heap)>1000000-count-1){return failure(2);};
    *steps=*steps-count-1;let id=v::len<usize>(heap)+1;
    if(!v::push<usize>(heap,count).is_ok){return failure(4);}var usize i=0;while(i<count){if(!v::push<usize>(heap,0).is_ok){return failure(4);}i+=1;}return result(id);
}
fn Outcome item(ref v::Vec<usize> heap,usize id,usize index){
    if(id==0 || id>v::len<usize>(heap)){return failure(4);}let count=*v::at<usize>(heap,id-1);if(index>=count){return trapped(1);}if(count>v::len<usize>(heap)-id){return failure(4);}return result(*v::at<usize>(heap,id+index));
}
fn Outcome field_index(ref ir::Program p,usize type,usize field,ref mut usize steps){var usize i=0;while(i<ir::layout_count(p)){if(*steps==0){return failure(2);};*steps=*steps-1;let s=ir::layout_at(p,i).value;if(s.type==type){return if(field>s.field_start && field<=s.field_start+s.field_count){result(field-s.field_start-1)}else{failure(4)};}i+=1;}return failure(4);}
fn Outcome read_place(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(id==0 || depth>=64){return failure(4);}let e=ir::node_at(b,id-1).value;if(e.kind==2 && e.binding!=0){return item(heap,*v::at<usize>(locals,e.binding),0);}
    let place=rx_place(p,b,id,locals,values,heap,steps,depth);if(place.error!=0 || place.trap!=0){return place;}return rx_get(heap,place.bits,steps);
}

fn Outcome write_place(ref ir::Program p,ref ir::Body b,usize id,usize bits,ref mut v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(id==0 || depth>=64){return failure(4);}let e=ir::node_at(b,id-1).value;if(e.kind==2 && e.binding!=0){return rx_set_cell(heap,*v::at<usize>(locals,e.binding),bits);}
    let place=rx_place(p,b,id,locals,values,heap,steps,depth);if(place.error!=0 || place.trap!=0){return place;}return rx_put(heap,place.bits,bits,steps);
}

fn Outcome aggregate(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    let e=ir::node_at(b,id-1).value;let t=clean::type_at(ir::cleanup_ir(p),e.type);var usize count=if(t.kind==16){t.second}else{0};
    if(t.kind==2 || t.kind==3){var usize i=0;var bool found=false;while(i<ir::layout_count(p)){if(*steps==0){return failure(2);};*steps=*steps-1;let s=ir::layout_at(p,i).value;if(s.type==e.type){count=s.field_count;found=true;break;}i+=1;}if(!found){return failure(4);}}
    let a=allocate(heap,count,steps);if(a.error!=0){return a;}
    if(e.kind==29){let bits=*v::at<usize>(values,lookup::node(b,e.left));var usize i=0;while(i<count){let slot=v::at_mut<usize>(heap,a.bits+i);*slot=bits;i+=1;}return a;}
    var usize link=e.right;var usize position=count;while(link!=0){let row=ir::node_at(b,lookup::node(b,link)-1).value;var usize child=lookup::node(b,row.left);if(child==0 || position==0){return failure(4);}position-=1;var usize index=position;if(t.kind==2 || t.kind==3){let resolved=field_index(p,e.type,row.field.id,steps);if(resolved.error!=0){return resolved;}index=resolved.bits;}let bits=*v::at<usize>(values,child);let slot=v::at_mut<usize>(heap,a.bits+index);*slot=bits;link=row.right;}return if(position==0){a}else{failure(4)};
}

// Tagged cells contain [tag, active payload...]. Enum tags are canonical
// variant field IDs; Option/Result use 1 for some/ok and 0 for none/err.
// Published cells are immutable, including payloads copied into switch locals.
fn Outcome tagged(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    let e=ir::node_at(b,id-1).value;let t=clean::type_at(ir::cleanup_ir(p),e.type);var usize tag=0;var usize count=0;
    if(t.kind==4){if(e.field.id==0 || e.field.id>ir::field_count(p)){return failure(4);}let f=ir::field_at(p,e.field.id-1);tag=e.field.id;count=f.payload_count;}
    else{let left=lookup::node(b,e.left);let name=ir::node_text(b,if(e.kind==28){id-1}else{left-1});tag=if(bytes::equal(name,b"some") || bytes::equal(name,b"ok")){1}else{0};count=if(t.kind==5 && tag==0){0}else{1};}
    let a=allocate(heap,count+1,steps);if(a.error!=0){return a;}{let slot=v::at_mut<usize>(heap,a.bits);*slot=tag;}
    var usize link=e.right;var usize used=0;while(link!=0){if(*steps==0){return failure(2);};*steps=*steps-1;let row=ir::node_at(b,lookup::node(b,link)-1).value;let child=lookup::node(b,row.left);if(child==0 || used>=count){return failure(4);}if(!admitted(p,ir::node_at(b,child-1).value.type)){return failure(1);}var usize index=1;
        if(t.kind==4){let f=ir::field_at(p,tag-1);if(row.field.id<=f.payload_start || row.field.id>f.payload_start+f.payload_count){return failure(4);}index=row.field.id-f.payload_start;}
        {let slot=v::at_mut<usize>(heap,a.bits+index);*slot=*v::at<usize>(values,child);}used+=1;link=row.right;
    }return if(used==count){a}else{failure(4)};
}
fn Outcome bind_payload(ref ir::Body b,usize original,usize bits,ref mut v::Vec<usize> locals,ref mut v::Vec<usize> heap){let id=lookup::node(b,original);if(id==0){return failure(4);}let e=ir::node_at(b,id-1).value;let local=e.place.root;if(e.kind!=17 || local==0 || local>ir::local_count(b)){return failure(4);}let slot=v::at_mut<usize>(heap,*v::at<usize>(locals,local));*slot=bits;return result(0);}
fn Outcome switch_payload(ref ir::Program p,ref ir::Body b,cfg::Block n,ref v::Vec<usize> values,ref mut v::Vec<usize> locals,ref mut v::Vec<usize> heap,ref mut usize steps){
    let e=ir::node_at(b,n.source-1).value;let subject=lookup::node(b,e.left);let arm=ir::node_at(b,n.target-1).value;let pat=lookup::node(b,arm.left);if(subject==0 || pat==0){return failure(4);}let pattern=ir::node_at(b,pat-1).value;let t=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,subject-1).value.type);if(t.kind!=4 && t.kind!=5 && t.kind!=6){return failure(1);}let bits=*v::at<usize>(values,subject);let actual=item(heap,bits,0);if(actual.error!=0 || actual.trap!=0){return actual;}
    var usize tag=pattern.field.id;if(t.kind==5 || t.kind==6){let name=ir::node_text(b,pat-1);tag=if(bytes::equal(name,b"some") || bytes::equal(name,b"ok")){1}else{0};}else if(t.kind!=4){return failure(1);}
    if(n.kind==18){return result(if(actual.bits==tag){1}else{0});}if(actual.bits!=tag){return failure(4);}
    if(t.kind==5 || t.kind==6){if(pattern.left==0){return result(0);}let payload=item(heap,bits,1);if(payload.error!=0 || payload.trap!=0){return payload;}return bind_payload(b,pattern.left,payload.bits,locals,heap);}
    let f=ir::field_at(p,tag-1);var usize link=pattern.right;while(link!=0){if(*steps==0){return failure(2);};*steps=*steps-1;let row=ir::node_at(b,lookup::node(b,link)-1).value;if(row.field.id<=f.payload_start || row.field.id>f.payload_start+f.payload_count){return failure(4);}let payload=item(heap,bits,row.field.id-f.payload_start);if(payload.error!=0 || payload.trap!=0){return payload;}let bound=bind_payload(b,row.left,payload.bits,locals,heap);if(bound.error!=0){return bound;}link=row.right;}return result(0);
}
// Checked ownership/CFGs decide which operand is live. This executes only
// those drops, stops on traps, and retains the receiver after cleanup writes.
fn Outcome drop_value(ref identified::Program owned,ref Plan plan,usize type,usize bits,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(depth>=256){return failure(3);}if(*steps==0){return failure(2);};*steps=*steps-1;let p=identified::ir_program(owned);let g=clean::glue_at(ir::cleanup_ir(p),type);let t=clean::type_at(ir::cleanup_ir(p),type);if(!admitted(p,type)){return failure(1);}
    if(g.kind==0){return result(0);}
    if(g.kind==2 || g.kind==3){let tag=item(heap,bits,0);if(tag.error!=0 || tag.trap!=0){return tag;}if(tag.bits>1){return failure(4);}if(g.kind==2 && tag.bits==0){return result(0);}let payload=item(heap,bits,1);if(payload.error!=0 || payload.trap!=0){return payload;}return drop_value(owned,plan,if(tag.bits==1){t.first}else{t.second},payload.bits,heap,steps,depth+1);}
    if(g.kind!=1){return failure(1);}var usize current=bits;
    if(g.nonempty){var v::Vec<usize> arguments=v::empty<usize>();if(!v::push<usize>(ref mut arguments,bits).is_ok){return failure(4);}let done=execute(owned,plan,g.frame,ref arguments,heap,steps,depth+1);if(done.error!=0 || done.trap!=0){return done;}current=done.bits;}
    var usize i=g.field_count;while(i>0){i-=1;if(*steps==0){return failure(2);};*steps=*steps-1;let f=clean::field_at(ir::cleanup_ir(p),g.field_start+i);let payload=item(heap,current,i);if(payload.error!=0 || payload.trap!=0){return payload;}let done=drop_value(owned,plan,f.type,payload.bits,heap,steps,depth+1);if(done.error!=0 || done.trap!=0){return done;}}return result(0);
}

fn bool slots(ref mut v::Vec<usize> values,usize count){if(!v::reserve<usize>(values,count).is_ok){return false;}while(v::len<usize>(values)<count){if(!v::push<usize>(values,0).is_ok){return false;}}return true;}
fn Outcome computed(math::RuntimeValue r){return if(!r.supported){failure(1)}else if(r.trap!=0){trapped(r.trap)}else{result(r.bits)};}
fn Outcome literal(ref ir::Body b,usize id,u32 type){
    if(type==6){return result(if(bytes::equal(ir::node_text(b,id-1),b"true")){1}else{0});}
    let text=ir::node_cooked(b,id-1);var usize bits=0;var usize i=0;if(text.len==0){return failure(4);}while(i<text.len){let digit=text[i] as usize;if(digit<48 || digit>57 || bits>(18446744073709551615-(digit-48))/10){return failure(4);}bits=bits*10+(digit-48);i+=1;}return result(bits);
}
fn Outcome invoke(ref identified::Program owned,ref Plan plan,usize frame,usize invocation,usize site,ref v::Vec<usize> values,ref v::Vec<usize> captures,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    let p=identified::ir_program(owned);let b=ir::at(p,frame);if(invocation==0 || invocation>ir::call_count(b)){return failure(4);}let call=ir::call_at(b,invocation-1);var usize work=0;
    let target=switch(identified::call_frame(owned,frame,invocation-1,ref mut work)){case .ok(n):{n}case .err(_):{return failure(4);}};if(target==0){return failure(1);}
    var v::Vec<usize> arguments=v::empty<usize>();var usize i=0;while(i<call.argument_count){let a=ir::argument_at(b,call.argument_start+i);if(!admitted(p,a.actual)){return failure(1);}let value=if(site==0){*v::at<usize>(values,a.expression)}else{*v::at<usize>(captures,cfg::deferred_at(v::at<cfg::Graph>(ref plan.graphs,frame-1),site).capture_start+i+1)};if(!v::push<usize>(ref mut arguments,value).is_ok){return failure(4);}i+=1;}
    return execute(owned,plan,target,ref arguments,heap,steps,depth+1);
}
fn Outcome expression(ref identified::Program owned,ref Plan plan,usize frame,usize id,ref v::Vec<usize> values,ref v::Vec<usize> locals,ref v::Vec<usize> captures,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    let p=identified::ir_program(owned);let b=ir::at(p,frame);let e=ir::node_at(b,id-1).value;let type=scalar(p,e.type);if(!admitted(p,e.type)){return failure(1);}let left=lookup::node(b,e.left);let right=lookup::node(b,e.right);let op=ir::node_text(b,id-1);
    let rx_type=clean::type_at(ir::cleanup_ir(p),e.type);
    if(e.kind==1 && (rx_type.kind==9 || rx_type.kind==10)){return rx_literal(b,id,heap,steps);}
    if(e.kind==21 || e.kind==22){return if(rx_view(p,e.type)){rx_array_view(p,b,left,locals,values,heap,steps)}else{rx_place(p,b,left,locals,values,heap,steps,0)};}
    if(e.kind==26){return rx_slice(p,b,id,locals,values,heap,steps);}
    if(e.kind==4 && bytes::equal(op,b"*")){return rx_get(heap,*v::at<usize>(values,left),steps);}
    if((e.kind==8 || e.kind==9) && left!=0){let rt=ir::node_at(b,left-1).value.type;
        if(rx_view(p,rt)){let base=*v::at<usize>(values,left);if(e.kind==8){let length=item(heap,base,3);if(length.error!=0 || length.trap!=0){return length;}if(bytes::equal(op,b"len")){return length;}if(bytes::equal(op,b"is_empty")){return result(if(length.bits==0){1}else{0});}return failure(1);}
            let selected=rx_view_index(heap,base,*v::at<usize>(values,right),steps);if(selected.error!=0 || selected.trap!=0){return selected;}return rx_get(heap,selected.bits,steps);}
        if(e.kind==8 && rx_reference(p,rt)){let base=rx_get(heap,*v::at<usize>(values,left),steps);if(base.error!=0 || base.trap!=0){return base;}let index=field_index(p,e.field.owner,e.field.id,steps);if(index.error!=0){return index;}return item(heap,base.bits,index.bits);}
    }

    let carrier=clean::type_at(ir::cleanup_ir(p),e.type);
    if(carrier.kind==4 && (e.kind==28 || e.kind==10) || (carrier.kind==5 || carrier.kind==6) && (e.kind==28 || e.kind==6 && left!=0 && ir::node_at(b,left-1).value.kind==28)){return tagged(p,b,id,values,heap,steps);}
    if(e.kind==30){let tag=item(heap,*v::at<usize>(values,left),0);if(tag.error!=0 || tag.trap!=0){return tag;}if(tag.bits!=1){return failure(4);}return item(heap,*v::at<usize>(values,left),1);}
    if(e.kind==10 || e.kind==24 || e.kind==29){return aggregate(p,b,id,values,heap,steps);}
    if(e.kind==8 && e.field.id==0 && bytes::equal(op,b"is_ok")){let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);if(rt.kind!=6){return failure(1);}return item(heap,*v::at<usize>(values,left),0);}
    if(e.kind==8 || e.kind==9){let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);if(e.kind==8 && rt.kind==16 && e.field.id==0 && bytes::equal(op,b"len")){return result(rt.second);}if(e.kind==9 && rt.kind!=16 || e.kind==8 && rt.kind!=2 && rt.kind!=3){return failure(1);}let index=if(e.kind==8){field_index(p,e.field.owner,e.field.id,steps)}else{result(*v::at<usize>(values,right))};if(index.error!=0){return index;}return item(heap,*v::at<usize>(values,left),index.bits);}
    if(e.kind==1){return literal(b,id,type);}if(e.kind==2){if(e.binding==0){return failure(1);}return result(*v::at<usize>(heap,*v::at<usize>(locals,e.binding)));}if(e.kind==3){return result(*v::at<usize>(values,left));}
    if(e.kind==4){if(e.minimum){return result(if(type==1){2147483648}else{9223372036854775808});}return computed(math::runtime_prefix(type,op,*v::at<usize>(values,left)));}
    if(e.kind==5){return computed(math::runtime_binary(scalar(p,ir::node_at(b,left-1).value.type),op,*v::at<usize>(values,left),*v::at<usize>(values,right)));}
    if(e.kind==23){return computed(math::runtime_cast(scalar(p,ir::node_at(b,left-1).value.type),type,*v::at<usize>(values,left)));}
    if(e.kind==6){if(e.builtin.kind==200){return location_value(plan,frame,id,heap,steps);}if(e.builtin.kind!=0){return failure(1);}var usize i=0;while(i<ir::call_count(b)){if(ir::call_at(b,i).expression==id){return invoke(owned,plan,frame,i+1,0,values,captures,heap,steps,depth);}i+=1;}return failure(4);}
    return failure(1);
}
fn Outcome execute(ref identified::Program owned,ref Plan plan,usize frame,ref v::Vec<usize> arguments,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(depth>=256){return failure(3);}let p=identified::ir_program(owned);let b=ir::at(p,frame);let g=v::at<cfg::Graph>(ref plan.graphs,frame-1);
    if(!admitted(p,clean::frame_at(ir::cleanup_ir(p),frame).result)){return failure(1);}
    var v::Vec<usize> values=v::empty<usize>();var v::Vec<usize> locals=v::empty<usize>();var v::Vec<usize> captures=v::empty<usize>();var v::Vec<usize> snapshots=v::empty<usize>();var v::Vec<usize> iter_views=v::empty<usize>();var v::Vec<usize> iter_indices=v::empty<usize>();var v::Vec<usize> iter_bindings=v::empty<usize>();
    if(!slots(ref mut values,ir::node_count(b)+1) || !slots(ref mut locals,ir::local_count(b)+1) || !slots(ref mut captures,cfg::capture_count(g)+1) || !slots(ref mut snapshots,ir::statement_count(b)+1) || !slots(ref mut iter_views,ir::statement_count(b)+1) || !slots(ref mut iter_indices,ir::statement_count(b)+1) || !slots(ref mut iter_bindings,ir::statement_count(b)+1)){return failure(4);}
    var usize rx_local=0;while(rx_local<v::len<usize>(ref locals)){let root=rx_cell(heap,0,steps);if(root.error!=0){return root;}{let slot=v::at_mut<usize>(ref mut locals,rx_local);*slot=root.bits;}rx_local+=1;}
    var usize parameter=0;var usize i=1;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(l.kind==1 || l.kind==10){if(parameter>=v::len<usize>(arguments)){return failure(4);}if(!admitted(p,l.type)){return failure(1);}let slot=v::at_mut<usize>(heap,*v::at<usize>(ref locals,i));*slot=*v::at<usize>(arguments,parameter);parameter+=1;}i+=1;}if(parameter!=v::len<usize>(arguments)){return failure(4);}
    var usize current=cfg::entry(g);while(current!=0){if(*steps==0){return failure(2);};*steps=*steps-1;let n=cfg::at(g,current);var Outcome r=result(0);var usize next=n.first;
        if(n.kind==1){r=expression(owned,plan,frame,n.source,ref values,ref locals,ref captures,heap,steps,depth);if(r.error==0 && r.trap==0){let slot=v::at_mut<usize>(ref mut values,n.source);*slot=r.bits;}}
        else if(n.kind==2){let stmt=ir::statement_at(b,n.target-1).value;let destination=lookup::node(b,stmt.target);let usize local=if(stmt.kind==2){lookup::local(b,stmt.entry)}else{0};var usize bits=*v::at<usize>(ref values,n.source);let op=ir::assignment_operator(b,n.target-1);if(stmt.kind==4 && !bytes::equal(op,b"=")){r=computed(math::runtime_binary(scalar(p,ir::node_at(b,destination-1).value.type),op[..op.len-1],*v::at<usize>(ref snapshots,n.target),bits));bits=r.bits;}if(r.error==0 && r.trap==0){if(local!=0){let slot=v::at_mut<usize>(heap,*v::at<usize>(ref locals,local));*slot=bits;}else{r=write_place(p,b,destination,bits,ref mut locals,ref values,heap,steps,0);}}}
        else if(n.kind==3){next=if(*v::at<usize>(ref values,n.source)!=0){n.first}else{n.second};}
        else if(n.kind==4){r=item(heap,*v::at<usize>(ref values,n.source),0);next=if(r.bits==1){n.first}else{n.second};}
        else if(n.kind==5){let d=clean::drop_at(ir::cleanup_ir(p),n.drop-1);let o=clean::operand_at(ir::cleanup_ir(p),d.operand);let id=if(o.kind==1){o.id}else{lookup::node(b,o.id)};if(id==0){return failure(4);}let bits=if(o.kind==1){*v::at<usize>(heap,*v::at<usize>(ref locals,id))}else{*v::at<usize>(ref values,id)};r=drop_value(owned,plan,o.type,bits,heap,steps,depth+1);}
        else if(n.kind==6){let f=clean::frame_at(ir::cleanup_ir(p),frame);return result(if(f.kind==2){*v::at<usize>(heap,*v::at<usize>(ref locals,f.receiver))}else if(n.source==0){0}else{*v::at<usize>(ref values,n.source)});}
        else if(n.kind==9){let bits=*v::at<usize>(ref values,n.source);r=item(heap,bits,0);if(r.error!=0 || r.trap!=0){return r;}if(r.bits!=0){return failure(4);}return result(bits);}
        else if(n.kind==18 || n.kind==19){r=switch_payload(p,b,n,ref values,ref mut locals,heap,steps);if(n.kind==18){next=if(r.bits!=0){n.first}else{n.second};}}
        else if(n.kind==8){if(*v::at<usize>(ref values,n.source)==0){return trapped(6);}}
        else if(n.kind==10 || n.kind==11){if(n.target!=0){let bits=*v::at<usize>(ref values,n.target);let slot=v::at_mut<usize>(ref mut values,n.source);*slot=bits;}}
        else if(n.kind==12){return trapped(7);}
        else if(n.kind==13){let bits=*v::at<usize>(ref values,n.source);let slot=v::at_mut<usize>(ref mut captures,n.target);*slot=bits;}
        else if(n.kind==14){r=invoke(owned,plan,frame,cfg::deferred_at(g,n.target).invocation,n.target,ref values,ref captures,heap,steps,depth);}
        else if(n.kind==20){
            r=rx_array_view(p,b,n.source,ref locals,ref values,heap,steps);if(r.error==0 && r.trap==0){var usize work=0;let binding=switch(iteration::binding(p,b,frame,n.target,ref mut work)){case .ok(n):{n}case .err(_):{return failure(4);}};if(work>*steps){return failure(2);};*steps=*steps-work;
                {let slot=v::at_mut<usize>(ref mut iter_views,n.target);*slot=r.bits;}{let slot=v::at_mut<usize>(ref mut iter_indices,n.target);*slot=0;}{let slot=v::at_mut<usize>(ref mut iter_bindings,n.target);*slot=binding;}
            }
        }
        else if(n.kind==21){
            let view=*v::at<usize>(ref iter_views,n.target);let index=*v::at<usize>(ref iter_indices,n.target);let length=item(heap,view,3);if(length.error!=0 || length.trap!=0){return length;}next=if(index<length.bits){n.first}else{n.second};
            if(index<length.bits){r=rx_view_index(heap,view,index,steps);if(r.error==0 && r.trap==0){let binding=*v::at<usize>(ref iter_bindings,n.target);if(!rx_reference(p,ir::local_at(b,binding).type)){r=rx_get(heap,r.bits,steps);}if(r.error==0 && r.trap==0){r=rx_set_cell(heap,*v::at<usize>(ref locals,binding),r.bits);let slot=v::at_mut<usize>(ref mut iter_indices,n.target);*slot=index+1;}}
            }
        }
        else if(n.kind==17){r=read_place(p,b,n.source,ref locals,ref values,heap,steps,0);if(r.error==0 && r.trap==0){let slot=v::at_mut<usize>(ref mut snapshots,n.target);*slot=r.bits;}}
        else if(n.kind!=7){return failure(1);}
        if(r.error!=0 || r.trap!=0){return r;}current=next;
    }return failure(4);
}
pub fn Outcome execute_void(ref identified::Program owned,ref Plan plan,usize frame){
    let p=identified::ir_program(owned);if(frame==0 || frame>ir::count(p) || v::len<cfg::Graph>(ref plan.graphs)!=ir::count(p)){return failure(4);}if(scalar(p,clean::frame_at(ir::cleanup_ir(p),frame).result)!=7){return failure(4);}
    let arguments=v::empty<usize>();var v::Vec<usize> heap=v::empty<usize>();var usize steps=1000000;return execute(owned,plan,frame,ref arguments,ref mut heap,ref mut steps,0);
}

@effects(console) fn i32 report_failure(ref str message,i32 code){return if(io::write_stderr(message).is_ok){code}else{4};}
@effects(console) pub fn i32 tests(ref compiler::Complete complete,ref test_options::TestOptions options,ref v::Vec<test_options::TestIdentity> tests,ref v::Vec<usize> chosen,ref input::Input projected,u32 lock_code){
    var v::Vec<u32> traps=v::empty<u32>();if(v::len<usize>(chosen)==0){return test_options::test_semantic_report(options,tests,chosen,projected,ref traps,lock_code);}
    let owned=compiler::checked_program(complete);let p=identified::ir_program(owned);let plan=switch(prepare_sources(p,projected)){case .ok(p):{p}case .err(code):{return report_failure(if(code==4){"fairyc: internal semantic test preparation failed\n"}else{"fairyc: requested workspace is outside the semantic test interpreter subset: candidate interpreter CFG preparation limit exceeded\n"},if(code==4){4}else{2});}};
    var usize i=0;while(i<v::len<usize>(chosen)){let test=v::at<test_options::TestIdentity>(tests,*v::at<usize>(chosen,i));var usize frame=1;while(frame<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),frame);if(f.kind==3 && f.module_index==test_options::test_module(test) && f.declaration==test_options::test_declaration(test)+1){break;}frame+=1;}
        let outcome=execute_void(owned,ref plan,frame);if(outcome.error!=0){if(outcome.error==4){return report_failure("fairyc: internal semantic test execution failed\n",4);}var bytes::Buffer message=bytes::empty();if(!test_options::test_append(ref mut message,b"fairyc: requested workspace is outside the semantic test interpreter subset: ") || !test_options::test_append(ref mut message,test_options::test_name(test)) || !test_options::test_append(ref mut message,b": ") || !test_options::test_append(ref mut message,detail(outcome.error)) || !test_options::test_append(ref mut message,b"\n")){return 4;}return if(io::write_stderr_bytes(bytes::view(ref message)).is_ok){2}else{4};}
        if(!v::push<u32>(ref mut traps,outcome.trap).is_ok){return 4;}i+=1;
    }return test_options::test_semantic_report(options,tests,chosen,projected,ref traps,lock_code);
}

// refs-next/v1: locals hold mutable one-word root cells; values stay immutable.
// Descriptor [root cell, frozen linked path, offset, length]. A path node is
// [parent path, field/index]. No host pointer or local-vector address escapes.
fn bool rx_view(ref ir::Program p,usize type){let t=clean::type_at(ir::cleanup_ir(p),type);return t.kind==9 || t.kind==10 || t.kind==11 || t.kind==17 || t.kind==18 || t.kind==15 && t.scalar>=18 && t.scalar<=29 && t.scalar!=22 && t.scalar!=23;}
fn bool rx_reference(ref ir::Program p,usize type){let t=clean::type_at(ir::cleanup_ir(p),type);return t.kind==7 || t.kind==8;}
fn Outcome rx_cell(ref mut v::Vec<usize> heap,usize bits,ref mut usize steps){let a=allocate(heap,1,steps);if(a.error!=0){return a;}{let slot=v::at_mut<usize>(heap,a.bits);*slot=bits;}return a;}
fn Outcome rx_set_cell(ref mut v::Vec<usize> heap,usize cell,usize bits){let old=item(heap,cell,0);if(old.error!=0 || old.trap!=0){return old;}{let slot=v::at_mut<usize>(heap,cell);*slot=bits;}return result(bits);}
fn Outcome rx_descriptor(ref mut v::Vec<usize> heap,usize root,usize path,usize offset,usize length,ref mut usize steps){
    let a=allocate(heap,4,steps);if(a.error!=0){return a;}
    {let slot=v::at_mut<usize>(heap,a.bits);*slot=root;}{let slot=v::at_mut<usize>(heap,a.bits+1);*slot=path;}{let slot=v::at_mut<usize>(heap,a.bits+2);*slot=offset;}{let slot=v::at_mut<usize>(heap,a.bits+3);*slot=length;}return a;
}
fn Outcome rx_path_read(ref v::Vec<usize> heap,usize root,usize path,ref mut usize steps,usize depth){
    if(depth>=64){return failure(4);}if(*steps==0){return failure(2);};*steps=*steps-1;if(path==0){return result(root);}
    let parent=item(heap,path,0);let index=item(heap,path,1);if(parent.error!=0 || index.error!=0 || parent.trap!=0 || index.trap!=0){return failure(4);}
    let value=rx_path_read(heap,root,parent.bits,steps,depth+1);if(value.error!=0 || value.trap!=0){return value;}return item(heap,value.bits,index.bits);
}
fn Outcome rx_get(ref v::Vec<usize> heap,usize descriptor,ref mut usize steps){
    let cell=item(heap,descriptor,0);let path=item(heap,descriptor,1);if(cell.error!=0 || path.error!=0 || cell.trap!=0 || path.trap!=0){return failure(4);}
    let root=item(heap,cell.bits,0);if(root.error!=0 || root.trap!=0){return root;}return rx_path_read(heap,root.bits,path.bits,steps,0);
}
fn Outcome rx_extend(ref mut v::Vec<usize> heap,usize descriptor,usize index,ref mut usize steps){
    let base=rx_get(heap,descriptor,steps);if(base.error!=0 || base.trap!=0){return base;}let checked=item(heap,base.bits,index);if(checked.error!=0 || checked.trap!=0){return checked;}
    let root=item(heap,descriptor,0);let old=item(heap,descriptor,1);let path=allocate(heap,2,steps);if(path.error!=0){return path;}
    {let slot=v::at_mut<usize>(heap,path.bits);*slot=old.bits;}{let slot=v::at_mut<usize>(heap,path.bits+1);*slot=index;}return rx_descriptor(heap,root.bits,path.bits,0,0,steps);
}
fn Outcome rx_path_write(ref mut v::Vec<usize> heap,usize root,usize path,usize bits,ref mut usize steps,usize depth){
    if(depth>=64){return failure(4);}if(*steps==0){return failure(2);};*steps=*steps-1;if(path==0){return result(bits);}
    let parent=item(heap,path,0);let index=item(heap,path,1);if(parent.error!=0 || index.error!=0 || parent.trap!=0 || index.trap!=0){return failure(4);}
    let base=rx_path_read(heap,root,parent.bits,steps,0);if(base.error!=0 || base.trap!=0){return base;}let checked=item(heap,base.bits,index.bits);if(checked.error!=0 || checked.trap!=0){return checked;}
    let count=*v::at<usize>(heap,base.bits-1);let copy=allocate(heap,count,steps);if(copy.error!=0){return copy;}var usize i=0;
    while(i<count){let value=if(i==index.bits){bits}else{*v::at<usize>(heap,base.bits+i)};{let slot=v::at_mut<usize>(heap,copy.bits+i);*slot=value;}i+=1;}
    return rx_path_write(heap,root,parent.bits,copy.bits,steps,depth+1);
}
fn Outcome rx_put(ref mut v::Vec<usize> heap,usize descriptor,usize bits,ref mut usize steps){
    let cell=item(heap,descriptor,0);let path=item(heap,descriptor,1);if(cell.error!=0 || path.error!=0 || cell.trap!=0 || path.trap!=0){return failure(4);}
    let root=item(heap,cell.bits,0);if(root.error!=0 || root.trap!=0){return root;}let updated=rx_path_write(heap,root.bits,path.bits,bits,steps,0);if(updated.error!=0 || updated.trap!=0){return updated;}return rx_set_cell(heap,cell.bits,updated.bits);
}
fn Outcome rx_view_index(ref mut v::Vec<usize> heap,usize descriptor,usize index,ref mut usize steps){
    let offset=item(heap,descriptor,2);let length=item(heap,descriptor,3);if(offset.error!=0 || length.error!=0 || offset.trap!=0 || length.trap!=0){return failure(4);}if(index>=length.bits){return trapped(1);}return rx_extend(heap,descriptor,offset.bits+index,steps);
}
fn Outcome rx_place(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(id==0 || depth>=64){return failure(4);}if(*steps==0){return failure(2);};*steps=*steps-1;
    let e=ir::node_at(b,id-1).value;let left=lookup::node(b,e.left);
    if(e.kind==2 && e.binding!=0){return rx_descriptor(heap,*v::at<usize>(locals,e.binding),0,0,0,steps);}
    if(e.kind==3){return rx_place(p,b,left,locals,values,heap,steps,depth+1);}
    if(e.kind==4 && bytes::equal(ir::node_text(b,id-1),b"*")){return result(*v::at<usize>(values,left));}
    if(e.kind!=8 && e.kind!=9 || left==0){return failure(1);}
    let rt=ir::node_at(b,left-1).value.type;
    if(e.kind==9 && rx_view(p,rt)){return rx_view_index(heap,*v::at<usize>(values,left),*v::at<usize>(values,lookup::node(b,e.right)),steps);}
    let base=if(rx_reference(p,rt)){result(*v::at<usize>(values,left))}else{rx_place(p,b,left,locals,values,heap,steps,depth+1)};if(base.error!=0 || base.trap!=0){return base;}
    let index=if(e.kind==8){field_index(p,e.field.owner,e.field.id,steps)}else{result(*v::at<usize>(values,lookup::node(b,e.right)))};if(index.error!=0 || index.trap!=0){return index;}return rx_extend(heap,base.bits,index.bits,steps);
}
fn Outcome rx_array_view(ref ir::Program p,ref ir::Body b,usize left,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);if(rx_view(p,ir::node_at(b,left-1).value.type)){return result(*v::at<usize>(values,left));}if(rt.kind!=16){return failure(1);}
    let place=rx_place(p,b,left,locals,values,heap,steps,0);if(place.error!=0 || place.trap!=0){return place;}
    let root=item(heap,place.bits,0);let path=item(heap,place.bits,1);return rx_descriptor(heap,root.bits,path.bits,0,rt.second,steps);
}
// Copies bytes into per-test storage. Also usable by future source_location().
fn Outcome rx_bytes(ref u8[] data,ref mut v::Vec<usize> heap,ref mut usize steps){
    let a=allocate(heap,data.len,steps);if(a.error!=0){return a;}var usize i=0;while(i<data.len){{let slot=v::at_mut<usize>(heap,a.bits+i);*slot=data[i] as usize;}i+=1;}
    let root=rx_cell(heap,a.bits,steps);if(root.error!=0){return root;}return rx_descriptor(heap,root.bits,0,0,data.len,steps);
}
fn Outcome rx_literal(ref ir::Body b,usize id,ref mut v::Vec<usize> heap,ref mut usize steps){return rx_bytes(ir::node_cooked(b,id-1),heap,steps);}
fn Outcome rx_slice(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> locals,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    let e=ir::node_at(b,id-1).value;let left=lookup::node(b,e.left);let pair=ir::node_at(b,lookup::node(b,e.right)-1).value;
    let base=rx_array_view(p,b,left,locals,values,heap,steps);if(base.error!=0 || base.trap!=0){return base;}let length=item(heap,base.bits,3);let offset=item(heap,base.bits,2);
    let usize start=if(pair.left==0){0}else{*v::at<usize>(values,lookup::node(b,pair.left))};let end=if(pair.right==0){length.bits}else{*v::at<usize>(values,lookup::node(b,pair.right))};
    if(start>end || end>length.bits){return trapped(1);}let root=item(heap,base.bits,0);let path=item(heap,base.bits,1);return rx_descriptor(heap,root.bits,path.bits,offset.bits+start,end-start,steps);
}
