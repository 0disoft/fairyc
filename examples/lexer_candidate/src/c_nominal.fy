module c_nominal;
import mir_model as m;
import hir_shapes as shapes;
import c_text as w;
import std::bytes as bytes;

// Arena IDs, not input identifiers, name the private C fields and records.
fn bool field(ref mut bytes::Buffer out, ref mut w::State s, usize id) {
    return w::text(out,s,b"fy_f_") && w::number(out,s,id);
}
fn bool definition(ref shapes::Table table, usize index, ref mut u8[] emitted, ref mut bytes::Buffer out, ref mut w::State s) {
    if (emitted[index] != 0) { return true; }
    let shapes::Shape shape = shapes::at(table,index); var usize i = 0;
    // The verified table is acyclic and has at most 64 nested value levels.
    while (i < shape.field_count) {
        let shapes::Field f = shapes::field_at(table,shape.field_start + i + 1);
        if (f.type >= 256 && !definition(table,shapes::find(table,f.type) - 1,emitted,out,s)) { return false; } i += 1;
    }
    if (!w::type(out,s,shape.type) || !w::text(out,s,b"{\n")) { return false; }
    if (shape.field_count == 0 && !w::text(out,s,b"    uint8_t fy_empty;\n")) { return false; }
    i = 0; while (i < shape.field_count) {
        let usize id = shape.field_start + i + 1;
        if (!w::text(out,s,b"    ") || !w::type(out,s,shapes::field_at(table,id).type) || !w::text(out,s,b" ") || !field(out,s,id) || !w::text(out,s,b";\n")) { return false; } i += 1;
    }
    if (!w::text(out,s,b"};\n")) { return false; } emitted[index] = 1; return true;
}
pub fn bool definitions(ref m::Program p, ref mut bytes::Buffer out, ref mut w::State s) {
    let ref shapes::Table table = m::nominal(p); var u8[512] emitted = [0;512]; var usize i = 0;
    while (i < shapes::count(table)) { if (!definition(table,i,ref mut emitted[..],out,s)) { return false; } i += 1; } return true;
}
pub fn bool operation(ref m::Program p, m::Operation o, ref mut bytes::Buffer out, ref mut w::State s) {
    if (!w::text(out,s,b"    ") || !w::local(out,s,o.result) || !w::text(out,s,b" = ")) { return false; }
    if (o.kind == 6) {
        let shapes::Shape shape = shapes::at(m::nominal(p),shapes::find(m::nominal(p),o.type) - 1);
        if (!w::text(out,s,b"(") || !w::type(out,s,o.type) || !w::text(out,s,b"){")) { return false; }
        if (o.argument_count == 0 && !w::text(out,s,b"0")) { return false; }
        var usize i = 0; while (i < o.argument_count) {
            if (i != 0 && !w::text(out,s,b",")) { return false; }
            if (!w::text(out,s,b".") || !field(out,s,shape.field_start + i + 1) || !w::text(out,s,b"=") || !w::local(out,s,m::edge_at(p,o.argument_start + i))) { return false; } i += 1;
        }
        return w::text(out,s,b"};\n");
    }
    if (!w::local(out,s,o.left)) { return false; }
    if (o.kind == 7) { return w::text(out,s,b".") && field(out,s,o.operator as usize) && w::text(out,s,b";\n"); }
    return w::text(out,s,b";\n    ") && w::local(out,s,o.result) && w::text(out,s,b".") && field(out,s,o.operator as usize)
        && w::text(out,s,b" = ") && w::local(out,s,o.right) && w::text(out,s,b";\n");
}
