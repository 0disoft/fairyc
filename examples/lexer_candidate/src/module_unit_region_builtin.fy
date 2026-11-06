module module_unit_region_builtin;
import std::bytes as bytes;
// Private typed operation IDs; source spelling alone never grants a Region capability.
pub fn u32 named(ref u8[] name){return if(bytes::equal(name,b"place")){300}else if(bytes::equal(name,b"alloc_zeroed_bytes")){301}else if(bytes::equal(name,b"copy_bytes")){302}else{0};}
pub fn bool allocation(u32 code){return code>=300 && code<=302;}
