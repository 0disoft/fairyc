module c_views;
import mir_model as m;
import array_types as arrays;
import c_text as w;
import std::bytes as bytes;

pub fn bool operation(ref m::Program p, m::Operation o, ref mut bytes::Buffer out, ref mut w::State s) {
    if (o.kind == 18) {
        return w::text(out,s,b"    ") && w::local(out,s,o.result) && w::text(out,s,b"=(struct fy_bytes){") && w::local(out,s,o.left) && w::text(out,s,b".fy_items,((uintptr_t)") && w::number(out,s,arrays::length(o.owner_type)) && w::text(out,s,b")};\n");
    }
    let usize end = m::edge_at(p,o.argument_start);
    // Volatile bounds prevent a compiler from treating a known trap branch as
    // dead C code under the warning-as-error low/optimized build contract.
    return w::text(out,s,b"    volatile uintptr_t fy_start_") && w::number(out,s,o.result) && w::text(out,s,b"=") && w::local(out,s,o.right) && w::text(out,s,b",fy_end_") && w::number(out,s,o.result) && w::text(out,s,b"=") && w::local(out,s,end)
        && w::text(out,s,b";\n    if(fy_start_") && w::number(out,s,o.result) && w::text(out,s,b">fy_end_") && w::number(out,s,o.result) && w::text(out,s,b"||fy_end_") && w::number(out,s,o.result) && w::text(out,s,b">") && w::local(out,s,o.left) && w::text(out,s,b".fy_length)fy_runtime_trap(UINT32_C(6));\n    ")
        && w::local(out,s,o.result) && w::text(out,s,b"=(struct fy_bytes){") && w::local(out,s,o.left) && w::text(out,s,b".fy_data+fy_start_") && w::number(out,s,o.result) && w::text(out,s,b",fy_end_") && w::number(out,s,o.result) && w::text(out,s,b"-fy_start_") && w::number(out,s,o.result) && w::text(out,s,b"};\n");
}
