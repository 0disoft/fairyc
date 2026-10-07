module pointer_types;
import syntax as syntax;
import reference_types as refs;
import type_state as state;
import std::bytes as bytes;

// Nullable inert raw pointers have their own closed type namespace. They do
// not carry safe-reference origins, acquire loans, or imply dereference rights.
pub fn bool element(u32 type){return type>=1 && type<=5 || type==7;}
pub fn bool valid(u32 type){return type>=1048578 && type<=1048591 && element((type-1048576)/2);}
pub fn u32 inner(u32 type){assert(valid(type));return (type-1048576)/2;}
pub fn bool mutable(u32 type){return valid(type) && type%2==1;}
pub fn u32 make(u32 type,bool exclusive){return if(element(type)){1048576+type*2+if(exclusive){1}else{0}}else{0};}
pub fn u32 resolve(ref syntax::Parsed p,usize root){
    let n=syntax::type_at(p,root);if(n.kind!=11 && n.kind!=12 || syntax::type_at(p,n.left).kind!=1){return 0;}
    let u32 type=state::primitive(p,n.left);if(type==7 && !bytes::equal(syntax::type_text(p,n.left),b"void")){return 0;}
    return make(type,n.kind==12);
}

// The caller separately validates direct-place syntax for opaque erasure.
// Safe view descriptors contribute their data address, never the descriptor.
pub fn bool cast_allowed(u32 source,u32 target){
    if(!valid(target)){return false;}
    if(refs::valid(source)){return (!mutable(target) || refs::mutable(source)) && (inner(target)==refs::inner(source) || inner(target)==7);}
    return inner(target)==3 && (source==9 && !mutable(target) || source==10 && !mutable(target) || source==11);
}
