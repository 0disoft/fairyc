module mir_output;
import mir_model as m;
import mir_verify as verify;
import hir_shapes as shapes;
import std::bytes as bytes;
import literal_types as literals;

fn bool row(ref mut bytes::Buffer out, ref u8[] tag, usize[20] values, usize count) {
    if (!bytes::append(out,tag).is_ok) { return false; }
    var usize i = 0;
    while (i < count) { if (!bytes::append_byte(out,9).is_ok || !bytes::append_usize(out,values[i]).is_ok) { return false; } i += 1; }
    return true;
}
fn bool end(ref mut bytes::Buffer out) { return bytes::append_byte(out,10).is_ok; }
// Private numeric TSV. Complete output is buffered before the caller publishes.
// 3 allocation, 4 transport limit. Schema field order is owned below.
pub fn u32 render(ref m::Program p, ref mut bytes::Buffer out) {
    let m::Problem problem = verify::check(p);
    if (problem.code != 0) { return problem.code; }
    let ref shapes::Table table = m::nominal(p); let bool nominal = shapes::count(table) != 0;
    let bool array = m::has_arrays(p);
    if (!bytes::append(out,if (shapes::has_resources(table)) { b"fairyc.resource-mir/v1\nP\t" } else if (shapes::has_payloads(table)) { b"fairyc.enum-payload-mir/v1\nP\t" } else if (shapes::has_enums(table)) { b"fairyc.enum-mir/v1\nP\t" } else if (m::has_views(p)) { b"fairyc.view-mir/v1\nP\t" } else if (m::has_literals(p)) { b"fairyc.literal-mir/v1\nP\t" } else if (array) { b"fairyc.array-mir/v1\nP\t" } else if (nominal) { b"fairyc.nominal-mir/v1\nP\t" } else { b"fairyc.scalar-mir/v1\nP\t" }).is_ok || !bytes::append_usize(out,m::module_index(p)).is_ok || !end(out)) { return 3; }
    var usize index = 0;
    while(index<m::drop_count(p)){let d=m::drop_at(p,index);let usize[20] fields=[d.type as usize,d.target,if(d.nonempty){1}else{0},0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];if(!row(out,b"K",fields,3) || !end(out)){return 3;}index+=1;}index=0;
    while (index < shapes::count(table)) {
        let shapes::Shape shape = shapes::at(table,index);
        let usize[20] fields = [shape.type as usize,shape.binding,shape.field_start,shape.field_count,if (shape.public) { 1 } else { 0 },0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        if (!row(out,b"N",fields,5) || !end(out)) { return 3; } index += 1;
    }
    index = 1;
    while (index <= shapes::field_count(table)) {
        let shapes::Field field = shapes::field_at(table,index);
        let usize[20] fields = [index,field.owner as usize,field.type as usize,field.start,field.end,field.payload_start,field.payload_count,0,0,0,0,0,0,0,0,0,0,0,0,0];
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        if (!row(out,b"D",fields,if(shapes::has_payloads(table)){7}else{5}) || !end(out)) { return 3; } index += 1;
    }
    var usize id = 1;
    while (id <= m::target_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let m::Target t = m::target_at(p,id);
        let usize[20] fields = [id,t.binding,t.module_index,t.declaration,t.result as usize,t.signature_start,t.signature_count,t.start,t.end,if (t.external) { 1 } else { 0 },0,0,0,0,0,0,0,0,0,0];
        if (!row(out,b"T",fields,10) || !end(out)) { return 3; } id += 1;
    }
    id = 0;
    while (id < m::signature_count(p)) {
        let usize[20] fields = [id,m::signature_at(p,id) as usize,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        if (!row(out,b"S",fields,2) || !end(out)) { return 3; } id += 1;
    }
    id = 1;
    while (id <= m::function_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let m::Function f = m::function_at(p,id);
        let usize[20] fields = [id,f.target,f.entry,f.local_start,f.local_count,f.block_start,f.block_count,f.parameter_start,f.parameter_count,f.start,f.end,if (f.public) { 1 } else { 0 },0,0,0,0,0,0,0,0];
        if (!row(out,b"F",fields,12) || !end(out)) { return 3; } id += 1;
    }
    id = 1;
    while (id <= m::local_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let m::Local l = m::local_at(p,id);
        let usize[20] fields = [id,l.function,l.binding,l.type as usize,if (l.parameter) { 1 } else { 0 },if (l.mutable) { 1 } else { 0 },l.start,l.end,0,0,0,0,0,0,0,0,0,0,0,0];
        if (!row(out,b"L",fields,8) || !end(out)) { return 3; } id += 1;
    }
    id = 1;
    while (id <= m::block_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let m::Block b = m::block_at(p,id);
        let usize[20] fields = [id,b.function,b.first,b.last,b.kind as usize,b.value,b.left,b.right,b.operator as usize,b.type as usize,b.target,b.alternative,b.check,b.callee,b.destination,b.argument_start,b.argument_count,b.trap_mask as usize,b.start,b.end];
        if (!row(out,b"B",fields,20) || !end(out)) { return 3; } id += 1;
    }
    id = 1;
    while (id <= m::operation_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let m::Operation o = m::operation_at(p,id);
        if (o.kind == 14 && m::literal_text(p,id).len > (2097152 - bytes::len(out) - 1024) / 2) { return 4; }
        let usize[20] fields = [id,o.function,o.block,o.kind as usize,o.result,o.type as usize,o.owner_type as usize,o.operator as usize,o.left,o.right,o.next,o.payload_start,o.payload_end,if (o.boolean) { 1 } else { 0 },o.start,o.end,0,0,0,0];
        if (!row(out,b"O",fields,16) || !bytes::append_byte(out,9).is_ok) { return 3; }
        if (o.kind == 14) { if (!literals::hex(out,m::literal_text(p,id))) { return 3; } }
        else if (o.payload_end > o.payload_start) { if (!bytes::append(out,m::literal_text(p,id)).is_ok) { return 3; } }
        else if (!bytes::append_byte(out,45).is_ok) { return 3; }
        if ((nominal || m::has_views(p)) && (!bytes::append_byte(out,9).is_ok || !bytes::append_usize(out,o.argument_start).is_ok || !bytes::append_byte(out,9).is_ok || !bytes::append_usize(out,o.argument_count).is_ok)) { return 3; }
        if (!end(out)) { return 3; } id += 1;
    }
    id = 0;
    while (id < m::edge_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let usize[20] fields = [id,m::edge_at(p,id),0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];
        if (!row(out,b"A",fields,2) || !end(out)) { return 3; } id += 1;
    }
    return 0;
}
