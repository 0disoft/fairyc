module ffi_symbols;
import pointer_types as pointers;
import std::bytes as bytes;

// Candidate scalar C ABI lane. SDK-owned fy_ symbols need authenticated SDK
// admission later; ordinary source cannot mint compiler/runtime identifiers.
pub fn bool scalar(u32 type,bool result){return type>=1 && type<=5 || result && type==7 || pointers::valid(type);}
fn bool letter(u8 c){return c>=65 && c<=90 || c>=97 && c<=122;}
pub fn bool valid(ref u8[] name){
    if(name.len==0 || name.len>256 || !letter(name[0])){return false;}
    var usize i=1;while(i<name.len){let u8 c=name[i];if(!letter(c) && !(c>=48 && c<=57) && c!=95){return false;}i+=1;}
    if(name.len>=3 && (bytes::equal(name[..3],b"fy_") || bytes::equal(name[..3],b"FY_") || bytes::equal(name[..3],b"INT")) || name.len>=4 && bytes::equal(name[..4],b"UINT")){return false;}
    let ref u8[] reserved=b"auto break case char const continue default do double else enum extern float for goto if inline int long register restrict return short signed sizeof static struct switch typedef union unsigned void volatile while _Alignas _Alignof _Atomic _Bool _Complex _Generic _Imaginary _Noreturn _Static_assert _Thread_local main bool true false uint8_t int32_t int64_t uint32_t uint64_t uintptr_t size_t ssize_t NULL offsetof CHAR_BIT SIZE_MAX memset memcpy memmove memcmp strlen ";
    i=0;while(i<reserved.len){let usize start=i;while(i<reserved.len && reserved[i]!=32){i+=1;}if(bytes::equal(name,reserved[start..i])){return false;}i+=1;}return true;
}
