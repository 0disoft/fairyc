module repr_c_declarations;
import c_abi_types as c_types;
import pointer_types as pointers;
import syntax as syntax;
import type_state as state;
import ffi_symbols as symbols;
import std::bytes as bytes;

// Bounded lane: a source-declared, inert repr(C) struct shape only.
//
// read decides the whole declaration. It approves only a kind-4 (struct)
// declaration whose sole attribute is exactly an @repr(C) marker, whose tag and
// field names are C identifiers, and whose fields all sit in the admitted value
// lane. layout re-checks the tag/field lane alone against the parsed arena and
// owned source bytes; the caller pairs it with read so a forged marker flag
// cannot approve a lane the verifier cannot re-derive from source text.
//
// Code contract (declaration-local, never a public compiler diagnostic ID):
//   0    not an inert repr(C) struct, or a fully approved one.
//   5106 malformed shape: a bad or duplicated @repr, a non-C spelling, any
//        other struct attribute such as @effects, a non-C tag or field name
//        (C keywords, runtime namespaces fy_/FY_/INT/UINT, libc names, and
//        main, all named by ffi_symbols::valid), a duplicate field name, or a
//        generic struct.
//   5102 well-formed @repr(C) whose value lane is not admitted here: an empty
//        body, a field type outside fixed scalar / closed C scalar / raw pointer
//        to scalar-or-void, or a non-value field kind.
// read reports the exact @repr(C) attribute span when it approves, the
// offending attribute/argument/name/type span when it rejects a field or name,
// and otherwise the declaration span. layout is the marker-independent lane
// predicate: a plain struct with admissible fields is true even though read
// reports repr_c=false, code=0 because it carries no repr(C) marker.
//
// ADR0415 whole-owner rule: inputs stay shared (ref) and results are owned, so
// no arena slot and no simultaneous mutable sibling view is created here.
pub struct Info { bool repr_c; u32 code; usize start; usize end; }

// Fixed scalar i32/i64/u8/u32/usize/bool, the closed C scalar set, or an
// existing nullable raw pointer to a scalar-or-void element. Nested repr(C)
// structs, arrays, borrows, resources, and tagged/generic types wait for later
// units and are rejected here.
fn bool value_type(ref syntax::Parsed p, usize root) {
    if (root == 0) { return false; }
    if (c_types::valid(c_types::resolve(p, root))) { return true; }
    if (pointers::resolve(p, root) != 0) { return true; }
    if (syntax::type_at(p, root).kind != 1) { return false; }
    let u32 primitive = state::primitive(p, root);
    return primitive >= 1 && primitive <= 6;
}

struct Lane { u32 code; usize start; usize end; }

// Tag and field-name lane plus value types. 5106 is a name/structure violation,
// 5102 a value-lane violation, 0 an admissible inert layout.
fn Lane lane(ref syntax::Parsed p, usize declaration) {
    let d = syntax::declaration_at(p, declaration);
    if (d.field_count == 0) { return Lane { .code = 5102, .start = d.name_start, .end = d.name_end }; }
    let ref u8[] tag = syntax::source_view(p)[d.name_start..d.name_end];
    if (!symbols::valid(tag)) { return Lane { .code = 5106, .start = d.name_start, .end = d.name_end }; }
    var usize i = 0;
    while (i < d.field_count) {
        let f = syntax::field_at(p, d.field_start + i);
        if (f.kind != 0) { return Lane { .code = 5102, .start = f.start, .end = f.end }; }
        let ref u8[] name = syntax::source_view(p)[f.name_start..f.name_end];
        if (!symbols::valid(name)) { return Lane { .code = 5106, .start = f.name_start, .end = f.name_end }; }
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
