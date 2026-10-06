module nominal_types;
import array_types as arrays;
import literal_types as literals;
import syntax as syntax;
import type_state as state;
import std::bytes as bytes;

// Private nominal ID = 256 + zero-based declaration index. Scalar tags stay
// unchanged. Identity is declaration based, never structural field equality.
pub fn u32 named(ref syntax::Parsed p, ref u8[] spelling) {
    var usize i = 0; while (i < syntax::declaration_count(p)) {
        let syntax::Declaration d = syntax::declaration_at(p,i);
        if (d.kind == 4 && bytes::equal(syntax::source_view(p)[d.name_start..d.name_end],spelling)) { return 256 + state::small_id(i); } i += 1;
    }
    return 0;
}
pub fn u32 resolve(ref syntax::Parsed p, usize root) {
    if (root == 0) { return 0; }
    if (syntax::type_at(p,root).kind == 2) { return literals::resolve(p,root); }
    if (syntax::type_at(p,root).kind == 8) { return arrays::resolve(p,root); }
    if (syntax::type_at(p,root).kind != 1) { return 0; }
    let ref u8[] spelling = syntax::type_text(p,root);
    let u32 primitive = state::primitive(p,root);
    if (primitive != 7 || bytes::equal(spelling,b"void")) { return primitive; }
    return named(p,spelling);
}
pub fn usize field(ref syntax::Parsed p, u32 type, ref u8[] name) {
    if (type < 256 || (type - 256) as usize >= syntax::declaration_count(p)) { return 0; }
    let syntax::Declaration d = syntax::declaration_at(p,(type - 256) as usize);
    if (d.kind != 4) { return 0; }
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
        if (type == 0 || type == 7 || arrays::valid(type) || literals::valid(type)) { return state::fail(16,f.start,f.end,0,type); }
        var usize j = d.field_start; while (j < i) {
            let previous = syntax::field_at(p,j);
            if (bytes::equal(syntax::source_view(p)[f.name_start..f.name_end],syntax::source_view(p)[previous.name_start..previous.name_end])) { return state::fail(17,f.name_start,f.name_end,0,0); } j += 1;
        }
        if (type >= 256) { let state::Problem problem = visit(p,(type - 256) as usize,colors,depth + 1); if (problem.code != 0) { return problem; } }
        i += 1;
    }
    colors[id] = 2; return state::ok();
}
pub fn state::Problem validate(ref syntax::Parsed p) {
    if (syntax::declaration_count(p) > 512) { return state::fail(10,0,syntax::source_view(p).len,0,0); }
    var u8[512] colors = [0;512]; var usize i = 0;
    while (i < syntax::declaration_count(p)) {
        let syntax::Declaration d = syntax::declaration_at(p,i);
        if (d.kind == 4) { let state::Problem problem = visit(p,i,ref mut colors[..],0); if (problem.code != 0) { return problem; } } i += 1;
    }
    return state::ok();
}
