module module_body_artifact_wire;
import candidate_artifact_wire as common;

import std::bytes as bytes;
// Body transport: 16MiB bytes and 32MiB work; source/IR limits stay unchanged.
// Cleanup errors: 1001 malformed, 1002 bound/work, 1003 allocation.
pub struct Cursor{usize offset;u32 code;}
pub struct Status{u32 code;}
pub fn usize byte_limit(){return 16777216;}
pub fn u32 status(ref Cursor c){return c.code;}
pub fn u32 output_status(ref Status p){return p.code;}
pub fn bool charge(ref mut usize work,usize amount){if(*work>33554432 || amount>33554432-*work){return false;};*work+=amount;return true;}
pub fn bytes::Buffer writer(ref u8[] marker,ref mut Status state,ref mut usize work){var bytes::Buffer data=bytes::empty();if(state.code==0){let u32 code=if(marker.len!=8){1001}else{raw(ref mut data,marker,work)};state.code=if(code==0 || code==1001){code}else if(code==4){1003}else{1002};}return data;}
pub fn void number(ref mut bytes::Buffer p,usize value,ref mut Status state,ref mut usize work){if(state.code!=0){return;}if(bytes::len(p)>byte_limit()-8){state.code=1002;return;}let code=length(p,value,work);if(code!=0){state.code=if(code==4){1003}else{1002};}}
pub fn Result<common::Payload,u32> finish(bytes::Buffer p,Status state){return common::finish(p,common::Status{.code=state.code});}
pub fn Result<Cursor,u32> cursor(ref u8[] raw,ref u8[] marker,ref mut usize work){if(!charge(work,9)){return .err(1002);}if(raw.len>byte_limit()){return .err(1002);}if(marker.len!=8 || raw.len<8 || !bytes::equal(raw[..8],marker)){return .err(1001);}return .ok(Cursor{.offset=8,.code=0});}
fn bool field(ref u8[] raw,ref mut Cursor c,ref mut usize work){if(c.code!=0){return false;}if(!charge(work,9)){c.code=1002;return false;}if(c.offset>raw.len || raw.len-c.offset<8){c.code=1001;return false;}return true;}
pub fn usize number_from(ref u8[] raw,ref mut Cursor c,ref mut usize work){if(!field(raw,c,work)){return 0;}var usize value=0;var usize i=0;while(i<8){value=value*256+(raw[c.offset+i] as usize);i+=1;}c.offset+=8;return value;}
pub fn u32 word_from(ref u8[] raw,ref mut Cursor c,ref mut usize work){if(!field(raw,c,work)){return 0;}var usize i=0;while(i<4){if(raw[c.offset+i]!=0){c.code=1001;return 0;}i+=1;}var u32 value=0;while(i<8){value=value*256+(raw[c.offset+i] as u32);i+=1;}c.offset+=8;return value;}
pub fn bool flag_from(ref u8[] raw,ref mut Cursor c,ref mut usize work){let value=number_from(raw,c,work);if(value>1){c.code=1001;}return value==1;}
pub fn usize count_from(ref u8[] raw,ref mut Cursor c,usize limit,usize width,ref mut usize work){let value=number_from(raw,c,work);if(c.code!=0){return 0;}if(value>limit){c.code=1002;return 0;}if(width==0 || c.offset>raw.len || value>(raw.len-c.offset)/width){c.code=1001;return 0;}return value;}
pub fn u32 end(ref u8[] raw,ref mut Cursor c,ref mut usize work){if(c.code==0){if(!charge(work,1)){c.code=1002;}else if(c.offset!=raw.len){c.code=1001;}}return c.code;}

fn u32 raw(ref mut bytes::Buffer out,ref u8[] value,ref mut usize work){if(value.len>byte_limit() || bytes::len(out)>byte_limit()-value.len || !charge(work,value.len+1)){return 5;}return if(bytes::append(out,value).is_ok){0}else{4};}
fn u8 octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn u32 length(ref mut bytes::Buffer out,usize value,ref mut usize work){var u8[8] encoded=[0;8];var usize rest=value;var usize i=0;while(i<8){encoded[7-i]=octet(rest%256);rest/=256;i+=1;}return raw(out,ref encoded,work);}
pub fn void blob(ref mut bytes::Buffer out,ref u8[] value,ref mut Status state,ref mut usize work){number(out,value.len,state,work);if(state.code==0){let code=raw(out,value,work);if(code!=0){state.code=if(code==4){1003}else{1002};}}}
pub fn bytes::Buffer blob_from(ref u8[] raw,ref mut Cursor c,usize limit,ref mut usize work){let n=count_from(raw,c,limit,1,work);if(c.code!=0){return bytes::empty();}if(!charge(work,n+1)){c.code=1002;return bytes::empty();}let copied=switch(bytes::copy(raw[c.offset..c.offset+n])){case .ok(p):{p}case .err(_):{c.code=1003;return bytes::empty();}};c.offset+=n;return copied;}
