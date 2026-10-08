module array_types;
import array_identity as identity;
import borrow_shapes as borrows;
import syntax as syntax;
import type_state as state;
import nominal_identity as nominal;
import resource_properties as properties;
import enum_types as enums;
import tagged_types as tagged;

// Private canonical IDs preserve the rank-one scalar IDs. Rank two uses a
// disjoint radix-1024 outer length and one rank-one element ID. Rank two remains
// 1..64 per dimension; rank one admits 1..4096 through separate closed IDs.
// Higher ranks, resource and view elements stay closed.
pub fn bool flat(u32 type) { return identity::flat(type); }
// Composite arrays own their element facts in HIR. Keys preserve nominal and
// scalar-tag identities; larger tags use their first type node. Rank two packs
// the child key and both dimensions, including fully inferred child arrays.
fn bool valid_key(u32 key) { return key<1066 || key>=2048 && key<6144; }
pub fn bool rank_two(u32 type) { return type>=4194304 && type<29360128 && valid_key((type-4194304)/4096); }
pub fn bool owned(u32 type) {
    return rank_two(type) || type>=524288 && type<917504 && valid_key((type-524288)/64) || large_owned(type);
}
pub fn bool large_owned(u32 type){return type>=67108928 && type<92274688 && (type-67108864)%4096>=64 && valid_key((type-67108864)/4096);}
pub fn u32 key(u32 type) { assert(owned(type)); return if(rank_two(type)){(type-4194304)/4096}else if(large_owned(type)){(type-67108864)/4096}else{(type-524288)/64}; }
pub fn bool valid(u32 type) { return owned(type) || flat(type) || type >= 5120 && type <= 70150 && (type - 4096) / 1024 >= 1 && (type - 4096) / 1024 <= 64 && flat(1024 + (type - 4096) % 1024); }
pub fn u32 element(u32 type) { assert(valid(type) && !owned(type)); return if (flat(type)) { identity::element(type) } else { 1024 + (type - 4096) % 1024 }; }
pub fn usize length(u32 type) { assert(valid(type)); return if(rank_two(type)){((type-4194304)%64+1) as usize}else if(large_owned(type)){((type-67108864)%4096+1) as usize}else if(owned(type)){((type-524288)%64+1) as usize}else if (flat(type)) { identity::length(type) } else { ((type - 4096) / 1024) as usize }; }
pub fn u32 encoded_element(u32 type) {
    if(rank_two(type)){return 524288+key(type)*64+(type-4194304)/64%64;}
    let u32 k=key(type);return if(k<512){256+k}else if(k<1024){131072+k-512}else if(k<1030){196609+k-1024}else if(k<1066){262145+k-1030}else{0};
}
pub fn u32 make(u32 element, usize count) {
    let u32 scalar=identity::make(element,count);if(scalar!=0){return scalar;}
    if (count < 1 || count > 64) { return 0; }
    return if (flat(element) && !identity::large_flat(element)) { 4096 + state::small_id(count) * 1024 + element - 1024 } else { 0 };
}
pub fn usize count(ref u8[] text) {
    if (text.len == 0) { return 0; }
    var usize base = 10; var usize i = 0; var usize n = 0;
    if (text.len >= 2 && text[0] == 48) { if (text[1] == 120 || text[1] == 88) { base = 16; i = 2; } else if (text[1] == 98 || text[1] == 66) { base = 2; i = 2; } }
    while (i < text.len) {
        let u8 b = text[i]; i += 1; if (b == 95) { continue; }
        if (!(b >= 48 && b <= 57) && !(b >= 65 && b <= 70) && !(b >= 97 && b <= 102)) { return 0; }
        let usize digit = if (b >= 48 && b <= 57) { (b - 48) as usize } else if (b >= 65 && b <= 70) { (b - 65 + 10) as usize } else { (b - 97 + 10) as usize };
        if (digit >= base || n > (4096 - digit) / base) { return 0; }
        n = n * base + digit;
    }
    return if (n >= 1 && n <= 4096) { n } else { 0 };
}
pub fn u32 resolve(ref syntax::Parsed p, usize root) {
    if(root==0 || root>syntax::type_count(p)){return 0;}
    let node = syntax::type_at(p,root);
    if (node.kind != 8 || syntax::type_at(p,node.right).kind != 9) { return 0; }
    let child = syntax::type_at(p,node.left); var u32 element = 0;
    if (child.kind == 1) {
        element=state::primitive(p,node.left);
        if(element==7){element=if(nominal::bound(p,node.left)!=0){nominal::bound(p,node.left)}else{nominal::named(p,syntax::type_text(p,node.left))};}
        if(element==0){element=enums::named(p,syntax::type_text(p,node.left));}
    }else if(child.kind==6){element=tagged::resolve(p,node.left);}
    else if(child.kind==8 && syntax::type_at(p,child.left).kind!=8){element=resolve(p,node.left);}
    return make_source(p,element,count(syntax::type_text(p,node.right)));
}
pub fn u32 make_source(ref syntax::Parsed p,u32 child,usize count) {
    if(borrows::count(child)!=0 || properties::needs_drop(child)){return 0;}
    let u32 legacy=make(child,count);if(legacy!=0){return legacy;}
    if(count<1 || count>4096){return 0;}var u32 k=16384;
    if(nominal::valid(child)){k=child-256;}
    else if(child>=131072 && child<=131583){k=512+child-131072;}
    else if(child>=196609 && child<=196614){k=1024+child-196609;}
    else if(child>=262145 && child<=262180){k=1030+child-262145;}
    else if(tagged::builtin(child)){
        let usize first=tagged::first(p,child);
        // Type nodes have a 4096-entry arena, unlike the 1024-entry
        // declaration/field conversion. Keep narrowing inside this bound.
        if(first>0 && first<=4096){k=2048;var usize left=first-1;while(left!=0){left-=1;k+=1;}}
    }else if(owned(child) && !rank_two(child) && !large_owned(child) && count<=64){return 4194304+key(child)*4096+state::small_id(length(child)-1)*64+state::small_id(count-1);}
    return if(k==16384){0}else if(count>64){67108864+k*4096+identity::count_id(count-1)}else{524288+k*64+state::small_id(count-1)};
}
pub fn u32 source_element(ref syntax::Parsed p,u32 type) {
    if(!valid(type)){return 0;}if(!owned(type)){return element(type);}
    let u32 simple=encoded_element(type);if(simple!=0){return simple;}
    let u32 k=key(type);let usize node=(k-2048) as usize+1;
    if(node>syntax::type_count(p)){return 0;}
    return tagged::resolve(p,node);
}
