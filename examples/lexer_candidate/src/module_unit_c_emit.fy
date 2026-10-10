module module_unit_c_emit;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_body_ir_lookup as lookup;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_scalar as scalar;
import c_text as w;
import std::bytes as bytes;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
fn usize parameters(ref ir::Body b){var usize count=0;var usize i=1;while(i<=ir::local_count(b)){if(ir::local_at(b,i).kind==1){count+=1;}i+=1;}return count;}
fn bool signature(ref ir::Program p,usize frame,ref mut bytes::Buffer out,ref mut w::State s){
    let b=ir::at(p,frame);let f=clean::frame_at(ir::cleanup_ir(p),frame);let result=scalar::scalar(p,f.result);
    if(result==0 || result>7 || f.kind==2){s.code=1004;return false;}
    if(!w::text(out,s,b"static ") || !w::type(out,s,result) || !w::text(out,s,b" ") || !w::function(out,s,frame) || !w::text(out,s,b"(")){return false;}
    var usize i=1;var usize count=0;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(l.kind==1){let type=scalar::scalar(p,l.type);if(type==0 || type==7){s.code=1004;return false;}if(count!=0 && !w::text(out,s,b",")){return false;}if(!w::type(out,s,type) || !w::text(out,s,b" ") || !w::local(out,s,i)){return false;}count+=1;}i+=1;}
    return (count!=0 || w::text(out,s,b"void")) && w::text(out,s,b")");
}
fn usize invocation(ref ir::Body b,usize expression,ref mut usize work){var usize i=0;while(i<ir::call_count(b)){if(!budget::charge(work,1)){return 0;}if(ir::call_at(b,i).expression==expression){return i+1;}i+=1;}return 0;}
fn usize target(ref ir::Program p,ref ir::Body b,usize caller,usize invocation,ref mut usize work){
    if(invocation==0){return 0;}let c=ir::call_at(b,invocation-1);if(!c.checked || c.kind!=1 && c.kind!=2){return 0;}
    let e=ir::node_at(b,c.expression-1).value;let f=clean::frame_at(ir::cleanup_ir(p),caller);var usize found=0;var usize i=1;
    while(i<=ir::count(p)){if(!budget::charge(work,1)){return 0;}let t=clean::frame_at(ir::cleanup_ir(p),i);if(t.module_index==c.target_module && t.declaration==c.target_declaration && (c.kind==1 && t.kind==3 || c.kind==2 && t.kind==1 && t.context==c.instance)){if(found!=0 || t.result!=e.type || (f.effects&t.effects)!=t.effects){return 0;}found=i;}i+=1;}
    if(found==0){return 0;}let body=ir::at(p,found);if(parameters(body)!=c.argument_count){return 0;}var usize position=0;i=1;
    while(i<=ir::local_count(body)){if(!budget::charge(work,1)){return 0;}let l=ir::local_at(body,i);if(l.kind==1){let a=ir::argument_at(b,c.argument_start+position);if(!a.checked || a.expected!=l.type || a.actual!=l.type || ir::node_at(b,a.expression-1).value.type!=l.type){return 0;}position+=1;}i+=1;}return found;
}
fn bool call(ref ir::Program p,ref cfg::Graph g,usize expression,usize site,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));let index=if(site==0){invocation(b,expression,work)}else{cfg::deferred_at(g,site).invocation};let callee=target(p,b,cfg::frame(g),index,work);
    if(callee==0){s.code=if(*work>=1048576){1002}else{1004};return false;}let c=ir::call_at(b,index-1);let type=scalar::scalar(p,ir::node_at(b,expression-1).value.type);
    if(site==0 && type!=7 && (!scalar::value(out,s,expression) || !w::text(out,s,b"="))){return false;}
    if(!w::function(out,s,callee) || !w::text(out,s,b"(")){return false;}var usize i=0;while(i<c.argument_count){if(i!=0 && !w::text(out,s,b",")){return false;}if(site==0){if(!scalar::value(out,s,ir::argument_at(b,c.argument_start+i).expression)){return false;}}else{if(!scalar::capture(out,s,cfg::deferred_at(g,site).capture_start+i+1)){return false;}}i+=1;}
    return w::text(out,s,b");\n");
}
fn bool store(ref ir::Program p,ref ir::Body b,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s){
    let stmt=ir::statement_at(b,n.target-1).value;let type=scalar::scalar(p,ir::node_at(b,n.source-1).value.type);var usize local=0;
    if(stmt.kind==2){local=lookup::local(b,stmt.entry);}else{let id=lookup::node(b,stmt.target);if(id!=0){let e=ir::node_at(b,id-1).value;if(e.kind==2){local=e.binding;}}}
    if(local==0 || type==0 || type==7 || ir::local_at(b,local).type!=ir::node_at(b,n.source-1).value.type){s.code=1004;return false;}
    let op=ir::assignment_operator(b,n.target-1);let plain=stmt.kind==2 || bytes::equal(op,b"=");
    if(!w::local(out,s,local) || !w::text(out,s,b"=")){return false;}
    if(plain){if(!scalar::value(out,s,n.source)){return false;}}
    else{if(op.len<2 || op[op.len-1]!=61){s.code=1001;return false;}let code=scalar::operator(op[..op.len-1]);if(code==0 || code==3 || code==4 || code>12 || type==6){s.code=1004;return false;}if(!scalar::math_start(out,s,type,code) || !w::local(out,s,local) || !w::text(out,s,b",") || !scalar::value(out,s,n.source) || !scalar::math_end(out,s,type)){return false;}}
    return w::text(out,s,b";\n");
}
fn bool function(ref ir::Program p,ref cfg::Graph g,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));if(!signature(p,cfg::frame(g),out,s) || !w::text(out,s,b"{\n")){return false;}var usize i=1;
    while(i<=ir::local_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let l=ir::local_at(b,i);let type=scalar::scalar(p,l.type);if(type==0 || type==7){s.code=1004;return false;}if(l.kind!=1 && (!w::type(out,s,type) || !w::text(out,s,b" ") || !w::local(out,s,i) || !w::text(out,s,b";\n"))){return false;}i+=1;}
    i=1;while(i<=ir::node_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let e=ir::node_at(b,i-1).value;if(e.state==1){let type=scalar::scalar(p,e.type);if(type==0){s.code=1004;return false;}if(type!=7 && (!w::type(out,s,type) || !w::text(out,s,b" ") || !scalar::value(out,s,i) || !w::text(out,s,b";\n"))){return false;}}i+=1;}
    i=1;while(i<=cfg::capture_count(g)){let c=cfg::capture_at(g,i);if(!w::type(out,s,scalar::scalar(p,c.type)) || !w::text(out,s,b" ") || !scalar::capture(out,s,i) || !w::text(out,s,b";\n")){return false;}i+=1;}
    if(!w::jump(out,s,cfg::entry(g))){return false;}i=1;
    while(i<=cfg::count(g)){if(!budget::charge(work,1)){s.code=1002;return false;}let n=cfg::at(g,i);if(!w::text(out,s,b"fy_b_") || !w::number(out,s,i) || !w::text(out,s,b":;\n")){return false;}
        if(n.kind==1){let e=ir::node_at(b,n.source-1).value;if(e.kind==6){if(e.builtin.kind!=0){s.code=1004;return false;}if(!call(p,g,n.source,0,out,s,work)){return false;}}else if(!scalar::expression(p,b,n.source,out,s)){return false;}}
        else if(n.kind==2){if(!store(p,b,n,out,s)){return false;}}
        else if(n.kind==3){if(!w::text(out,s,b"if(") || !scalar::value(out,s,n.source) || !w::text(out,s,b"){") || !w::jump(out,s,n.first) || !w::text(out,s,b"}else{") || !w::jump(out,s,n.second) || !w::text(out,s,b"}\n")){return false;}}
        else if(n.kind==6){if(!w::text(out,s,b"return") || n.source!=0 && (!w::text(out,s,b" ") || !scalar::value(out,s,n.source)) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==8){if(!w::text(out,s,b"if(!") || !scalar::value(out,s,n.source) || !w::text(out,s,b")fy_runtime_trap(5);\n")){return false;}}
        else if(n.kind==10 || n.kind==11){let type=scalar::scalar(p,ir::node_at(b,n.source-1).value.type);if(type!=7){if(n.target==0){s.code=1001;return false;}if(!scalar::value(out,s,n.source) || !w::text(out,s,b"=") || !scalar::value(out,s,n.target) || !w::text(out,s,b";\n")){return false;}}}
        else if(n.kind==12){if(!w::text(out,s,b"fy_runtime_trap(7);\n")){return false;}}
        else if(n.kind==13){if(!scalar::capture(out,s,n.target) || !w::text(out,s,b"=") || !scalar::value(out,s,n.source) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==14){if(!call(p,g,n.source,n.target,out,s,work)){return false;}}
        else if(n.kind!=7){s.code=1004;return false;}
        if(n.kind!=3 && n.kind!=6 && n.kind!=12 && !w::jump(out,s,n.first)){return false;}i+=1;
    }return w::text(out,s,b"}\n");
}
// A private, scalar-only transport from owned bodies through freshly lowered
// CFGs. No parser/source owner or source join is used. Output is transactional.
pub fn clean::Problem render(ref ir::Program p,usize entry,ref mut bytes::Buffer out){
    let checked=proof::check(p);if(checked.code!=0){return checked;}if(entry==0 || entry>ir::count(p)){return fail(1001,0,0,0);}let f=clean::frame_at(ir::cleanup_ir(p),entry);if(scalar::scalar(p,f.result)!=1 || parameters(ir::at(p,entry))!=0){return fail(1001,f.module_index,0,0);}
    var v::Vec<cfg::Graph> graphs=v::empty<cfg::Graph>();var usize work=0;var usize i=1;while(i<=ir::count(p)){let g=switch(flow::lower(p,i)){case .ok(g):{g}case .err(e):{return e;}};if(!budget::charge(ref mut work,cfg::count(ref g))){return fail(1002,0,0,0);}if(!v::push<cfg::Graph>(ref mut graphs,g).is_ok){return fail(1003,0,0,0);}i+=1;}
    var bytes::Buffer text=bytes::empty();var w::State s=w::State{.code=0};
    if(!w::text(ref mut text,ref mut s,b"/* fairyc.candidate-module-scalar-c/v1 */\n#include <stdint.h>\n#include <limits.h>\n#include <string.h>\n_Static_assert(CHAR_BIT==8 && UINTPTR_MAX==UINT64_MAX,\"requires 64-bit target\");\n_Static_assert(INT32_MIN==(-INT32_MAX-1) && INT64_MIN==(-INT64_MAX-1),\"requires two's complement\");\n_Noreturn void fy_runtime_trap(uint32_t category);\n") || !scalar::helpers(ref mut text,ref mut s)){return fail(if(s.code==3){1003}else{1002},0,0,0);}
    i=1;while(i<=ir::count(p)){if(!signature(p,i,ref mut text,ref mut s) || !w::text(ref mut text,ref mut s,b";\n")){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},clean::frame_at(ir::cleanup_ir(p),i).module_index,ir::start(ir::at(p,i)),ir::end(ir::at(p,i)));}i+=1;}
    i=1;while(i<=ir::count(p)){if(!function(p,v::at<cfg::Graph>(ref graphs,i-1),ref mut text,ref mut s,ref mut work)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},clean::frame_at(ir::cleanup_ir(p),i).module_index,ir::start(ir::at(p,i)),ir::end(ir::at(p,i)));}i+=1;}
    if(!w::text(ref mut text,ref mut s,b"int32_t fy_candidate_entry(void){return ") || !w::function(ref mut text,ref mut s,entry) || !w::text(ref mut text,ref mut s,b"();}\n#ifndef FY_CANDIDATE_NO_MAIN\nint main(void){return (int)fy_candidate_entry();}\n#endif\n")){return fail(if(s.code==3){1003}else{1002},0,0,0);}
    if(bytes::len(out)>2097152 || bytes::len(ref text)>2097152-bytes::len(out)){return fail(1002,0,0,0);}return fail(if(bytes::append(out,bytes::view(ref text)).is_ok){0}else{1003},0,0,0);
}
