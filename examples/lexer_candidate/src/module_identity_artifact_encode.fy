module module_identity_artifact_encode;
import module_identity_artifact_model as model;
import module_identity_artifact_extract as snapshot_extract;
import module_project_identified_ir as project;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
pub fn Result<wire::Payload,u32> encode(ref model::Snapshot p,ref clean::Program context,ref mut usize work){let checked=model::check(p,context,work);if(checked!=0){return .err(checked);}var wire::Status s=wire::Status{.code=0};var out=wire::writer(b"FYIDS000",ref mut s,work);wire::number(ref mut out,model::type_count(p),ref mut s,work);var usize i=1;
    while(i<=model::type_count(p)+model::frame_count(p)){if(i==model::type_count(p)+1){wire::number(ref mut out,model::frame_count(p),ref mut s,work);}let row=if(i<=model::type_count(p)){model::type_at(p,i)}else{model::frame_at(p,i-model::type_count(p))};wire::number(ref mut out,row.node,ref mut s,work);wire::number(ref mut out,row.id.start,ref mut s,work);wire::number(ref mut out,row.id.end,ref mut s,work);if(wire::output_status(ref s)!=0){return .err(wire::output_status(ref s));}i+=1;}
    if(model::frame_count(p)==0){wire::number(ref mut out,0,ref mut s,work);}wire::number(ref mut out,model::names(p).len,ref mut s,work);if(wire::output_status(ref s)!=0){return .err(wire::output_status(ref s));}let code=budget::raw(ref mut out,model::names(p),work);if(code!=0){return .err(if(code==4){1003}else{1002});}return wire::finish(out,s);
}
pub fn Result<wire::Payload,u32> project_owner(ref project::Program p,ref mut usize work){let snapshot=switch(snapshot_extract::extract(project::source_identities(p),ir::cleanup_ir(project::ir_program(p)),work)){case .ok(p):{p}case .err(code):{return .err(code);}};return encode(ref snapshot,ir::cleanup_ir(project::ir_program(p)),work);}
