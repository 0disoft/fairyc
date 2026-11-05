module project_pack_context;
import project_pack_options as options;
import project_pack_projection as projection;
import project_catalog_json as w;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_path as work;
import std::bytes as bytes;

pub resource Context{bytes::Buffer json;bytes::Buffer completeness;cleanup{}}
pub fn ref u8[] json(ref Context p){return bytes::view(ref p.json);}
pub fn ref u8[] completeness(ref Context p){return bytes::view(ref p.completeness);}
fn bool wanted(ref u8[] relation,usize mask,u32 version){
    if(version>1 && (bytes::equal(relation,b"generic-specializations") || bytes::equal(relation,b"generic-template-bodies"))){return true;}
    if(bytes::equal(relation,b"borrows")){return (mask & 6)!=0;}if(bytes::equal(relation,b"calls")){return (mask & 24)!=0;}
    var usize i=0;while(i<10){if((mask & (1<<i))!=0 && bytes::equal(relation,options::relation(i))){return true;}i+=1;}return false;
}
pub fn Result<Context,u32> build(ref validate::Partition p,usize mask,ref mut usize meter){
    let t=validate::tree(p);let h=validate::header(p);var completeness=bytes::empty();var usize i=0;var usize included=0;
    while(i<validate::completeness_count(p)){if(!work::charge(meter,1)){return .err(3);}let row=validate::completeness(p,i);if(wanted(wire::string(t,row,b"relation"),mask,validate::version(p))){
        if(included>0){discard w::raw(ref mut completeness,b",",meter)?;}let fact=projection::completeness(p,i,meter)?;discard w::raw(ref mut completeness,bytes::view(ref fact),meter)?;included+=1;
    }i+=1;}
    var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;let generic=wire::string(t,h,b"generic_interface_hash");if(generic.len>0){discard w::pair(ref mut out,b"generic_interface_hash",generic,meter)?;discard w::raw(ref mut out,b",",meter)?;}
    discard projection::fields(ref mut out,t,h,b"logical_schema=logical_schema|compiler_build=compiler_build|language=language|package_uid=package_uid|package_instance=package_instance|package_coordinate=package_coordinate|package_source=package_source|interface_hash=interface_hash|generation=generation_status",meter)?;
    discard w::raw(ref mut out,b",\"map_omissions\":",meter)?;discard w::strings(ref mut out,t,wire::field(t,h,b"omissions"),true,meter)?;
    discard w::raw(ref mut out,b",\"completeness\":",meter)?;if(included==0){discard w::raw(ref mut out,b"null",meter)?;}else{discard w::raw(ref mut out,b"[",meter)?;discard w::raw(ref mut out,bytes::view(ref completeness),meter)?;discard w::raw(ref mut out,b"]",meter)?;}
    discard w::raw(ref mut out,b"}",meter)?;return .ok(Context{.json=out,.completeness});
}
