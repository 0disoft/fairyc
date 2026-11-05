module project_catalog_projection;
import project_catalog_options as options;
import project_catalog_json as w;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;

fn Result<bool,u32> completeness(ref mut bytes::Buffer out,ref json::Tree t,usize row,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;discard w::pair(out,b"Relation",wire::string(t,row,b"relation"),meter)?;
    discard w::raw(out,b",",meter)?;discard w::pair(out,b"Status",wire::string(t,row,b"status"),meter)?;
    discard w::raw(out,b",\"Resolved\":",meter)?;discard w::number(out,wire::integer(t,row,b"resolved"),meter)?;
    discard w::raw(out,b",\"External\":",meter)?;discard w::number(out,wire::integer(t,row,b"external"),meter)?;
    discard w::raw(out,b",\"Unknown\":",meter)?;discard w::number(out,wire::integer(t,row,b"unknown"),meter)?;
    discard w::raw(out,b",\"Reasons\":",meter)?;discard w::strings(out,t,wire::field(t,row,b"reasons"),true,meter)?;return w::raw(out,b"}",meter);
}
pub fn Result<bool,u32> context(ref mut bytes::Buffer out,ref validate::Partition p,bool normalized,ref mut usize meter){
    let t=validate::tree(p);let h=validate::header(p);discard w::raw(out,b"{",meter)?;
    let generic=wire::string(t,h,b"generic_interface_hash");if(generic.len>0){discard w::pair(out,b"generic_interface_hash",generic,meter)?;discard w::raw(out,b",",meter)?;}
    let names=b"logical_schema|compiler_build|language|package_uid|package_instance|package_coordinate|package_source|interface_hash";var usize start=0;
    while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}if(start>0){discard w::raw(out,b",",meter)?;}discard w::pair(out,names[start..end],wire::string(t,h,names[start..end]),meter)?;start=end+1;}
    discard w::raw(out,b",",meter)?;discard w::pair(out,b"generation",wire::string(t,h,b"generation_status"),meter)?;
    discard w::raw(out,b",\"map_omissions\":",meter)?;discard w::strings(out,t,wire::field(t,h,b"omissions"),!normalized,meter)?;
    var selected=bytes::empty();var usize count=0;var usize i=0;
    if(validate::version(p)>1){while(i<validate::completeness_count(p)){if(!work::charge(meter,1)){return .err(3);}let row=validate::completeness(p,i);let kind=wire::string(t,row,b"relation");
        if(bytes::equal(kind,b"generic-specializations") || bytes::equal(kind,b"generic-template-bodies")){if(count>0){discard w::raw(ref mut selected,b",",meter)?;}discard completeness(ref mut selected,t,row,meter)?;count+=1;}i+=1;
    }}
    discard w::raw(out,b",\"completeness\":",meter)?;
    if(count==0 && !normalized){discard w::raw(out,b"null",meter)?;}else{discard w::raw(out,b"[",meter)?;discard w::raw(out,bytes::view(ref selected),meter)?;discard w::raw(out,b"]",meter)?;}return w::raw(out,b"}",meter);
}
pub fn Result<bool,u32> filters(ref mut bytes::Buffer out,ref options::Options request,usize total,ref mut usize meter){
    if(options::module_name(request).len>0){discard w::raw(out,b",",meter)?;discard w::pair(out,b"module",options::module_name(request),meter)?;}
    if(options::name(request).len>0){discard w::raw(out,b",",meter)?;discard w::pair(out,b"name",options::name(request),meter)?;}
    discard w::raw(out,b",\"kinds\":[",meter)?;var usize i=0;var bool first=true;
    while(i<5){if(options::has_kind(request,i)){if(!first){discard w::raw(out,b",",meter)?;}discard w::string(out,options::kind(i),meter)?;first=false;}i+=1;}
    discard w::raw(out,b"]",meter)?;if(options::after(request).len>0){discard w::raw(out,b",",meter)?;discard w::pair(out,b"after",options::after(request),meter)?;}
    discard w::raw(out,b",\"total\":",meter)?;return w::number(out,total,meter);
}
pub fn Result<bytes::Buffer,u32> entry(ref validate::Partition p,usize index,ref mut usize meter){
    let t=validate::tree(p);let symbol=validate::symbol(p,index);let key=wire::field(t,symbol,b"declaration");let declaration=validate::symbol_declaration(p,index);var out=bytes::empty();
    discard w::raw(ref mut out,b"{\"symbol\":{\"Declaration\":{",meter)?;
    discard w::pair(ref mut out,b"PackageInstance",wire::string(t,key,b"package_instance"),meter)?;
    discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"Module",wire::string(t,key,b"module"),meter)?;
    discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"Kind",wire::string(t,key,b"kind"),meter)?;
    discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"Name",wire::string(t,key,b"name"),meter)?;
    discard w::raw(ref mut out,b"},",meter)?;discard w::pair(ref mut out,b"Visible",wire::string(t,symbol,b"visible"),meter)?;
    discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"Full",wire::string(t,symbol,b"full"),meter)?;discard w::raw(ref mut out,b"}",meter)?;
    let names=b"module|kind|name";var usize start=0;while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,names[start..end],wire::string(t,key,names[start..end]),meter)?;start=end+1;}
    discard w::raw(ref mut out,b",\"public\":",meter)?;discard w::boolean(ref mut out,wire::boolean(t,declaration,b"public"),meter)?;
    discard w::raw(ref mut out,b",\"unsafe\":",meter)?;discard w::boolean(ref mut out,wire::boolean(t,declaration,b"unsafe"),meter)?;
    discard w::raw(ref mut out,b",\"foreign\":",meter)?;discard w::boolean(ref mut out,wire::boolean(t,declaration,b"foreign"),meter)?;
    discard w::raw(ref mut out,b"}",meter)?;return .ok(out);
}
