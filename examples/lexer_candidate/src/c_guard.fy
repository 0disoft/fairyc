module c_guard;
import mir_model as m;
import c_text as w;
import c_scalar as scalar;
import std::bytes as bytes;

fn bool failed(ref mut bytes::Buffer out, ref mut w::State s, m::Block b, usize category) {
    return w::text(out,s,b"){fy_pending_trap=UINT32_C(") && w::number(out,s,category) && w::text(out,s,b");") && w::jump(out,s,b.alternative) && w::text(out,s,b"}\n");
}
fn bool arithmetic(ref mut bytes::Buffer out, ref mut w::State s, m::Block b) {
    if (!w::text(out,s,b"    if(")) { return false; }
    if (b.type == 1) {
        if (!w::text(out,s,b"((int64_t)") || !w::local(out,s,b.left) || !w::operator(out,s,b.operator) || !w::text(out,s,b"(int64_t)") || !w::local(out,s,b.right)
            || !w::text(out,s,b")>INT32_MAX||((int64_t)") || !w::local(out,s,b.left) || !w::operator(out,s,b.operator) || !w::text(out,s,b"(int64_t)") || !w::local(out,s,b.right) || !w::text(out,s,b")<INT32_MIN")) { return false; }
    } else if (b.type == 2) {
        if (!w::text(out,s,if (b.operator == 1) { b"fy_i64_add_overflow(" } else if (b.operator == 2) { b"fy_i64_sub_overflow(" } else { b"fy_i64_mul_overflow(" })
            || !w::local(out,s,b.left) || !w::text(out,s,b",") || !w::local(out,s,b.right) || !w::text(out,s,b")")) { return false; }
    } else if (b.operator == 1) {
        if (!w::local(out,s,b.left) || !w::text(out,s,b">") || !w::maximum(out,s,b.type) || !w::text(out,s,b"-") || !w::local(out,s,b.right)) { return false; }
    } else if (b.operator == 2) {
        if (!w::local(out,s,b.left) || !w::text(out,s,b"<") || !w::local(out,s,b.right)) { return false; }
    } else {
        if (!w::local(out,s,b.right) || !w::text(out,s,b"!=0&&") || !w::local(out,s,b.left) || !w::text(out,s,b">") || !w::maximum(out,s,b.type) || !w::text(out,s,b"/") || !w::local(out,s,b.right)) { return false; }
    }
    return failed(out,s,b,1);
}
pub fn bool emit(ref mut bytes::Buffer out, ref mut w::State s, m::Block b) {
    if (b.operator == 0) {
        if (!w::text(out,s,b"    if(!") || !w::local(out,s,b.left) || !failed(out,s,b,5)) { return false; }
    } else if (b.right == 0) {
        if (!w::text(out,s,b"    if(") || !w::local(out,s,b.left) || !w::text(out,s,b"==") || !w::minimum(out,s,b.type) || !failed(out,s,b,1)) { return false; }
    } else if (b.operator == 1 || b.operator == 2 || b.operator == 5) {
        if (!arithmetic(out,s,b)) { return false; }
    } else if (b.operator == 6 || b.operator == 7) {
        if (!w::text(out,s,b"    if(") || !w::local(out,s,b.right) || !w::text(out,s,b"==0") || !failed(out,s,b,2)) { return false; }
        if (b.type == 1 || b.type == 2) {
            if (!w::text(out,s,b"    if(") || !w::local(out,s,b.left) || !w::text(out,s,b"==") || !w::minimum(out,s,b.type)
                || !w::text(out,s,b"&&") || !w::local(out,s,b.right) || !w::text(out,s,b"==-1") || !failed(out,s,b,3)) { return false; }
        }
    } else {
        if (!w::text(out,s,b"    if(")) { return false; }
        if ((b.type == 1 || b.type == 2) && (!w::local(out,s,b.right) || !w::text(out,s,b"<0||"))) { return false; }
        if (!w::local(out,s,b.right) || !w::text(out,s,b">=") || !w::number(out,s,if (b.type == 3) { 8 } else if (b.type == 1 || b.type == 4) { 32 } else { 64 }) || !failed(out,s,b,4)) { return false; }
        if (b.operator == 11) {
            if (!w::text(out,s,b"    if(")) { return false; }
            if ((b.type == 1 || b.type == 2) && (!w::local(out,s,b.left) || !w::text(out,s,b"<0||"))) { return false; }
            if (!scalar::cast_local(out,s,b.left,b.type) || !w::text(out,s,b">((") || !w::unsigned_type(out,s,b.type)
                || !w::text(out,s,b")") || !w::maximum(out,s,b.type) || !w::text(out,s,b">>") || !w::local(out,s,b.right) || !w::text(out,s,b")") || !failed(out,s,b,1)) { return false; }
        }
    }
    return w::text(out,s,b"    ") && w::jump(out,s,b.target);
}
