module tagged_types;
import syntax as syntax;
import type_state as state;
import std::bytes as bytes;

// Builtin identities are separate from user enums and array encodings.
// Scalar Option instances are canonical across repeated spellings.
pub fn bool option(u32 type) { return type >= 196609 && type <= 196614; }
pub fn u32 inner(u32 type) { assert(option(type)); return type - 196608; }
pub fn u32 resolve(ref syntax::Parsed p, usize root) {
    if (root == 0) { return 0; } let node=syntax::type_at(p,root);
    if (node.kind != 6 || !bytes::equal(syntax::type_text(p,node.left),b"Option") || syntax::type_argument_count(p,root) != 1) { return 0; }
    let usize argument=syntax::type_argument(p,root,0);
    if (syntax::type_at(p,argument).kind != 1) { return 0; }
    let u32 child=state::primitive(p,argument);return if(child>=1 && child<=6){196608+child}else{0};
}
pub fn usize first(ref syntax::Parsed p,u32 type) {
    var usize i=1;while(i<=syntax::type_count(p)){if(resolve(p,i)==type){return i;}i+=1;}return 0;
}
pub fn usize count(ref syntax::Parsed p) {
    var usize n=0;var u32 type=196609;while(type<=196614){if(first(p,type)!=0){n+=1;}type+=1;}return n;
}
pub fn usize payload(ref syntax::Parsed p,u32 type) {
    assert(option(type) && first(p,type)!=0);var usize offset=syntax::field_count(p);var u32 at=196609;
    while(at<type){if(first(p,at)!=0){offset+=3;}at+=1;}return offset+1;
}
pub fn usize member(ref syntax::Parsed p,u32 type,bool some) { return payload(p,type)+if(some){1}else{2}; }
