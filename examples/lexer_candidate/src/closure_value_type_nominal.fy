module closure_value_type_nominal;
import closure_value_type_source as closure;
import module_catalog as catalog;
import syntax as syntax;
import std::vec as v;

// Inspect only the authenticated current source snapshot. The catalog exposes
// identities, not provider bytes; imported nominal layouts remain downstream
// responsibilities and gain no executable admission from this graph leaf.
// 0: no closure; 1: ordinary closure-bearing struct; 2: forbidden/failed shape.
// On a nominal cycle, independently find whether it carries a closure. Treating
// every back edge as empty would miss Option<Self> next to a closure field.
fn bool contains_type(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,ref mut v::Vec<usize> seen,usize depth){
    if(root==0){return false;}if(root>syntax::type_count(p) || depth>=64){return true;}let n=syntax::type_at(p,root);if(n.kind==17){return true;}
    if(n.kind==1 || n.kind==6){let found=catalog::lookup(c,module_index,syntax::type_text(p,if(n.kind==1){root}else{n.left}));if(found.code==0 && contains(c,p,module_index,found.item,seen,depth+1)){return true;}if(n.kind==1){return false;}}
    if(n.kind==2 || n.kind==3 || n.kind==4 || n.kind==5 || n.kind==8 || n.kind==11 || n.kind==12 || n.kind==15){if(contains_type(c,p,module_index,n.left,seen,depth+1)){return true;}}
    if(n.kind==6 || n.kind==15){var usize link=n.right;var usize steps=0;while(link!=0){if(link>syntax::type_count(p) || steps>=64){return true;}let arg=syntax::type_at(p,link);if(contains_type(c,p,module_index,arg.left,seen,depth+1)){return true;}link=arg.right;steps+=1;}}
    return false;
}
fn bool contains(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize item,ref mut v::Vec<usize> seen,usize depth){
    if(depth>=64){return true;}let d=catalog::declaration_at(c,item);if(d.module_index!=module_index){return false;}if(d.kind!=4 && d.kind!=6 && d.kind!=11){return false;}
    var usize i=0;while(i<v::len<usize>(seen)){if(*v::at<usize>(seen,i)==item){return false;}i+=1;}if(!v::push<usize>(seen,item).is_ok){return true;}
    if(syntax::problem_count(p)!=0 || d.declaration==0 || d.declaration>syntax::declaration_count(p)){return true;}let owner=syntax::declaration_at(p,d.declaration-1);i=0;
    while(i<owner.field_count){if(contains_type(c,p,d.module_index,syntax::field_at(p,owner.field_start+i).type_root,seen,depth+1)){return true;}i+=1;}discard v::pop<usize>(seen);return false;
}
fn u32 typ(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,ref mut v::Vec<usize> seen,usize depth){
    if(root==0){return 0;}if(root>syntax::type_count(p) || depth>=64){return 2;}let n=syntax::type_at(p,root);
    if(n.kind==17){return if(closure::signature(p,root)){1}else{2};}
    if(n.kind==1 || n.kind==6){let name=syntax::type_text(p,if(n.kind==6){n.left}else{root});let found=catalog::lookup(c,module_index,name);
        if(found.code==0){let result=visit(c,p,module_index,found.item,seen,depth+1);if(result!=0){return result;}}
        if(n.kind==1){return 0;}
    }
    if(n.kind==2 || n.kind==3 || n.kind==4 || n.kind==5 || n.kind==8 || n.kind==11 || n.kind==12){let child=typ(c,p,module_index,n.left,seen,depth+1);return if(child==0){0}else{2};}
    if(n.kind==6){var usize link=n.right;var usize steps=0;while(link!=0){if(link>syntax::type_count(p) || steps>=64){return 2;}let arg=syntax::type_at(p,link);if(typ(c,p,module_index,arg.left,seen,depth+1)!=0){return 2;}link=arg.right;steps+=1;}}
    if(n.kind==15){if(typ(c,p,module_index,n.left,seen,depth+1)!=0){return 2;}var usize link=n.right;var usize steps=0;while(link!=0){if(link>syntax::type_count(p) || steps>=64){return 2;}let arg=syntax::type_at(p,link);if(typ(c,p,module_index,arg.left,seen,depth+1)!=0){return 2;}link=arg.right;steps+=1;}}
    return 0;
}
fn u32 visit(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize item,ref mut v::Vec<usize> seen,usize depth){
    if(depth>=64){return 2;}let d=catalog::declaration_at(c,item);if(d.module_index!=module_index){return 0;}if(d.kind!=4 && d.kind!=6 && d.kind!=11){return 0;}
    var usize i=0;while(i<v::len<usize>(seen)){if(*v::at<usize>(seen,i)==item){var probe=v::empty<usize>();return if(contains(c,p,module_index,item,ref mut probe,0)){1}else{0};}i+=1;}
    if(!v::push<usize>(seen,item).is_ok){return 2;}
    if(syntax::problem_count(p)!=0 || d.declaration==0 || d.declaration>syntax::declaration_count(p)){return 2;}
    let owner=syntax::declaration_at(p,d.declaration-1);var u32 result=0;i=0;
    while(i<owner.field_count){let f=syntax::field_at(p,owner.field_start+i);let child=typ(c,p,d.module_index,f.type_root,seen,depth+1);
        if(child!=0){if(child==2 || owner.kind!=4 || owner.generic_parameters!=0 || closure::repr_c(p,d.declaration-1)){return 2;}result=1;}i+=1;
    }
    discard v::pop<usize>(seen);return result;
}
pub fn u32 classify(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize declaration){
    let item=catalog::identity(c,module_index,declaration);if(item==0){return 0;}var seen=v::empty<usize>();return visit(c,p,module_index,item,ref mut seen,0);
}
