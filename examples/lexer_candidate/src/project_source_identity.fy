module project_source_identity;
import project_source_set as sources;
import project_manifest_values as rules;
import sha256_stream as sha;
import std::bytes as bytes;

fn u8 octet(usize value){let ref u8[] digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn bool length(ref mut sha::State state,usize value){var u8[8] encoded=[0;8];var usize index=0;var usize remaining=value;while(index<8){encoded[7-index]=octet(remaining%256);remaining/=256;index+=1;}let ref u8[] input=ref encoded;return remaining==0 && sha::update(state,input);}

// Sources owns canonical, unique paths in exact-byte order. No filesystem read,
// decoding, module-name ordering or manifest metadata enters this digest.
// Errors: 1 retained-set bound/order invariant, 2 hash bound, 3 allocation.
pub fn Result<bytes::Buffer,u32> compute(ref sources::Sources input){
    let usize count=sources::count(input);if(count==0 || count>4096){return .err(1);}var sha::State state=sha::empty();
    if(!sha::update(ref mut state,b"fairyc.package-source\x00\x01") || !length(ref mut state,count)){return .err(2);}var usize index=0;var usize total=0;
    while(index<count){let ref u8[] path=sources::path(input,index);let ref u8[] body=sources::source(input,index);
        if(path.len==0 || path.len>4095 || body.len>8388608 || body.len>67108864-total){return .err(1);}if(index>0 && !rules::before(sources::path(input,index-1),path)){return .err(1);}total+=body.len;
        if(!length(ref mut state,path.len) || !sha::update(ref mut state,path) || !length(ref mut state,body.len) || !sha::update(ref mut state,body)){return .err(2);}index+=1;
    }
    var bytes::Buffer result=bytes::empty();if(!bytes::append(ref mut result,b"fysrc1-sha256-").is_ok || !sha::append_hex(ref state,ref mut result)){return .err(3);}return .ok(result);
}
