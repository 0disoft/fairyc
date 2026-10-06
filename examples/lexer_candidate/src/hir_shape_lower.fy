module hir_shape_lower;
import hir_shapes as shapes;
import type_check as typing;
import nominal_types as nominal;
import name_resolution as names;
import syntax as syntax;
import std::vec as v;

pub fn Result<shapes::Table,u32> collect(ref typing::Typed input) {
    let ref names::Resolved resolved = typing::resolved(input);
    let ref syntax::Parsed parsed = names::parsed(resolved);
    var v::Vec<shapes::Shape> records = v::empty<shapes::Shape>();
    var v::Vec<shapes::Field> fields = v::empty<shapes::Field>();
    var usize index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration d = syntax::declaration_at(parsed,index);
        if (d.kind == 4 || d.kind == 11) {
            var usize binding = 0; var usize id = 1;
            while (id <= names::symbol_count(resolved)) {
                let names::Symbol s = names::symbol_at(resolved,id);
                if ((s.kind == 6 || s.kind == 7) && s.declaration == index + 1) { binding = id; } id += 1;
            }
            if (binding == 0 || d.field_start != v::len<shapes::Field>(ref fields)) { return .err(4); }
            let u32 owner = typing::symbol_type(input,binding);
            if (!v::push<shapes::Shape>(ref mut records,shapes::Shape { .type = owner, .binding, .field_start = d.field_start, .field_count = d.field_count, .public = d.public }).is_ok) { return .err(3); }
            var usize j = 0;
            while (j < d.field_count) {
                let f = syntax::field_at(parsed,d.field_start + j);
                if (!v::push<shapes::Field>(ref mut fields,shapes::Field { .owner, .type = nominal::resolve(parsed,f.type_root), .start = f.name_start, .end = f.name_end, .payload_start = f.payload_start, .payload_count = f.payload_count }).is_ok) { return .err(3); } j += 1;
            }
        } else if (d.kind != 1 && d.kind != 2 && d.kind != 3) { return .err(4); }
        index += 1;
    }
    return .ok(shapes::assemble(records,fields));
}
