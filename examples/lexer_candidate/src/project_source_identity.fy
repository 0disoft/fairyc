module project_source_identity;
import project_source_set as sources;
import project_manifest_values as rules;
import sha256_stream as sha;
import project_content_hash as hash;
import std::bytes as bytes;

// Sources owns canonical, unique paths in exact-byte order. No filesystem read,
// decoding, module-name ordering or manifest metadata enters this digest.
// Errors: 1 retained-set bound/order invariant, 2 hash bound, 3 allocation.
pub fn Result<bytes::Buffer,u32> compute(ref sources::Sources input){
    let usize count=sources::count(input);if(count==0 || count>4096){return .err(1);}var sha::State state=sha::empty();
    if(!hash::begin(ref mut state,count)){return .err(2);}var usize index=0;var usize total=0;
    while(index<count){let ref u8[] path=sources::path(input,index);let ref u8[] body=sources::source(input,index);
        if(path.len==0 || path.len>4095 || body.len>8388608 || body.len>67108864-total){return .err(1);}if(index>0 && !rules::before(sources::path(input,index-1),path)){return .err(1);}total+=body.len;
        if(!hash::file(ref mut state,path,body)){return .err(2);}index+=1;
    }
    return hash::finish(ref state);
}
