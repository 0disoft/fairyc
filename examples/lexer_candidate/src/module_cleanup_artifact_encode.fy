module module_cleanup_artifact_encode;
import module_unit_cleanup_mir_model as ir;
import module_unit_cleanup_mir_verify as proof;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
// FYCLEAN0: project flag, type count, types, one glue per type, then
// frame/field/operand/drop counts and rows. Every scalar occupies eight bytes.
// Producer work is transient; decoding creates a fresh verifier history.
pub fn Result<wire::Payload,u32> encode(ref ir::Program p,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}var usize measured=0;let checked=proof::check_metered(p,ref mut measured);if(!budget::charge(work,measured)){return .err(1002);}if(checked.code!=0){return .err(checked.code);}var wire::Status status=wire::Status{.code=0};var out=wire::writer(b"FYCLEAN0",ref mut status,work);wire::number(ref mut out,if(ir::project(p)){1}else{0},ref mut status,work);var usize i=0;
    wire::number(ref mut out,ir::type_count(p),ref mut status,work);
    i=0;while(i<ir::type_count(p)){let row=ir::type_at(p,i+1);
        wire::number(ref mut out,row.kind as usize,ref mut status,work);
        wire::number(ref mut out,row.scalar as usize,ref mut status,work);
        wire::number(ref mut out,row.module_index,ref mut status,work);
        wire::number(ref mut out,row.declaration,ref mut status,work);
        wire::number(ref mut out,row.first,ref mut status,work);
        wire::number(ref mut out,row.second,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    i=0;while(i<ir::type_count(p)){let row=ir::glue_at(p,i+1);
        wire::number(ref mut out,row.kind as usize,ref mut status,work);
        wire::number(ref mut out,row.frame,ref mut status,work);
        wire::number(ref mut out,if(row.nonempty){1}else{0},ref mut status,work);
        wire::number(ref mut out,row.effects as usize,ref mut status,work);
        wire::number(ref mut out,row.abilities as usize,ref mut status,work);
        wire::number(ref mut out,row.field_start,ref mut status,work);
        wire::number(ref mut out,row.field_count,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    wire::number(ref mut out,ir::frame_count(p),ref mut status,work);
    i=0;while(i<ir::frame_count(p)){let row=ir::frame_at(p,i+1);
        wire::number(ref mut out,row.kind as usize,ref mut status,work);
        wire::number(ref mut out,row.context,ref mut status,work);
        wire::number(ref mut out,row.module_index,ref mut status,work);
        wire::number(ref mut out,row.declaration,ref mut status,work);
        wire::number(ref mut out,row.result,ref mut status,work);
        wire::number(ref mut out,row.effects as usize,ref mut status,work);
        wire::number(ref mut out,row.receiver,ref mut status,work);
        wire::number(ref mut out,row.receiver_type,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    wire::number(ref mut out,ir::field_count(p),ref mut status,work);
    i=0;while(i<ir::field_count(p)){let row=ir::field_at(p,i);
        wire::number(ref mut out,row.owner,ref mut status,work);
        wire::number(ref mut out,row.ordinal,ref mut status,work);
        wire::number(ref mut out,row.type,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    wire::number(ref mut out,ir::operand_count(p),ref mut status,work);
    i=0;while(i<ir::operand_count(p)){let row=ir::operand_at(p,i+1);
        wire::number(ref mut out,row.frame,ref mut status,work);
        wire::number(ref mut out,row.kind as usize,ref mut status,work);
        wire::number(ref mut out,row.id,ref mut status,work);
        wire::number(ref mut out,row.type,ref mut status,work);
        wire::number(ref mut out,row.start,ref mut status,work);
        wire::number(ref mut out,row.end,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    wire::number(ref mut out,ir::drop_count(p),ref mut status,work);
    i=0;while(i<ir::drop_count(p)){let row=ir::drop_at(p,i);
        wire::number(ref mut out,row.operand,ref mut status,work);
        wire::number(ref mut out,row.exit as usize,ref mut status,work);
        wire::number(ref mut out,row.start,ref mut status,work);
        wire::number(ref mut out,row.end,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    return wire::finish(out,status);
}
