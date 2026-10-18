module module_unit_interpreter;
import module_unit_interpreter_values as runtime;
import module_unit_body_ir_model as ir;
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
fn runtime::Outcome location_value(ref Plan plan,usize frame,usize node,ref mut v::Vec<usize> heap,ref mut usize steps){
    var usize i=0;while(i<v::len<Location>(ref plan.locations)){if(*steps==0){return runtime::failure(2);};*steps=*steps-1;let row=v::at<Location>(ref plan.locations,i);if(row.frame==frame && row.node==node){
        let data=bytes::view(ref row.text);let a=runtime::allocate(heap,data.len,steps);if(a.error!=0){return a;}var usize j=0;while(j<data.len){{let slot=v::at_mut<usize>(heap,a.bits+j);*slot=data[j] as usize;}j+=1;}let root=runtime::rx_cell(heap,a.bits,steps);if(root.error!=0){return root;}return runtime::rx_descriptor(heap,root.bits,0,0,data.len,steps);
    }i+=1;}return runtime::failure(1);
}

// Tagged cells contain [tag, active payload...]. Enum tags are canonical
// variant field IDs; Option/Result use 1 for some/ok and 0 for none/err.
// Published cells are immutable, including payloads copied into switch locals.
fn runtime::Outcome tagged(ref ir::Program p,ref ir::Body b,usize id,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    let e=ir::node_at(b,id-1).value;let t=clean::type_at(ir::cleanup_ir(p),e.type);var usize tag=0;var usize count=0;
    if(t.kind==4){if(e.field.id==0 || e.field.id>ir::field_count(p)){return runtime::failure(4);}let f=ir::field_at(p,e.field.id-1);tag=e.field.id;count=f.payload_count;}
    else{let left=ir::node(b,e.left);let name=ir::node_text(b,if(e.kind==28){id-1}else{left-1});tag=if(bytes::equal(name,b"some") || bytes::equal(name,b"ok")){1}else{0};count=if(t.kind==5 && tag==0){0}else{1};}
    let a=runtime::allocate(heap,count+1,steps);if(a.error!=0){return a;}{let slot=v::at_mut<usize>(heap,a.bits);*slot=tag;}
    var usize link=e.right;var usize used=0;while(link!=0){if(*steps==0){return runtime::failure(2);};*steps=*steps-1;let row=ir::node_at(b,ir::node(b,link)-1).value;let child=ir::node(b,row.left);if(child==0 || used>=count){return runtime::failure(4);}if(!runtime::admitted(p,ir::node_at(b,child-1).value.type)){return runtime::failure(1);}var usize index=1;
        if(t.kind==4){let f=ir::field_at(p,tag-1);if(row.field.id<=f.payload_start || row.field.id>f.payload_start+f.payload_count){return runtime::failure(4);}index=row.field.id-f.payload_start;}
        {let slot=v::at_mut<usize>(heap,a.bits+index);*slot=*v::at<usize>(values,child);}used+=1;link=row.right;
    }return if(used==count){a}else{runtime::failure(4)};
}
fn runtime::Outcome bind_payload(ref ir::Body b,usize original,usize bits,ref mut v::Vec<usize> locals,ref mut v::Vec<usize> heap){let id=ir::node(b,original);if(id==0){return runtime::failure(4);}let e=ir::node_at(b,id-1).value;let local=e.place.root;if(e.kind!=17 || local==0 || local>ir::local_count(b)){return runtime::failure(4);}let slot=v::at_mut<usize>(heap,*v::at<usize>(locals,local));*slot=bits;return runtime::result(0);}
fn runtime::Outcome switch_payload(ref ir::Program p,ref ir::Body b,cfg::Block n,ref v::Vec<usize> values,ref mut v::Vec<usize> locals,ref mut v::Vec<usize> heap,ref mut usize steps){
    let e=ir::node_at(b,n.source-1).value;let subject=ir::node(b,e.left);let arm=ir::node_at(b,n.target-1).value;let pat=ir::node(b,arm.left);if(subject==0 || pat==0){return runtime::failure(4);}let pattern=ir::node_at(b,pat-1).value;let t=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,subject-1).value.type);if(t.kind!=4 && t.kind!=5 && t.kind!=6){return runtime::failure(1);}let bits=*v::at<usize>(values,subject);let actual=runtime::item(heap,bits,0);if(actual.error!=0 || actual.trap!=0){return actual;}
    var usize tag=pattern.field.id;if(t.kind==5 || t.kind==6){let name=ir::node_text(b,pat-1);tag=if(bytes::equal(name,b"some") || bytes::equal(name,b"ok")){1}else{0};}else if(t.kind!=4){return runtime::failure(1);}
    if(n.kind==18){return runtime::result(if(actual.bits==tag){1}else{0});}if(actual.bits!=tag){return runtime::failure(4);}
    if(t.kind==5 || t.kind==6){if(pattern.left==0){return runtime::result(0);}let payload=runtime::item(heap,bits,1);if(payload.error!=0 || payload.trap!=0){return payload;}return bind_payload(b,pattern.left,payload.bits,locals,heap);}
    let f=ir::field_at(p,tag-1);var usize link=pattern.right;while(link!=0){if(*steps==0){return runtime::failure(2);};*steps=*steps-1;let row=ir::node_at(b,ir::node(b,link)-1).value;if(row.field.id<=f.payload_start || row.field.id>f.payload_start+f.payload_count){return runtime::failure(4);}let payload=runtime::item(heap,bits,row.field.id-f.payload_start);if(payload.error!=0 || payload.trap!=0){return payload;}let bound=bind_payload(b,row.left,payload.bits,locals,heap);if(bound.error!=0){return bound;}link=row.right;}return runtime::result(0);
}
// Checked ownership/CFGs decide which operand is live. This executes only
// those drops, stops on traps, and retains the receiver after cleanup writes.
fn runtime::Outcome drop_value(ref identified::Program owned,ref Plan plan,usize type,usize bits,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(depth>=256){return runtime::failure(3);}if(*steps==0){return runtime::failure(2);};*steps=*steps-1;let p=identified::ir_program(owned);let g=clean::glue_at(ir::cleanup_ir(p),type);let t=clean::type_at(ir::cleanup_ir(p),type);if(!runtime::admitted(p,type)){return runtime::failure(1);}
    if(g.kind==0){return runtime::result(0);}
    if(g.kind==2 || g.kind==3){let tag=runtime::item(heap,bits,0);if(tag.error!=0 || tag.trap!=0){return tag;}if(tag.bits>1){return runtime::failure(4);}if(g.kind==2 && tag.bits==0){return runtime::result(0);}let payload=runtime::item(heap,bits,1);if(payload.error!=0 || payload.trap!=0){return payload;}return drop_value(owned,plan,if(tag.bits==1){t.first}else{t.second},payload.bits,heap,steps,depth+1);}
    if(g.kind!=1){return runtime::failure(1);}var usize current=bits;
    if(g.nonempty){var v::Vec<usize> arguments=v::empty<usize>();if(!v::push<usize>(ref mut arguments,bits).is_ok){return runtime::failure(4);}let done=execute(owned,plan,g.frame,ref arguments,heap,steps,depth+1);if(done.error!=0 || done.trap!=0){return done;}current=done.bits;}
    var usize i=g.field_count;while(i>0){i-=1;if(*steps==0){return runtime::failure(2);};*steps=*steps-1;let f=clean::field_at(ir::cleanup_ir(p),g.field_start+i);let payload=runtime::item(heap,current,i);if(payload.error!=0 || payload.trap!=0){return payload;}let done=drop_value(owned,plan,f.type,payload.bits,heap,steps,depth+1);if(done.error!=0 || done.trap!=0){return done;}}return runtime::result(0);
}

fn runtime::Outcome invoke(ref identified::Program owned,ref Plan plan,usize frame,usize invocation,usize site,ref v::Vec<usize> values,ref v::Vec<usize> captures,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    let p=identified::ir_program(owned);let b=ir::at(p,frame);if(invocation==0 || invocation>ir::call_count(b)){return runtime::failure(4);}let call=ir::call_at(b,invocation-1);var usize work=0;
    let target=switch(identified::call_frame(owned,frame,invocation-1,ref mut work)){case .ok(n):{n}case .err(_):{return runtime::failure(4);}};if(target==0){return runtime::failure(1);}
    var v::Vec<usize> arguments=v::empty<usize>();var usize i=0;while(i<call.argument_count){let a=ir::argument_at(b,call.argument_start+i);if(!runtime::admitted(p,a.actual)){return runtime::failure(1);}let value=if(site==0){*v::at<usize>(values,a.expression)}else{*v::at<usize>(captures,cfg::deferred_at(v::at<cfg::Graph>(ref plan.graphs,frame-1),site).capture_start+i+1)};if(!v::push<usize>(ref mut arguments,value).is_ok){return runtime::failure(4);}i+=1;}
    return execute(owned,plan,target,ref arguments,heap,steps,depth+1);
}
fn runtime::Outcome expression(ref identified::Program owned,ref Plan plan,usize frame,usize id,ref v::Vec<usize> values,ref v::Vec<usize> locals,ref v::Vec<usize> captures,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    let p=identified::ir_program(owned);let b=ir::at(p,frame);let e=ir::node_at(b,id-1).value;let type=runtime::scalar(p,e.type);if(!runtime::admitted(p,e.type)){return runtime::failure(1);}let left=ir::node(b,e.left);let right=ir::node(b,e.right);let op=ir::node_text(b,id-1);
    let rx_type=clean::type_at(ir::cleanup_ir(p),e.type);
    if(e.kind==1 && (rx_type.kind==9 || rx_type.kind==10)){return runtime::rx_literal(b,id,heap,steps);}
    if(e.kind==21 || e.kind==22){return if(runtime::rx_view(p,e.type)){runtime::rx_array_view(p,b,left,locals,values,heap,steps)}else{runtime::rx_place(p,b,left,locals,values,heap,steps,0)};}
    if(e.kind==26){return runtime::rx_slice(p,b,id,locals,values,heap,steps);}
    if(e.kind==4 && bytes::equal(op,b"*")){return runtime::rx_get(heap,*v::at<usize>(values,left),steps);}
    if((e.kind==8 || e.kind==9) && left!=0){let rt=ir::node_at(b,left-1).value.type;
        if(runtime::rx_view(p,rt)){let base=*v::at<usize>(values,left);if(e.kind==8){let length=runtime::item(heap,base,3);if(length.error!=0 || length.trap!=0){return length;}if(bytes::equal(op,b"len")){return length;}if(bytes::equal(op,b"is_empty")){return runtime::result(if(length.bits==0){1}else{0});}return runtime::failure(1);}
            let selected=runtime::rx_view_index(heap,base,*v::at<usize>(values,right),steps);if(selected.error!=0 || selected.trap!=0){return selected;}return runtime::rx_get(heap,selected.bits,steps);}
        if(e.kind==8 && runtime::rx_reference(p,rt)){let base=runtime::rx_get(heap,*v::at<usize>(values,left),steps);if(base.error!=0 || base.trap!=0){return base;}let index=runtime::field_index(p,e.field.owner,e.field.id,steps);if(index.error!=0){return index;}return runtime::item(heap,base.bits,index.bits);}
    }

    let carrier=clean::type_at(ir::cleanup_ir(p),e.type);
    if(carrier.kind==4 && (e.kind==28 || e.kind==10) || (carrier.kind==5 || carrier.kind==6) && (e.kind==28 || e.kind==6 && left!=0 && ir::node_at(b,left-1).value.kind==28)){return tagged(p,b,id,values,heap,steps);}
    if(e.kind==30){let tag=runtime::item(heap,*v::at<usize>(values,left),0);if(tag.error!=0 || tag.trap!=0){return tag;}if(tag.bits!=1){return runtime::failure(4);}return runtime::item(heap,*v::at<usize>(values,left),1);}
    if(e.kind==10 || e.kind==24 || e.kind==29){return runtime::aggregate(p,b,id,values,heap,steps);}
    if(e.kind==8 && e.field.id==0 && bytes::equal(op,b"is_ok")){let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);if(rt.kind!=6){return runtime::failure(1);}return runtime::item(heap,*v::at<usize>(values,left),0);}
    if(e.kind==8 || e.kind==9){let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);if(e.kind==8 && rt.kind==16 && e.field.id==0 && bytes::equal(op,b"len")){return runtime::result(rt.second);}if(e.kind==9 && rt.kind!=16 || e.kind==8 && rt.kind!=2 && rt.kind!=3){return runtime::failure(1);}let index=if(e.kind==8){runtime::field_index(p,e.field.owner,e.field.id,steps)}else{runtime::result(*v::at<usize>(values,right))};if(index.error!=0){return index;}return runtime::item(heap,*v::at<usize>(values,left),index.bits);}
    if(e.kind==1){return runtime::literal(b,id,type);}if(e.kind==2){if(e.binding==0){return runtime::failure(1);}return runtime::result(*v::at<usize>(heap,*v::at<usize>(locals,e.binding)));}if(e.kind==3){return runtime::result(*v::at<usize>(values,left));}
    if(e.kind==4){if(e.minimum){return runtime::result(if(type==1){2147483648}else{9223372036854775808});}return runtime::computed(math::runtime_prefix(type,op,*v::at<usize>(values,left)));}
    if(e.kind==5){return runtime::computed(math::runtime_binary(runtime::scalar(p,ir::node_at(b,left-1).value.type),op,*v::at<usize>(values,left),*v::at<usize>(values,right)));}
    if(e.kind==23){return runtime::computed(math::runtime_cast(runtime::scalar(p,ir::node_at(b,left-1).value.type),type,*v::at<usize>(values,left)));}
    if(e.kind==6){if(e.builtin.kind==300){return runtime::region_place(b,id,locals,values,heap,steps);}if(e.builtin.kind==200){return location_value(plan,frame,id,heap,steps);}if(e.builtin.kind!=0){return runtime::failure(1);}var usize i=0;while(i<ir::call_count(b)){if(ir::call_at(b,i).expression==id){return invoke(owned,plan,frame,i+1,0,values,captures,heap,steps,depth);}i+=1;}return runtime::failure(4);}
    return runtime::failure(1);
}
// Internal checked-frame entry also permits abstract Arena arguments in tests.
pub fn runtime::Outcome execute(ref identified::Program owned,ref Plan plan,usize frame,ref v::Vec<usize> arguments,ref mut v::Vec<usize> heap,ref mut usize steps,usize depth){
    if(depth>=256){return runtime::failure(3);}let p=identified::ir_program(owned);if(frame==0 || frame>ir::count(p) || v::len<cfg::Graph>(ref plan.graphs)!=ir::count(p)){return runtime::failure(4);}let b=ir::at(p,frame);let g=v::at<cfg::Graph>(ref plan.graphs,frame-1);
    if(!runtime::admitted(p,clean::frame_at(ir::cleanup_ir(p),frame).result)){return runtime::failure(1);}
    var v::Vec<usize> values=v::empty<usize>();var v::Vec<usize> locals=v::empty<usize>();var v::Vec<usize> captures=v::empty<usize>();var v::Vec<usize> snapshots=v::empty<usize>();var v::Vec<usize> iter_views=v::empty<usize>();var v::Vec<usize> iter_indices=v::empty<usize>();var v::Vec<usize> iter_bindings=v::empty<usize>();
    if(!runtime::slots(ref mut values,ir::node_count(b)+1) || !runtime::slots(ref mut locals,ir::local_count(b)+1) || !runtime::slots(ref mut captures,cfg::capture_count(g)+1) || !runtime::slots(ref mut snapshots,ir::statement_count(b)+1) || !runtime::slots(ref mut iter_views,ir::statement_count(b)+1) || !runtime::slots(ref mut iter_indices,ir::statement_count(b)+1) || !runtime::slots(ref mut iter_bindings,ir::statement_count(b)+1)){return runtime::failure(4);}
    var usize rx_local=0;while(rx_local<v::len<usize>(ref locals)){let root=runtime::rx_cell(heap,0,steps);if(root.error!=0){return root;}{let slot=v::at_mut<usize>(ref mut locals,rx_local);*slot=root.bits;}rx_local+=1;}
    var usize parameter=0;var usize i=1;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(l.kind==1 || l.kind==10){if(parameter>=v::len<usize>(arguments)){return runtime::failure(4);}if(!runtime::admitted(p,l.type)){return runtime::failure(1);}let slot=v::at_mut<usize>(heap,*v::at<usize>(ref locals,i));*slot=*v::at<usize>(arguments,parameter);parameter+=1;}i+=1;}if(parameter!=v::len<usize>(arguments)){return runtime::failure(4);}
    var v::Vec<usize> active_regions=v::empty<usize>();if(!runtime::slots(ref mut active_regions,ir::region_count(b)+1)){return runtime::failure(4);}var usize region_count=0;
    var usize current=cfg::entry(g);while(current!=0){if(*steps==0){return runtime::failure(2);};*steps=*steps-1;let n=cfg::at(g,current);var runtime::Outcome r=runtime::result(0);var usize next=n.first;
        if(n.kind==1){r=expression(owned,plan,frame,n.source,ref values,ref locals,ref captures,heap,steps,depth);if(r.error==0 && r.trap==0){let slot=v::at_mut<usize>(ref mut values,n.source);*slot=r.bits;}}
        else if(n.kind==2){let stmt=ir::statement_at(b,n.target-1).value;let destination=ir::node(b,stmt.target);let usize local=if(stmt.kind==2){ir::local(b,stmt.entry)}else{0};var usize bits=*v::at<usize>(ref values,n.source);let op=ir::assignment_operator(b,n.target-1);if(stmt.kind==4 && !bytes::equal(op,b"=")){r=runtime::computed(math::runtime_binary(runtime::scalar(p,ir::node_at(b,destination-1).value.type),op[..op.len-1],*v::at<usize>(ref snapshots,n.target),bits));bits=r.bits;}if(r.error==0 && r.trap==0){if(local!=0){let slot=v::at_mut<usize>(heap,*v::at<usize>(ref locals,local));*slot=bits;}else{r=runtime::write_place(p,b,destination,bits,ref mut locals,ref values,heap,steps,0);}}}
        else if(n.kind==3){next=if(*v::at<usize>(ref values,n.source)!=0){n.first}else{n.second};}
        else if(n.kind==4){r=runtime::item(heap,*v::at<usize>(ref values,n.source),0);next=if(r.bits==1){n.first}else{n.second};}
        else if(n.kind==5){let d=clean::drop_at(ir::cleanup_ir(p),n.drop-1);let o=clean::operand_at(ir::cleanup_ir(p),d.operand);let id=if(o.kind==1){o.id}else{ir::node(b,o.id)};if(id==0){return runtime::failure(4);}let bits=if(o.kind==1){*v::at<usize>(heap,*v::at<usize>(ref locals,id))}else{*v::at<usize>(ref values,id)};r=drop_value(owned,plan,o.type,bits,heap,steps,depth+1);}
        else if(n.kind==6){if(region_count!=0){return runtime::failure(4);}let f=clean::frame_at(ir::cleanup_ir(p),frame);return runtime::result(if(f.kind==2){*v::at<usize>(heap,*v::at<usize>(ref locals,f.receiver))}else if(n.source==0){0}else{*v::at<usize>(ref values,n.source)});}
        else if(n.kind==9){if(region_count!=0){return runtime::failure(4);}let bits=*v::at<usize>(ref values,n.source);r=runtime::item(heap,bits,0);if(r.error!=0 || r.trap!=0){return r;}if(r.bits!=0){return runtime::failure(4);}return runtime::result(bits);}
        else if(n.kind==18 || n.kind==19){r=switch_payload(p,b,n,ref values,ref mut locals,heap,steps);if(n.kind==18){next=if(r.bits!=0){n.first}else{n.second};}}
        else if(n.kind==8){if(*v::at<usize>(ref values,n.source)==0){return runtime::trapped(6);}}
        else if(n.kind==10 || n.kind==11){if(n.target!=0){let bits=*v::at<usize>(ref values,n.target);let slot=v::at_mut<usize>(ref mut values,n.source);*slot=bits;}}
        else if(n.kind==12){return runtime::trapped(7);}
        else if(n.kind==13){let bits=*v::at<usize>(ref values,n.source);let slot=v::at_mut<usize>(ref mut captures,n.target);*slot=bits;}
        else if(n.kind==14){r=invoke(owned,plan,frame,cfg::deferred_at(g,n.target).invocation,n.target,ref values,ref captures,heap,steps,depth);}
        else if(n.kind==15){
            if(*v::at<usize>(ref active_regions,n.target)!=0){return runtime::failure(4);}r=runtime::region_begin(*v::at<usize>(ref values,n.source),heap,steps);
            if(r.error==0 && r.trap==0){let scope=ir::region_at(b,n.target-1);let stored=runtime::rx_set_cell(heap,*v::at<usize>(ref locals,scope.capability),r.bits);if(stored.error!=0 || stored.trap!=0){return stored;}{let slot=v::at_mut<usize>(ref mut active_regions,n.target);*slot=r.bits;}region_count+=1;}
        }
        else if(n.kind==16){
            let reset=ir::reset_at(b,n.target-1);let activation=*v::at<usize>(ref active_regions,reset.region_id);if(activation==0 || region_count==0){return runtime::failure(4);}r=runtime::region_reset(activation,heap);
            if(r.error==0 && r.trap==0){{let slot=v::at_mut<usize>(ref mut active_regions,reset.region_id);*slot=0;}region_count-=1;}
        }
        else if(n.kind==20){
            r=runtime::rx_array_view(p,b,n.source,ref locals,ref values,heap,steps);if(r.error==0 && r.trap==0){var usize work=0;let binding=switch(iteration::binding(p,b,frame,n.target,ref mut work)){case .ok(n):{n}case .err(_):{return runtime::failure(4);}};if(work>*steps){return runtime::failure(2);};*steps=*steps-work;
                {let slot=v::at_mut<usize>(ref mut iter_views,n.target);*slot=r.bits;}{let slot=v::at_mut<usize>(ref mut iter_indices,n.target);*slot=0;}{let slot=v::at_mut<usize>(ref mut iter_bindings,n.target);*slot=binding;}
            }
        }
        else if(n.kind==21){
            let view=*v::at<usize>(ref iter_views,n.target);let index=*v::at<usize>(ref iter_indices,n.target);let length=runtime::item(heap,view,3);if(length.error!=0 || length.trap!=0){return length;}next=if(index<length.bits){n.first}else{n.second};
            if(index<length.bits){r=runtime::rx_view_index(heap,view,index,steps);if(r.error==0 && r.trap==0){let binding=*v::at<usize>(ref iter_bindings,n.target);if(!runtime::rx_reference(p,ir::local_at(b,binding).type)){r=runtime::rx_get(heap,r.bits,steps);}if(r.error==0 && r.trap==0){r=runtime::rx_set_cell(heap,*v::at<usize>(ref locals,binding),r.bits);let slot=v::at_mut<usize>(ref mut iter_indices,n.target);*slot=index+1;}}
            }
        }
        else if(n.kind==17){r=runtime::read_place(p,b,n.source,ref locals,ref values,heap,steps,0);if(r.error==0 && r.trap==0){let slot=v::at_mut<usize>(ref mut snapshots,n.target);*slot=r.bits;}}
        else if(n.kind!=7){return runtime::failure(1);}
        if(r.error!=0 || r.trap!=0){return r;}current=next;
    }return runtime::failure(4);
}
pub fn runtime::Outcome execute_void(ref identified::Program owned,ref Plan plan,usize frame){
    let p=identified::ir_program(owned);if(frame==0 || frame>ir::count(p) || v::len<cfg::Graph>(ref plan.graphs)!=ir::count(p)){return runtime::failure(4);}if(runtime::scalar(p,clean::frame_at(ir::cleanup_ir(p),frame).result)!=7){return runtime::failure(4);}
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

// Copies bytes into per-test storage. Also usable by future source_location().
