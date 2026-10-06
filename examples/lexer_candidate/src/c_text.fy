module c_text;
import std::bytes as bytes;

pub struct State { u32 code; }
pub fn bool text(ref mut bytes::Buffer out, ref mut State s, ref u8[] value) {
    if (s.code != 0) { return false; }
    if (bytes::len(out) > 2097152 || value.len > 2097152 - bytes::len(out)) { s.code = 4; return false; }
    if (!bytes::append(out,value).is_ok) { s.code = 3; return false; } return true;
}
pub fn bool number(ref mut bytes::Buffer out, ref mut State s, usize value) {
    if (s.code != 0) { return false; }
    if (bytes::len(out) > 2097152 - 20) { s.code = 4; return false; }
    if (!bytes::append_usize(out,value).is_ok) { s.code = 3; return false; } return true;
}
pub fn bool type(ref mut bytes::Buffer out, ref mut State s, u32 code) {
    if (code == 11) { return text(out,s,b"struct fy_bytes_mut"); }
    if (code == 9 || code == 10) { return text(out,s,if (code == 9) { b"struct fy_text" } else { b"struct fy_bytes" }); }
    if (code >= 1024) { return text(out,s,b"struct fy_a_") && number(out,s,code as usize); }
    if (code >= 256) { return text(out,s,b"struct fy_s_") && number(out,s,code as usize); }
    return text(out,s,if (code == 1 || code == 6) { b"int32_t" } else if (code == 2) { b"int64_t" }
        else if (code == 3) { b"uint8_t" } else if (code == 4) { b"uint32_t" } else if (code == 5) { b"uintptr_t" } else { b"void" });
}
pub fn bool unsigned_type(ref mut bytes::Buffer out, ref mut State s, u32 code) {
    return text(out,s,if (code == 2) { b"uint64_t" } else if (code == 5) { b"uintptr_t" } else { b"uint32_t" });
}
pub fn bool local(ref mut bytes::Buffer out, ref mut State s, usize id) { return text(out,s,b"fy_l_") && number(out,s,id); }
pub fn bool function(ref mut bytes::Buffer out, ref mut State s, usize id) { return text(out,s,b"fy_fn_") && number(out,s,id); }
pub fn bool jump(ref mut bytes::Buffer out, ref mut State s, usize id) { return text(out,s,b"goto fy_b_") && number(out,s,id) && text(out,s,b";\n"); }
pub fn bool operator(ref mut bytes::Buffer out, ref mut State s, u32 code) {
    return text(out,s,if (code == 1) { b"+" } else if (code == 2) { b"-" } else if (code == 3) { b"!" } else if (code == 4) { b"~" }
        else if (code == 5) { b"*" } else if (code == 6) { b"/" } else if (code == 7) { b"%" } else if (code == 8) { b"&" }
        else if (code == 9) { b"|" } else if (code == 10) { b"^" } else if (code == 11) { b"<<" } else if (code == 12) { b">>" }
        else if (code == 13) { b"==" } else if (code == 14) { b"!=" } else if (code == 15) { b"<" } else if (code == 16) { b"<=" }
        else if (code == 17) { b">" } else { b">=" });
}
pub fn bool maximum(ref mut bytes::Buffer out, ref mut State s, u32 code) {
    return text(out,s,if (code == 1) { b"INT32_MAX" } else if (code == 2) { b"INT64_MAX" } else if (code == 3) { b"UINT8_MAX" } else if (code == 4) { b"UINT32_MAX" } else { b"UINTPTR_MAX" });
}
pub fn bool minimum(ref mut bytes::Buffer out, ref mut State s, u32 code) { return text(out,s,if (code == 1) { b"INT32_MIN" } else { b"INT64_MIN" }); }
pub fn bool divisor(ref mut bytes::Buffer out, ref mut State s, u32 code) { return text(out,s,b"fy_div_") && number(out,s,code as usize); }
