module mir_verify_nominal;
import mir_model as m;
import hir_shapes as shapes;
import enum_types as enums;
import array_types as arrays;

fn bool local(ref m::Program p, usize id, usize function, u32 type) {
    return id > 0 && id <= m::local_count(p) && m::local_at(p,id).function == function && m::local_at(p,id).type == type;
}
pub fn bool operation(ref m::Program p, m::Operation o) {
    let ref shapes::Table table = m::nominal(p);
    if (o.payload_start != 0 || o.payload_end != 0 || o.boolean || !local(p,o.result,o.function,o.type)) { return false; }
    if (o.kind == 6) {
        let usize at = shapes::find(table,o.type);
        if (at == 0 || enums::valid(o.type) || arrays::valid(o.type) || o.owner_type != o.type || o.left != 0 || o.right != 0 || o.operator != 0) { return false; }
        let shapes::Shape shape = shapes::at(table,at - 1);
        if (o.argument_count != shape.field_count) { return false; }
        var usize i = 0;
        while (i < o.argument_count) {
            let shapes::Field field = shapes::field_at(table,shape.field_start + i + 1);
            if (!local(p,m::edge_at(p,o.argument_start + i),o.function,field.type)) { return false; } i += 1;
        }
        return true;
    }
    if(arrays::valid(o.owner_type)){return false;}
    let usize field_id = o.operator as usize;
    if (o.argument_start != 0 || o.argument_count != 0 || field_id == 0 || field_id > shapes::field_count(table)) { return false; }
    let shapes::Field field = shapes::field_at(table,field_id);
    if (enums::valid(field.owner)) { return false; }
    if (o.kind == 7) { return !shapes::needs_drop(o.type) && o.type == field.type && o.owner_type == field.owner && local(p,o.left,o.function,field.owner) && o.right == 0; }
    return o.kind == 8 && o.type == field.owner && o.owner_type == field.owner && (!shapes::needs_drop(o.type) || o.result==o.left) && local(p,o.left,o.function,field.owner) && local(p,o.right,o.function,field.type);
}
