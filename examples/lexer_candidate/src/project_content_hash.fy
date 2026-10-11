module project_content_hash;
import sha256_stream as sha;
import std::bytes as bytes;

fn u8 octet(usize value){let ref u8[] digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn bool length(ref mut sha::State state,usize value){var u8[8] encoded=[0;8];var usize index=0;var usize remaining=value;while(index<8){encoded[7-index]=octet(remaining%256);remaining/=256;index+=1;}let ref u8[] input=ref encoded;return remaining==0 && sha::update(state,input);}
// Callers validate unique canonical paths and append in exact path-byte order.
pub fn bool begin(ref mut sha::State state,usize count){return sha::update(state,b"fairyc.package-source\x00\x01") && length(state,count);}
pub fn bool file(ref mut sha::State state,ref u8[] path,ref u8[] body){return length(state,path.len) && sha::update(state,path) && length(state,body.len) && sha::update(state,body);}
pub fn Result<bytes::Buffer,u32> finish(ref sha::State state){var bytes::Buffer output=bytes::empty();if(!bytes::append(ref mut output,b"fysrc1-sha256-").is_ok || !sha::append_hex(state,ref mut output)){return .err(3);}return .ok(output);}
