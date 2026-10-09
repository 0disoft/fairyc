module storage_intrinsics;
import scalar_views as views;
import syntax as syntax;
import type_state as scalar;
import sdk_scopes as sdk;
import reference_types as refs;
import pointer_types as pointers;
import resource_identity as resources;
import nominal_identity as nominal;
import std::bytes as bytes;

// Private SDK storage operations relocate scalars, structs and resource values.
// Safe element borrows retain the owner independently of the raw slot.
pub fn u32 named(ref u8[] name){return if(bytes::equal(name,b"storage_size")){1}else if(bytes::equal(name,b"storage_alignment")){2}else if(bytes::equal(name,b"storage_take")){3}else if(bytes::equal(name,b"storage_put")){4}else if(bytes::equal(name,b"storage_element_address")){5}else if(bytes::equal(name,b"storage_borrow")){6}else if(bytes::equal(name,b"storage_borrow_mut")){7}else{0};}
// A private slice type argument describes a view, not a stored unsized value.
pub fn usize annotation_root(ref syntax::Parsed p,usize specialization,usize root,bool vector){let e=syntax::expression_at(p,specialization);let u32 code=named(syntax::expression_text(p,e.left));let n=syntax::type_at(p,root);return if(vector && (code==6 || code==7) && n.kind==4){n.left}else{root};}
pub fn u32 call(ref syntax::Parsed p,usize id){
    if(id==0){return 0;}let e=syntax::expression_at(p,id);if(e.kind!=6){return 0;}
    let g=syntax::expression_at(p,e.left);if(g.kind!=20){return 0;}let base=syntax::expression_at(p,g.left);
    let u32 code=if(base.kind==2){named(syntax::expression_text(p,g.left))}else{0};
    if((code==6 || code==7) && g.right!=0){let arg=syntax::type_at(p,g.right);if(arg.left!=0 && syntax::type_at(p,arg.left).kind==4){return code+2;}}
    return code;
}
pub fn u32 element(ref syntax::Parsed p,usize id){
    let e=syntax::expression_at(p,id);let g=syntax::expression_at(p,e.left);if(g.right==0){return 0;}let arg=syntax::type_at(p,g.right);
    if(arg.right!=0 || arg.left==0){return 0;}let item=syntax::type_at(p,arg.left);let usize root=if(call(p,id)>=8 && item.kind==4){item.left}else{arg.left};
    if(root==0 || syntax::type_at(p,root).kind!=1){return 0;}
    let u32 type=scalar::primitive(p,root);if(type>=1 && type<=6){return type;}
    let u32 bound=nominal::bound(p,root);return if(resources::valid(bound) || nominal::valid(bound)){bound}else{0};
}
pub fn bool scalar_element(u32 element){return element>=1 && element<=6;}
pub fn bool stored_element(u32 element){return scalar_element(element) || nominal::valid(element) || resources::valid(element);}
pub fn u32 result(u32 code,u32 element){return if(!stored_element(element) || code==0 || code>9){0}else if(code<=2){5}else if(code==3){element}else if(code==4){7}else if(code==5){pointers::make(7,true)}else if(code>=8){views::make(element,code==9)}else{refs::make(element,code==7)};}
pub fn usize arguments(u32 code){return if(code<=2){0}else if(code==3){1}else if(code>=8){3}else{2};}
pub fn bool owner(u32 code,u32 type){return code>=6 && code<=9 && refs::valid(type) && resources::valid(refs::inner(type)) && refs::mutable(type)==(code==7 || code==9);}
pub fn u32 argument_type(u32 code,u32 element,usize index){return if(index==2){5}else if(index==0 || code>=6){pointers::make(7,true)}else if(code==5){5}else{element};}
pub fn bool allowed(ref syntax::Parsed p,usize id){let e=syntax::expression_at(p,id);return sdk::parsed_storage(p,e.start,e.end,call(p,id));}
