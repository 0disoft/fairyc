module module_unit_body_ir_closures;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import copied_closure_tables as private;
import copied_closure_admission as admission;
import std::bytes as bytes;

pub fn bool leaf(ref clean::Program p,usize type){if(type==0 || type>clean::type_count(p)){return false;}let t=clean::type_at(p,type);return clean::closures(p) && t.kind==23 && t.scalar==0 && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0;}
pub fn bool i32(ref clean::Program p,usize type){if(type==0 || type>clean::type_count(p)){return false;}let t=clean::type_at(p,type);return t.kind==1 && t.scalar==1 && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0;}
fn clean::Problem fail(usize module_index,usize start,usize end,ref usize work){return clean::Problem{.code=if(*work>=1048576){1002}else{1001},.module_index,.start,.end};}
fn bool arithmetic(ref u8[] op){return bytes::equal(op,b"+") || bytes::equal(op,b"-") || bytes::equal(op,b"*") || bytes::equal(op,b"/") || bytes::equal(op,b"%");}
// The detached representation retains the exact bounded pure helper family.
// Current-source identity is separately proved at the publication boundary.
fn bool pure(ref ir::Program p,usize frame,usize captures,ref mut usize work){
    let b=ir::at(p,frame);let cleaned=ir::cleanup_ir(p);var usize parameters=0;var usize locals=0;var usize returns=0;var usize i=1;
    if(ir::node_count(b)>64 || ir::call_count(b)!=0 || ir::region_count(b)!=0 || ir::reset_count(b)!=0 || ir::placement_count(b)!=0){return false;}
    while(i<=ir::local_count(b)){if(!budget::charge(work,1)){return false;}let l=ir::local_at(b,i);if(!i32(cleaned,l.type) || l.expected!=0 && l.expected!=l.type){return false;}if(l.kind==1){parameters+=1;}else if(l.kind==2){locals+=1;if(l.initializer==0){return false;}}else{return false;}i+=1;}
    if(parameters!=captures+1 || locals>16){return false;}i=0;
    while(i<ir::node_count(b)){if(!budget::charge(work,1)){return false;}let n=ir::node_at(b,i);let e=n.value;
        if(e.state!=1 || !i32(cleaned,e.type) || e.expected!=0 && e.expected!=e.type || e.target!=0 || e.builtin.kind!=0 || e.field.id!=0){return false;}
        if(e.kind==1){if(n.literal_kind!=5){return false;}}
        else if(e.kind==2){if(e.binding==0 || e.binding>ir::local_count(b)){return false;}let l=ir::local_at(b,e.binding);if(e.place.kind!=1 || e.place.root!=e.binding || e.place.mutable || l.kind==2 && e.start<l.end){return false;}}
        else if(e.kind==3){if(e.left==0 || e.right!=0){return false;}}
        else if(e.kind==4){let op=ir::node_text(b,i);if(e.left==0 || e.right!=0 || !(bytes::equal(op,b"+") || bytes::equal(op,b"-"))){return false;}}
        else if(e.kind==5){if(e.left==0 || e.right==0 || !arithmetic(ir::node_text(b,i))){return false;}}
        else{return false;}i+=1;
    }
    i=0;var usize declarations=0;var usize blocks=0;var usize links=0;
    while(i<ir::statement_count(b)){if(!budget::charge(work,1)){return false;}let s=ir::statement_at(b,i).value;
        if(s.kind==1){blocks+=1;if(s.second!=locals+1 || s.expression!=0){return false;}}
        else if(s.kind==7){links+=1;}
        else if(s.kind==3){returns+=1;let value=ir::node(b,s.expression);if(value==0 || !i32(cleaned,ir::node_at(b,value-1).value.type)){return false;}}
        else if(s.kind==2){let local=ir::local(b,s.entry);if(local==0 || ir::local_at(b,local).kind!=2 || ir::local_at(b,local).initializer!=s.expression){return false;}declarations+=1;}
        else{return false;}i+=1;
    }return blocks==1 && returns==1 && declarations==locals && links==locals+1;
}
// A private literal adapter never targets a generic, unsafe, effectful or foreign
// callable, nor an unrelated module/frame with the same declaration number.
pub fn usize helper(ref ir::Program p,usize caller_frame,usize declaration,usize captures,ref mut usize work){
    if(caller_frame==0 || caller_frame>ir::count(p) || declaration==0 || captures>4 || !clean::closures(ir::cleanup_ir(p))){return 0;}let caller=clean::frame_at(ir::cleanup_ir(p),caller_frame);var usize found=0;var usize i=1;
    while(i<=ir::count(p)){if(!budget::charge(work,1)){return 0;}let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.module_index==caller.module_index && f.declaration==declaration){if(found!=0 || f.kind!=3 || f.effects!=0 || !i32(ir::cleanup_ir(p),f.result)){return 0;}found=i;}i+=1;}
    if(found==0){return 0;}var usize contracts=0;i=0;
    while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return 0;}let c=ir::callable_at(p,i);if(c.module_index==caller.module_index && c.declaration==declaration){
        if(c.template || c.foreign || c.unsafe_call || c.effects!=0 || c.must_use || c.parameter_count!=captures+1 || !i32(ir::cleanup_ir(p),c.result) || c.parameter_start>ir::parameter_count(p) || c.parameter_count>ir::parameter_count(p)-c.parameter_start){return 0;}
        var usize j=0;while(j<c.parameter_count){if(!budget::charge(work,1) || !i32(ir::cleanup_ir(p),ir::parameter_at(p,c.parameter_start+j))){return 0;}j+=1;}contracts+=1;
    }i+=1;}return if(contracts==1 && pure(p,found,captures,work)){found}else{0};
}
fn bool construction(ref ir::Program p,ref ir::Body b,usize frame,usize index,ref mut usize work){
    let n=ir::node_at(b,index);let e=n.value;let cleaned=ir::cleanup_ir(p);let callee=ir::node(b,e.left);
    if(!clean::closures(cleaned) || e.state!=1 || !leaf(cleaned,e.type) || e.expected!=0 && e.expected!=e.type || e.binding!=0 || e.builtin.kind!=0 || e.field.id!=0 || e.place.kind!=0 || callee==0 || e.target==0){return false;}
    let target=ir::node_at(b,callee-1);if(target.value.kind!=2 || target.value.state!=2 || target.value.type!=0 || target.value.binding!=0 || target.syntax.token!=private::helper_flag()+e.target || !admission::reserved(ir::node_text(b,callee-1))){return false;}
    var usize count=0;var usize link=e.right;var usize previous=0;
    while(link!=0){if(count>=4 || !budget::charge(work,1)){return false;}let id=ir::node(b,link);if(id==0){return false;}let pair=ir::node_at(b,id-1).value;let child=ir::node(b,pair.left);
        if(pair.kind!=7 || pair.right>=link || child==0){return false;}let capture=ir::node_at(b,child-1);let v=capture.value;
        if(v.kind!=2 || v.state!=1 || !i32(cleaned,v.type) || v.binding==0 || v.binding>ir::local_count(b) || v.place.kind!=1 || v.place.root!=v.binding || v.place.mutable){return false;}
        let local=ir::local_at(b,v.binding);if(local.kind!=2 || local.type!=v.type || local.initializer==0 || local.source_index==0 || capture.syntax.token!=private::snapshot_flag()+local.source_index || previous!=0 && local.source_index>=previous){return false;}previous=local.source_index;count+=1;link=pair.right;
    }return helper(p,frame,e.target,count,work)!=0;
}
fn bool invocation(ref ir::Program p,ref ir::Body b,usize index,ref mut usize work){
    let c=ir::call_at(b,index);let cleaned=ir::cleanup_ir(p);
    if(!clean::closures(cleaned) || !c.checked || c.target_module!=0 || c.target_declaration!=0 || c.instance!=0 || c.argument_count!=1 || c.expression==0 || c.expression>ir::node_count(b) || c.argument_start>=ir::argument_count(b) || !budget::charge(work,1)){return false;}
    let e=ir::node_at(b,c.expression-1).value;let id=ir::node(b,e.left);if(e.kind!=6 || e.state!=1 || !i32(cleaned,e.type) || e.target!=0 || e.builtin.kind!=0 || id==0){return false;}let callee=ir::node_at(b,id-1).value;
    if(callee.kind!=2 || callee.state!=1 || !leaf(cleaned,callee.type) || callee.binding==0 || callee.binding>ir::local_count(b) || callee.place.kind!=1 || callee.place.root!=callee.binding || callee.place.mutable){return false;}
    let local=ir::local_at(b,callee.binding);if((local.kind!=1 && local.kind!=2) || local.type!=callee.type || callee.place.scope!=local.scope){return false;}
    if(local.kind==2){let initial=ir::node(b,local.initializer);if(initial==0 || callee.start<local.end || ir::node_at(b,initial-1).value.state!=1 || ir::node_at(b,initial-1).value.type!=local.type){return false;}}let a=ir::argument_at(b,c.argument_start);
    return a.invocation==index+1 && a.position==0 && a.checked && !a.shared && a.label_start==0 && a.label_end==0 && i32(cleaned,a.expected) && a.actual==a.expected && a.expression!=0 && a.expression<=ir::node_count(b) && ir::node_at(b,a.expression-1).value.type==a.actual;
}
pub fn clean::Problem check_body(ref ir::Program p,ref ir::Body b,usize frame,ref mut usize work){
    if(frame==0 || frame>ir::count(p)){return fail(0,0,0,work);}let f=clean::frame_at(ir::cleanup_ir(p),frame);var usize i=0;
    while(i<ir::node_count(b)){if(!budget::charge(work,1)){return fail(f.module_index,0,0,work);}let e=ir::node_at(b,i).value;if(e.kind==36 && !construction(p,b,frame,i,work)){return fail(f.module_index,e.start,e.end,work);}i+=1;}
    i=1;while(i<=ir::local_count(b)){if(!budget::charge(work,1)){return fail(f.module_index,0,0,work);}let l=ir::local_at(b,i);if(leaf(ir::cleanup_ir(p),l.type)){
        if(l.kind==1){if(l.initializer!=0){return fail(f.module_index,l.start,l.end,work);}}
        else if(l.kind==2){let initial=ir::node(b,l.initializer);if(initial==0 || ir::node_at(b,initial-1).value.state!=1 || ir::node_at(b,initial-1).value.type!=l.type){return fail(f.module_index,l.start,l.end,work);}}
        else{return fail(f.module_index,l.start,l.end,work);}
    }i+=1;}
    i=0;while(i<ir::call_count(b)){if(!budget::charge(work,1)){return fail(f.module_index,0,0,work);}let c=ir::call_at(b,i);if(c.kind==4 && !invocation(p,b,i,work)){return fail(f.module_index,c.start,c.end,work);}i+=1;}return clean::Problem{.code=0,.module_index=0,.start=0,.end=0};
}
pub fn clean::Problem check(ref ir::Program p,usize frame,ref mut usize work){if(frame==0 || frame>ir::count(p)){return fail(0,0,0,work);}return check_body(p,ir::at(p,frame),frame,work);}
fn bool carrier(ref ir::Program p,usize type,ref mut usize work){var usize i=0;while(i<ir::layout_count(p)){if(!budget::charge(work,1)){return true;}let s=ir::layout_at(p,i).value;if(s.type==type){var usize j=0;while(j<s.field_count){if(!budget::charge(work,1)){return true;}if(leaf(ir::cleanup_ir(p),ir::field_at(p,s.field_start+j).type)){return true;}j+=1;}}i+=1;}return false;}
// Ordinary direct struct storage is the sole nominal carrier. Closing its
// wrapper/FFI paths is independent of the source-level annotation checker.
pub fn clean::Problem storage(ref ir::Program p,ref mut usize work){
    let cleaned=ir::cleanup_ir(p);if(!clean::closures(cleaned)){return clean::Problem{.code=0,.module_index=0,.start=0,.end=0};}var usize i=0;
    while(i<ir::layout_count(p)){let s=ir::layout_at(p,i).value;var usize j=0;while(j<s.field_count){if(!budget::charge(work,1)){return fail(s.module_index,s.start,s.end,work);}let f=ir::field_at(p,s.field_start+j);
        if(leaf(cleaned,f.type)){if(clean::type_at(cleaned,s.type).kind!=2 || s.repr_c){return fail(s.module_index,f.start,f.end,work);}}
        else if(carrier(p,f.type,work)){return fail(s.module_index,f.start,f.end,work);}j+=1;}i+=1;}
    i=1;while(i<=clean::type_count(cleaned)){if(!budget::charge(work,1)){return fail(0,0,0,work);}let t=clean::type_at(cleaned,i);if((t.kind>=5 && t.kind<=8 || t.kind==12 || t.kind==16 || t.kind==17 || t.kind==18) && carrier(p,t.first,work) || t.kind==6 && carrier(p,t.second,work)){return fail(t.module_index,0,0,work);}i+=1;}
    i=0;while(i<ir::callable_count(p)){let c=ir::callable_at(p,i);if(!budget::charge(work,1)){return fail(c.module_index,0,0,work);}if(c.foreign){if(leaf(cleaned,c.result) || carrier(p,c.result,work)){return fail(c.module_index,0,0,work);}var usize j=0;while(j<c.parameter_count){let t=ir::parameter_at(p,c.parameter_start+j);if(!budget::charge(work,1) || leaf(cleaned,t) || carrier(p,t,work)){return fail(c.module_index,0,0,work);}j+=1;}}i+=1;}
    return clean::Problem{.code=0,.module_index=0,.start=0,.end=0};
}
