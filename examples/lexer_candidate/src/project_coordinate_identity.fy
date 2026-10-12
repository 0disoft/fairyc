module project_coordinate_identity;
import project_manifest_values as rules;
import project_path as work;
import sha256_stream as sha;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Public namespace only: source bytes and instance IDs are absent from this tuple.
pub resource Identity{bytes::Buffer canonical;bytes::Buffer digest;cleanup{}}
pub fn ref u8[] canonical(ref Identity value){return bytes::view(ref value.canonical);}
pub fn ref u8[] digest(ref Identity value){return bytes::view(ref value.digest);}
// Same private error meanings as project_instance_identity: 1 empty component,
// 2 invalid UTF-8, 3 feature order/duplicates, 4 empty feature, 5 allocation, 6 bound.
fn u32 component(ref u8[] value,ref mut usize meter){if(value.len==0){return 1;}if(value.len>=16777216 || !work::charge(meter,value.len+1)){return 6;}return if(text::validate_utf8(value).is_ok){0}else{2};}
fn u32 raw(ref mut bytes::Buffer out,ref u8[] value,ref mut usize meter){if(value.len>=16777216 || !work::charge(meter,value.len+1)){return 6;}return if(bytes::append(out,value).is_ok){0}else{5};}
fn u8 octet(usize value){let ref u8[] digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn u32 length(ref mut bytes::Buffer out,usize value,ref mut usize meter){var u8[8] encoded=[0;8];var usize rest=value;var usize at=0;while(at<8){encoded[7-at]=octet(rest%256);rest/=256;at+=1;}return if(rest!=0){6}else{raw(out,ref encoded,meter)};}
fn u32 field(ref mut bytes::Buffer out,ref u8[] tag,ref u8[] value,ref mut usize meter){var u32 code=raw(out,tag,meter);if(code!=0){return code;}code=length(out,value.len,meter);return if(code!=0){code}else{raw(out,value,meter)};}
pub fn Result<Identity,u32> compute(ref u8[] uid,ref u8[] version,ref v::Vec<bytes::Buffer> features,ref mut usize meter){
    var u32 code=component(uid,meter);if(code!=0){return .err(code);}code=component(version,meter);if(code!=0){return .err(code);}let count=v::len<bytes::Buffer>(features);if(count>=16777216 || !work::charge(meter,count+1)){return .err(6);}
    var usize at=1;while(at<count){let a=bytes::view(v::at<bytes::Buffer>(features,at-1));let b=bytes::view(v::at<bytes::Buffer>(features,at));if(a.len>=16777216 || b.len>=16777216 || !work::charge(meter,a.len+b.len+1)){return .err(6);}if(!rules::before(a,b)){return .err(3);}at+=1;}
    at=0;while(at<count){let f=bytes::view(v::at<bytes::Buffer>(features,at));if(f.len==0){return .err(4);}code=component(f,meter);if(code!=0){return .err(code);}at+=1;}
    var bytes::Buffer encoded=bytes::empty();code=raw(ref mut encoded,b"fairyc.package-coordinate\x00\x01",meter);if(code!=0){return .err(code);}code=field(ref mut encoded,b"u",uid,meter);if(code!=0){return .err(code);}code=field(ref mut encoded,b"v",version,meter);if(code!=0){return .err(code);}code=raw(ref mut encoded,b"f",meter);if(code!=0){return .err(code);}code=length(ref mut encoded,count,meter);if(code!=0){return .err(code);}
    at=0;while(at<count){let f=bytes::view(v::at<bytes::Buffer>(features,at));code=length(ref mut encoded,f.len,meter);if(code!=0){return .err(code);}code=raw(ref mut encoded,f,meter);if(code!=0){return .err(code);}at+=1;}
    if(!work::charge(meter,bytes::len(ref encoded)+1+77)){return .err(6);}var sha::State state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref encoded))){return .err(6);}var bytes::Buffer key=bytes::empty();if(!bytes::append(ref mut key,b"fypc1-sha256-").is_ok || !sha::append_hex(ref state,ref mut key)){return .err(5);}return .ok(Identity{.canonical=encoded,.digest=key});
}
pub fn bool valid_text(ref u8[] value){if(value.len!=77 || !bytes::equal(value[..13],b"fypc1-sha256-")){return false;}var usize at=13;while(at<77){let c=value[at];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}at+=1;}return true;}
