module mir_verify_enums;
import mir_model as m;
import hir_shapes as shapes;
import enum_types as enums;

fn bool local(ref m::Program p, usize id, usize function, u32 type) {
    return id > 0 && id <= m::local_count(p) && m::local_at(p,id).function == function && m::local_at(p,id).type == type;
}
pub fn bool operation(ref m::Program p, m::Operation o) {
    let usize at = shapes::find(m::nominal(p),o.owner_type);
    if (!enums::valid(o.owner_type) || at == 0 || o.operator as usize >= shapes::at(m::nominal(p),at - 1).field_count
        || o.payload_start != 0 || o.payload_end != 0 || o.argument_start != 0 || o.argument_count != 0 || o.boolean || o.right != 0
        || !local(p,o.result,o.function,o.type)) { return false; }
    if (o.kind == 21) { return o.type == o.owner_type && o.left == 0; }
    return o.kind == 22 && o.type == 6 && local(p,o.left,o.function,o.owner_type);
}
