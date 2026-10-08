module storage_intrinsics;
import syntax as syntax;
import type_state as scalar;
import sdk_scopes as sdk;
import std::bytes as bytes;

// Private SDK storage operations. The first executable lane accepts copy
// scalars; nominal/resource elements and owner-bound storage borrows stay closed.
pub fn u32 named(ref u8[] name){return if(bytes::equal(name,b"storage_size")){1}else if(bytes::equal(name,b"storage_alignment")){2}else if(bytes::equal(name,b"storage_take")){3}else if(bytes::equal(name,b"storage_put")){4}else{0};}
pub fn u32 call(ref syntax::Parsed p,usize id){
    if(id==0){return 0;}let e=syntax::expression_at(p,id);if(e.kind!=6){return 0;}
    let g=syntax::expression_at(p,e.left);if(g.kind!=20){return 0;}let base=syntax::expression_at(p,g.left);
    return if(base.kind==2){named(syntax::expression_text(p,g.left))}else{0};
}
pub fn u32 element(ref syntax::Parsed p,usize id){
    let e=syntax::expression_at(p,id);let g=syntax::expression_at(p,e.left);if(g.right==0){return 0;}let arg=syntax::type_at(p,g.right);
    if(arg.right!=0 || arg.left==0 || syntax::type_at(p,arg.left).kind!=1){return 0;}
    let u32 type=scalar::primitive(p,arg.left);return if(type>=1 && type<=6){type}else{0};
}
pub fn bool allowed(ref syntax::Parsed p,usize id){let e=syntax::expression_at(p,id);return sdk::parsed_storage(p,e.start,e.end,call(p,id));}
