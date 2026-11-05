module project_catalog_json;
import project_json_string as quoted;
import project_json_tree as json;
import project_atlas_wire as wire;
import project_path as work;
import std::bytes as bytes;

// Catalog CLI policy permits up to 1 GiB. Shared work remains independently
// bounded; this writer does not inherit the unrelated 4 MiB map-output limit.
pub fn Result<bool,u32> raw(ref mut bytes::Buffer out,ref u8[] s,ref mut usize meter){
    if(bytes::len(out)>1073741824 || s.len>1073741824-bytes::len(out) || !work::charge(meter,s.len+1)){return .err(3);}
    return if(bytes::append(out,s).is_ok){.ok(true)}else{.err(2)};
}
pub fn Result<bool,u32> string(ref mut bytes::Buffer out,ref u8[] s,ref mut usize meter){let q=quoted::quote(s,meter)?;return raw(out,bytes::view(ref q),meter);}
pub fn Result<bool,u32> pair(ref mut bytes::Buffer out,ref u8[] name,ref u8[] value,ref mut usize meter){discard string(out,name,meter)?;discard raw(out,b":",meter)?;return string(out,value,meter);}
pub fn Result<bool,u32> number(ref mut bytes::Buffer out,usize n,ref mut usize meter){var temp=bytes::empty();if(!bytes::append_usize(ref mut temp,n).is_ok){return .err(2);}return raw(out,bytes::view(ref temp),meter);}
pub fn Result<bool,u32> boolean(ref mut bytes::Buffer out,bool value,ref mut usize meter){return raw(out,if(value){b"true"}else{b"false"},meter);}
// Only semantically admitted canonical string arrays may enter this projection.
// Go query records retain nil Reasons; only top-level catalog arrays normalize.
pub fn Result<bool,u32> strings(ref mut bytes::Buffer out,ref json::Tree t,usize at,bool nullable,ref mut usize meter){
    if(nullable && wire::count(t,at)==0){return raw(out,b"null",meter);}
    let first=json::token(t,at).start;let last=json::token(t,json::next(t,at)-1).end;
    return raw(out,json::source(t)[first..last],meter);
}
