module module_state_graph_transport;
import project_compiler_core as core;
import project_compiler_state_graph as complete;
import module_state_graph_native as native;
import module_state_graph_native_assets as runtime_assets;
import module_state_graph_frame_assets as frame;
import module_state_graph_transport_plan as facts;
import module_state_graph_native_text as w;
import project_identity_bytes as binary;
import project_cache_encoding as budget;
import sha256_stream as sha;
import std::bytes as bytes;

// Entry lowering and framing retain independent, caller-owned 16MiB ledgers.
pub struct Work{usize entry;usize framing;}
pub fn Work empty_work(){return Work{.entry=0,.framing=0};}
// Keep Complete alive for later path/response admission. Neither a C buffer nor
// caller-paired hashes can construct this owner. Storage remains a separate port.
pub resource Prepared{core::Complete checked;native::Artifact entry;facts::Plan plan;bytes::Buffer source;bytes::Buffer program;cleanup{}}
pub fn ref u8[] source(ref Prepared p){return bytes::view(ref p.source);}
pub fn ref u8[] program_hash(ref Prepared p){return bytes::view(ref p.program);}
pub fn ref native::Artifact entry(ref Prepared p){return ref p.entry;}
pub fn ref core::Complete checked(ref Prepared p){return ref p.checked;}
pub fn ref facts::Plan plan(ref Prepared p){return ref p.plan;}
fn bool hash_part(ref mut sha::State state,ref u8[] raw,ref mut usize work){return budget::charge(work,raw.len+1) && sha::update(state,raw);}
// Artifact owns canonical JSON of [a-z0-9-] categories (or null), so each
// category contributes exactly two quotes. Do not rebuild its slot/trap plan.
fn Result<usize,u32> trap_count(ref native::Artifact entry,ref mut usize work){let raw=native::trap_json(entry);if(!budget::charge(work,raw.len+1)){return .err(1002);}var usize quotes=0;var usize i=0;while(i<raw.len){if(raw[i]==34){quotes+=1;}i+=1;}return .ok(quotes/2);}
fn Result<bytes::Buffer,u32> digest(ref native::Artifact entry,ref facts::Plan p,ref frame::Bundle framing,ref mut usize work){
    var suffix=bytes::empty();if(budget::length(ref mut suffix,facts::operation_bound(p),work)!=0 || budget::length(ref mut suffix,facts::max_literal(p),work)!=0){return .err(1002);}
    var hash=sha::empty();if(!hash_part(ref mut hash,b"fairyc.native-state-graph-transport/v1\x00",work) || !hash_part(ref mut hash,native::program_hash(entry),work) || !hash_part(ref mut hash,bytes::view(ref suffix),work) || !hash_part(ref mut hash,frame::content(framing),work)){return .err(1002);}
    return switch(binary::words(sha::finish(ref hash),work)){case .ok(p):{.ok(p)}case .err(code):{.err(if(code==11){1003}else{1002})}};
}
fn bool known(ref facts::Plan p,ref mut bytes::Buffer out,ref mut w::State s){
    let count=facts::cell_count(p);if(count==0){return w::text(out,s,b"static uint8_t fygr_codec(const uint8_t key[32]) { (void)key; return 0u; }\n");}
    if(!w::text(out,s,b"typedef struct fygr_known { uint8_t key[32]; uint8_t codec; } fygr_known;\nstatic const fygr_known fygr_keys[] = {\n")){return false;}
    var usize i=0;while(i<count){let cell=facts::cell_at(p,i);if(!w::text(out,s,b"{{") || !w::digest(out,s,cell.key) || !w::text(out,s,b"},") || !w::number(out,s,cell.codec) || !w::text(out,s,b"u},\n")){return false;}i+=1;}
    return w::text(out,s,b"};\nstatic uint8_t fygr_codec(const uint8_t key[32]) { size_t low = 0u, high = ") && w::number(out,s,count) && w::text(out,s,b"u; while (low < high) { size_t middle = low+(high-low)/2u; int order = memcmp(fygr_keys[middle].key,key,32u); if (order < 0) { low = middle+1u; } else { high = middle; } } if (low < ") && w::number(out,s,count) && w::text(out,s,b"u && memcmp(fygr_keys[low].key,key,32u) == 0) { return fygr_keys[low].codec; } return 0u; }\n");
}
fn Result<bytes::Buffer,u32> framed(ref native::Artifact entry,ref facts::Plan p,ref u8[] program,usize trap_count,ref frame::Bundle framing,ref mut usize work){
    var out=bytes::empty();var s=w::State{.code=0,.work=*work,.site=0,.term=0,.audit=true};
    discard w::text(ref mut out,ref mut s,native::source(entry));discard w::text(ref mut out,ref mut s,b"\n#include <string.h>\nstatic const uint8_t fygr_identity[96] = {");
    discard w::octets(ref mut out,ref mut s,native::layout_hash(entry));discard w::text(ref mut out,ref mut s,b", ");discard w::octets(ref mut out,ref mut s,native::body_hash(entry));discard w::text(ref mut out,ref mut s,b", ");discard w::octets(ref mut out,ref mut s,program);
    discard w::text(ref mut out,ref mut s,b"};\n#define FYGR_WORK ");discard w::number(ref mut out,ref mut s,facts::operation_bound(p));discard w::text(ref mut out,ref mut s,b"u\n#define FYGR_LITERAL ");discard w::number(ref mut out,ref mut s,facts::max_literal(p));discard w::text(ref mut out,ref mut s,b"u\n#define FYGR_TRAP_COUNT ");discard w::number(ref mut out,ref mut s,trap_count);discard w::text(ref mut out,ref mut s,b"u\n");
    discard known(p,ref mut out,ref mut s);discard w::text(ref mut out,ref mut s,frame::content(framing));*work=s.work;if(s.code!=0){return .err(s.code);}return .ok(out);
}
pub fn Result<Prepared,u32> prepare(core::Complete checked,ref runtime_assets::Bundle runtime,ref frame::Bundle framing,ref u8[] target,ref mut Work work){
    if(!budget::charge(ref mut work.entry,0) || !budget::charge(ref mut work.framing,1)){return .err(1002);}if(!core::is_state_graph(ref checked) || !bytes::equal(target,b"x86_64-windows-msvc")){return .err(1001);}
    let proof=complete::program(core::checked_state_graph(ref checked));
    let plan=switch(facts::build(proof,ref mut work.framing)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let entry=switch(native::emit(ref checked,runtime,target,true,ref mut work.entry)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let count=switch(trap_count(ref entry,ref mut work.framing)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let program=switch(digest(ref entry,ref plan,framing,ref mut work.framing)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let source=switch(framed(ref entry,ref plan,bytes::view(ref program),count,framing,ref mut work.framing)){case .ok(p):{p}case .err(code):{return .err(code);}};
    return .ok(Prepared{.checked,.entry,.plan,.source,.program});
}
