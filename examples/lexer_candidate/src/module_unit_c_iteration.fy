module module_unit_c_iteration;
import module_unit_cfg_model as cfg;
import module_unit_cfg_iteration as proof;
import module_unit_body_ir_model as ir;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as m;
import c_text as w;
import std::bytes as bytes;

fn bool slot(ref mut bytes::Buffer out,ref mut w::State s,usize id,ref u8[] field){return w::text(out,s,b"fy_iteration_") && w::number(out,s,id) && w::text(out,s,b"_") && w::text(out,s,field);}
pub fn bool locals(ref ir::Program p,ref cfg::Graph g,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));var usize i=1;
    while(i<=cfg::count(g)){if(!m::charge(work,1)){s.code=1002;return false;}let n=cfg::at(g,i);if(n.kind==20){
        let local=switch(proof::binding(p,b,cfg::frame(g),n.target,work)){case .ok(n):{n}case .err(e):{s.code=e.code;return false;}};
        let l=ir::local_at(b,local);let t=clean::type_at(ir::cleanup_ir(p),l.type);let borrowed=t.kind==7 || t.kind==8;
        if(!aggregate::type(p,if(borrowed){t.first}else{l.type},out,s) || !w::text(out,s,if(t.kind==8){b" *"}else{b" const *"}) || !slot(out,s,n.target,b"data") || !w::text(out,s,b";uintptr_t ") || !slot(out,s,n.target,b"length") || !w::text(out,s,b",") || !slot(out,s,n.target,b"index") || !w::text(out,s,b";\n")){return false;}
    }i+=1;}return true;
}
pub fn bool block(ref ir::Program p,ref cfg::Graph g,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));let local=switch(proof::binding(p,b,cfg::frame(g),n.target,work)){case .ok(n):{n}case .err(e):{s.code=e.code;return false;}};
    let carrier=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,n.source-1).value.type);
    if(n.kind==20){
        if(!slot(out,s,n.target,b"data") || !w::text(out,s,b"=")){return false;}
        if(carrier.kind==16){if(!aggregate::place(p,b,n.source,out,s,0) || !w::text(out,s,b".fy_items")){return false;}}
        else if(!scalar::value(out,s,n.source) || !w::text(out,s,b".fy_data")){return false;}
        if(!w::text(out,s,b";\n") || !slot(out,s,n.target,b"length") || !w::text(out,s,b"=")){return false;}
        if(carrier.kind==16){if(!w::number(out,s,carrier.second)){return false;}}
        else if(!scalar::value(out,s,n.source) || !w::text(out,s,b".fy_length")){return false;}
        return w::text(out,s,b";\n") && slot(out,s,n.target,b"index") && w::text(out,s,b"=0;\n");
    }
    let t=clean::type_at(ir::cleanup_ir(p),ir::local_at(b,local).type);let borrowed=t.kind==7 || t.kind==8;
    return w::text(out,s,b"if(") && slot(out,s,n.target,b"index") && w::text(out,s,b"<") && slot(out,s,n.target,b"length") && w::text(out,s,b"){") && w::local(out,s,local) && w::text(out,s,b"=") && (!borrowed || w::text(out,s,b"&")) && slot(out,s,n.target,b"data") && w::text(out,s,b"[") && slot(out,s,n.target,b"index") && w::text(out,s,b"];++") && slot(out,s,n.target,b"index") && w::text(out,s,b";\n") && w::jump(out,s,n.first) && w::text(out,s,b"}else{") && w::jump(out,s,n.second) && w::text(out,s,b"}\n");
}
