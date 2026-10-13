module module_unit_c_calls;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_model as cfg;
import module_unit_value_model as budget;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import module_unit_c_sdk as sdk;
import module_unit_c_callable_index as contracts;
import module_unit_c_frame_index as targets;
import module_unit_c_names as names;
import ffi_symbols as ffi;
import c_text as w;
import std::bytes as bytes;
import std::vec as v;

// Checked callable contracts, names and argument transport shared by C emitters.
pub fn usize parameters(ref ir::Body b){var usize count=0;var usize i=1;while(i<=ir::local_count(b)){if(ir::local_at(b,i).kind==1){count+=1;}i+=1;}return count;}
pub fn bool signature(ref ir::Program p,ref names::Table named,usize frame,ref mut bytes::Buffer out,ref mut w::State s,bool shared){
    let b=ir::at(p,frame);let f=clean::frame_at(ir::cleanup_ir(p),frame);let result=scalar::scalar(p,f.result);
    if(!w::text(out,s,if(shared){b"extern "}else{b"static "}) || !aggregate::type(p,f.result,out,s) || !w::text(out,s,b" ") || !names::function(named,frame,out,s) || !w::text(out,s,b"(")){return false;}
    if(f.kind==2){return aggregate::type(p,f.receiver_type,out,s) && w::text(out,s,b" *") && w::local(out,s,f.receiver) && w::text(out,s,b")");}
    var usize i=1;var usize count=0;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(l.kind==1){if(scalar::scalar(p,l.type)==7){s.code=1004;return false;}if(count!=0 && !w::text(out,s,b",")){return false;}if(!aggregate::type(p,l.type,out,s) || !w::text(out,s,b" ") || !w::local(out,s,i)){return false;}count+=1;}i+=1;}
    return (count!=0 || w::text(out,s,b"void")) && w::text(out,s,b")");
}
fn usize invocation(ref ir::Body b,usize expression,ref mut usize work){var usize i=0;while(i<ir::call_count(b)){if(!budget::charge(work,1)){return 0;}if(ir::call_at(b,i).expression==expression){return i+1;}i+=1;}return 0;}
fn bool foreign_type(ref ir::Program p,ref sdk::Plan plan,bool authenticated,usize id,bool result){if(id==0 || id>clean::type_count(ir::cleanup_ir(p))){return false;}if(authenticated && sdk::allowed(p,plan,id,result)){return true;}let t=clean::type_at(ir::cleanup_ir(p),id);return if(t.kind==12){t.first==0 && ffi::scalar(t.scalar,result)}else{ffi::scalar(scalar::scalar(p,id),result)};}
fn usize target(ref ir::Program p,ref targets::Index body_index,ref ir::Body b,usize caller,usize invocation,ref mut usize work,ref mut w::State s){
    if(invocation==0){return 0;}let c=ir::call_at(b,invocation-1);if(!c.checked || c.kind!=1 && c.kind!=2){return 0;}
    let e=ir::node_at(b,c.expression-1).value;let f=clean::frame_at(ir::cleanup_ir(p),caller);let found=targets::find(p,body_index,c.target_module,c.target_declaration,c.instance,c.kind==2,work,ref mut s.code);
    if(found!=0){let t=clean::frame_at(ir::cleanup_ir(p),found);if(t.result!=e.type || (f.effects&t.effects)!=t.effects){return 0;}}var usize i=1;
    if(found==0){return 0;}let body=ir::at(p,found);if(parameters(body)!=c.argument_count){return 0;}var usize position=0;i=1;
    while(i<=ir::local_count(body)){if(!budget::charge(work,1)){return 0;}let l=ir::local_at(body,i);if(l.kind==1){let a=ir::argument_at(b,c.argument_start+position);if(!a.checked || a.expected!=l.type || ir::node_at(b,a.expression-1).value.state!=1 || a.actual!=ir::node_at(b,a.expression-1).value.type || a.actual!=l.type && (!a.shared || !aggregate::shared(p,l.type,a.actual)) || a.actual==l.type && a.shared){return 0;}position+=1;}i+=1;}return found;
}
fn usize foreign(ref ir::Program p,ref sdk::Plan plan,ref contracts::Index indexed,ref ir::Body b,usize caller,usize invocation,ref mut usize work,ref mut w::State s){
    if(invocation==0){return 0;}let call=ir::call_at(b,invocation-1);if(call.kind!=1 || !call.checked){return 0;}let e=ir::node_at(b,call.expression-1).value;let frame=clean::frame_at(ir::cleanup_ir(p),caller);
    let id=contracts::find(p,indexed,call.target_module,call.target_declaration,work,ref mut s.code);if(id==0){return 0;}let i=id-1;let c=ir::callable_at(p,i);
    let authenticated=sdk::admitted(sdk::rule(plan,i));if(!c.foreign || c.template || !c.unsafe_call || c.result!=e.type || c.parameter_count!=call.argument_count || (frame.effects&c.effects)!=c.effects || !authenticated && !ffi::valid(ir::linker_name(p,i)) || !foreign_type(p,plan,authenticated,c.result,true)){return 0;}var usize j=0;while(j<c.parameter_count){if(!budget::charge(work,1)){s.code=1002;return 0;}let t=ir::parameter_at(p,c.parameter_start+j);let a=ir::argument_at(b,call.argument_start+j);if(!foreign_type(p,plan,authenticated,t,false) || !a.checked || a.expected!=t || a.actual!=t || ir::node_at(b,a.expression-1).value.type!=t){return 0;}j+=1;}return id;
}
pub fn bool foreign_signatures(ref ir::Program p,ref sdk::Plan plan,ref contracts::Index indexed,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    var v::Vec<usize> seen=v::empty<usize>();var usize frame=1;while(frame<=ir::count(p)){let b=ir::at(p,frame);var usize i=1;while(i<=ir::call_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let id=foreign(p,plan,indexed,b,frame,i,work,s);if(s.code!=0){return false;}if(id!=0){let c=ir::callable_at(p,id-1);var bool duplicate=false;var usize j=0;while(j<v::len<usize>(ref seen)){if(!budget::charge(work,1)){s.code=1002;return false;}let before=*v::at<usize>(ref seen,j);if(bytes::equal(ir::linker_name(p,before-1),ir::linker_name(p,id-1))){let old=ir::callable_at(p,before-1);if(old.result!=c.result || old.parameter_count!=c.parameter_count){s.code=1001;return false;}var usize k=0;while(k<c.parameter_count){if(ir::parameter_at(p,c.parameter_start+k)!=ir::parameter_at(p,old.parameter_start+k)){s.code=1001;return false;}k+=1;}duplicate=true;break;}j+=1;}
            if(!duplicate){if(!v::push<usize>(ref mut seen,id).is_ok){s.code=1003;return false;}if(sdk::rule(plan,id-1)!=0){if(!sdk::signature(p,plan,id,out,s,work)){return false;}}else{if(!w::text(out,s,b"extern ") || !aggregate::type(p,c.result,out,s) || !w::text(out,s,b" ") || !w::text(out,s,ir::linker_name(p,id-1)) || !w::text(out,s,b"(")){return false;}j=0;while(j<c.parameter_count){if(j!=0 && !w::text(out,s,b",")){return false;}if(!aggregate::type(p,ir::parameter_at(p,c.parameter_start+j),out,s)){return false;}j+=1;}if(c.parameter_count==0 && !w::text(out,s,b"void")){return false;}if(!w::text(out,s,b");\n")){return false;}}}}i+=1;}frame+=1;}return true;
}
pub fn bool call(ref ir::Program p,ref names::Table named,ref sdk::Plan plan,ref contracts::Index indexed,ref targets::Index body_index,ref cfg::Graph g,usize expression,usize site,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));let index=if(site==0){invocation(b,expression,work)}else{cfg::deferred_at(g,site).invocation};let callee=target(p,body_index,b,cfg::frame(g),index,work,s);
    let usize external=if(callee==0){foreign(p,plan,indexed,b,cfg::frame(g),index,work,s)}else{0};
    if(s.code!=0){return false;}if(callee==0 && external==0){s.code=if(*work>=1048576){1002}else{1004};return false;}let c=ir::call_at(b,index-1);let type=scalar::scalar(p,ir::node_at(b,expression-1).value.type);
    if(site==0 && type!=7 && (!scalar::value(out,s,expression) || !w::text(out,s,b"="))){return false;}
    if(!(if(external==0){names::function(named,callee,out,s)}else if(sdk::rule(plan,external-1)!=0){sdk::name(out,s,external)}else{w::text(out,s,ir::linker_name(p,external-1))}) || !w::text(out,s,b"(")){return false;}var usize i=0;while(i<c.argument_count){if(i!=0 && !w::text(out,s,b",")){return false;}if(site==0){if(!aggregate::argument(p,ir::argument_at(b,c.argument_start+i),out,s)){return false;}}else{if(!scalar::capture(out,s,cfg::deferred_at(g,site).capture_start+i+1)){return false;}}i+=1;}
    return w::text(out,s,b");\n");
}
