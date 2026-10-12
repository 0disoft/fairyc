module generic_identity;
import module_public_interface_write as write;
import module_state_layout as budget;
import sha256_stream as sha;
import std::bytes as bytes;

// Private generic identities share the existing 1 MiB work/output allowance.
// Codes: 9 invalid identity, 10 exhausted work, 11 allocation failure.
pub fn u32 text(ref mut bytes::Buffer out,ref u8[] value,ref mut usize work){let code=write::number(out,value.len,work);return if(code!=0){code}else{write::raw(out,value,work)};}
pub fn bool valid_hex(ref u8[] hex,bool nonzero){if(hex.len!=64){return false;}var bool present=false;var usize i=0;while(i<64){let c=hex[i];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}if(c!=48){present=true;}i+=1;}return !nonzero || present;}
fn u8 nibble(u8 c){return if(c<=57){c-48}else{c-87};}
pub fn u32 digest(ref mut bytes::Buffer out,ref u8[] hex,bool nonzero,ref mut usize work){if(!budget::charge(work,65)){return 10;}if(!valid_hex(hex,nonzero)){return 9;}var u8[32] raw=[0;32];var usize i=0;while(i<32){raw[i]=(nibble(hex[i*2])<<4)|nibble(hex[i*2+1]);i+=1;}return write::raw(out,ref raw,work);}
pub fn Result<bytes::Buffer,u32> hash(ref bytes::Buffer encoded,ref u8[] prefix,ref mut usize work){if(!budget::charge(work,bytes::len(encoded)+1)){return .err(10);}var sha::State state=sha::empty();if(!sha::update(ref mut state,bytes::view(encoded))){return .err(9);}var bytes::Buffer result=bytes::empty();let code=write::raw(ref mut result,prefix,work);if(code!=0){return .err(code);}if(!budget::charge(work,65)){return .err(10);}if(!sha::append_hex(ref state,ref mut result)){return .err(11);}return .ok(result);}
pub fn Result<bytes::Buffer,u32> digest_text(ref u8[] domain,ref u8[] value,ref mut usize work){var bytes::Buffer out=bytes::empty();var u32 code=text(ref mut out,b"fairyc.genericplan.digest.v0",work);if(code!=0){return .err(code);}code=text(ref mut out,domain,work);if(code!=0){return .err(code);}code=text(ref mut out,value,work);if(code!=0){return .err(code);}return hash(ref out,b"",work);}
pub fn bool valid_id(ref u8[] value,ref u8[] prefix){return value.len==prefix.len+64 && bytes::equal(value[..prefix.len],prefix) && valid_hex(value[prefix.len..],false);}
