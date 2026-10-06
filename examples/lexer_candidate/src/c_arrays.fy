module c_arrays;
import mir_model as m;
import array_types as arrays;
import c_text as w;
import std::bytes as bytes;

pub fn bool present(ref m::Program p) { return m::has_arrays(p); }
pub fn bool definitions(ref m::Program p, ref mut bytes::Buffer out, ref mut w::State s) {
    var bool[520] used = [false;520]; var usize i = 1;
    while (i <= m::local_count(p)) { let u32 t = m::local_at(p,i).type; if (arrays::valid(t)) { used[(t-1024) as usize] = true; } i += 1; }
    i = 1; while (i <= m::target_count(p)) { let u32 t = m::target_at(p,i).result; if (arrays::valid(t)) { used[(t-1024) as usize] = true; } i += 1; }
    var u32 t = 1033;
    while (t <= 1542) {
        if (arrays::valid(t) && used[(t-1024) as usize]) {
            if (!w::type(out,s,t) || !w::text(out,s,b"{ ") || !w::type(out,s,arrays::element(t)) || !w::text(out,s,b" fy_items[") || !w::number(out,s,arrays::length(t)) || !w::text(out,s,b"]; };\n")) { return false; }
        } t += 1;
    }
    return true;
}
pub fn bool operation(ref m::Program p, m::Operation o, ref mut bytes::Buffer out, ref mut w::State s) {
    if (o.kind == 12) {
        // Keep MSVC's reachability analysis from turning a known failing index
        // into C4702 after the trap. The source expression still executes once.
        return w::text(out,s,b"    volatile uintptr_t fy_index_") && w::number(out,s,o.result) && w::text(out,s,b"=") && w::local(out,s,o.left)
            && w::text(out,s,b";\n    if(fy_index_") && w::number(out,s,o.result) && w::text(out,s,b">=") && w::number(out,s,arrays::length(o.owner_type))
            && w::text(out,s,b")fy_runtime_trap(UINT32_C(6));\n    ") && w::local(out,s,o.result) && w::text(out,s,b"=fy_index_") && w::number(out,s,o.result) && w::text(out,s,b";\n");
    }
    if (!w::text(out,s,b"    ") || !w::local(out,s,o.result) || !w::text(out,s,b"=")) { return false; }
    if (o.kind == 9) {
        if (!w::text(out,s,b"(") || !w::type(out,s,o.type) || !w::text(out,s,b"){.fy_items={")) { return false; }
        var usize i = 0; while (i < arrays::length(o.type)) {
            if (i != 0 && !w::text(out,s,b",")) { return false; }
            if (!w::local(out,s,m::edge_at(p,o.argument_start + if (o.boolean) { 0 } else { i }))) { return false; } i += 1;
        }
        return w::text(out,s,b"}};\n");
    }
    if (o.kind == 13) { return w::text(out,s,b"((uintptr_t)") && w::number(out,s,arrays::length(o.owner_type)) && w::text(out,s,b");\n"); }
    if (!w::local(out,s,o.left)) { return false; }
    if (o.kind == 10) { return w::text(out,s,b".fy_items[") && w::local(out,s,o.right) && w::text(out,s,b"];\n"); }
    return w::text(out,s,b";\n    ") && w::local(out,s,o.result) && w::text(out,s,b".fy_items[") && w::local(out,s,o.right) && w::text(out,s,b"]=") && w::local(out,s,m::edge_at(p,o.argument_start)) && w::text(out,s,b";\n");
}
