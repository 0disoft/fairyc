module c_text;
import scalar_views as views;
import c_abi_types as c_types;
import pointer_types as pointers;
import reference_types as refs;
import std::bytes as bytes;
import enum_types as enums;
import tagged_types as tagged;
import resource_identity as resources;
import hir_shapes as shapes;

pub struct State { u32 code; }
pub fn bool text(ref mut bytes::Buffer out, ref mut State s, ref u8[] value) {
    if (s.code != 0) { return false; }
    if (bytes::len(out) > 2097152 || value.len > 2097152 - bytes::len(out)) { s.code = 4; return false; }
    if (!bytes::append(out,value).is_ok) { s.code = 3; return false; } return true;
}
pub fn bool number(ref mut bytes::Buffer out, ref mut State s, usize value) {
    if (s.code != 0) { return false; }
    if (bytes::len(out) > 2097152 - 20) { s.code = 4; return false; }
    if (!bytes::append_usize(out,value).is_ok) { s.code = 3; return false; } return true;
}
pub fn bool type(ref mut bytes::Buffer out, ref mut State s, u32 code) {
    if(c_types::valid(code)){return text(out,s,c_types::spelling(code));}
    if(pointers::valid(code)){return type(out,s,pointers::inner(code)) && text(out,s,if(pointers::mutable(code)){b" *"}else{b" const *"});}
    if(refs::valid(code)){return type(out,s,refs::inner(code)) && text(out,s,if(refs::mutable(code)){b" *"}else{b" const *"});}
    if(code==12){return text(out,s,b"struct fy_cstr");}
    if(views::extended(code)){return text(out,s,b"struct fy_slice_") && number(out,s,code as usize);}
    if (code == 11) { return text(out,s,b"struct fy_bytes_mut"); }
    if (code == 9 || code == 10) { return text(out,s,if (code == 9) { b"struct fy_text" } else { b"struct fy_bytes" }); }
    if (tagged::option(code)) { return text(out,s,b"struct fy_o_") && number(out,s,tagged::inner(code) as usize); }
    if (tagged::result(code)) { return text(out,s,b"struct fy_r_") && number(out,s,tagged::success(code) as usize) && text(out,s,b"_") && number(out,s,tagged::failure(code) as usize); }
    if (enums::valid(code)) { return text(out,s,b"struct fy_e_") && number(out,s,code as usize); }
    if(resources::valid(code)){return text(out,s,b"struct fy_resource_") && number(out,s,code as usize);}
    if (code >= 1024) { return text(out,s,b"struct fy_a_") && number(out,s,code as usize); }
    if (code >= 256) { return text(out,s,b"struct fy_s_") && number(out,s,code as usize); }
    return text(out,s,if (code == 1 || code == 6) { b"int32_t" } else if (code == 2) { b"int64_t" }
        else if (code == 3) { b"uint8_t" } else if (code == 4) { b"uint32_t" } else if (code == 5) { b"uintptr_t" } else { b"void" });
}
pub fn bool unsigned_type(ref mut bytes::Buffer out, ref mut State s, u32 code) {
    return text(out,s,if (code == 2) { b"uint64_t" } else if (code == 5) { b"uintptr_t" } else { b"uint32_t" });
}
// Only verified owned shape facts provide external C tag/field spellings.
// Ordinary nominal names remain private IDs, including inside nested records.
pub fn bool owned_type(ref shapes::Table table,ref u8[] source,ref mut bytes::Buffer out,ref mut State s,u32 code){
    if(pointers::valid(code)){return owned_type(table,source,out,s,pointers::inner(code)) && text(out,s,if(pointers::mutable(code)){b" *"}else{b" const *"});}
    if(refs::valid(code)){return owned_type(table,source,out,s,refs::inner(code)) && text(out,s,if(refs::mutable(code)){b" *"}else{b" const *"});}
    let usize slot=shapes::abi_find(table,code);
    if(slot==0){return type(out,s,code);}let a=shapes::abi_at(table,slot-1);
    return text(out,s,b"struct ") && text(out,s,source[a.name_start..a.name_end]);
}
pub fn bool field(ref shapes::Table table,ref u8[] source,ref mut bytes::Buffer out,ref mut State s,usize id){
    let f=shapes::field_at(table,id);
    return if(shapes::abi_find(table,f.owner)!=0){text(out,s,source[f.start..f.end])}else{text(out,s,b"fy_f_") && number(out,s,id)};
}
pub fn bool local(ref mut bytes::Buffer out, ref mut State s, usize id) { return text(out,s,b"fy_l_") && number(out,s,id); }
pub fn bool function(ref mut bytes::Buffer out, ref mut State s, usize id) { return text(out,s,b"fy_fn_") && number(out,s,id); }
pub fn bool jump(ref mut bytes::Buffer out, ref mut State s, usize id) { return text(out,s,b"goto fy_b_") && number(out,s,id) && text(out,s,b";\n"); }
pub fn bool operator(ref mut bytes::Buffer out, ref mut State s, u32 code) {
    return text(out,s,if (code == 1) { b"+" } else if (code == 2) { b"-" } else if (code == 3) { b"!" } else if (code == 4) { b"~" }
        else if (code == 5) { b"*" } else if (code == 6) { b"/" } else if (code == 7) { b"%" } else if (code == 8) { b"&" }
        else if (code == 9) { b"|" } else if (code == 10) { b"^" } else if (code == 11) { b"<<" } else if (code == 12) { b">>" }
        else if (code == 13) { b"==" } else if (code == 14) { b"!=" } else if (code == 15) { b"<" } else if (code == 16) { b"<=" }
        else if (code == 17) { b">" } else { b">=" });
}
pub fn bool maximum(ref mut bytes::Buffer out, ref mut State s, u32 code) {
    return text(out,s,if (code == 1) { b"INT32_MAX" } else if (code == 2) { b"INT64_MAX" } else if (code == 3) { b"UINT8_MAX" } else if (code == 4) { b"UINT32_MAX" } else { b"UINTPTR_MAX" });
}
pub fn bool minimum(ref mut bytes::Buffer out, ref mut State s, u32 code) { return text(out,s,if (code == 1) { b"INT32_MIN" } else { b"INT64_MIN" }); }
pub fn bool divisor(ref mut bytes::Buffer out, ref mut State s, u32 code) { return text(out,s,b"fy_div_") && number(out,s,code as usize); }
// Package-qualified source names distinguish files in different package instances.
// The enclosing function defines FY_UNIT_SOURCE; only actual checks carry its bytes.
pub fn bool location(ref mut bytes::Buffer out,ref mut State s,usize start,usize end){
    return text(out,s,b"FY_UNIT_SOURCE,(uint32_t)(sizeof FY_UNIT_SOURCE-1),UINT64_C(") && number(out,s,start) && text(out,s,b"),UINT64_C(") && number(out,s,end) && text(out,s,b")");
}
pub fn bool trap(ref mut bytes::Buffer out,ref mut State s,u32 category,usize start,usize end){
    return text(out,s,b"fy_unit_trap(") && number(out,s,category as usize) && text(out,s,b",") && location(out,s,start,end) && text(out,s,b")");
}
