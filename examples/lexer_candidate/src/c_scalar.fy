module c_scalar;
import mir_model as m;
import c_text as w;
import c_flow as flow;
import c_nominal as nominal;
import c_arrays as arrays;
import std::bytes as bytes;

pub fn bool cast_local(ref mut bytes::Buffer out, ref mut w::State s, usize id, u32 type) {
    return text_cast(out,s,type) && w::local(out,s,id) && w::text(out,s,b")");
}
fn bool text_cast(ref mut bytes::Buffer out, ref mut w::State s, u32 type) { return w::text(out,s,b"((") && w::unsigned_type(out,s,type) && w::text(out,s,b")"); }
fn bool bits(ref mut bytes::Buffer out, ref mut w::State s, u32 type) { return w::text(out,s,if (type == 1) { b"fy_bits_to_i32(" } else { b"fy_bits_to_i64(" }); }
fn bool constant(ref m::Program p, usize id, m::Operation o, ref mut bytes::Buffer out, ref mut w::State s) {
    if (o.type == 6) { return w::text(out,s,if (o.boolean) { b"INT32_C(1)" } else { b"INT32_C(0)" }); }
    let ref u8[] value = m::literal_text(p,id);
    if (o.type == 1 && bytes::equal(value,b"-2147483648") || o.type == 2 && bytes::equal(value,b"-9223372036854775808")) { return w::minimum(out,s,o.type); }
    let bool negative = value[0] == 45;
    if (o.type == 3 || o.type == 5) { if (!w::text(out,s,b"((") || !w::type(out,s,o.type) || !w::text(out,s,b")")) { return false; } }
    if (negative && !w::text(out,s,b"(-")) { return false; }
    if (!w::text(out,s,if (o.type == 1) { b"INT32_C(" } else if (o.type == 2) { b"INT64_C(" } else if (o.type == 5) { b"UINT64_C(" } else { b"UINT32_C(" })
        || !w::text(out,s,value[if (negative) { 1 } else { 0 }..]) || !w::text(out,s,b")")) { return false; }
    return (!negative || w::text(out,s,b")")) && (o.type != 3 && o.type != 5 || w::text(out,s,b")"));
}
pub fn bool operation(ref m::Program p, usize id, ref mut bytes::Buffer out, ref mut w::State s) {
    let m::Operation o = m::operation_at(p,id);
    if (o.kind >= 9) { return arrays::operation(p,o,out,s); }
    if (o.kind >= 6 && o.kind <= 8) { return nominal::operation(p,o,out,s); }
    if (o.kind == 5) { return true; }
    if (o.kind == 4 && (o.operator == 6 || o.operator == 7)) {
        if (!w::text(out,s,b"    ") || !w::divisor(out,s,o.owner_type) || !w::text(out,s,b" = ") || !w::local(out,s,o.right) || !w::text(out,s,b";\n")) { return false; }
    }
    if (!w::text(out,s,b"    ") || !w::local(out,s,o.result) || !w::text(out,s,b" = ")) { return false; }
    if (o.kind == 1) { if (!constant(p,id,o,out,s)) { return false; } }
    else if (o.kind == 2) { if (!w::local(out,s,o.left)) { return false; } }
    else if (o.kind == 3) {
        if (o.operator == 1) { if (!w::local(out,s,o.left)) { return false; } }
        else if (o.operator == 3) { if (!w::text(out,s,b"(!") || !w::local(out,s,o.left) || !w::text(out,s,b") ? INT32_C(1) : INT32_C(0)")) { return false; } }
        else if (o.owner_type == 1 || o.owner_type == 2) {
            if (!bits(out,s,o.owner_type) || !w::text(out,s,if (o.operator == 2) { b"0 - " } else { b"~" }) || !cast_local(out,s,o.left,o.owner_type) || !w::text(out,s,b")")) { return false; }
        } else {
            if (!w::text(out,s,b"(") || !w::type(out,s,o.type) || !w::text(out,s,b")~") || !cast_local(out,s,o.left,o.owner_type)) { return false; }
        }
    } else if (o.operator >= 13) {
        if (!w::text(out,s,b"(") || !w::local(out,s,o.left) || !w::operator(out,s,o.operator) || !w::local(out,s,o.right) || !w::text(out,s,b") ? INT32_C(1) : INT32_C(0)")) { return false; }
    } else if (o.operator == 6 || o.operator == 7) {
        if (!w::local(out,s,o.left) || !w::operator(out,s,o.operator) || !w::divisor(out,s,o.owner_type)) { return false; }
    } else if (o.operator == 12 && (o.owner_type == 1 || o.owner_type == 2)) {
        if (!w::text(out,s,if (o.owner_type == 1) { b"fy_sar_i32(" } else { b"fy_sar_i64(" }) || !w::local(out,s,o.left) || !w::text(out,s,b",") || !w::local(out,s,o.right) || !w::text(out,s,b")")) { return false; }
    } else {
        let bool signed = o.owner_type == 1 || o.owner_type == 2;
        if (signed) { if (!bits(out,s,o.owner_type)) { return false; } }
        else { if (!w::text(out,s,b"(") || !w::type(out,s,o.type) || !w::text(out,s,b")(")) { return false; } }
        if (!cast_local(out,s,o.left,o.owner_type) || !w::operator(out,s,o.operator) || !cast_local(out,s,o.right,o.owner_type) || !w::text(out,s,b")")) { return false; }
    }
    return w::text(out,s,b";\n");
}

pub fn bool helpers(ref m::Program p, ref flow::Plan plan, ref mut bytes::Buffer out, ref mut w::State s) {
    var bool bit32 = false; var bool bit64 = false; var bool sar32 = false; var bool sar64 = false;
    var bool add64 = false; var bool sub64 = false; var bool mul64 = false; var usize id = 1;
    while (id <= m::operation_count(p)) {
        let m::Operation o = m::operation_at(p,id);
        if (!flow::live(plan,o.block)) { id += 1; continue; }
        let bool bit = o.kind == 3 && (o.operator == 2 || o.operator == 4) || o.kind == 4 && (o.operator <= 5 || o.operator >= 8 && o.operator <= 12);
        bit32 = bit32 || bit && o.owner_type == 1; bit64 = bit64 || bit && o.owner_type == 2;
        sar32 = sar32 || o.kind == 4 && o.operator == 12 && o.owner_type == 1;
        sar64 = sar64 || o.kind == 4 && o.operator == 12 && o.owner_type == 2;
        add64 = add64 || o.kind == 5 && o.right != 0 && o.operator == 1 && o.owner_type == 2;
        sub64 = sub64 || o.kind == 5 && o.right != 0 && o.operator == 2 && o.owner_type == 2;
        mul64 = mul64 || o.kind == 5 && o.right != 0 && o.operator == 5 && o.owner_type == 2; id += 1;
    }
    return (!bit32 || w::text(out,s,b"static inline int32_t fy_bits_to_i32(uint32_t bits){int32_t value;memcpy(&value,&bits,sizeof value);return value;}\n"))
        && (!bit64 || w::text(out,s,b"static inline int64_t fy_bits_to_i64(uint64_t bits){int64_t value;memcpy(&value,&bits,sizeof value);return value;}\n"))
        && (!sar32 || w::text(out,s,b"static inline int32_t fy_sar_i32(int32_t value,int32_t amount){uint32_t bits=(uint32_t)value;if(amount==0||value>=0)return fy_bits_to_i32(bits>>(uint32_t)amount);return fy_bits_to_i32((bits>>(uint32_t)amount)|(UINT32_MAX<<(32u-(uint32_t)amount)));}\n"))
        && (!sar64 || w::text(out,s,b"static inline int64_t fy_sar_i64(int64_t value,int64_t amount){uint64_t bits=(uint64_t)value;if(amount==0||value>=0)return fy_bits_to_i64(bits>>(uint64_t)amount);return fy_bits_to_i64((bits>>(uint64_t)amount)|(UINT64_MAX<<(64u-(uint64_t)amount)));}\n"))
        && (!add64 || w::text(out,s,b"static inline int fy_i64_add_overflow(int64_t left,int64_t right){return(right>0&&left>INT64_MAX-right)||(right<0&&left<INT64_MIN-right);}\n"))
        && (!sub64 || w::text(out,s,b"static inline int fy_i64_sub_overflow(int64_t left,int64_t right){return(right<0&&left>INT64_MAX+right)||(right>0&&left<INT64_MIN+right);}\n"))
        && (!mul64 || w::text(out,s,b"static inline int fy_i64_mul_overflow(int64_t left,int64_t right){if(left>0)return right>0?left>INT64_MAX/right:right<INT64_MIN/left;if(left<0)return right>0?left<INT64_MIN/right:right<0&&left<INT64_MAX/right;return 0;}\n"));
}
