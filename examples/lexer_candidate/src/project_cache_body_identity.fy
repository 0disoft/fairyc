module project_cache_body_identity;
import project_compiler_core as core;
import module_project_identified_ir as checked;
import module_project_interfaces as interfaces;
import module_unit_body_ir_model as ir;
import module_state_layout as state;
import generic_package_bindings as bindings;
import project_identity_bytes as binary;
import project_cache_encoding as enc;
import sha256_stream as sha;
import std::bytes as bytes;

pub resource Identity{bytes::Buffer coordinate;bytes::Buffer canonical;bytes::Buffer digest;cleanup{}}
pub fn ref u8[] coordinate(ref Identity p){return bytes::view(ref p.coordinate);}
pub fn ref u8[] canonical(ref Identity p){return bytes::view(ref p.canonical);}
pub fn ref u8[] digest(ref Identity p){return bytes::view(ref p.digest);}
fn u32 mapped(u32 code){return if(code==5){10}else if(code==4){11}else{9};}
fn Result<bytes::Buffer,u32> namespace(ref checked::Program p,usize owner,ref mut usize work){
    let source=checked::source_provenance(p);var bytes::Buffer found=bytes::empty();var bytes::Buffer coordinate=bytes::empty();var bool present=false;var usize at=0;
    while(at<bindings::count(source)){if(!enc::charge(work,1)){return .err(10);}let row=bindings::module_at(source,at);if(row.owner==owner){let text=bindings::text(source,row.namespace);if(!enc::charge(work,text.len+1)){return .err(10);}if(!present){let raw=switch(binary::decode(text,b"package_coordinate_v1_",true,work)){case .ok(b):{b}case .err(code):{return .err(code);}};let copied=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{return .err(11);}};found=copied;coordinate=raw;present=true;}else if(!bytes::equal(bytes::view(ref found),text)){return .err(9);}}at+=1;}
    if(!present){return .err(9);}return .ok(coordinate);
}
// The ordinary Complete factory admits state declarations, never transaction
// operations. Its immutable checked IR carries the revalidated layout plan.
// This computes the exact zero-transaction member of the Go body-hash contract;
// it cannot be used for future transaction-bearing or experimental-state IR.
pub fn Result<Identity,u32> derive(ref core::Complete p,usize owner,ref mut usize work){
    if(owner>=512){return .err(9);}if(!enc::charge(work,0)){return .err(10);}let records=core::public_interfaces(p);var bool published=false;var usize at=0;
    while(at<interfaces::count(records)){if(!enc::charge(work,1)){return .err(10);}if(interfaces::owner(interfaces::at(records,at))==owner){if(published){return .err(9);}published=true;}at+=1;}if(!published){return .err(9);}
    let program=core::checked_program(p);let coord=switch(namespace(program,owner,work)){case .ok(b):{b}case .err(code):{return .err(code);}};let layouts=ir::state_layouts(checked::ir_program(program));var usize selected=state::count(layouts);at=0;
    while(at<state::count(layouts)){if(!enc::charge(work,1)){return .err(10);}if(state::at(layouts,at).owner==owner){if(selected!=state::count(layouts)){return .err(9);}selected=at;}at+=1;}
    var bytes::Buffer encoded=bytes::empty();var u32 code=enc::raw(ref mut encoded,b"fairyc.contract-state-package-body\x00\x01",work);if(code!=0){return .err(mapped(code));}code=enc::raw(ref mut encoded,bytes::view(ref coord),work);if(code!=0){return .err(mapped(code));}
    code=enc::raw(ref mut encoded,if(selected==state::count(layouts)){b"\x00"}else{b"\x01"},work);if(code!=0){return .err(mapped(code));}
    if(selected!=state::count(layouts)){let hash=switch(binary::words(state::at(layouts,selected).hash,work)){case .ok(b):{b}case .err(code):{return .err(code);}};code=enc::raw(ref mut encoded,bytes::view(ref hash),work);if(code!=0){return .err(mapped(code));}}
    code=enc::length(ref mut encoded,0,work);if(code!=0){return .err(mapped(code));}if(!enc::charge(work,bytes::len(ref encoded)+82)){return .err(10);}var sha::State hash=sha::empty();if(!sha::update(ref mut hash,bytes::view(ref encoded))){return .err(10);}var bytes::Buffer digest=bytes::empty();if(!bytes::append(ref mut digest,b"fycsbody1-sha256-").is_ok || !sha::append_hex(ref hash,ref mut digest)){return .err(11);}return .ok(Identity{.coordinate=coord,.canonical=encoded,.digest});
}
