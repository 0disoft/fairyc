module native_mir_artifact_wire;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import std::bytes as bytes;

// Exact private native-MIR family. The old project/unit envelopes stay closed.
// 00 has no closure records; 01 requires an actual closure feature and carries
// only symbolic helper/capture/operand facts, never an invoke pointer.
pub fn Result<bool,u32> family(ref u8[] raw){
    if(raw.len>wire::byte_limit()){return .err(1002);}
    if(raw.len<8){return .err(1001);}
    if(bytes::equal(raw[..8],b"FYNMIR00")){return .ok(false);}
    if(bytes::equal(raw[..8],b"FYNMIR01")){return .ok(true);}
    return .err(1001);
}
pub fn void blob(ref mut bytes::Buffer out,ref u8[] value,ref mut wire::Status s,ref mut usize work){
    wire::number(out,value.len,s,work);if(s.code!=0){return;}
    if(value.len>wire::byte_limit() || bytes::len(out)>wire::byte_limit()-value.len){s.code=1002;return;}
    let code=budget::raw(out,value,work);if(code!=0){s.code=if(code==4){1003}else{1002};}
}
pub fn Result<bytes::Buffer,u32> blob_from(ref u8[] raw,ref mut wire::Cursor c,usize limit,ref mut usize work){
    let size=wire::number_from(raw,c,work);if(c.code!=0){return .err(c.code);}
    if(size>limit){return .err(1002);}
    if(c.offset>raw.len || size>raw.len-c.offset){return .err(1001);}
    if(!budget::charge(work,size+1)){return .err(1002);}
    let out=switch(bytes::copy(raw[c.offset..c.offset+size])){case .ok(p):{p}case .err(_):{return .err(1003);}};
    c.offset+=size;return .ok(out);
}
