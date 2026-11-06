module project_state_graph_cache_key;
import project_compiler_core as core;
import project_compiler_state_graph as graph;
import project_cache_key as compatibility;
import project_cache_environment as environment;
import project_cache_encoding as enc;
import project_identity_bytes as binary;
import module_state_graph_source as authored;
import module_state_graph_source_identity as bound;
import module_state_graph_hash_context as context;
import module_state_graph_body_hash as body;
import module_project_interfaces as interfaces;
import syntax as syntax;
import sha256_stream as sha;
import std::bytes as bytes;

// Private versioned identity prerequisite for the sealed single-module graph
// source family. No dependency graph, workspace, admission or execution authority.
pub resource Identity{bytes::Buffer canonical;bytes::Buffer digest;cleanup{}}
pub fn ref u8[] canonical(ref Identity p){return bytes::view(ref p.canonical);}
pub fn ref u8[] digest(ref Identity p){return bytes::view(ref p.digest);}
fn u32 encoding_error(u32 code){return if(code==5){1002}else if(code==4){1003}else{1001};}
fn u32 identity_error(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn Result<bytes::Buffer,u32> decoded(ref u8[] text,ref u8[] prefix,ref mut usize work){return switch(binary::decode(text,prefix,true,work)){case .ok(p):{.ok(p)}case .err(code):{.err(identity_error(code))}};}
fn ref u8[] marker(){return b"fairyc.state-graph-cache-key\x00\x01";}
fn ref u8[] prefix(){return b"fysgraphcache1-sha256-";}
// A detached compatibility snapshot is data, not environment authority. Source,
// coordinate, portable body and both interfaces come only from this Complete.
// Canonical: marker, eight u64-BE framed UTF-8 compatibility strings in existing
// environment_field order, then six raw SHA256 members: stdlib(fypi1), selected
// coordinate(fypc1), exact retained source, body(fycsbody1), public(fyif5), source
// interface(fygif0). Numeric owner IDs and cold/restored storage are excluded.
pub fn Result<Identity,u32> compute(ref core::Complete complete,ref compatibility::Environment compatibility,ref u8[] standard_library,ref mut usize work){
    if(!enc::charge(work,1)){return .err(1002);}if(!core::is_state_graph(complete)){return .err(1001);}
    let graph=core::checked_state_graph(complete);let records=core::public_interfaces(complete);
    if(interfaces::count(records)!=1){return .err(1001);}let record=interfaces::at(records,0);
    if(interfaces::owner(record)!=graph::owner(graph)){return .err(1001);}
    // Check the entire output length before any compatibility-sized allocation.
    var usize size=marker().len+192;var usize i=0;
    while(i<8){let value=compatibility::environment_field(compatibility,i);let code=enc::string_check(value,work);if(code!=0){return .err(encoding_error(code));}
        if(size>16777207 || value.len>16777207-size){return .err(1002);}size+=8+value.len;i+=1;}
    let standard=switch(decoded(standard_library,b"fypi1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let identity=graph::body_identity(graph);let coordinate=context::coordinate(bound::context(identity));
    let valid=enc::digest_check(coordinate,work);if(valid!=0){return .err(encoding_error(valid));}
    let body_hash=switch(decoded(body::digest(bound::identity(identity)),b"fycsbody1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let public_hash=switch(decoded(interfaces::ordinary_hash(record),b"fyif5-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let source_interface=switch(decoded(interfaces::source_hash(record),b"fygif0-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let raw=syntax::source_view(authored::parsed(graph::source(graph)));
    if(raw.len>=16777216 || !enc::charge(work,raw.len+1)){return .err(1002);}var sha::State source=sha::empty();if(!sha::update(ref mut source,raw)){return .err(1002);}
    let source_hash=switch(binary::words(sha::finish(ref source),work)){case .ok(p):{p}case .err(code):{return .err(identity_error(code));}};
    var bytes::Buffer out=bytes::empty();var u32 code=enc::raw(ref mut out,marker(),work);if(code!=0){return .err(encoding_error(code));}i=0;
    while(i<8){code=enc::string(ref mut out,compatibility::environment_field(compatibility,i),work);if(code!=0){return .err(encoding_error(code));}i+=1;}
    code=enc::raw(ref mut out,bytes::view(ref standard),work);if(code!=0){return .err(encoding_error(code));}
    code=enc::raw(ref mut out,coordinate,work);if(code!=0){return .err(encoding_error(code));}
    code=enc::raw(ref mut out,bytes::view(ref source_hash),work);if(code!=0){return .err(encoding_error(code));}
    code=enc::raw(ref mut out,bytes::view(ref body_hash),work);if(code!=0){return .err(encoding_error(code));}
    code=enc::raw(ref mut out,bytes::view(ref public_hash),work);if(code!=0){return .err(encoding_error(code));}
    code=enc::raw(ref mut out,bytes::view(ref source_interface),work);if(code!=0){return .err(encoding_error(code));}
    if(bytes::len(ref out)!=size || !enc::charge(work,size+1+prefix().len+65)){return .err(1002);}var sha::State hash=sha::empty();if(!sha::update(ref mut hash,bytes::view(ref out))){return .err(1002);}
    var bytes::Buffer digest=bytes::empty();if(!bytes::append(ref mut digest,prefix()).is_ok || !sha::append_hex(ref hash,ref mut digest)){return .err(1003);}return .ok(Identity{.canonical=out,.digest});
}
// Production adapter consumes the retained compiler/profile facts; it does not
// accept caller labels in place of project_cache_environment's sealed owner.
pub fn Result<Identity,u32> derive(ref core::Complete complete,ref environment::Environment environment,ref mut usize work){
    if(!enc::charge(work,0)){return .err(1002);}if(!core::is_state_graph(complete)){return .err(1001);}
    let snapshot=switch(environment::key_environment(environment,work)){case .ok(p):{p}case .err(code):{return .err(identity_error(code));}};
    return compute(complete,ref snapshot,environment::standard_library(environment),work);
}
