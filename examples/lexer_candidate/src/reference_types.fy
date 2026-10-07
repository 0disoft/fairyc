module reference_types;
import syntax as syntax;
import nominal_identity as nominal;
import resource_identity as resources;
import type_state as state;
// Safe references occupy the high-half namespace, disjoint from tagged IDs.
pub fn bool referent(u32 type){return type>=1 && type<=6 || type>=256 && type<=767 || resources::valid(type);}
pub fn bool valid(u32 type){return type>=2147483648 && type<2147681280 && referent((type-2147483648)/2);}
pub fn u32 inner(u32 type){assert(valid(type));return (type-2147483648)/2;}
pub fn bool mutable(u32 type){return valid(type) && type%2==1;}
pub fn u32 make(u32 type,bool exclusive){return if(referent(type)){2147483648+type*2+if(exclusive){1}else{0}}else{0};}
pub fn u32 resolve(ref syntax::Parsed p,usize root){let n=syntax::type_at(p,root);if(n.kind!=2 && n.kind!=3 || syntax::type_at(p,n.left).kind!=1){return 0;}
    let text=syntax::type_text(p,n.left);let u32 scalar=state::primitive(p,n.left);let u32 type=if(scalar!=7){scalar}else{nominal::named(p,text)};return make(type,n.kind==3);
}
