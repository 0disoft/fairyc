module generic_transport_scalars;
import std::bytes as bytes;

// Keep the original scalar, callable, literal and slice tags unchanged.
// These are value transport types, not arithmetic or direct extern-C scalars.
pub fn bool extended(u32 code){return code>=32 && code<=38;}
pub fn bool valid(u32 code){return code>=1 && code<=6 || extended(code);}
pub fn ref u8[] name(u32 code){
    assert(extended(code));
    if(code==32){return b"i8";}if(code==33){return b"i16";}if(code==34){return b"isize";}
    if(code==35){return b"u16";}if(code==36){return b"u64";}if(code==37){return b"f32";}return b"f64";
}
pub fn u32 named(ref u8[] text){var u32 code=32;while(code<=38){if(bytes::equal(text,name(code))){return code;}code+=1;}return 0;}
pub fn ref u8[] c_name(u32 code){
    assert(extended(code));
    if(code==32){return b"int8_t";}if(code==33){return b"int16_t";}if(code==34){return b"intptr_t";}
    if(code==35){return b"uint16_t";}if(code==36){return b"uint64_t";}if(code==37){return b"float";}return b"double";
}
