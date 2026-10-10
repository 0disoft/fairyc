module module_unit_c_switches;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_enums as proof;
import module_unit_cfg_model as cfg;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_enums as enums;
import module_unit_c_aggregate as aggregate;
import module_unit_c_scalar as scalar;
import c_text as w;
import std::bytes as bytes;

fn usize subject(ref ir::Body b,cfg::Block n,ref mut w::State s){let e=ir::node_at(b,n.source-1).value;let id=lookup::node(b,e.left);if(e.kind!=15 || id==0 || ir::node_at(b,id-1).value.type!=e.target){s.code=1001;return 0;}return id;}
fn usize pattern(ref ir::Body b,cfg::Block n,ref mut w::State s){let a=ir::node_at(b,n.target-1).value;let id=lookup::node(b,a.left);if(a.kind!=18 || id==0){s.code=1001;return 0;}return id;}
pub fn bool branch(ref ir::Program p,ref ir::Body b,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let value=subject(b,n,s);let pat=pattern(b,n,s);if(value==0 || pat==0){return false;}let t=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,value-1).value.type);let e=ir::node_at(b,pat-1).value;var usize tag=0;
    if(t.kind==4){let index=proof::layout(p,e.expected,work,ref mut s.code);if(index==0 || !proof::member(p,e.field,e.expected,3)){if(s.code==0){s.code=1001;}return false;}let shape=ir::layout_at(p,index-1).value;var usize i=shape.field_start;while(i+1<e.field.id){if(!budget::charge(work,1)){s.code=1002;return false;}if(ir::field_at(p,i).kind==3){tag+=1;}i+=1;}}
    else if(t.kind==5 || t.kind==6){let name=ir::node_text(b,pat-1);if(!budget::charge(work,name.len+1)){s.code=1002;return false;}if(bytes::equal(name,if(t.kind==5){b"some"}else{b"ok"})){tag=1;}else if(!bytes::equal(name,if(t.kind==5){b"none"}else{b"err"})){s.code=1001;return false;}}
    else{s.code=1001;return false;}
    return w::text(out,s,b"if(") && scalar::value(out,s,value) && w::text(out,s,b".fy_tag==") && w::number(out,s,tag) && w::text(out,s,b"){") && w::jump(out,s,n.first) && w::text(out,s,b"}else{") && w::jump(out,s,n.second) && w::text(out,s,b"}\n");
}
fn bool destination(ref ir::Body b,usize original,ref mut bytes::Buffer out,ref mut w::State s){let id=lookup::node(b,original);if(id==0){s.code=1001;return false;}let e=ir::node_at(b,id-1).value;let local=e.place.root;if(e.kind!=17 || local==0 || local>ir::local_count(b) || ir::local_at(b,local).kind!=4 || ir::local_at(b,local).type!=e.target){s.code=1001;return false;}return w::local(out,s,local) && w::text(out,s,b"=");}
// Only a verified case-success edge enters this block. Anonymous payloads
// use the same physical local slots and cleanup paths as named payloads.
pub fn bool seed(ref ir::Program p,ref ir::Body b,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let value=subject(b,n,s);let pat=pattern(b,n,s);if(value==0 || pat==0){return false;}let t=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,value-1).value.type);let e=ir::node_at(b,pat-1).value;
    if(t.kind==5 || t.kind==6){if(e.left==0){return t.kind==5;}return destination(b,e.left,out,s) && scalar::value(out,s,value) && w::text(out,s,if(bytes::equal(ir::node_text(b,pat-1),b"err")){b".fy_payload.fy_err;\n"}else{b".fy_payload.fy_ok;\n"});}
    if(t.kind!=4 || !proof::member(p,e.field,e.expected,3)){s.code=1001;return false;}var usize link=e.right;
    while(link!=0){if(!budget::charge(work,1)){s.code=1002;return false;}let pair=lookup::node(b,link);if(pair==0){s.code=1001;return false;}let row=ir::node_at(b,pair-1).value;
        if(row.kind!=33 || !proof::member(p,row.field,e.expected,4)){s.code=1001;return false;}
        if(!destination(b,row.left,out,s) || !scalar::value(out,s,value) || !w::text(out,s,b".fy_payload.") || !enums::variant(out,s,e.field.id) || !w::text(out,s,b".") || !aggregate::field(out,s,row.field.id) || !w::text(out,s,b";\n")){return false;}link=row.right;
    }return true;
}
