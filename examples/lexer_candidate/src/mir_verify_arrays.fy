module mir_verify_arrays;
import mir_model as m;
import array_types as arrays;
import hir_shapes as shapes;
import std::vec as v;

fn bool local(ref m::Program p, usize id, usize function, u32 type) {
    return id > 0 && id <= m::local_count(p) && m::local_at(p,id).function == function && m::local_at(p,id).type == type;
}
pub fn bool operation(ref m::Program p, m::Operation o) {
    if (o.payload_start != 0 || o.payload_end != 0 || o.operator != 0 || !arrays::valid(o.owner_type) || !shapes::valid_type(m::nominal(p),o.owner_type,false) || !local(p,o.result,o.function,o.type)) { return false; }
    if (o.kind == 9) {
        if (o.type != o.owner_type || o.left != 0 || o.right != 0 || o.argument_count != if (o.boolean) { 1 } else { arrays::length(o.type) }) { return false; }
        var usize i = 0; while (i < o.argument_count) { if (!local(p,m::edge_at(p,o.argument_start+i),o.function,shapes::array_element(m::nominal(p),o.type))) { return false; } i += 1; } return true;
    }
    if (o.boolean) { return false; }
    if (o.kind == 12) { return o.type == 5 && local(p,o.left,o.function,5) && o.result != o.left && o.right == 0 && o.argument_start == 0 && o.argument_count == 0; }
    if (!local(p,o.left,o.function,o.owner_type)) { return false; }
    if (o.kind == 13) { return o.type == 5 && o.right == 0 && o.argument_start == 0 && o.argument_count == 0; }
    if (!local(p,o.right,o.function,5)) { return false; }
    if (o.kind == 10) { return o.type == shapes::array_element(m::nominal(p),o.owner_type) && o.argument_start == 0 && o.argument_count == 0; }
    return o.kind == 11 && o.type == o.owner_type && o.argument_count == 1 && local(p,m::edge_at(p,o.argument_start),o.function,shapes::array_element(m::nominal(p),o.type));
}
// Definite initialization proves dominance; this pass additionally rejects
// forged, reassigned or wrong-length checked indexes before C emission.
pub fn m::Problem check(ref m::Program p) {
    var v::Vec<u32> owners = v::empty<u32>(); var v::Vec<usize> definitions = v::empty<usize>();
    var usize i = 0; while (i <= m::local_count(p)) { if (!v::push<u32>(ref mut owners,0).is_ok || !v::push<usize>(ref mut definitions,0).is_ok) { return m::Problem { .code = 3, .start = 0, .end = 0 }; } i += 1; }
    i = 1; while (i <= m::operation_count(p)) {
        let o = m::operation_at(p,i);
        if (o.result != 0) { let ref mut usize n = v::at_mut<usize>(ref mut definitions,o.result); *n += 1; }
        if (o.kind == 12) { let ref mut u32 type = v::at_mut<u32>(ref mut owners,o.result); *type = o.owner_type; } i += 1;
    }
    i = 1; while (i <= m::block_count(p)) { let b = m::block_at(p,i); if (b.destination != 0) { let ref mut usize n = v::at_mut<usize>(ref mut definitions,b.destination); *n += 1; } i += 1; }
    i = 1; while (i <= m::operation_count(p)) {
        let o = m::operation_at(p,i);
        if ((o.kind == 10 || o.kind == 11) && (*v::at<u32>(ref owners,o.right) != o.owner_type || *v::at<usize>(ref definitions,o.right) != 1 || m::local_at(p,o.right).parameter)) { return m::Problem { .code = 1501, .start = o.start, .end = o.end }; } i += 1;
    }
    return m::Problem { .code = 0, .start = 0, .end = 0 };
}
