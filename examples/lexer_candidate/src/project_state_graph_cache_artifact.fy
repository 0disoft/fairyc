module project_state_graph_cache_artifact;
import project_state_graph_cache_key as key;
import project_compiler_core as core;
import project_compiler_state_graph as graph;
import module_state_graph_source_artifact as source_artifact;
import module_public_type_text as coordinates;
import project_cache_key as compatibility;
import project_cache_environment as environment;
import project_cache_encoding as budget;
import candidate_artifact_wire as wire;
import std::bytes as bytes;

// A private transport family. A matching header never grants Complete authority:
// restore checks current source/coordinate and recomputes the entire cache key.
struct Sections{usize key_end;usize packet_start;}
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn u32 append(ref mut bytes::Buffer out,ref u8[] value,ref mut usize work){
    if(value.len>wire::byte_limit() || bytes::len(out)>wire::byte_limit()-value.len){return 1002;}
    let code=budget::raw(out,value,work);return if(code==0){0}else if(code==4){1003}else{1002};
}
fn Result<wire::Payload,u32> packet(ref core::Complete complete,ref key::Identity identity,ref mut usize work){
    if(!core::is_state_graph(complete)){return .err(1001);}
    let saved=switch(source_artifact::encode(graph::source(core::checked_state_graph(complete)),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var status=wire::Status{.code=0};var out=wire::writer(b"FYSGCAC0",ref mut status,work);
    wire::number(ref mut out,key::canonical(identity).len,ref mut status,work);if(status.code!=0){return .err(status.code);}
    let first=append(ref mut out,key::canonical(identity),work);if(first!=0){return .err(first);}
    wire::number(ref mut out,wire::content(ref saved).len,ref mut status,work);if(status.code!=0){return .err(status.code);}
    let second=append(ref mut out,wire::content(ref saved),work);if(second!=0){return .err(second);}
    return wire::finish(out,status);
}
fn Result<Sections,u32> sections(ref u8[] saved,ref mut usize work){
    var c=switch(wire::cursor(saved,b"FYSGCAC0",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let key_size=wire::count_from(saved,ref mut c,wire::byte_limit(),1,work);if(c.code!=0){return .err(c.code);}c.offset+=key_size;let key_end=c.offset;
    let packet_size=wire::count_from(saved,ref mut c,wire::byte_limit(),1,work);if(c.code!=0){return .err(c.code);}let packet_start=c.offset;c.offset+=packet_size;
    let code=wire::end(saved,ref mut c,work);if(code!=0){return .err(code);}return .ok(Sections{.key_end,.packet_start});
}
fn u32 matches(ref u8[] saved,Sections parts,ref key::Identity identity,ref mut usize work){
    let current=key::canonical(identity);if(!budget::charge(work,parts.key_end-16+current.len+1)){return 1002;}
    return if(bytes::equal(saved[16..parts.key_end],current)){0}else{1001};
}
// Explicit compatibility snapshots are deterministic transport inputs, not a
// claim that a caller-provided snapshot describes this running compiler.
pub fn Result<wire::Payload,u32> encode_snapshot(ref core::Complete complete,ref compatibility::Environment env,ref u8[] standard_library,ref mut usize work){
    let identity=switch(key::compute(complete,env,standard_library,work)){case .ok(p):{p}case .err(code):{return .err(code);}};return packet(complete,ref identity,work);
}
pub fn Result<core::Complete,u32> restore_snapshot(ref u8[] saved,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref compatibility::Environment env,ref u8[] standard_library,ref mut usize work){
    let parts=switch(sections(saved,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let complete=switch(core::restore_complete_state_graph(saved[parts.packet_start..],actual,table,owner,work)){case .ok(p):{p}case .err(p):{return .err(p.code);}};
    let identity=switch(key::compute(ref complete,env,standard_library,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let code=matches(saved,parts,ref identity,work);if(code!=0){return .err(code);}return .ok(complete);
}
// Production adapters take the existing sealed current environment owner.
pub fn Result<wire::Payload,u32> encode(ref core::Complete complete,ref environment::Environment env,ref mut usize work){
    let snapshot=switch(environment::key_environment(env,work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};return encode_snapshot(complete,ref snapshot,environment::standard_library(env),work);
}
pub fn Result<core::Complete,u32> restore(ref u8[] saved,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref environment::Environment env,ref mut usize work){
    let snapshot=switch(environment::key_environment(env,work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};return restore_snapshot(saved,actual,table,owner,ref snapshot,environment::standard_library(env),work);
}
