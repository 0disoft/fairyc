module module_unit_c_region;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_model as cfg;
import module_unit_c_sdk as sdk;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import module_unit_value_model as budget;
import c_text as w;
import std::bytes as bytes;

fn bool name(ref mut bytes::Buffer out,ref mut w::State s,usize id){return w::text(out,s,b"fy_unit_region_") && w::number(out,s,id);}
fn bool plain(ref ir::Program p,usize id,usize depth,ref mut usize work,ref mut w::State s){if(depth>=64 || !budget::charge(work,1)){s.code=1002;return false;}let t=clean::type_at(ir::cleanup_ir(p),id);if(t.kind==3 || t.kind==13){s.code=1001;return false;}
    if(t.kind==1){return t.scalar!=7;}if(t.kind==12){return true;}if(t.kind==5 || t.kind==6 || t.kind==16){return plain(p,t.first,depth+1,work,s) && (t.kind!=6 || plain(p,t.second,depth+1,work,s));}
    if(t.kind!=2 && t.kind!=4){s.code=1001;return false;}var usize i=0;while(i<ir::layout_count(p)){if(!budget::charge(work,1)){s.code=1002;return false;}let shape=ir::layout_at(p,i).value;if(shape.type==id){var usize j=0;while(j<shape.field_count){if(!budget::charge(work,1)){s.code=1002;return false;}let f=ir::field_at(p,shape.field_start+j);if((f.kind==0 || f.kind==4) && !plain(p,f.type,depth+1,work,s)){return false;}j+=1;}return true;}i+=1;}s.code=1001;return false;
}
pub fn bool definitions(ref ir::Program p,ref sdk::Plan plan,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){var bool used=false;var usize frame=1;
    while(frame<=ir::count(p)){let b=ir::at(p,frame);var usize i=0;while(i<ir::placement_count(b)){let x=ir::placement_at(b,i);if(!budget::charge(work,1)){s.code=1002;return false;}if(x.error_type!=sdk::allocation_error(plan) || x.error_type==0 || !plain(p,x.owner_type,0,work,s)){if(s.code==0){s.code=1001;}return false;}used=true;i+=1;}i=0;while(i<ir::region_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}if(sdk::arena_type(plan)==0 || ir::region_at(b,i).arena_type!=sdk::arena_type(plan)){s.code=1001;return false;}used=true;i+=1;}frame+=1;}
    if(!used){return true;}
    // Runtime Region keeps its Arena address through reset. Its wire owner lives
    // in the caller's lexical slot, never inside a short-lived call adapter.
    return w::text(out,s,b"struct fy_std_region_v0{struct fy_std_arena_v0 *arena;struct fy_std_arena_node_v0 *mark;uint64_t generation;uint8_t active;};\nstruct fy_unit_region_bridge{struct fy_std_arena_v0 arena;struct fy_std_region_v0 region;void *owner;};\nextern uint32_t fy_std_region_begin_v0(struct fy_std_arena_v0 *,struct fy_std_region_v0 *);\nextern uint32_t fy_std_region_reset_v0(struct fy_std_region_v0 *);\nextern uint32_t fy_std_region_allocate_zeroed_v0(struct fy_std_region_v0 *,size_t,size_t,void **);\nstatic void fy_unit_region_begin(struct fy_unit_region_bridge *slot,void *owner,const char *source,uint32_t source_length,uint64_t start,uint64_t end){slot->owner=owner;fy_unit_copy(&slot->arena,owner,sizeof slot->arena);if(fy_std_region_begin_v0(&slot->arena,&slot->region))fy_unit_trap(7,source,source_length,start,end);}\nstatic void fy_unit_region_reset(struct fy_unit_region_bridge *slot,const char *source,uint32_t source_length,uint64_t start,uint64_t end){if(fy_std_region_reset_v0(&slot->region))fy_unit_trap(7,source,source_length,start,end);fy_unit_copy(slot->owner,&slot->arena,sizeof slot->arena);}\n");
}
pub fn bool locals(ref ir::Body b,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){var usize i=1;while(i<=ir::region_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}if(!w::text(out,s,b"struct fy_unit_region_bridge ") || !name(out,s,i) || !w::text(out,s,b";\n")){return false;}i+=1;}return true;}
pub fn bool block(ref ir::Body b,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s){let id=if(n.kind==15){n.target}else{ir::reset_at(b,n.target-1).region_id};return w::text(out,s,if(n.kind==15){b"fy_unit_region_begin(&"}else{b"fy_unit_region_reset(&"}) && name(out,s,id) && (n.kind!=15 || w::text(out,s,b",") && scalar::value(out,s,n.source)) && w::text(out,s,b",") && w::location(out,s,n.start,n.end) && w::text(out,s,b");\n");}
pub fn bool value(ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){let e=ir::node_at(b,id-1).value;if(!scalar::value(out,s,id) || !w::text(out,s,b"=")){return false;}
    if(e.kind==3){let left=ir::node(b,e.left);if(left==0 || ir::node_at(b,left-1).value.type!=e.type){s.code=1001;return false;}return scalar::value(out,s,left) && w::text(out,s,b";\n");}
    if(e.kind!=2 || e.binding==0 || ir::local_at(b,e.binding).type!=e.type){s.code=1001;return false;}let local=ir::local_at(b,e.binding);if(local.kind==1){return w::local(out,s,e.binding) && w::text(out,s,b";\n");}if(local.kind!=6){s.code=1001;return false;}var usize i=0;
    while(i<ir::region_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}if(ir::region_at(b,i).capability==e.binding){return w::text(out,s,b"&") && name(out,s,i+1) && w::text(out,s,b";\n");}i+=1;}s.code=1001;return false;
}
pub fn bool placement(ref ir::Program p,ref sdk::Plan plan,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){var usize i=0;
    while(i<ir::placement_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let x=ir::placement_at(b,i);if(ir::node(b,x.expression)==id){let argument=ir::node(b,x.argument);
        if(!w::text(out,s,b"{void *storage=NULL;uint32_t status=fy_std_region_allocate_zeroed_v0(&")){return false;}
        if(x.region_id==0){if(!w::local(out,s,x.capability) || !w::text(out,s,b"->region")){return false;}}else if(!name(out,s,x.region_id) || !w::text(out,s,b".region")){return false;}
        if(!w::text(out,s,b",sizeof(") || !aggregate::type(p,x.owner_type,out,s) || !w::text(out,s,b"),_Alignof(") || !aggregate::type(p,x.owner_type,out,s) || !w::text(out,s,b"),&storage);if(!status){fy_unit_copy(storage,&") || !scalar::value(out,s,argument) || !w::text(out,s,b",sizeof(") || !aggregate::type(p,x.owner_type,out,s) || !w::text(out,s,b"));") || !scalar::value(out,s,id) || !w::text(out,s,b"=(") || !aggregate::type(p,x.result_type,out,s) || !w::text(out,s,b"){.fy_tag=1,.fy_payload.fy_ok=storage};}else{") || !scalar::value(out,s,id) || !w::text(out,s,b"=(") || !aggregate::type(p,x.result_type,out,s) || !w::text(out,s,b"){.fy_tag=0,.fy_payload.fy_err=(") || !aggregate::type(p,x.error_type,out,s) || !w::text(out,s,b"){") || !w::text(out,s,b".") || !aggregate::field(out,s,sdk::status_field(p,plan)) || !w::text(out,s,b"=status}};}\n}\n")){return false;}return true;
    }i+=1;}s.code=1001;return false;
}
