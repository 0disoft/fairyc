module module_state_graph_native;
import project_compiler_core as core;
import project_compiler_state_graph as complete;
import module_state_graph_source_identity as bound;
import module_state_graph_body_hash as identity;
import module_state_graph_program as program;
import module_state_graph_model as m;
import module_state_layout as layout;
import module_state_graph_native_plan as plan;
import module_state_graph_native_assets as assets;
import module_state_graph_native_bodies as bodies;
import module_state_graph_native_text as w;
import project_identity_bytes as binary;
import project_cache_encoding as budget;
import sha256_stream as sha;
import std::bytes as bytes;

// Current Complete is mandatory; detached IR, cache packets and caller hashes
// cannot authorize publication. This owner contains C, not a live transaction.
pub resource Artifact{bytes::Buffer source;bytes::Buffer layout;bytes::Buffer body;bytes::Buffer program;bytes::Buffer traps;bool audit;cleanup{}}
pub fn ref u8[] source(ref Artifact p){return bytes::view(ref p.source);}
pub fn ref u8[] layout_hash(ref Artifact p){return bytes::view(ref p.layout);}
pub fn ref u8[] body_hash(ref Artifact p){return bytes::view(ref p.body);}
pub fn ref u8[] program_hash(ref Artifact p){return bytes::view(ref p.program);}
pub fn ref u8[] trap_json(ref Artifact p){return bytes::view(ref p.traps);}
pub fn bool audited(ref Artifact p){return p.audit;}
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn bool observe(ref mut bytes::Buffer out,ref mut w::State s){
    let audit=s.audit;
    return w::text(out,s,if(audit){b"\ntypedef int32_t (*fycs_graph_observer_v1)(void *, uint64_t, uint32_t, const fycs_graph_value_v1 *, const fycs_graph_value_v1 *, uint8_t);\n"}else{b"\ntypedef int32_t (*fycs_graph_observer_v1)(void *, uint64_t, uint32_t, const uint8_t[32], uint8_t, const uint8_t *, size_t, uint8_t);\n"}) && w::text(out,s,b"static const uint8_t fygr_bool_v1[2] = {0u,1u};\n");
}
fn Result<bytes::Buffer,u32> copy(ref u8[] raw,ref mut usize work){if(!budget::charge(work,raw.len+1)){return .err(1002);}return switch(bytes::copy(raw)){case .ok(p):{.ok(p)}case .err(_):{.err(1003)}};}
fn bool hash_part(ref mut sha::State hash,ref u8[] raw,ref mut usize work){return budget::charge(work,raw.len+1) && sha::update(hash,raw);}
pub fn Result<Artifact,u32> emit(ref core::Complete checked,ref assets::Bundle runtime,ref u8[] target,bool audit,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}
    if(!bytes::equal(target,b"x86_64-windows-msvc") || !core::is_state_graph(checked)){return .err(1001);}
    let owned=core::checked_state_graph(checked);let proof=complete::program(owned);
    let prepared=switch(plan::build(proof,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let header=switch(assets::declarations(runtime,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var out=bytes::empty();var s=w::State{.code=0,.work=*work,.site=0,.term=0,.audit};
    discard w::text(ref mut out,ref mut s,bytes::view(ref header));discard observe(ref mut out,ref mut s);
    var usize i=0;while(i<plan::body_count(ref prepared) && s.code==0){discard bodies::declaration(i,ref mut out,ref mut s);i+=1;}
    i=0;while(i<plan::body_count(ref prepared) && s.code==0){discard bodies::emit(proof,ref prepared,i,ref mut out,ref mut s);i+=1;}
    discard bodies::entry(ref mut out,ref mut s);*work=s.work;if(s.code!=0){return .err(s.code);}
    let layout=switch(binary::words(layout::at(program::layouts(proof),m::header(program::root(proof)).layout).hash,work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};
    let body=switch(binary::decode(identity::digest(bound::identity(complete::body_identity(owned))),b"fycsbody1-sha256-",true,work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};
    let traps=switch(copy(plan::trap_json(ref prepared),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var hash=sha::empty();
    if(!hash_part(ref mut hash,b"fairyc.native-state-graph-entry/v1\x00",work) || !hash_part(ref mut hash,target,work) || !hash_part(ref mut hash,b"\x00",work) || !hash_part(ref mut hash,bytes::view(ref layout),work) || !hash_part(ref mut hash,bytes::view(ref body),work) || !hash_part(ref mut hash,bytes::view(ref traps),work) || !hash_part(ref mut hash,bytes::view(ref out),work)){return .err(1002);}
    let program=switch(binary::words(sha::finish(ref hash),work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};
    return .ok(Artifact{.source=out,.layout,.body,.program,.traps,.audit});
}
