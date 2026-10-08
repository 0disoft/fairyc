module tagged_types;
import reference_types as refs;
import literal_types as literals;
import syntax as syntax;
import type_state as state;
import tagged_encoding as encoding;
import nominal_identity as nominal;
import std::bytes as bytes;
import std::vec as v;

// Canonical scalar and nested builtin identities, independent of source order.
pub fn bool option(u32 type){return encoding::option(type);}
pub fn bool result(u32 type){return encoding::result(type);}
pub fn bool builtin(u32 type){return option(type) || result(type);}
pub fn u32 inner(u32 type){return encoding::inner(type);}
pub fn u32 success(u32 type){return encoding::success(type);}
pub fn u32 failure(u32 type){return encoding::failure(type);}
pub fn u32 make_result(u32 success,u32 failure){return encoding::make_result(success,failure);}
pub fn u32 make_option(u32 child){return encoding::make_option(child);}
fn u32 enum_leaf(ref syntax::Parsed p,usize root){
    let name=syntax::type_text(p,root);let usize owner=syntax::module_index(p,syntax::type_at(p,root).start);
    var usize i=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);
        if(d.kind==11 && syntax::module_index(p,d.start)==owner && bytes::equal(name,syntax::source_view(p)[d.name_start..d.name_end])){return 131072+state::small_id(i);}i+=1;
    }return 0;
}
fn u32 resolved(ref syntax::Parsed p,usize root,usize depth){
    if(root==0 || depth>=16){return 0;}let node=syntax::type_at(p,root);
    if(node.kind==1){let u32 scalar=state::primitive(p,root);return if(scalar>=1 && scalar<=6){scalar}else if(nominal::bound(p,root)!=0){nominal::bound(p,root)}else if(enum_leaf(p,root)!=0){enum_leaf(p,root)}else{nominal::named(p,syntax::type_text(p,root))};}
    if(node.kind==2 || node.kind==3){let u32 literal=literals::resolve(p,root);return if(literal!=0){literal}else{refs::resolve(p,root)};}
    if(node.kind!=6){return 0;}let usize count=syntax::type_argument_count(p,root);
    if(bytes::equal(syntax::type_text(p,node.left),b"Option") && count==1){
        return make_option(resolved(p,syntax::type_argument(p,root,0),depth+1));
    }
    if(bytes::equal(syntax::type_text(p,node.left),b"Result") && count==2){
        return make_result(resolved(p,syntax::type_argument(p,root,0),depth+1),resolved(p,syntax::type_argument(p,root,1),depth+1));
    }return 0;
}
pub fn u32 resolve(ref syntax::Parsed p,usize root){let u32 type=resolved(p,root,0);return if(builtin(type)){type}else{0};}
pub fn usize first(ref syntax::Parsed p,u32 type){
    var usize i=1;while(i<=syntax::type_count(p)){if(resolve(p,i)==type){return i;}i+=1;}return 0;
}
// Nested arguments themselves have parsed type nodes; retain each distinct
// instantiated layout even if it appears only beneath another annotation.
pub fn usize field_slots(ref syntax::Parsed p){
    var usize total=0;var usize i=1;while(i<=syntax::type_count(p)){let u32 type=resolve(p,i);
        if(type!=0 && first(p,type)==i){total+=if(option(type)){3}else{4};}i+=1;
    }return total;
}
pub fn usize payload_for(ref syntax::Parsed p,u32 type,bool first_variant){
    assert(builtin(type) && first(p,type)!=0);var usize offset=syntax::field_count(p);var usize i=1;
    while(i<=syntax::type_count(p)){let u32 prior=resolve(p,i);
        if(prior!=0 && prior<type && first(p,prior)==i){offset+=if(option(prior)){3}else{4};}i+=1;
    }return offset+if(option(type) || first_variant){1}else{3};
}
pub fn usize payload(ref syntax::Parsed p,u32 type){return payload_for(p,type,true);}
pub fn u32 payload_type(u32 type,bool first_variant){return if(option(type)){inner(type)}else if(first_variant){success(type)}else{failure(type)};}
pub fn usize member(ref syntax::Parsed p,u32 type,bool first_variant){
    return if(option(type)){payload(p,type)+if(first_variant){1}else{2}}else{payload_for(p,type,first_variant)+1};
}
pub fn usize count(ref syntax::Parsed p){var usize n=0;var usize i=1;while(i<=syntax::type_count(p)){let u32 type=resolve(p,i);if(option(type) && first(p,type)==i){n+=1;}i+=1;}return n;}
pub fn usize result_count(ref syntax::Parsed p){var usize n=0;var usize i=1;while(i<=syntax::type_count(p)){let u32 type=resolve(p,i);if(result(type) && first(p,type)==i){n+=1;}i+=1;}return n;}
pub fn bool instances(ref syntax::Parsed p,ref mut v::Vec<u32> types){
    var usize i=1;while(i<=syntax::type_count(p)){let u32 type=resolve(p,i);
        if(type!=0 && first(p,type)==i){if(v::len<u32>(types)>=512 || !v::push<u32>(types,type).is_ok){return false;}}i+=1;
    }
    i=1;while(i<v::len<u32>(types)){let u32 value=*v::at<u32>(types,i);var usize j=i;
        while(j!=0 && *v::at<u32>(types,j-1)>value){let u32 old=*v::at<u32>(types,j-1);{let ref mut u32 slot=v::at_mut<u32>(types,j);*slot=old;}j-=1;}
        {let ref mut u32 slot=v::at_mut<u32>(types,j);*slot=value;}i+=1;
    }return true;
}
