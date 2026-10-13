module project_cache_encoding;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Independent 16MiB phase ledger. No encoded prefix is published on failure.
pub fn bool charge(ref mut usize work,usize cost){if(*work>16777216 || cost>16777216-*work){return false;};*work+=cost;return true;}
// Private errors: 1 empty/invalid UTF-8, 2 missing digest, 3 noncanonical set,
// 4 allocation, 5 bound. Input facts confer no artifact-reuse authority.
pub fn u32 string_check(ref u8[] value,ref mut usize work){if(value.len>=16777216 || !charge(work,value.len+1)){return 5;}return if(value.len!=0 && text::validate_utf8(value).is_ok){0}else{1};}
pub fn u32 raw(ref mut bytes::Buffer output,ref u8[] value,ref mut usize work){if(value.len>=16777216 || !charge(work,value.len+1)){return 5;}return if(bytes::append(output,value).is_ok){0}else{4};}
fn u8 octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
pub fn u32 length(ref mut bytes::Buffer output,usize value,ref mut usize work){var u8[8] encoded=[0;8];var usize rest=value;var usize at=0;while(at<8){encoded[7-at]=octet(rest%256);rest/=256;at+=1;}return if(rest!=0){5}else{raw(output,ref encoded,work)};}
pub fn u32 string(ref mut bytes::Buffer output,ref u8[] value,ref mut usize work){let code=length(output,value.len,work);return if(code!=0){code}else{raw(output,value,work)};}
pub fn bool before(ref u8[] a,ref u8[] b){var usize at=0;while(at<a.len && at<b.len){if(a[at]!=b[at]){return a[at]<b[at];}at+=1;}return a.len<b.len;}
pub fn u32 digest_check(ref u8[] value,ref mut usize work){if(!charge(work,value.len+1)){return 5;}if(value.len!=32){return 2;}var usize at=0;while(at<32){if(value[at]!=0){return 0;}at+=1;}return 2;}
pub fn u32 set_check(ref v::Vec<bytes::Buffer> values,ref mut usize work){let count=v::len<bytes::Buffer>(values);if(!charge(work,count+1)){return 5;}var usize at=0;while(at<count){let current=bytes::view(v::at<bytes::Buffer>(values,at));let code=string_check(current,work);if(code!=0){return code;}if(at!=0){let previous=bytes::view(v::at<bytes::Buffer>(values,at-1));if(!charge(work,previous.len+current.len+1)){return 5;}if(!before(previous,current)){return 3;}}at+=1;}return 0;}
pub fn u32 strings(ref mut bytes::Buffer output,ref v::Vec<bytes::Buffer> values,ref mut usize work){let count=v::len<bytes::Buffer>(values);var u32 code=length(output,count,work);var usize at=0;while(at<count && code==0){code=string(output,bytes::view(v::at<bytes::Buffer>(values,at)),work);at+=1;}return code;}
