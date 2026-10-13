module module_program_artifact_wire;
import candidate_artifact_wire as common;
import candidate_limits as limits;
import std::bytes as bytes;

// Aggregate transport bounds, independent of each component's existing bounds.
// Errors: 1001 malformed, 1002 bound/work, 1003 allocation.
pub struct Cursor{usize offset;u32 code;}
pub struct Status{u32 code;}
pub struct Span{usize start;usize end;}
pub fn usize byte_limit(){return 268435456;}
pub fn usize work_limit(){return 1073741824;}
pub fn bool charge(ref mut usize work,usize cost){if(*work>work_limit() || cost>work_limit()-*work){return false;};*work+=cost;return true;}
pub fn u32 status(ref Cursor c){return c.code;}
fn u8 octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn void raw(ref mut bytes::Buffer out,ref u8[] value,ref mut Status s,ref mut usize work){if(s.code!=0){return;}if(value.len>byte_limit() || bytes::len(out)>byte_limit()-value.len || !charge(work,value.len+1)){s.code=1002;return;}if(!bytes::append(out,value).is_ok){s.code=1003;}}
pub fn bytes::Buffer writer_marker(ref u8[] marker,ref mut Status s,ref mut usize work){var out=bytes::empty();if(marker.len!=8){s.code=1001;return out;}raw(ref mut out,marker,s,work);return out;}
pub fn bytes::Buffer writer(ref mut Status s,ref mut usize work){return writer_marker(b"FYPROG00",s,work);}
pub fn void number(ref mut bytes::Buffer out,usize value,ref mut Status s,ref mut usize work){var u8[8] encoded=[0;8];var usize rest=value;var usize i=0;while(i<8){encoded[7-i]=octet(rest%256);rest/=256;i+=1;}raw(out,ref encoded,s,work);}
pub fn void member(ref mut bytes::Buffer out,ref u8[] value,ref mut Status s,ref mut usize work){number(out,value.len,s,work);raw(out,value,s,work);}
pub fn Result<common::Payload,u32> finish(bytes::Buffer out,Status s){return common::finish(out,common::Status{.code=s.code});}
pub fn Result<Cursor,u32> cursor_marker(ref u8[] raw,ref u8[] marker,ref mut usize work){if(!charge(work,9)){return .err(1002);}if(raw.len>byte_limit()){return .err(1002);}if(marker.len!=8 || raw.len<8 || !bytes::equal(raw[..8],marker)){return .err(1001);}return .ok(Cursor{.offset=8,.code=0});}
pub fn Result<Cursor,u32> cursor(ref u8[] raw,ref mut usize work){return cursor_marker(raw,b"FYPROG00",work);}
pub fn usize number_from(ref u8[] raw,ref mut Cursor c,ref mut usize work){if(c.code!=0){return 0;}if(!charge(work,9)){c.code=1002;return 0;}if(c.offset>raw.len || raw.len-c.offset<8){c.code=1001;return 0;}var usize value=0;var usize i=0;while(i<8){value=value*256+(raw[c.offset+i] as usize);i+=1;}c.offset+=8;return value;}
pub fn Span member_from(ref u8[] raw,ref mut Cursor c,usize limit,ref u8[] marker,ref mut usize work){let size=number_from(raw,c,work);let first=c.offset;if(c.code==0){if(size>limit){c.code=1002;}else if(size<8 || first>raw.len || size>raw.len-first){c.code=1001;}else if(!charge(work,9)){c.code=1002;}else if(!bytes::equal(raw[first..first+8],marker)){c.code=1001;}else{c.offset+=size;}}return Span{.start=first,.end=c.offset};}
pub fn usize bodies_from(ref u8[] raw,ref mut Cursor c,ref mut usize work){let count=number_from(raw,c,work);if(c.code==0){if(count>limits::project_frames()){c.code=1002;}else if(c.offset>raw.len || count>(raw.len-c.offset)/16){c.code=1001;}}return count;}
pub fn u32 end(ref u8[] raw,ref mut Cursor c,ref mut usize work){if(c.code==0){if(!charge(work,1)){c.code=1002;}else if(c.offset!=raw.len){c.code=1001;}}return c.code;}
// Validate every framing boundary before any nested owner is allocated.
pub fn u32 preflight(ref u8[] raw,ref mut usize work){var c=switch(cursor(raw,work)){case .ok(c):{c}case .err(code):{return code;}};
    discard member_from(raw,ref mut c,8388608,b"FYCLEAN0",work);discard member_from(raw,ref mut c,8388608,b"FYSDK000",work);discard member_from(raw,ref mut c,8388608,b"FYMETA00",work);discard member_from(raw,ref mut c,8388608,b"FYSTATE0",work);
    let count=bodies_from(raw,ref mut c,work);var usize i=0;while(i<count && c.code==0){discard member_from(raw,ref mut c,16777216,b"FYBODY00",work);i+=1;}return end(raw,ref mut c,work);
}
