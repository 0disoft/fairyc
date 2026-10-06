module c_literals;
import mir_model as m;
import c_text as w;
import std::bytes as bytes;

pub fn bool definitions(ref m::Program p, ref mut bytes::Buffer out, ref mut w::State s) {
    return !m::has_literals(p) || w::text(out,s,b"struct fy_text { const uint8_t *fy_data; uintptr_t fy_length; };\nstruct fy_bytes { const uint8_t *fy_data; uintptr_t fy_length; };\n");
}
pub fn bool operation(ref m::Program p, usize id, m::Operation o, ref mut bytes::Buffer out, ref mut w::State s) {
    if (o.kind == 14) {
        let ref u8[] data = m::literal_text(p,id);
        if (!w::text(out,s,b"    static const uint8_t fy_literal_") || !w::number(out,s,id) || !w::text(out,s,b"[]={")) { return false; }
        var usize i = 0; while (i < data.len) { if (i != 0 && !w::text(out,s,b",")) { return false; } if (!w::number(out,s,data[i] as usize)) { return false; } i += 1; }
        if (data.len == 0 && !w::text(out,s,b"0")) { return false; }
        return w::text(out,s,b"};\n    ") && w::local(out,s,o.result) && w::text(out,s,b"=(") && w::type(out,s,o.type) && w::text(out,s,b"){fy_literal_") && w::number(out,s,id) && w::text(out,s,b",((uintptr_t)") && w::number(out,s,data.len) && w::text(out,s,b")};\n");
    }
    if (o.kind == 16) {
        return w::text(out,s,b"    volatile uintptr_t fy_index_") && w::number(out,s,o.result) && w::text(out,s,b"=") && w::local(out,s,o.right) && w::text(out,s,b";\n    if(fy_index_") && w::number(out,s,o.result) && w::text(out,s,b">=") && w::local(out,s,o.left) && w::text(out,s,b".fy_length)fy_runtime_trap(UINT32_C(6));\n    ") && w::local(out,s,o.result) && w::text(out,s,b"=fy_index_") && w::number(out,s,o.result) && w::text(out,s,b";\n");
    }
    if (!w::text(out,s,b"    ") || !w::local(out,s,o.result) || !w::text(out,s,b"=") || !w::local(out,s,o.left)) { return false; }
    if (o.kind == 17) { return w::text(out,s,b".fy_data[") && w::local(out,s,o.right) && w::text(out,s,b"];\n"); }
    return w::text(out,s,if (o.boolean) { b".fy_length==0;\n" } else { b".fy_length;\n" });
}
