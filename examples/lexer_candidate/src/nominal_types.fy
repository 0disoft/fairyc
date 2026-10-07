module nominal_types;
import pointer_types as pointers;
import borrow_shapes as borrows;
import reference_types as refs;
import array_types as arrays;
import tagged_types as tagged;
import enum_types as enums;
import literal_types as literals;
import syntax as syntax;
import nominal_identity as identity;
import resource_identity as resources;
import type_state as state;
import std::bytes as bytes;

// Private nominal ID = 256 + zero-based declaration index. Scalar tags stay
// unchanged. Identity is declaration based, never structural field equality.
pub fn u32 named(ref syntax::Parsed p, ref u8[] spelling) {
    return identity::named(p,spelling);
}
pub fn u32 resolve(ref syntax::Parsed p, usize root) {
    if (root == 0) { return 0; }
    if(syntax::type_at(p,root).kind==11 || syntax::type_at(p,root).kind==12){return pointers::resolve(p,root);}
    if (syntax::type_at(p,root).kind == 6) { let u32 type=tagged::resolve(p,root);return if(borrows::count(type)<=1){type}else{0}; }
    if ((syntax::type_at(p,root).kind == 2 || syntax::type_at(p,root).kind == 3)) { let u32 literal=literals::resolve(p,root);return if(literal!=0){literal}else{refs::resolve(p,root)}; }
    if (syntax::type_at(p,root).kind == 8) { return arrays::resolve(p,root); }
    if (syntax::type_at(p,root).kind != 1) { return 0; }
    let ref u8[] spelling = syntax::type_text(p,root);
    let u32 primitive = state::primitive(p,root);
    if (primitive != 7 || bytes::equal(spelling,b"void")) { return primitive; }
    let u32 enumeration = enums::named(p,spelling);
    return if (enumeration != 0) { enumeration } else { named(p,spelling) };
}
pub fn usize field(ref syntax::Parsed p, u32 type, ref u8[] name) {
    if(!identity::valid(type) && !resources::valid(type)){return 0;}
    let usize index=(type-if(resources::valid(type)){98304}else{256}) as usize;
    if(index>=syntax::declaration_count(p)){return 0;}
    let syntax::Declaration d = syntax::declaration_at(p,index);
    if (d.kind != if(resources::valid(type)){6}else{4}) { return 0; }
    var usize i = d.field_start; while (i < d.field_start + d.field_count) {
        let f = syntax::field_at(p,i);
        if (bytes::equal(syntax::source_view(p)[f.name_start..f.name_end],name)) { return i + 1; } i += 1;
    }
    return 0;
}
fn state::Problem visit(ref syntax::Parsed p, usize id, ref mut u8[] colors, usize depth) {
    let syntax::Declaration d = syntax::declaration_at(p,id);
    if (colors[id] == 2) { return state::ok(); }
    if (colors[id] == 1) { return state::fail(20,d.name_start,d.name_end,0,0); }
    if (depth >= 64) { return state::fail(10,d.start,d.end,0,0); }
    colors[id] = 1; var usize i = d.field_start;
    while (i < d.field_start + d.field_count) {
        let f = syntax::field_at(p,i); let u32 type = resolve(p,f.type_root);
        if (type == 0 || type == 7 || borrows::count(type)!=0) { return state::fail(16,f.start,f.end,0,type); }
        if((d.kind==4 || arrays::valid(type)) && contains_resource(p,type,0)){return state::fail(43,f.start,f.end,0,type);}
        var usize j = d.field_start; while (j < i) {
            let previous = syntax::field_at(p,j);
            if (bytes::equal(syntax::source_view(p)[f.name_start..f.name_end],syntax::source_view(p)[previous.name_start..previous.name_end])) { return state::fail(17,f.name_start,f.name_end,0,0); } j += 1;
        }
        let state::Problem problem = visit_type(p,type,colors,depth + 1);
        if (problem.code != 0) { return problem; }
        i += 1;
    }
    colors[id] = 2; return state::ok();
}
fn state::Problem visit_type(ref syntax::Parsed p, u32 type, ref mut u8[] colors, usize depth) {
    if(depth>=64){return state::fail(10,0,syntax::source_view(p).len,0,0);}
    if(identity::valid(type) || resources::valid(type)){return visit(p,(type-if(resources::valid(type)){98304}else{256}) as usize,colors,depth);}
    if(tagged::option(type)){return visit_type(p,tagged::inner(type),colors,depth+1);}
    if(tagged::result(type)){
        let state::Problem success=visit_type(p,tagged::success(type),colors,depth+1);
        return if(success.code!=0){success}else{visit_type(p,tagged::failure(type),colors,depth+1)};
    }
    if(arrays::owned(type)){return visit_type(p,arrays::source_element(p,type),colors,depth+1);}
    return state::ok(); // Scalar arrays and scalar-only user-enum layouts.
}
// The reference admits nested field stores only through a plain stored graph.
// Tagged-containing structs keep direct field replacement, but not deep stores.
pub fn bool contains_tagged(ref syntax::Parsed p, u32 type, usize depth) {
    if (enums::valid(type) || tagged::builtin(type) || depth >= 64) { return true; }
    if(arrays::owned(type)){return contains_tagged(p,arrays::source_element(p,type),depth+1);}
    if (!identity::valid(type)) { return false; }
    let d = syntax::declaration_at(p,(type - 256) as usize); var usize i = d.field_start;
    while (i < d.field_start + d.field_count) {
        if (contains_tagged(p,resolve(p,syntax::field_at(p,i).type_root),depth + 1)) { return true; } i += 1;
    }
    return false;
}
pub fn state::Problem validate(ref syntax::Parsed p) {
    if (syntax::declaration_count(p) > 512) { return state::fail(10,0,syntax::source_view(p).len,0,0); }
    var u8[512] colors = [0;512]; var usize i = 0;
    while (i < syntax::declaration_count(p)) {
        let syntax::Declaration d = syntax::declaration_at(p,i);
        if (d.kind == 4 || d.kind==6) { let state::Problem problem = visit(p,i,ref mut colors[..],0); if (problem.code != 0) { return problem; } } i += 1;
    }
    return state::ok();
}
pub fn bool contains_resource(ref syntax::Parsed p,u32 type,usize depth) {
    if(depth>=64){return true;}
    if(resources::valid(type)){return true;}
    if(tagged::option(type)){return contains_resource(p,tagged::inner(type),depth+1);}
    if(tagged::result(type)){return contains_resource(p,tagged::success(type),depth+1) || contains_resource(p,tagged::failure(type),depth+1);}
    if(arrays::owned(type)){return contains_resource(p,arrays::source_element(p,type),depth+1);}
    return false;
}
