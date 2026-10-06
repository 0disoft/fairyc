module c_nominal;
import tagged_types as tagged;
import mir_model as m;
import hir_shapes as shapes;
import array_types as arrays;
import enum_types as enums;
import c_text as w;
import std::bytes as bytes;

// Arena IDs, not input identifiers, name the private C fields and records.
fn bool field(ref mut bytes::Buffer out, ref mut w::State s, usize id) {
    return w::text(out,s,b"fy_f_") && w::number(out,s,id);
}
fn bool definition(ref shapes::Table table, usize index, ref mut u8[] emitted, ref mut bytes::Buffer out, ref mut w::State s) {
    if (emitted[index] != 0) { return true; }
    let shapes::Shape shape = shapes::at(table,index); var usize i = 0;
    // Nested tagged payloads must be fully defined before their parent record.
    while(i<shape.field_count){let f=shapes::field_at(table,shape.field_start+i+1);
        if(f.type>=256 && !arrays::valid(f.type) && !definition(table,shapes::find(table,f.type)-1,emitted,out,s)){return false;}i+=1;
    }i=0;
    if (enums::valid(shape.type)) {
        if (!w::type(out,s,shape.type) || !w::text(out,s,b"{ uint32_t fy_tag;\n")) { return false; }
        var bool payload = false;
        while (i < shape.field_count) { payload = payload || shapes::field_at(table,shape.field_start+i+1).payload_count != 0; i += 1; }
        if (payload) {
            if (!w::text(out,s,b"    union {\n")) { return false; } i = 0;
            while (i < shape.field_count) {
                let usize member = shape.field_start+i+1; let f = shapes::field_at(table,member);
                if (f.payload_count != 0) {
                    if (!w::text(out,s,b"        struct {\n")) { return false; } var usize j = 0;
                    while (j < f.payload_count) {
                        let usize id = f.payload_start+j+1;
                        if (!w::text(out,s,b"            ") || !w::type(out,s,shapes::field_at(table,id).type) || !w::text(out,s,b" ") || !field(out,s,id) || !w::text(out,s,b";\n")) { return false; } j += 1;
                    }
                    if (!w::text(out,s,b"        } ") || !field(out,s,member) || !w::text(out,s,b";\n")) { return false; }
                } i += 1;
            }
            if (!w::text(out,s,b"    } fy_payload;\n")) { return false; }
        }
        if (!w::text(out,s,b"};\n")) { return false; } emitted[index] = 1; return true;
    }
    // The verified table is acyclic and has at most 64 nested value levels.
    while (i < shape.field_count) {
        let shapes::Field f = shapes::field_at(table,shape.field_start + i + 1);
        if (f.type >= 256 && !arrays::valid(f.type) && !definition(table,shapes::find(table,f.type) - 1,emitted,out,s)) { return false; } i += 1;
    }
    if (!w::type(out,s,shape.type) || !w::text(out,s,b"{\n")) { return false; }
    if (shape.field_count == 0 && !w::text(out,s,b"    uint8_t fy_empty;\n")) { return false; }
    i = 0; while (i < shape.field_count) {
        let usize id = shape.field_start + i + 1;
        if (!w::text(out,s,b"    ") || !w::type(out,s,shapes::field_at(table,id).type) || !w::text(out,s,b" ") || !field(out,s,id) || !w::text(out,s,b";\n")) { return false; } i += 1;
    }
    if (!w::text(out,s,b"};\n")) { return false; } emitted[index] = 1; return true;
}
pub fn bool enum_operation(ref m::Program p, m::Operation o, ref mut bytes::Buffer out, ref mut w::State s) {
    let ref shapes::Table table = m::nominal(p);
    let usize member = if (o.kind == 23) { shapes::payload_member(table,o.owner_type,o.operator as usize) }
        else { shapes::member_at(table,o.owner_type,o.operator as usize) };
    if (o.kind == 21 && o.type >= 1073741824 && tagged::builtin(o.type)) {
        // Avoid implicit recursive aggregate initialization of inactive arms.
        if (!w::text(out,s,b"    memset(&") || !w::local(out,s,o.result) || !w::text(out,s,b",0,sizeof(")
            || !w::local(out,s,o.result) || !w::text(out,s,b"));\n    ") || !w::local(out,s,o.result)
            || !w::text(out,s,b".fy_tag=UINT32_C(") || !w::number(out,s,o.operator as usize) || !w::text(out,s,b");\n")) { return false; }
        let f = shapes::field_at(table,member); var usize i = 0;
        while (i < o.argument_count) {
            if (!w::text(out,s,b"    ") || !w::local(out,s,o.result) || !w::text(out,s,b".fy_payload.") || !field(out,s,member)
                || !w::text(out,s,b".") || !field(out,s,f.payload_start+i+1) || !w::text(out,s,b"=")
                || !w::local(out,s,m::edge_at(p,o.argument_start+i)) || !w::text(out,s,b";\n")) { return false; } i += 1;
        }
        return true;
    }
    if (o.kind == 23) {
        if (!w::text(out,s,b"    if (") || !w::local(out,s,o.left) || !w::text(out,s,b".fy_tag != UINT32_C(")
            || !w::number(out,s,shapes::member_tag(table,o.owner_type,member)) || !w::text(out,s,b")) fy_runtime_trap(7);\n")) { return false; }
    }
    if (!w::text(out,s,b"    ") || !w::local(out,s,o.result) || !w::text(out,s,b" = ")) { return false; }
    if (o.kind == 21) {
        if (!w::text(out,s,b"(") || !w::type(out,s,o.type) || !w::text(out,s,b"){.fy_tag=UINT32_C(") || !w::number(out,s,o.operator as usize) || !w::text(out,s,b")")) { return false; }
        let f = shapes::field_at(table,member); var usize i = 0;
        if (o.argument_count != 0 && (!w::text(out,s,b",.fy_payload={.") || !field(out,s,member) || !w::text(out,s,b"={"))) { return false; }
        while (i < o.argument_count) {
            if (i != 0 && !w::text(out,s,b",")) { return false; }
            if (!w::text(out,s,b".") || !field(out,s,f.payload_start+i+1) || !w::text(out,s,b"=") || !w::local(out,s,m::edge_at(p,o.argument_start+i))) { return false; } i += 1;
        }
        return (o.argument_count == 0 || w::text(out,s,b"}}")) && w::text(out,s,b"};\n");
    }
    if (!w::local(out,s,o.left)) { return false; }
    if (o.kind == 22) { return w::text(out,s,b".fy_tag == UINT32_C(") && w::number(out,s,o.operator as usize) && w::text(out,s,b");\n"); }
    return w::text(out,s,b".fy_payload.") && field(out,s,member) && w::text(out,s,b".") && field(out,s,o.operator as usize) && w::text(out,s,b";\n");
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
