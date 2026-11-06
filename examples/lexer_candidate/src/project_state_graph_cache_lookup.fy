module project_state_graph_cache_lookup;
import project_cache_key as compatibility;
import project_cache_environment as environment;
import project_cache_encoding as enc;
import project_identity_bytes as binary;
import module_public_type_text as coordinates;
import module_state_graph_source as source;
import project_compiler_core as core;
import project_compiler_state_graph as graph;
import module_state_graph_source_identity as source_identity;
import module_state_graph_hash_context as graph_hash_context;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

// A pre-compilation disk routing hint for untrusted graph candidates. This owns
// identity bytes only: it neither admits source nor constructs a Complete.
pub resource Identity{bytes::Buffer canonical;bytes::Buffer value;cleanup{}}
pub fn ref u8[] canonical(ref Identity p){return bytes::view(ref p.canonical);}
pub fn ref u8[] value(ref Identity p){return bytes::view(ref p.value);}
fn u32 encoding_error(u32 code){return if(code==5){1002}else if(code==4){1003}else{1001};}
fn u32 identity_error(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn Result<bytes::Buffer,u32> decoded(ref u8[] text,ref u8[] prefix,ref mut usize work){return switch(binary::decode(text,prefix,true,work)){case .ok(p):{.ok(p)}case .err(code):{.err(identity_error(code))}};}
fn ref u8[] marker(){return b"fairyc.state-graph-cache-lookup\x00\x01";}
fn ref u8[] prefix(){return b"fylookup0-sha256-";}
// Marker, eight u64-BE length-framed validated UTF-8 environment strings in
// environment_field order, then raw stdlib, selected coordinate and exact-source
// SHA256 digests. Numeric owner IDs are deliberately absent. The distinct marker
// separates graph candidates from ordinary lookups using the same disk route.
pub fn Result<Identity,u32> compute(ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref compatibility::Environment compatibility,ref u8[] stdlibIdentity,ref mut usize work){
    if(!enc::charge(work,1)){return .err(1002);}if(actual.len>source::source_limit()){return .err(1002);}if(owner>=512){return .err(1001);}
    // Bound the entire output before any compatibility-sized allocation.
    var usize size=marker().len+96;var usize i=0;
    while(i<8){let field=compatibility::environment_field(compatibility,i);let code=enc::string_check(field,work);if(code!=0){return .err(encoding_error(code));}
        if(size>16777207 || field.len>16777207-size){return .err(1002);}size+=8+field.len;i+=1;}
    let standard=switch(decoded(stdlibIdentity,b"fypi1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    // The namespace authority intentionally validates only the selected owner.
    // Its private 1MiB meter is transferred into this shared 16MiB phase ledger.
    var usize measured=0;let checked=coordinates::validate_owner(table,owner,ref mut measured);
    if(!enc::charge(work,measured)){return .err(1002);}if(checked!=0){return .err(identity_error(checked));}
    var usize at=0;while(at<coordinates::coordinate_count(table)){
        if(!enc::charge(work,1)){return .err(1002);}if(coordinates::coordinate_at(table,at).owner==owner){break;}at+=1;}
    if(at==coordinates::coordinate_count(table)){return .err(1001);}
    let coordinate=switch(decoded(coordinates::coordinate_text(table,at),b"fypc1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    if(!enc::charge(work,actual.len+1)){return .err(1002);}var sha::State source_hash=sha::empty();if(!sha::update(ref mut source_hash,actual)){return .err(1002);}
    let source_digest=switch(binary::words(sha::finish(ref source_hash),work)){case .ok(p):{p}case .err(code):{return .err(identity_error(code));}};
    var bytes::Buffer out=bytes::empty();var u32 code=enc::raw(ref mut out,marker(),work);if(code!=0){return .err(encoding_error(code));}i=0;
    while(i<8){code=enc::string(ref mut out,compatibility::environment_field(compatibility,i),work);if(code!=0){return .err(encoding_error(code));}i+=1;}
    code=enc::raw(ref mut out,bytes::view(ref standard),work);if(code!=0){return .err(encoding_error(code));}
    code=enc::raw(ref mut out,bytes::view(ref coordinate),work);if(code!=0){return .err(encoding_error(code));}
    code=enc::raw(ref mut out,bytes::view(ref source_digest),work);if(code!=0){return .err(encoding_error(code));}
    if(bytes::len(ref out)!=size || !enc::charge(work,size+1+prefix().len+65)){return .err(1002);}var sha::State hash=sha::empty();if(!sha::update(ref mut hash,bytes::view(ref out))){return .err(1002);}
    var bytes::Buffer route=bytes::empty();if(!bytes::append(ref mut route,prefix()).is_ok || !sha::append_hex(ref hash,ref mut route)){return .err(1003);}return .ok(Identity{.canonical=out,.value=route});
}
// Adapter for retained compiler/profile facts, without any source admission.
pub fn Result<Identity,u32> derive(ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref environment::Environment environment,ref mut usize work){
    if(!enc::charge(work,0)){return .err(1002);}
    let snapshot=switch(environment::key_environment(environment,work)){case .ok(p):{p}case .err(code):{return .err(identity_error(code));}};
    return compute(actual,table,owner,ref snapshot,environment::standard_library(environment),work);
}
// Save-side adapter: source and coordinate come together from the sealed graph,
// so callers cannot pair Complete A with source/coordinate B to pollute a route.
pub fn Result<Identity,u32> for_complete(ref core::Complete complete,ref compatibility::Environment compat,ref u8[] stdlib,ref mut usize work){
    if(!enc::charge(work,1)){return .err(1002);}if(!core::is_state_graph(complete)){return .err(1001);}
    let current=core::checked_state_graph(complete);
    let coordinate=graph_hash_context::coordinate(source_identity::context(graph::body_identity(current)));
    let checked=enc::digest_check(coordinate,work);if(checked!=0){return .err(encoding_error(checked));}
    // Only a fixed 77-byte canonical text and a single row are allocated. Owner
    // zero is a local namespace slot; the admitted numeric owner is immaterial.
    if(!enc::charge(work,79)){return .err(1002);}var names=bytes::empty();if(!bytes::append(ref mut names,b"fypc1-sha256-").is_ok){return .err(1003);}
    let digits=b"0123456789abcdef";var usize i=0;while(i<32){let octet=coordinate[i];if(!bytes::append_byte(ref mut names,digits[(octet>>4) as usize]).is_ok || !bytes::append_byte(ref mut names,digits[(octet&15) as usize]).is_ok){return .err(1003);}i+=1;}
    var rows=v::empty<coordinates::Coordinate>();if(!v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=77}).is_ok){return .err(1003);}
    let table=coordinates::assemble(names,rows);
    return compute(source::source(graph::source(current)),ref table,0,compat,stdlib,work);
}
