module project_identity_bytes;
import project_cache_encoding as budget;
import std::bytes as bytes;

// Decode a complete lower-case identity, preserving its domain/version prefix.
// 9 malformed/missing identity, 10 phase work, 11 allocation.
fn u8 nibble(u8 c){return if(c<=57){c-48}else{c-87};}
pub fn Result<bytes::Buffer,u32> decode(ref u8[] text,ref u8[] prefix,bool nonzero,ref mut usize work){
    if(prefix.len>1024 || text.len!=prefix.len+64){return .err(9);}if(!budget::charge(work,text.len+prefix.len+33)){return .err(10);}if(!bytes::equal(text[..prefix.len],prefix)){return .err(9);}var usize at=prefix.len;var bool present=false;
    while(at<text.len){let c=text[at];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return .err(9);}present=present || c!=48;at+=1;}if(nonzero && !present){return .err(9);}
    var bytes::Buffer output=bytes::empty();at=prefix.len;while(at<text.len){if(!bytes::append_byte(ref mut output,(nibble(text[at])<<4)|nibble(text[at+1])).is_ok){return .err(11);}at+=2;}return .ok(output);
}
pub fn Result<bytes::Buffer,u32> words(u32[8] hash,ref mut usize work){
    if(!budget::charge(work,33)){return .err(10);}let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";var bytes::Buffer output=bytes::empty();var usize at=0;var u32 shift=24;
    while(at<32){let word=hash[at/4]>>shift;let octet=(digits[((word>>4)&15) as usize]<<4)|digits[(word&15) as usize];if(!bytes::append_byte(ref mut output,octet).is_ok){return .err(11);}at+=1;if(at%4==0){shift=24;}else{shift-=8;}}return .ok(output);
}
