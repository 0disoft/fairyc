module pointer_types;
import syntax as syntax;
import reference_types as refs;
import type_state as state;
import nominal_identity as nominal;
import std::bytes as bytes;

import pointer_identity as identity;
pub fn bool valid(u32 type){return identity::valid(type);}
pub fn u32 inner(u32 type){return identity::inner(type);}
pub fn bool mutable(u32 type){return identity::mutable(type);}
pub fn u32 make(u32 type,bool exclusive){return identity::make(type,exclusive);}
pub fn u32 resolve(ref syntax::Parsed p,usize root){
    let n=syntax::type_at(p,root);if(n.kind!=11 && n.kind!=12){return 0;}
    let child=syntax::type_at(p,n.left);if(n.kind==12 && child.kind==12 && syntax::type_at(p,child.left).kind==1 && bytes::equal(syntax::type_text(p,child.left),b"void")){return identity::make(1048591,true);}
    if(child.kind!=1){return 0;}
    var u32 type=state::primitive(p,n.left);if(type==7 && !bytes::equal(syntax::type_text(p,n.left),b"void")){let u32 bound=nominal::bound(p,n.left);type=if(bound!=0){bound}else{nominal::named(p,syntax::type_text(p,n.left))};if(!nominal::valid(type)){return 0;}}
    return make(type,n.kind==12);
}

// The caller separately validates direct-place syntax for opaque erasure.
// Safe view descriptors contribute their data address, never the descriptor.
pub fn bool cast_allowed(u32 source,u32 target){
    if(!valid(target)){return false;}
    if(refs::valid(source)){return (!mutable(target) || refs::mutable(source)) && (inner(target)==refs::inner(source) || inner(target)==7);}
    return inner(target)==3 && (source==9 && !mutable(target) || source==10 && !mutable(target) || source==12 && !mutable(target) || source==11);
}
