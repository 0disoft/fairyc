module closure_value_type_source;
import syntax as syntax;
import std::bytes as bytes;

// Syntax 17 is safe fn; syntax 15 remains the independent C callback family.
fn bool direct_i32(ref syntax::Parsed p,usize root){if(root==0 || root>syntax::type_count(p)){return false;}let n=syntax::type_at(p,root);return n.kind==1 && n.left==0 && n.right==0 && bytes::equal(syntax::type_text(p,root),b"i32");}
pub fn bool signature(ref syntax::Parsed p,usize root){
    if(root==0 || root>syntax::type_count(p)){return false;}let n=syntax::type_at(p,root);
    if(n.kind!=17 || n.left>=root || !direct_i32(p,n.left) || n.right==0 || n.right>=root){return false;}
    let link=syntax::type_at(p,n.right);return link.kind==16 && link.right==0 && link.left<n.right && direct_i32(p,link.left);
}
pub fn bool repr_c(ref syntax::Parsed p,usize declaration){
    let d=syntax::declaration_at(p,declaration);var usize i=0;
    while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"repr")){return true;}i+=1;}return false;
}
// A child extracted from a forbidden wrapper is not an independent source root.
// Generic source functions remain forbidden even with a direct written i32 fn.
pub fn bool root(ref syntax::Parsed p,usize root){
    if(root==0 || root>syntax::type_count(p)){return false;}let n=syntax::type_at(p,root);var usize i=0;
    while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);
        if(n.start>=d.start && n.end<=d.end){
            if(d.generic_parameters!=0){return false;}
            if(d.kind==3){if(d.result_type==root){return true;}var usize j=0;while(j<d.entry_count){let e=syntax::entry_at(p,d.entry_start+j);if((e.kind==1 || e.kind==2) && e.type_root==root){return true;}j+=1;}return false;}
            if(d.kind==4 && !repr_c(p,i)){var usize j=0;while(j<d.field_count){let f=syntax::field_at(p,d.field_start+j);if(f.kind==0 && f.type_root==root){return true;}j+=1;}}
            return false;
        }i+=1;
    }return false;
}
