module repr_c_declarations;
import c_abi_types as c_types;
import pointer_types as pointers;
import syntax as syntax;
import type_state as state;
import ffi_symbols as symbols;
import std::bytes as bytes;

// Private lane notes: docs/architecture/candidate-repr-c-model-v0.md.
pub struct Info { bool repr_c; u32 code; usize start; usize end; }

fn bool value_type(ref syntax::Parsed p, usize root) {
    if (root == 0) { return false; }
    if (c_types::valid(c_types::resolve(p, root))) { return true; }
    if (pointers::resolve(p, root) != 0) { return true; }
    if (syntax::type_at(p, root).kind != 1) { return false; }
    let u32 primitive = state::primitive(p, root);
    return primitive >= 1 && primitive <= 6;
}

struct Lane { u32 code; usize start; usize end; }

fn Lane lane(ref syntax::Parsed p, usize declaration) {
    let d = syntax::declaration_at(p, declaration);
    if (d.field_count == 0) { return Lane { .code = 5102, .start = d.name_start, .end = d.name_end }; }
    let ref u8[] tag = syntax::source_view(p)[d.name_start..d.name_end];
    if (!symbols::record(tag)) { return Lane { .code = 5106, .start = d.name_start, .end = d.name_end }; }
    var usize i = 0;
    while (i < d.field_count) {
        let f = syntax::field_at(p, d.field_start + i);
        if (f.kind != 0) { return Lane { .code = 5102, .start = f.start, .end = f.end }; }
        let ref u8[] name = syntax::source_view(p)[f.name_start..f.name_end];
        if (!symbols::member(name)) { return Lane { .code = 5106, .start = f.name_start, .end = f.name_end }; }
        if (!value_type(p, f.type_root)) {
            let t = syntax::type_at(p, f.type_root);
            return Lane { .code = 5102, .start = t.start, .end = t.end };
        }
        var usize j = 0;
        while (j < i) {
            let old = syntax::field_at(p, d.field_start + j);
            if (old.kind == 0 && bytes::equal(syntax::source_view(p)[old.name_start..old.name_end], name)) {
                return Lane { .code = 5106, .start = f.name_start, .end = f.name_end };
            }
            j += 1;
        }
        i += 1;
    }
    return Lane { .code = 0, .start = 0, .end = 0 };
}

pub fn bool layout(ref syntax::Parsed p, usize declaration) {
    if (syntax::declaration_at(p, declaration).kind != 4) { return false; }
    return lane(p, declaration).code == 0;
}

pub fn Info read(ref syntax::Parsed p, usize declaration) {
    let d = syntax::declaration_at(p, declaration);
    var Info info = Info { .repr_c = false, .code = 0, .start = d.start, .end = d.end };
    if (d.kind != 4 || d.attribute_count == 0) { return info; }
    if (d.generic_parameters != 0) { info.code = 5106; return info; }
    if (d.attribute_count > 1) {
        let extra = syntax::attribute_at(p, d.attribute_start + 1);
        info.code = 5106; info.start = extra.start; info.end = extra.end; return info;
    }
    let a = syntax::attribute_at(p, d.attribute_start);
    if (!bytes::equal(syntax::source_view(p)[a.name_start..a.name_end], b"repr") || a.argument_count != 1) {
        info.code = 5106; info.start = a.start; info.end = a.end; return info;
    }
    let arg = syntax::attribute_argument_at(p, a.argument_start);
    if (arg.kind != 1 || !bytes::equal(syntax::attribute_argument_text(p, a.argument_start), b"C")) {
        info.code = 5106; info.start = arg.start; info.end = arg.end; return info;
    }
    let l = lane(p, declaration);
    if (l.code != 0) { info.code = l.code; info.start = l.start; info.end = l.end; return info; }
    info.repr_c = true; info.start = a.start; info.end = a.end; return info;
}
