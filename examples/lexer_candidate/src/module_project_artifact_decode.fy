module module_project_artifact_decode;
import module_project_artifact_model as model;
import module_program_artifact_decode as programs;
import module_source_artifact_decode as sources;
import module_identity_artifact_decode as identities;
import module_unit_body_ir_model as ir;
import module_program_artifact_wire as wire;

pub fn u32 preflight(ref u8[] raw,ref mut usize work){var c=switch(wire::cursor_marker(raw,b"FYPRJ000",work)){case .ok(c):{c}case .err(code):{return code;}};discard wire::member_from(raw,ref mut c,wire::byte_limit(),b"FYPROG00",work);discard wire::member_from(raw,ref mut c,8388608,b"FYPROV00",work);discard wire::member_from(raw,ref mut c,8388608,b"FYIDS000",work);return wire::end(raw,ref mut c,work);}
pub fn Result<model::Restored,u32> decode(ref u8[] raw,ref mut usize work){let checked=preflight(raw,work);if(checked!=0){return .err(checked);}var c=switch(wire::cursor_marker(raw,b"FYPRJ000",work)){case .ok(c):{c}case .err(code):{return .err(code);}};let program_span=wire::member_from(raw,ref mut c,wire::byte_limit(),b"FYPROG00",work);let source_span=wire::member_from(raw,ref mut c,8388608,b"FYPROV00",work);let identity_span=wire::member_from(raw,ref mut c,8388608,b"FYIDS000",work);let end=wire::end(raw,ref mut c,work);if(end!=0){return .err(end);}var usize measured=0;
    let program=switch(programs::decode(raw[program_span.start..program_span.end],ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}
    measured=0;let provenance=switch(sources::decode(raw[source_span.start..source_span.end],ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}
    measured=0;let named=switch(identities::decode(raw[identity_span.start..identity_span.end],ir::cleanup_ir(ref program),ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}return model::assemble(program,provenance,named,work);
}
