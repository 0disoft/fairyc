module module_metadata_artifact_encode;
import module_metadata_artifact_model as model;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_sdk_sources as sdk;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
// FYMETA00: module/type counts, layouts/fields/callables/parameters, exact names.
pub fn Result<wire::Payload,u32> encode(ref model::Metadata p,ref clean::Program cleaned,usize modules,ref mut usize work){
    let code=model::check(p,cleaned,modules,work);if(code!=0){return .err(code);}
    var wire::Status status=wire::Status{.code=0};var out=wire::writer(b"FYMETA00",ref mut status,work);
    wire::number(ref mut out,modules,ref mut status,work);wire::number(ref mut out,clean::type_count(cleaned),ref mut status,work);var usize i=0;
    wire::number(ref mut out,model::layout_count(p),ref mut status,work);i=0;while(i<model::layout_count(p)){let row=model::layout_at(p,i);
        wire::number(ref mut out,row.value.type,ref mut status,work);
        wire::number(ref mut out,row.value.module_index,ref mut status,work);
        wire::number(ref mut out,row.value.declaration,ref mut status,work);
        wire::number(ref mut out,row.value.field_start,ref mut status,work);
        wire::number(ref mut out,row.value.field_count,ref mut status,work);
        wire::number(ref mut out,row.value.abilities as usize,ref mut status,work);
        wire::number(ref mut out,if(row.value.repr_c){1}else{0},ref mut status,work);
        wire::number(ref mut out,row.value.start,ref mut status,work);
        wire::number(ref mut out,row.value.end,ref mut status,work);
        wire::number(ref mut out,if(row.value.cleanup_nonempty){1}else{0},ref mut status,work);
        wire::number(ref mut out,row.value.cleanup_effects as usize,ref mut status,work);
        wire::number(ref mut out,row.name_start,ref mut status,work);
        wire::number(ref mut out,row.name_end,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    wire::number(ref mut out,model::field_count(p),ref mut status,work);i=0;while(i<model::field_count(p)){let row=model::field_at(p,i);
        wire::number(ref mut out,row.kind as usize,ref mut status,work);
        wire::number(ref mut out,row.type,ref mut status,work);
        wire::number(ref mut out,row.module_index,ref mut status,work);
        wire::number(ref mut out,row.declaration,ref mut status,work);
        wire::number(ref mut out,row.source_index,ref mut status,work);
        wire::number(ref mut out,row.start,ref mut status,work);
        wire::number(ref mut out,row.end,ref mut status,work);
        wire::number(ref mut out,row.name_start,ref mut status,work);
        wire::number(ref mut out,row.name_end,ref mut status,work);
        wire::number(ref mut out,row.payload_start,ref mut status,work);
        wire::number(ref mut out,row.payload_count,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    wire::number(ref mut out,model::callable_count(p),ref mut status,work);i=0;while(i<model::callable_count(p)){let row=model::callable_at(p,i);
        wire::number(ref mut out,row.module_index,ref mut status,work);
        wire::number(ref mut out,row.declaration,ref mut status,work);
        wire::number(ref mut out,if(row.template){1}else{0},ref mut status,work);
        wire::number(ref mut out,row.result,ref mut status,work);
        wire::number(ref mut out,row.parameter_start,ref mut status,work);
        wire::number(ref mut out,row.parameter_count,ref mut status,work);
        wire::number(ref mut out,row.effects as usize,ref mut status,work);
        wire::number(ref mut out,if(row.must_use){1}else{0},ref mut status,work);
        wire::number(ref mut out,if(row.unsafe_call){1}else{0},ref mut status,work);
        wire::number(ref mut out,if(row.foreign){1}else{0},ref mut status,work);
        wire::number(ref mut out,row.link_start,ref mut status,work);
        wire::number(ref mut out,row.link_end,ref mut status,work);
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;}
    wire::number(ref mut out,model::parameter_count(p),ref mut status,work);i=0;while(i<model::parameter_count(p)){wire::number(ref mut out,model::parameter_at(p,i),ref mut status,work);i+=1;}
    wire::number(ref mut out,model::names(p).len,ref mut status,work);if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}
    let written=budget::raw(ref mut out,model::names(p),work);if(written!=0){return .err(if(written==4){1003}else{1002});}return wire::finish(out,status);
}
pub fn Result<wire::Payload,u32> program(ref ir::Program p,ref mut usize work){let metadata=switch(model::extract(p,work)){case .ok(p):{p}case .err(code):{return .err(code);}};return encode(ref metadata,ir::cleanup_ir(p),sdk::module_count(ir::sdk_units(p)),work);}
