module module_unit_c_emit;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_body_ir_lookup as lookup;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import module_unit_c_literals as literals;
import module_unit_c_switches as switches;
import module_unit_c_sdk as sdk;
import module_unit_c_region as lifecycle;
import ffi_symbols as ffi;
import c_text as w;
import std::bytes as bytes;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
fn bool snapshot(ref mut bytes::Buffer out,ref mut w::State s,usize id){return w::text(out,s,b"fy_snapshot_") && w::number(out,s,id);}
fn usize parameters(ref ir::Body b){var usize count=0;var usize i=1;while(i<=ir::local_count(b)){if(ir::local_at(b,i).kind==1){count+=1;}i+=1;}return count;}
fn bool signature(ref ir::Program p,usize frame,ref mut bytes::Buffer out,ref mut w::State s){
    let b=ir::at(p,frame);let f=clean::frame_at(ir::cleanup_ir(p),frame);let result=scalar::scalar(p,f.result);
    if(!w::text(out,s,b"static ") || !aggregate::type(p,f.result,out,s) || !w::text(out,s,b" ") || !w::function(out,s,frame) || !w::text(out,s,b"(")){return false;}
    if(f.kind==2){return aggregate::type(p,f.receiver_type,out,s) && w::text(out,s,b" *") && w::local(out,s,f.receiver) && w::text(out,s,b")");}
    var usize i=1;var usize count=0;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(l.kind==1){if(scalar::scalar(p,l.type)==7){s.code=1004;return false;}if(count!=0 && !w::text(out,s,b",")){return false;}if(!aggregate::type(p,l.type,out,s) || !w::text(out,s,b" ") || !w::local(out,s,i)){return false;}count+=1;}i+=1;}
    return (count!=0 || w::text(out,s,b"void")) && w::text(out,s,b")");
}
fn usize invocation(ref ir::Body b,usize expression,ref mut usize work){var usize i=0;while(i<ir::call_count(b)){if(!budget::charge(work,1)){return 0;}if(ir::call_at(b,i).expression==expression){return i+1;}i+=1;}return 0;}
fn bool foreign_type(ref ir::Program p,ref sdk::Plan plan,bool authenticated,usize id,bool result){if(id==0 || id>clean::type_count(ir::cleanup_ir(p))){return false;}if(authenticated && sdk::allowed(p,plan,id,result)){return true;}let t=clean::type_at(ir::cleanup_ir(p),id);return if(t.kind==12){t.first==0 && ffi::scalar(t.scalar,result)}else{ffi::scalar(scalar::scalar(p,id),result)};}
fn usize target(ref ir::Program p,ref ir::Body b,usize caller,usize invocation,ref mut usize work){
    if(invocation==0){return 0;}let c=ir::call_at(b,invocation-1);if(!c.checked || c.kind!=1 && c.kind!=2){return 0;}
    let e=ir::node_at(b,c.expression-1).value;let f=clean::frame_at(ir::cleanup_ir(p),caller);var usize found=0;var usize i=1;
    while(i<=ir::count(p)){if(!budget::charge(work,1)){return 0;}let t=clean::frame_at(ir::cleanup_ir(p),i);if(t.module_index==c.target_module && t.declaration==c.target_declaration && (c.kind==1 && t.kind==3 || c.kind==2 && t.kind==1 && t.context==c.instance)){if(found!=0 || t.result!=e.type || (f.effects&t.effects)!=t.effects){return 0;}found=i;}i+=1;}
    if(found==0){return 0;}let body=ir::at(p,found);if(parameters(body)!=c.argument_count){return 0;}var usize position=0;i=1;
    while(i<=ir::local_count(body)){if(!budget::charge(work,1)){return 0;}let l=ir::local_at(body,i);if(l.kind==1){let a=ir::argument_at(b,c.argument_start+position);if(!a.checked || a.expected!=l.type || ir::node_at(b,a.expression-1).value.state!=1 || a.actual!=ir::node_at(b,a.expression-1).value.type || a.actual!=l.type && (!a.shared || !aggregate::shared(p,l.type,a.actual)) || a.actual==l.type && a.shared){return 0;}position+=1;}i+=1;}return found;
}
fn usize foreign(ref ir::Program p,ref sdk::Plan plan,ref ir::Body b,usize caller,usize invocation,ref mut usize work){
    if(invocation==0){return 0;}let call=ir::call_at(b,invocation-1);if(call.kind!=1 || !call.checked){return 0;}let e=ir::node_at(b,call.expression-1).value;let frame=clean::frame_at(ir::cleanup_ir(p),caller);var usize i=0;
    while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return 0;}let c=ir::callable_at(p,i);if(c.module_index==call.target_module && c.declaration==call.target_declaration){
        let authenticated=sdk::admitted(sdk::rule(plan,i));if(!c.foreign || c.template || !c.unsafe_call || c.result!=e.type || c.parameter_count!=call.argument_count || (frame.effects&c.effects)!=c.effects || !authenticated && !ffi::valid(ir::linker_name(p,i)) || !foreign_type(p,plan,authenticated,c.result,true)){return 0;}var usize j=0;while(j<c.parameter_count){if(!budget::charge(work,1)){return 0;}let t=ir::parameter_at(p,c.parameter_start+j);let a=ir::argument_at(b,call.argument_start+j);if(!foreign_type(p,plan,authenticated,t,false) || !a.checked || a.expected!=t || a.actual!=t || ir::node_at(b,a.expression-1).value.type!=t){return 0;}j+=1;}return i+1;}i+=1;}return 0;
}
fn bool foreign_signatures(ref ir::Program p,ref sdk::Plan plan,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    var v::Vec<usize> seen=v::empty<usize>();var usize frame=1;while(frame<=ir::count(p)){let b=ir::at(p,frame);var usize i=1;while(i<=ir::call_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let id=foreign(p,plan,b,frame,i,work);if(id!=0){let c=ir::callable_at(p,id-1);var bool duplicate=false;var usize j=0;while(j<v::len<usize>(ref seen)){if(!budget::charge(work,1)){s.code=1002;return false;}let before=*v::at<usize>(ref seen,j);if(bytes::equal(ir::linker_name(p,before-1),ir::linker_name(p,id-1))){let old=ir::callable_at(p,before-1);if(old.result!=c.result || old.parameter_count!=c.parameter_count){s.code=1001;return false;}var usize k=0;while(k<c.parameter_count){if(ir::parameter_at(p,c.parameter_start+k)!=ir::parameter_at(p,old.parameter_start+k)){s.code=1001;return false;}k+=1;}duplicate=true;break;}j+=1;}
            if(!duplicate){if(!v::push<usize>(ref mut seen,id).is_ok){s.code=1003;return false;}if(sdk::rule(plan,id-1)!=0){if(!sdk::signature(p,plan,id,out,s,work)){return false;}}else{if(!w::text(out,s,b"extern ") || !aggregate::type(p,c.result,out,s) || !w::text(out,s,b" ") || !w::text(out,s,ir::linker_name(p,id-1)) || !w::text(out,s,b"(")){return false;}j=0;while(j<c.parameter_count){if(j!=0 && !w::text(out,s,b",")){return false;}if(!aggregate::type(p,ir::parameter_at(p,c.parameter_start+j),out,s)){return false;}j+=1;}if(c.parameter_count==0 && !w::text(out,s,b"void")){return false;}if(!w::text(out,s,b");\n")){return false;}}}}i+=1;}frame+=1;}return true;
}
fn bool call(ref ir::Program p,ref sdk::Plan plan,ref cfg::Graph g,usize expression,usize site,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));let index=if(site==0){invocation(b,expression,work)}else{cfg::deferred_at(g,site).invocation};let callee=target(p,b,cfg::frame(g),index,work);
    let usize external=if(callee==0){foreign(p,plan,b,cfg::frame(g),index,work)}else{0};
    if(callee==0 && external==0){s.code=if(*work>=1048576){1002}else{1004};return false;}let c=ir::call_at(b,index-1);let type=scalar::scalar(p,ir::node_at(b,expression-1).value.type);
    if(site==0 && type!=7 && (!scalar::value(out,s,expression) || !w::text(out,s,b"="))){return false;}
    if(!(if(external==0){w::function(out,s,callee)}else if(sdk::rule(plan,external-1)!=0){sdk::name(out,s,external)}else{w::text(out,s,ir::linker_name(p,external-1))}) || !w::text(out,s,b"(")){return false;}var usize i=0;while(i<c.argument_count){if(i!=0 && !w::text(out,s,b",")){return false;}if(site==0){if(!aggregate::argument(p,ir::argument_at(b,c.argument_start+i),out,s)){return false;}}else{if(!scalar::capture(out,s,cfg::deferred_at(g,site).capture_start+i+1)){return false;}}i+=1;}
    return w::text(out,s,b");\n");
}
fn bool store(ref ir::Program p,ref ir::Body b,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s){
    let stmt=ir::statement_at(b,n.target-1).value;let type=scalar::scalar(p,ir::node_at(b,n.source-1).value.type);var usize local=0;
    if(stmt.kind==2){local=lookup::local(b,stmt.entry);}else{let id=lookup::node(b,stmt.target);if(id!=0){let e=ir::node_at(b,id-1).value;if(e.kind==2){local=e.binding;}}}
    if(stmt.kind==2 && (local==0 || ir::local_at(b,local).type!=ir::node_at(b,n.source-1).value.type) || type==7){s.code=1001;return false;}
    let target=lookup::node(b,stmt.target);if(stmt.kind==4 && (target==0 || ir::node_at(b,target-1).value.type!=ir::node_at(b,n.source-1).value.type)){s.code=1001;return false;}
    let op=ir::assignment_operator(b,n.target-1);let plain=stmt.kind==2 || bytes::equal(op,b"=");
    if(!(if(stmt.kind==2){w::local(out,s,local)}else{aggregate::place(p,b,target,out,s,0)}) || !w::text(out,s,b"=")){return false;}
    if(plain){if(!scalar::value(out,s,n.source)){return false;}}
    else{if(op.len<2 || op[op.len-1]!=61){s.code=1001;return false;}let code=scalar::operator(op[..op.len-1]);if(code==0 || code==3 || code==4 || code>12 || type==0 || type==6){s.code=1004;return false;}if(!scalar::math_start(out,s,type,code) || !snapshot(out,s,n.target) || !w::text(out,s,b",") || !scalar::value(out,s,n.source) || !scalar::math_end(out,s,type)){return false;}}
    return w::text(out,s,b";\n");
}
fn bool function(ref ir::Program p,ref sdk::Plan plan,ref cfg::Graph g,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work,ref mut usize failed_start,ref mut usize failed_end){
    let b=ir::at(p,cfg::frame(g));if(!signature(p,cfg::frame(g),out,s) || !w::text(out,s,b"{\n")){return false;}var usize i=1;
    while(i<=ir::local_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let l=ir::local_at(b,i);if(scalar::scalar(p,l.type)==7){s.code=1004;return false;}if(l.kind!=1 && l.kind!=10 && l.kind!=6 && (!aggregate::type(p,l.type,out,s) || !w::text(out,s,b" ") || !w::local(out,s,i) || !w::text(out,s,b";\n"))){return false;}i+=1;}
    if(!lifecycle::locals(b,out,s,work)){return false;}
    i=1;while(i<=ir::node_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let e=ir::node_at(b,i-1).value;if(e.state==1){let type=scalar::scalar(p,e.type);if(type!=7 && (!aggregate::type(p,e.type,out,s) || !w::text(out,s,b" ") || !scalar::value(out,s,i) || !w::text(out,s,b";\n"))){return false;}}i+=1;}
    i=1;while(i<=cfg::capture_count(g)){let c=cfg::capture_at(g,i);if(!w::type(out,s,scalar::scalar(p,c.type)) || !w::text(out,s,b" ") || !scalar::capture(out,s,i) || !w::text(out,s,b";\n")){return false;}i+=1;}
    i=1;while(i<=cfg::count(g)){let n=cfg::at(g,i);if(n.kind==17 && (!aggregate::type(p,ir::node_at(b,n.source-1).value.type,out,s) || !w::text(out,s,b" ") || !snapshot(out,s,n.target) || !w::text(out,s,b";\n"))){return false;}i+=1;}
    if(!w::jump(out,s,cfg::entry(g))){return false;}i=1;
    while(i<=cfg::count(g)){if(!budget::charge(work,1)){s.code=1002;return false;}let n=cfg::at(g,i);*failed_start=n.start;*failed_end=n.end;if(!w::text(out,s,b"fy_b_") || !w::number(out,s,i) || !w::text(out,s,b":;\n")){return false;}
        if(n.kind==1){let e=ir::node_at(b,n.source-1).value;let left=lookup::node(b,e.left);let variant=e.kind==6 && left!=0 && ir::node_at(b,left-1).value.kind==28;
            if(e.kind==6 && e.builtin.kind==300){if(!lifecycle::placement(p,plan,b,n.source,out,s,work)){return false;}}
            else if(clean::type_at(ir::cleanup_ir(p),e.type).kind==19){if(!lifecycle::value(b,n.source,out,s,work)){return false;}}
            else if(e.kind==6 && !variant){if(e.builtin.kind!=0){s.code=1004;return false;}if(!call(p,plan,g,n.source,0,out,s,work)){return false;}}
            else if(e.kind==1 && literals::descriptor(clean::type_at(ir::cleanup_ir(p),e.type))){if(!literals::expression(p,b,n.source,out,s,work)){return false;}}
            else if((e.kind==1 || e.kind==23) && clean::type_at(ir::cleanup_ir(p),e.type).kind==12 || e.kind==2 || e.kind==3 || e.kind==8 || e.kind==9 || e.kind==10 || e.kind==21 || e.kind==22 || e.kind==24 || e.kind==26 || e.kind==28 || e.kind==29 || e.kind==30 || e.kind==4 && bytes::equal(ir::node_text(b,n.source-1),b"*") || variant){if(!aggregate::expression(p,b,n.source,out,s,work)){return false;}}
            else if(!scalar::expression(p,b,n.source,out,s)){return false;}}
        else if(n.kind==2){if(!store(p,b,n,out,s)){return false;}}
        else if(n.kind==3){if(!w::text(out,s,b"if(") || !scalar::value(out,s,n.source) || !w::text(out,s,b"){") || !w::jump(out,s,n.first) || !w::text(out,s,b"}else{") || !w::jump(out,s,n.second) || !w::text(out,s,b"}\n")){return false;}}
        else if(n.kind==4){if(!w::text(out,s,b"if(") || !scalar::value(out,s,n.source) || !w::text(out,s,b".fy_tag){") || !w::jump(out,s,n.first) || !w::text(out,s,b"}else{") || !w::jump(out,s,n.second) || !w::text(out,s,b"}\n")){return false;}}
        else if(n.kind==5){let d=clean::drop_at(ir::cleanup_ir(p),n.drop-1);let o=clean::operand_at(ir::cleanup_ir(p),d.operand);if(!aggregate::drop_call(p,o.type,out,s) || !w::text(out,s,b"&") || !(if(o.kind==1){w::local(out,s,o.id)}else{scalar::value(out,s,lookup::node(b,o.id))}) || !w::text(out,s,b");\n")){return false;}}
        else if(n.kind==6){if(!w::text(out,s,b"return") || n.source!=0 && (!w::text(out,s,b" ") || !scalar::value(out,s,n.source)) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==8){if(!w::text(out,s,b"if(!") || !scalar::value(out,s,n.source) || !w::text(out,s,b")fy_runtime_trap(5);\n")){return false;}}
        else if(n.kind==9){let f=clean::frame_at(ir::cleanup_ir(p),cfg::frame(g));if(!w::text(out,s,b"return (") || !aggregate::type(p,f.result,out,s) || !w::text(out,s,b"){.fy_tag=0,.fy_payload.fy_err=") || !scalar::value(out,s,n.source) || !w::text(out,s,b".fy_payload.fy_err};\n")){return false;}}
        else if(n.kind==10 || n.kind==11){let type=scalar::scalar(p,ir::node_at(b,n.source-1).value.type);if(type!=7){if(n.target==0){s.code=1001;return false;}if(!scalar::value(out,s,n.source) || !w::text(out,s,b"=") || !scalar::value(out,s,n.target) || !w::text(out,s,b";\n")){return false;}}}
        else if(n.kind==12){if(!w::text(out,s,b"fy_runtime_trap(7);\n")){return false;}}
        else if(n.kind==13){if(!scalar::capture(out,s,n.target) || !w::text(out,s,b"=") || !scalar::value(out,s,n.source) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==14){if(!call(p,plan,g,n.source,n.target,out,s,work)){return false;}}
        else if(n.kind==15 || n.kind==16){if(!lifecycle::block(b,n,out,s)){return false;}}
        else if(n.kind==17){if(!snapshot(out,s,n.target) || !w::text(out,s,b"=") || !aggregate::place(p,b,n.source,out,s,0) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==18){if(!switches::branch(p,b,n,out,s,work)){return false;}}
        else if(n.kind==19){if(!switches::seed(p,b,n,out,s,work)){return false;}}
        else if(n.kind!=7){s.code=1004;return false;}
        if(n.kind!=3 && n.kind!=4 && n.kind!=6 && n.kind!=9 && n.kind!=12 && n.kind!=18 && !w::jump(out,s,n.first)){return false;}i+=1;
    }return w::text(out,s,b"}\n");
}
// A private transport from owned bodies through freshly lowered
// CFGs. No parser/source owner or source join is used. Output is transactional.
pub fn clean::Problem render(ref ir::Program p,usize entry,ref mut bytes::Buffer out){
    let checked=proof::check(p);if(checked.code!=0){return checked;}if(entry==0 || entry>ir::count(p)){return fail(1001,0,0,0);}let f=clean::frame_at(ir::cleanup_ir(p),entry);if(scalar::scalar(p,f.result)!=1 || parameters(ir::at(p,entry))!=0){return fail(1001,f.module_index,0,0);}
    var v::Vec<cfg::Graph> graphs=v::empty<cfg::Graph>();var usize work=0;var usize i=1;while(i<=ir::count(p)){let g=switch(flow::lower(p,i)){case .ok(g):{g}case .err(e):{return e;}};if(!budget::charge(ref mut work,cfg::count(ref g))){return fail(1002,0,0,0);}if(!v::push<cfg::Graph>(ref mut graphs,g).is_ok){return fail(1003,0,0,0);}i+=1;}
    let sdk_plan=switch(sdk::build(p,ref mut work)){case .ok(p):{p}case .err(e):{return e;}};
    var bytes::Buffer text=bytes::empty();var w::State s=w::State{.code=0};
    if(!w::text(ref mut text,ref mut s,b"/* fairyc.candidate-module-owned-c/v1 */\n#include <stdint.h>\n#include <limits.h>\n#include <string.h>\n_Static_assert(CHAR_BIT==8 && UINTPTR_MAX==UINT64_MAX,\"requires 64-bit target\");\n_Static_assert(INT32_MIN==(-INT32_MAX-1) && INT64_MIN==(-INT64_MAX-1),\"requires two's complement\");\n_Noreturn void fy_runtime_trap(uint32_t category);\n") || !scalar::helpers(ref mut text,ref mut s)){return fail(if(s.code==3){1003}else{1002},0,0,0);}
    // Keep aggregate reads from live owner storage behind an opaque CRT copy
    // on MSVC. Inline aggregate copies cached a pre-view-write array element
    // under /O2; the scalar helpers keep their normal intrinsic behavior.
    if(!w::text(ref mut text,ref mut s,b"#ifdef _MSC_VER\n#pragma function(memcpy)\n__declspec(noinline)\n#endif\nstatic void fy_unit_copy(void *destination,const void *source,uintptr_t count){memcpy(destination,source,count);}\n#ifdef _MSC_VER\n#pragma intrinsic(memcpy)\n#endif\n")){return fail(if(s.code==3){1003}else{1002},0,0,0);}
    let marks=switch(aggregate::definitions(p,ref mut text,ref mut s,ref mut work)){case .ok(m):{m}case .err(e):{return fail(if(e<1000){if(e==3){1003}else{1002}}else{e},0,0,0);}};
    if(!literals::definitions(p,ref mut text,ref mut s,ref mut work)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},0,0,0);}
    if(!w::text(ref mut text,ref mut s,b"#include <stddef.h>\n") || !sdk::definitions(p,ref sdk_plan,ref mut text,ref mut s,ref mut work) || !lifecycle::definitions(p,ref sdk_plan,ref mut text,ref mut s,ref mut work) || !foreign_signatures(p,ref sdk_plan,ref mut text,ref mut s,ref mut work)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},0,0,0);}
    i=1;while(i<=ir::count(p)){if(!signature(p,i,ref mut text,ref mut s) || !w::text(ref mut text,ref mut s,b";\n")){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},clean::frame_at(ir::cleanup_ir(p),i).module_index,ir::start(ir::at(p,i)),ir::end(ir::at(p,i)));}i+=1;}
    if(!aggregate::drops(p,ref marks,ref mut text,ref mut s,ref mut work)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},0,0,0);}
    i=1;while(i<=ir::count(p)){var usize failed_start=ir::start(ir::at(p,i));var usize failed_end=ir::end(ir::at(p,i));if(!function(p,ref sdk_plan,v::at<cfg::Graph>(ref graphs,i-1),ref mut text,ref mut s,ref mut work,ref mut failed_start,ref mut failed_end)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},clean::frame_at(ir::cleanup_ir(p),i).module_index,failed_start,failed_end);}i+=1;}
    if(!w::text(ref mut text,ref mut s,b"int32_t fy_candidate_entry(void){return ") || !w::function(ref mut text,ref mut s,entry) || !w::text(ref mut text,ref mut s,b"();}\n#ifndef FY_CANDIDATE_NO_MAIN\nint main(void){return (int)fy_candidate_entry();}\n#endif\n")){return fail(if(s.code==3){1003}else{1002},0,0,0);}
    if(bytes::len(out)>2097152 || bytes::len(ref text)>2097152-bytes::len(out)){return fail(1002,0,0,0);}return fail(if(bytes::append(out,bytes::view(ref text)).is_ok){0}else{1003},0,0,0);
}
