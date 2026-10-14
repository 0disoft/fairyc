module module_unit_c_function;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cfg_model as cfg;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import module_unit_c_literals as literals;
import module_unit_c_switches as switches;
import module_unit_c_sdk as sdk;
import module_unit_c_bytes as byte_compare;
import module_unit_c_callable_index as contracts;
import module_unit_c_frame_index as targets;
import module_unit_c_region as lifecycle;
import module_unit_c_raw_views as raw_views;
import module_unit_c_storage as storage;
import module_unit_c_iteration as iteration;
import module_unit_c_borrowed_arrays as borrowed_arrays;
import module_unit_c_names as names;
import module_unit_c_calls as calls;
import c_text as w;
import std::bytes as bytes;

fn bool snapshot(ref mut bytes::Buffer out,ref mut w::State s,usize id){return w::text(out,s,b"fy_snapshot_") && w::number(out,s,id);}
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
pub fn bool render(ref ir::Program p,ref names::Table named,ref sdk::Plan plan,ref contracts::Index indexed,ref targets::Index body_index,ref cfg::Graph g,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work,ref mut usize failed_start,ref mut usize failed_end,bool shared){
    let b=ir::at(p,cfg::frame(g));if(!calls::signature(p,named,cfg::frame(g),out,s,shared) || !w::text(out,s,b"{\n")){return false;}var usize i=1;
    if(cfg::frame(g)==sdk::byte_equal_frame(plan)){return byte_compare::body(b,out,s);}
    let invocations=switch(calls::invocation_index(b,work)){case .ok(p):{p}case .err(code):{s.code=code;return false;}};
    let arrays=switch(borrowed_arrays::build(p,b,g,work)){case .ok(a):{a}case .err(code):{s.code=code;return false;}};
    while(i<=ir::local_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let l=ir::local_at(b,i);if(scalar::scalar(p,l.type)==7){s.code=1004;return false;}if(l.kind!=1 && l.kind!=10 && l.kind!=6 && (!aggregate::type(p,l.type,out,s) || !w::text(out,s,b" ") || !w::local(out,s,i) || !w::text(out,s,b";\n"))){return false;}i+=1;}
    if(!iteration::locals(g,out,s,work) || !lifecycle::locals(b,out,s,work)){return false;}
    i=1;while(i<=ir::node_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let e=ir::node_at(b,i-1).value;if(e.state==1 && !borrowed_arrays::omitted(ref arrays,i)){let type=scalar::scalar(p,e.type);if(type!=7 && (!aggregate::type(p,e.type,out,s) || !w::text(out,s,b" ") || !scalar::value(out,s,i) || !w::text(out,s,b";\n"))){return false;}}i+=1;}
    i=1;while(i<=cfg::capture_count(g)){let c=cfg::capture_at(g,i);if(!w::type(out,s,scalar::scalar(p,c.type)) || !w::text(out,s,b" ") || !scalar::capture(out,s,i) || !w::text(out,s,b";\n")){return false;}i+=1;}
    i=1;while(i<=cfg::count(g)){let n=cfg::at(g,i);if(n.kind==17 && (!aggregate::type(p,ir::node_at(b,n.source-1).value.type,out,s) || !w::text(out,s,b" ") || !snapshot(out,s,n.target) || !w::text(out,s,b";\n"))){return false;}i+=1;}
    if(!w::jump(out,s,cfg::entry(g))){return false;}i=1;
    while(i<=cfg::count(g)){if(!budget::charge(work,1)){s.code=1002;return false;}let n=cfg::at(g,i);*failed_start=n.start;*failed_end=n.end;if(!w::text(out,s,b"fy_b_") || !w::number(out,s,i) || !w::text(out,s,b":;\n")){return false;}
        if(n.kind==1){let e=ir::node_at(b,n.source-1).value;let left=lookup::node(b,e.left);let variant=e.kind==6 && left!=0 && ir::node_at(b,left-1).value.kind==28;
            if(borrowed_arrays::omitted(ref arrays,n.source)){}
            else if(e.kind==8 && borrowed_arrays::field_omitted(ref arrays,left)){if(!borrowed_arrays::read_field(p,b,n.source,out,s)){return false;}}
            else if(e.kind==9 && borrowed_arrays::omitted(ref arrays,left)){if(!borrowed_arrays::read(p,b,n.source,out,s)){return false;}}
            else if(e.kind==6 && e.builtin.kind>=101 && e.builtin.kind<=107){if(!storage::expression(p,b,n.source,out,s,work)){return false;}}
            else if(e.kind==6 && e.builtin.kind>=9 && e.builtin.kind<=12){if(!raw_views::expression(p,b,n.source,out,s,work)){return false;}}
            else if(e.kind==6 && e.builtin.kind==300){if(!lifecycle::placement(p,plan,b,n.source,out,s,work)){return false;}}
            else if(clean::type_at(ir::cleanup_ir(p),e.type).kind==19){if(!lifecycle::value(b,n.source,out,s,work)){return false;}}
            else if(e.kind==6 && !variant){if(e.builtin.kind!=0){s.code=1004;return false;}if(!calls::call(p,named,plan,indexed,body_index,ref invocations,g,n.source,0,out,s,work)){return false;}}
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
        else if(n.kind==14){if(!calls::call(p,named,plan,indexed,body_index,ref invocations,g,n.source,n.target,out,s,work)){return false;}}
        else if(n.kind==15 || n.kind==16){if(!lifecycle::block(b,n,out,s)){return false;}}
        else if(n.kind==17){if(!snapshot(out,s,n.target) || !w::text(out,s,b"=") || !aggregate::place(p,b,n.source,out,s,0) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==18){if(!switches::branch(p,b,n,out,s,work)){return false;}}
        else if(n.kind==19){if(!switches::seed(p,b,n,out,s,work)){return false;}}
        else if(n.kind==20 || n.kind==21){if(!iteration::block(p,g,n,out,s,work)){return false;}}
        else if(n.kind!=7){s.code=1004;return false;}
        if(n.kind!=3 && n.kind!=4 && n.kind!=6 && n.kind!=9 && n.kind!=12 && n.kind!=18 && n.kind!=21 && !w::jump(out,s,n.first)){return false;}i+=1;
    }return w::text(out,s,b"}\n");
}
