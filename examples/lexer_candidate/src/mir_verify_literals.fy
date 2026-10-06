module mir_verify_literals;
import mir_model as m;
import literal_types as literals;
import std::vec as v;

fn bool local(ref m::Program p, usize id, usize function, u32 type) { return id > 0 && id <= m::local_count(p) && m::local_at(p,id).function == function && m::local_at(p,id).type == type; }
pub fn bool operation(ref m::Program p, usize id, m::Operation o) {
    if (o.operator != 0 || o.argument_start != 0 || o.argument_count != 0 || !local(p,o.result,o.function,o.type)) { return false; }
    if (o.kind == 14) { return literals::valid(o.type) && o.owner_type == o.type && !o.boolean && o.left == 0 && o.right == 0 && (o.type != 9 || literals::text(m::literal_text(p,id))); }
    if (o.payload_start != 0 || o.payload_end != 0 || !literals::valid(o.owner_type) || !local(p,o.left,o.function,o.owner_type)) { return false; }
    if (o.kind == 15) { return o.type == if (o.boolean) { 6 } else { 5 } && o.right == 0; }
    if (o.boolean || o.owner_type != 10 || !local(p,o.right,o.function,5)) { return false; }
    return if (o.kind == 16) { o.type == 5 && o.result != o.right } else { o.kind == 17 && o.type == 3 };
}
// The checked index records one frozen descriptor, not merely its view type.
// Definition counting prevents later descriptor/index reassignment; definite
// initialization then proves that the matching check dominates every load.
pub fn m::Problem check(ref m::Program p) {
    var v::Vec<usize> definitions = v::empty<usize>(); var v::Vec<usize> owners = v::empty<usize>();
    var usize i = 0; while (i <= m::local_count(p)) { if (!v::push<usize>(ref mut definitions,0).is_ok || !v::push<usize>(ref mut owners,0).is_ok) { return m::Problem { .code=3,.start=0,.end=0 }; } i += 1; }
    i = 1; while (i <= m::operation_count(p)) {
        let o = m::operation_at(p,i); if (o.result != 0) { let ref mut usize n = v::at_mut<usize>(ref mut definitions,o.result); *n += 1; }
        if (o.kind == 16) { let ref mut usize owner = v::at_mut<usize>(ref mut owners,o.result); *owner = o.left; } i += 1;
    }
    i = 1; while (i <= m::block_count(p)) { let b = m::block_at(p,i); if (b.destination != 0) { let ref mut usize n = v::at_mut<usize>(ref mut definitions,b.destination); *n += 1; } i += 1; }
    i = 1; while (i <= m::operation_count(p)) {
        let o = m::operation_at(p,i);
        if (o.kind == 17 && (*v::at<usize>(ref owners,o.right) != o.left || *v::at<usize>(ref definitions,o.right) != 1 || *v::at<usize>(ref definitions,o.left) != 1 || m::local_at(p,o.right).parameter || m::local_at(p,o.left).parameter)) { return m::Problem { .code=1501,.start=o.start,.end=o.end }; } i += 1;
    } return m::Problem { .code=0,.start=0,.end=0 };
}
