module project_state_graph_memory_cache;
import project_compiler_core as core;
import project_compiler_state_graph as graph;
import module_state_graph_source as source;
import module_state_graph_source_artifact as artifact;
import module_public_type_text as coordinates;
import project_identity_bytes as binary;
import project_cache_encoding as budget;
import candidate_artifact_wire as wire;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

// Session-local packets only. No Complete importer, disk or ambient environment.
// The owning Vec is the session: row zero holds limits; later sealed rows own
// one packet each. Whole vector moves avoid mutable resource-field projection.
pub resource Session{usize entry_limit;usize byte_limit;bool header;bytes::Buffer digest;bytes::Buffer coordinate;v::Vec<wire::Payload> packets;cleanup{}}
pub struct Observation{bool hit;}
pub fn Result<v::Vec<Session>,u32> create(usize entries,usize bytes){
    if(entries>512 || bytes>wire::byte_limit()){return .err(1002);}
    var out=v::empty<Session>();if(!v::push<Session>(ref mut out,Session{.entry_limit=entries,.byte_limit=bytes,.header=true,.digest=bytes::empty(),.coordinate=bytes::empty(),.packets=v::empty<wire::Payload>()}).is_ok){return .err(1003);}return .ok(out);
}
pub fn usize count(ref v::Vec<Session> p){return if(v::len<Session>(p)==0){0}else{v::len<Session>(p)-1};}
// Includes the two 32-byte keys as well as every retained envelope byte.
pub fn usize stored_bytes(ref v::Vec<Session> p){var usize used=0;var usize at=1;while(at<v::len<Session>(p)){let row=v::at<Session>(p,at);if(v::len<wire::Payload>(ref row.packets)==1){used+=wire::content(v::at<wire::Payload>(ref row.packets,0)).len+64;}at+=1;}return used;}
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn Result<bytes::Buffer,u32> coordinate(ref coordinates::Coordinates table,usize owner,ref mut usize work){
    if(owner>=512){return .err(1001);}
    var usize measured=0;let checked=coordinates::validate_owner(table,owner,ref mut measured);
    if(!budget::charge(work,measured)){return .err(1002);}if(checked!=0){return .err(mapped(checked));}
    var usize at=0;while(at<coordinates::coordinate_count(table)){
        if(!budget::charge(work,1)){return .err(1002);}
        if(coordinates::coordinate_at(table,at).owner==owner){return switch(binary::decode(coordinates::coordinate_text(table,at),b"fypc1-sha256-",true,work)){case .ok(p):{.ok(p)}case .err(code):{.err(mapped(code))}};}
        at+=1;
    }return .err(1001);
}
// All paths share the caller's ledger. A failed restore never starts a fresh one.
pub fn Result<core::Complete,u32> build(ref mut v::Vec<Session> session,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref mut Observation observation,ref mut usize work){
    observation.hit=false;
    if(!budget::charge(work,1) || actual.len>source::source_limit()){return .err(1002);}
    if(v::len<Session>(session)==0 || v::len<Session>(session)>513){return .err(1001);}let header=v::at<Session>(session,0);
    let entry_limit=header.entry_limit;let byte_limit=header.byte_limit;
    if(!header.header || entry_limit>512 || byte_limit>wire::byte_limit() || count(session)>entry_limit || v::len<wire::Payload>(ref header.packets)!=0 || bytes::len(ref header.digest)!=0 || bytes::len(ref header.coordinate)!=0){return .err(1001);}
    var usize used=0;var usize index=1;while(index<v::len<Session>(session)){
        if(!budget::charge(work,1)){return .err(1002);}let row=v::at<Session>(session,index);
        if(row.header || v::len<wire::Payload>(ref row.packets)!=1 || bytes::len(ref row.digest)!=32 || bytes::len(ref row.coordinate)!=32){return .err(1001);}
        let size=wire::content(v::at<wire::Payload>(ref row.packets,0)).len+64;if(size>byte_limit-used){return .err(1001);}used+=size;index+=1;
    }
    let selected=switch(coordinate(table,owner,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    if(!budget::charge(work,actual.len+1)){return .err(1002);}var state=sha::empty();if(!sha::update(ref mut state,actual)){return .err(1002);}
    let digest=switch(binary::words(sha::finish(ref state),work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};
    var usize at=1;while(at<v::len<Session>(session)){
        // Both fixed-size key comparisons, including a failed lookup, are metered.
        if(!budget::charge(work,129)){return .err(1002);}
        let entry=v::at<Session>(session,at);
        if(bytes::equal(bytes::view(ref entry.digest),bytes::view(ref digest)) && bytes::equal(bytes::view(ref entry.coordinate),bytes::view(ref selected))){
            let complete=switch(core::restore_complete_state_graph(wire::content(v::at<wire::Payload>(ref entry.packets,0)),actual,table,owner,work)){case .ok(p):{p}case .err(p):{return .err(p.code);}};
            observation.hit=true;return .ok(complete);
        }at+=1;
    }
    let complete=switch(core::build_complete_state_graph(actual,table,owner,work)){case .ok(p):{p}case .err(p):{return .err(p.code);}};
    if(count(session)>=entry_limit || byte_limit-used<64){return .ok(complete);}
    let packet=switch(artifact::encode(graph::source(core::checked_state_graph(ref complete)),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let size=wire::content(ref packet).len+64;
    if(size>byte_limit-used){return .ok(complete);}
    // Encoding/key construction charged owned byte copies. Also charge a
    // conservative 256 bytes per moved row if Vec grows, plus packet/row moves.
    // This covers the scalar, two Buffer and Vec descriptors on supported hosts.
    if(!budget::charge(work,(v::len<Session>(session)+1)*256+129)){return .err(1002);}
    var packets=v::empty<wire::Payload>();if(!v::push<wire::Payload>(ref mut packets,packet).is_ok){return .err(1003);}
    if(!v::push<Session>(session,Session{.entry_limit=0,.byte_limit=0,.header=false,.digest,.coordinate=selected,.packets}).is_ok){return .err(1003);}
    return .ok(complete);
}
