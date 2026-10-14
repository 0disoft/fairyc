module module_project_artifact_encode;
import module_project_identified_ir as project;
import module_unit_body_ir_model as ir;
import module_source_artifact_model as source_model;
import module_source_artifact_encode as sources;
import module_identity_artifact_model as identity_model;
import module_identity_artifact_extract as snapshot_extract;
import module_identity_artifact_encode as identities;
import module_program_artifact_encode as programs;
import module_project_artifact_model as model;
import module_program_artifact_wire as wire;
import candidate_artifact_wire as component;

pub fn Result<component::Payload,u32> encode(ref project::Program p,ref mut usize work){if(!wire::charge(work,0)){return .err(1002);}var usize measured=0;let stamps=switch(source_model::extract(project::source_provenance(p),ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}measured=0;let ids=switch(snapshot_extract::extract(project::source_identities(p),ir::cleanup_ir(project::ir_program(p)),ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}
    return components(project::ir_program(p),ref stamps,ref ids,work);
}
pub fn Result<component::Payload,u32> restored(ref model::Restored p,ref mut usize work){return components(model::program(p),model::source_provenance(p),model::source_identities(p),work);}
fn Result<component::Payload,u32> components(ref ir::Program p,ref source_model::Snapshot stamps,ref identity_model::Snapshot ids,ref mut usize work){
    let checked=model::check(p,stamps,ids,work);if(checked!=0){return .err(checked);}var wire::Status s=wire::Status{.code=0};var out=wire::writer_marker(b"FYPRJ000",ref mut s,work);
    var usize measured=0;let program=switch(programs::encode(p,ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}wire::member(ref mut out,component::content(ref program),ref mut s,work);if(s.code!=0){return .err(s.code);}
    measured=0;let provenance=switch(sources::encode(stamps,ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}wire::member(ref mut out,component::content(ref provenance),ref mut s,work);if(s.code!=0){return .err(s.code);}
    measured=0;let named=switch(identities::encode(ids,ir::cleanup_ir(p),ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}wire::member(ref mut out,component::content(ref named),ref mut s,work);return wire::finish(out,s);
}
