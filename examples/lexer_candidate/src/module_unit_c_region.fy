module module_unit_c_region;
import module_unit_body_ir_model as ir;
import module_unit_cfg_model as cfg;
import module_unit_c_sdk as sdk;
import module_unit_c_scalar as scalar;
import module_unit_value_model as budget;
import c_text as w;
import std::bytes as bytes;

fn bool name(ref mut bytes::Buffer out,ref mut w::State s,usize id){return w::text(out,s,b"fy_unit_region_") && w::number(out,s,id);}
pub fn bool definitions(ref ir::Program p,ref sdk::Plan plan,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){var bool used=false;var usize frame=1;
    while(frame<=ir::count(p)){let b=ir::at(p,frame);if(ir::placement_count(b)!=0){s.code=1004;return false;}var usize i=0;while(i<ir::region_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}if(sdk::arena_type(plan)==0 || ir::region_at(b,i).arena_type!=sdk::arena_type(plan)){s.code=1001;return false;}used=true;i+=1;}frame+=1;}
    if(!used){return true;}
    // Runtime Region keeps its Arena address through reset. Its wire owner lives
    // in the caller's lexical slot, never inside a short-lived call adapter.
    return w::text(out,s,b"struct fy_std_region_v0{struct fy_std_arena_v0 *arena;struct fy_std_arena_node_v0 *mark;uint64_t generation;uint8_t active;};\nstruct fy_unit_region_bridge{struct fy_std_arena_v0 arena;struct fy_std_region_v0 region;void *owner;};\nextern uint32_t fy_std_region_begin_v0(struct fy_std_arena_v0 *,struct fy_std_region_v0 *);\nextern uint32_t fy_std_region_reset_v0(struct fy_std_region_v0 *);\nstatic void fy_unit_region_begin(struct fy_unit_region_bridge *slot,void *owner){slot->owner=owner;fy_unit_copy(&slot->arena,owner,sizeof slot->arena);if(fy_std_region_begin_v0(&slot->arena,&slot->region))fy_runtime_trap(7);}\nstatic void fy_unit_region_reset(struct fy_unit_region_bridge *slot){if(fy_std_region_reset_v0(&slot->region))fy_runtime_trap(7);fy_unit_copy(slot->owner,&slot->arena,sizeof slot->arena);}\n");
}
pub fn bool locals(ref ir::Body b,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){var usize i=1;while(i<=ir::region_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}if(!w::text(out,s,b"struct fy_unit_region_bridge ") || !name(out,s,i) || !w::text(out,s,b";\n")){return false;}i+=1;}return true;}
pub fn bool block(ref ir::Body b,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s){let id=if(n.kind==15){n.target}else{ir::reset_at(b,n.target-1).region_id};return w::text(out,s,if(n.kind==15){b"fy_unit_region_begin(&"}else{b"fy_unit_region_reset(&"}) && name(out,s,id) && (n.kind!=15 || w::text(out,s,b",") && scalar::value(out,s,n.source)) && w::text(out,s,b");\n");}
