module module_state_graph_body_hash;
import module_state_graph_program as program;
import module_state_graph_model as m;
import module_state_graph_hash_context as context;
import module_state_graph_hash_encoding as enc;
import module_state_layout as state;
import project_identity_bytes as binary;
import project_cache_encoding as budget;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

// Exact one-transaction Go package body identity. Detached identity does not
// bind current Complete/source, authorize cache reuse, or execute storage.
pub resource Identity{bytes::Buffer canonical;bytes::Buffer digest;cleanup{}}
pub fn ref u8[] canonical(ref Identity value){return bytes::view(ref value.canonical);}
pub fn ref u8[] digest(ref Identity value){return bytes::view(ref value.digest);}
pub fn Result<Identity,u32> derive(ref program::Verified proof,ref context::Context identity,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}let root=program::root(proof);if(context::effects(identity)!=m::header(root).effects){return .err(1001);}
    let helpers=program::helper_count(proof);var u32 mode=if(helpers==0){0}else{1};var usize i=0;
    if(!budget::charge(work,v::len<m::Operation>(m::operations(root))+1)){return .err(1002);}if(enc::has_borrows(root)){mode=2;}
    while(i<helpers){let h=program::helper(proof,i);if(!budget::charge(work,v::len<m::Operation>(m::operations(h))+1)){return .err(1002);}if(enc::has_borrows(h)){mode=2;}i+=1;}
    var enc::Status status=enc::Status{.code=0};var bytes::Buffer encoded=bytes::empty();enc::raw(ref mut encoded,b"fairyc.contract-state-package-body\x00\x01",ref mut status,work);enc::raw(ref mut encoded,context::coordinate(identity),ref mut status,work);enc::raw(ref mut encoded,b"\x01",ref mut status,work);
    let layout_hash=switch(binary::words(state::at(program::layouts(proof),m::header(root).layout).hash,work)){case .ok(p):{p}case .err(code):{return .err(if(code==11){1003}else{1002});}};enc::raw(ref mut encoded,bytes::view(ref layout_hash),ref mut status,work);enc::number(ref mut encoded,1,ref mut status,work);enc::text(ref mut encoded,context::descriptor(identity),ref mut status,work);
    if(mode!=0){enc::text(ref mut encoded,if(mode==2){b"borrows-v1"}else{b"helpers-v1"},ref mut status,work);}enc::executable(ref mut encoded,proof,root,mode,ref mut status,work);
    if(mode!=0){enc::number(ref mut encoded,helpers,ref mut status,work);i=0;while(i<helpers && status.code==0){enc::text(ref mut encoded,program::helper_name(proof,i),ref mut status,work);enc::executable(ref mut encoded,proof,program::helper(proof,i),mode,ref mut status,work);i+=1;}}
    if(status.code!=0){return .err(status.code);}if(!budget::charge(work,bytes::len(ref encoded)+82)){return .err(1002);}var sha::State hash=sha::empty();if(!sha::update(ref mut hash,bytes::view(ref encoded))){return .err(1002);}var bytes::Buffer digest=bytes::empty();if(!bytes::append(ref mut digest,b"fycsbody1-sha256-").is_ok || !sha::append_hex(ref hash,ref mut digest)){return .err(1003);}return .ok(Identity{.canonical=encoded,.digest});
}
