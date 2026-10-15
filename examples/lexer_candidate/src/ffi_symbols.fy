module ffi_symbols;
import c_abi_types as c_types;
import pointer_types as pointers;
import std::bytes as bytes;

// Candidate scalar and pointer C ABI lane. SDK-owned fy_ symbols need authenticated SDK
// admission later; ordinary source cannot mint compiler/runtime identifiers.
pub fn bool scalar(u32 type,bool result){return type>=1 && type<=5 || result && type==7 || pointers::valid(type) || c_types::valid(type);}
fn bool letter(u8 c){return c>=65 && c<=90 || c>=97 && c<=122;}
pub fn bool valid(ref u8[] name){
    if(name.len==0 || name.len>256 || !letter(name[0])){return false;}
    var usize i=1;while(i<name.len){let u8 c=name[i];if(!letter(c) && !(c>=48 && c<=57) && c!=95){return false;}i+=1;}
    if(name.len>=3 && (bytes::equal(name[..3],b"fy_") || bytes::equal(name[..3],b"FY_") || bytes::equal(name[..3],b"INT")) || name.len>=4 && bytes::equal(name[..4],b"UINT")){return false;}
    let ref u8[] reserved=b"auto break case char const continue default do double else enum extern float for goto if inline int long register restrict return short signed sizeof static struct switch typedef union unsigned void volatile while _Alignas _Alignof _Atomic _Bool _Complex _Generic _Imaginary _Noreturn _Static_assert _Thread_local main bool true false uint8_t int32_t int64_t uint32_t uint64_t uintptr_t size_t ssize_t NULL offsetof CHAR_BIT SIZE_MAX memset memcpy memmove memcmp strlen ";
    i=0;while(i<reserved.len){let usize start=i;while(i<reserved.len && reserved[i]!=32){i+=1;}if(bytes::equal(name,reserved[start..i])){return false;}i+=1;}return true;
}

// Record/member names follow ADR 0320 and internal/cbackend/c_identifiers.go.
// External symbols retain the separate target-authenticated valid() lane above.
fn bool listed(ref u8[] name, ref u8[] names) {
    var usize i = 0;
    while (i < names.len) {
        let usize start = i;
        while (i < names.len && names[i] != 32) { i += 1; }
        if (bytes::equal(name, names[start..i])) { return true; }
        i += 1;
    }
    return false;
}
fn bool valid_record_or_member(ref u8[] name, bool member) {
    if (name.len == 0 || !letter(name[0]) && name[0] != 95) { return false; }
    if (name[0] == 95) {
        if (!member) { return false; }
        if (name.len > 1 && (name[1] == 95 || name[1] >= 65 && name[1] <= 90)) { return false; }
    }
    var usize i = 1;
    while (i < name.len) {
        let u8 c = name[i];
        if (!letter(c) && !(c >= 48 && c <= 57) && c != 95) { return false; }
        i += 1;
    }
    if (name.len >= 3 && bytes::equal(name[..3], b"FY_")) { return false; }
    if (listed(name, b"auto break case char const continue default do double else enum extern float for goto if inline int long register restrict return short signed sizeof static struct switch typedef union unsigned void volatile while _Alignas _Alignof _Atomic _Bool _Complex _Generic _Imaginary _Noreturn _Static_assert _Thread_local ")) { return false; }
    if (listed(name, b"uint8_t int32_t uint32_t uint64_t uintptr_t size_t ssize_t UINT8_C INT32_C UINT32_C UINT64_C INT32_MIN INT32_MAX UINT8_MAX UINT32_MAX UINTPTR_MAX ")) { return false; }
    return !listed(name, b"NULL offsetof INT8_C INT16_C INT64_C UINT16_C UINT32_C UINT64_C INT8_MIN INT16_MIN INT64_MIN INT8_MAX INT16_MAX INT64_MAX UINT16_MAX UINT64_MAX INT_LEAST8_MIN INT_LEAST16_MIN INT_LEAST32_MIN INT_LEAST64_MIN INT_LEAST8_MAX INT_LEAST16_MAX INT_LEAST32_MAX INT_LEAST64_MAX UINT_LEAST8_MAX UINT_LEAST16_MAX UINT_LEAST32_MAX UINT_LEAST64_MAX INT_FAST8_MIN INT_FAST16_MIN INT_FAST32_MIN INT_FAST64_MIN INT_FAST8_MAX INT_FAST16_MAX INT_FAST32_MAX INT_FAST64_MAX UINT_FAST8_MAX UINT_FAST16_MAX UINT_FAST32_MAX UINT_FAST64_MAX INTPTR_MIN INTPTR_MAX UINTPTR_MAX INTMAX_MIN INTMAX_MAX UINTMAX_MAX PTRDIFF_MIN PTRDIFF_MAX SIG_ATOMIC_MIN SIG_ATOMIC_MAX SIZE_MAX WCHAR_MIN WCHAR_MAX WINT_MIN WINT_MAX ");
}
pub fn bool record(ref u8[] name) {
    return valid_record_or_member(name, false) && !listed(name, b"fy_region fy_byte_slice fy_mut_byte_slice fy_i64_slice fy_mut_i64_slice fy_i32_slice fy_mut_i32_slice fy_u32_slice fy_mut_u32_slice fy_usize_slice fy_mut_usize_slice fy_struct_slice fy_str ");
}
pub fn bool member(ref u8[] name) { return valid_record_or_member(name, true); }
