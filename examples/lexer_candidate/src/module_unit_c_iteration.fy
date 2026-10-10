module module_unit_c_iteration;
import module_unit_cfg_model as cfg;
import module_unit_cfg_iteration as proof;
import module_unit_body_ir_model as ir;
import module_unit_c_scalar as scalar;
import module_unit_value_model as m;
import c_text as w;
import std::bytes as bytes;

fn bool slot(ref mut bytes::Buffer out,ref mut w::State s,usize id,ref u8[] field){return w::text(out,s,b"fy_iteration_") && w::number(out,s,id) && w::text(out,s,b"_") && w::text(out,s,field);}
pub fn bool locals(ref cfg::Graph g,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){var usize i=1;while(i<=cfg::count(g)){if(!m::charge(work,1)){s.code=1002;return false;}let n=cfg::at(g,i);if(n.kind==20){if(!w::text(out,s,b"uint8_t const *") || !slot(out,s,n.target,b"data") || !w::text(out,s,b";uintptr_t ") || !slot(out,s,n.target,b"length") || !w::text(out,s,b",") || !slot(out,s,n.target,b"index") || !w::text(out,s,b";\n")){return false;}}i+=1;}return true;}
pub fn bool block(ref ir::Program p,ref cfg::Graph g,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){let b=ir::at(p,cfg::frame(g));let local=switch(proof::binding(p,b,cfg::frame(g),n.target,work)){case .ok(n):{n}case .err(e):{s.code=e.code;return false;}};
    if(n.kind==20){return slot(out,s,n.target,b"data") && w::text(out,s,b"=") && scalar::value(out,s,n.source) && w::text(out,s,b".fy_data;\n") && slot(out,s,n.target,b"length") && w::text(out,s,b"=") && scalar::value(out,s,n.source) && w::text(out,s,b".fy_length;\n") && slot(out,s,n.target,b"index") && w::text(out,s,b"=0;\n");}
    return w::text(out,s,b"if(") && slot(out,s,n.target,b"index") && w::text(out,s,b"<") && slot(out,s,n.target,b"length") && w::text(out,s,b"){") && w::local(out,s,local) && w::text(out,s,b"=") && slot(out,s,n.target,b"data") && w::text(out,s,b"[") && slot(out,s,n.target,b"index") && w::text(out,s,b"];++") && slot(out,s,n.target,b"index") && w::text(out,s,b";\n") && w::jump(out,s,n.first) && w::text(out,s,b"}else{") && w::jump(out,s,n.second) && w::text(out,s,b"}\n");
}
