module raw_view_intrinsics;
import syntax as syntax;
import reference_types as refs;
import pointer_types as pointers;
import std::bytes as bytes;

// Closed compiler-known constructors; data supplies bytes, never provenance.
// CStr waits for the candidate's CStr descriptor and storage-length semantics.
pub fn u32 named(ref u8[] name){
    return if(bytes::equal(name,b"str::from_raw_parts")){9}
        else if(bytes::equal(name,b"byte_slice::from_raw_parts")){10}
        else if(bytes::equal(name,b"byte_slice::from_raw_parts_mut")){11}else{0};
}
pub fn u32 call(ref syntax::Parsed p,usize id){
    if(id==0){return 0;}let e=syntax::expression_at(p,id);
    return if(e.kind==6 && syntax::expression_at(p,e.left).kind==2){named(syntax::expression_text(p,e.left))}else{0};
}
pub fn bool owner(u32 type,u32 result){
    return if(result==11){type==11 || refs::valid(type) && refs::mutable(type)}
        else{type==9 || type==10 || refs::valid(type) && !refs::mutable(type)};
}
pub fn bool data(u32 type,u32 result){
    return pointers::valid(type) && (pointers::inner(type)==3 || pointers::inner(type)==7)
        && (result!=11 || pointers::mutable(type));
}
