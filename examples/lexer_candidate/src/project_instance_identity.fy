module project_instance_identity;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_manifest as manifest;
import project_manifest_values as rules;
import project_path as work;
import sha256_stream as sha;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

pub resource Identity{bytes::Buffer canonical;bytes::Buffer digest;cleanup{}}
pub fn ref u8[] canonical(ref Identity identity){return bytes::view(ref identity.canonical);}
pub fn ref u8[] digest(ref Identity identity){return bytes::view(ref identity.digest);}
// 1 empty component, 2 UTF-8, 3 feature order/duplicates, 4 empty feature,
// 5 allocation, 6 implementation/work bound. Components are opaque UTF-8.
pub fn ref str reason(u32 code){if(code==1){return "empty-component";}if(code==2){return "invalid-utf8";}if(code==3){return "features-not-sorted-unique";}if(code==4){return "empty-feature";}if(code==5){return "allocation-failed";}return "implementation-bound";}
fn u32 component(ref u8[] value,ref mut usize meter){if(value.len==0){return 1;}if(!work::charge(meter,value.len+1)){return 6;}return if(text::validate_utf8(value).is_ok){0}else{2};}
fn u8 octet(usize value){let ref u8[] digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn bool append(ref mut bytes::Buffer output,ref u8[] value){let usize size=bytes::len(output);return size<=100663296 && value.len<=100663296-size && bytes::append(output,value).is_ok;}
fn bool length(ref mut bytes::Buffer output,usize value){var u8[8] encoded=[0;8];var usize rest=value;var usize at=0;while(at<8){encoded[7-at]=octet(rest%256);rest/=256;at+=1;}let ref u8[] input=ref encoded;return rest==0 && append(output,input);}
fn bool field(ref mut bytes::Buffer output,ref u8[] tag,ref u8[] value){return append(output,tag) && length(output,value.len) && append(output,value);}
pub fn Result<Identity,u32> compute(ref u8[] uid,ref u8[] source,ref u8[] version,ref v::Vec<bytes::Buffer> features,ref mut usize meter){
    var u32 code=component(uid,meter);if(code!=0){return .err(code);}code=component(source,meter);if(code!=0){return .err(code);}code=component(version,meter);if(code!=0){return .err(code);}
    let usize count=v::len<bytes::Buffer>(features);var usize at=1;while(at<count){let ref u8[] a=bytes::view(v::at<bytes::Buffer>(features,at-1));let ref u8[] b=bytes::view(v::at<bytes::Buffer>(features,at));if(!work::charge(meter,a.len+b.len+1)){return .err(6);}if(!rules::before(a,b)){return .err(3);}at+=1;}
    at=0;while(at<count){let ref u8[] feature=bytes::view(v::at<bytes::Buffer>(features,at));if(feature.len==0){return .err(4);}if(!work::charge(meter,feature.len+1)){return .err(6);}if(!text::validate_utf8(feature).is_ok){return .err(2);}at+=1;}
    var bytes::Buffer encoded=bytes::empty();if(!append(ref mut encoded,b"fairyc.package-instance\x00\x01") || !field(ref mut encoded,b"u",uid) || !field(ref mut encoded,b"s",source) || !field(ref mut encoded,b"v",version) || !append(ref mut encoded,b"f") || !length(ref mut encoded,count)){return .err(5);}
    at=0;while(at<count){let ref u8[] feature=bytes::view(v::at<bytes::Buffer>(features,at));if(!length(ref mut encoded,feature.len) || !append(ref mut encoded,feature)){return .err(5);}at+=1;}
    if(!work::charge(meter,bytes::len(ref encoded)+1)){return .err(6);}var sha::State state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref encoded))){return .err(6);}var bytes::Buffer digest=bytes::empty();if(!bytes::append(ref mut digest,b"fypi1-sha256-").is_ok || !sha::append_hex(ref state,ref mut digest)){return .err(5);}return .ok(Identity{.canonical=encoded,.digest});
}
pub fn bool valid_text(ref u8[] value){if(value.len!=77 || !bytes::equal(value[..13],b"fypi1-sha256-")){return false;}var usize at=13;while(at<77){let u8 c=value[at];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}at+=1;}return true;}
// 0 different digest, 1 exact tuple, 2 claimed digest collision, 3 malformed ID.
// A digest match never skips comparison of the trusted canonical tuple bytes.
pub fn u32 compare_record(ref Identity known,ref u8[] claimed_digest,ref u8[] claimed_canonical){if(!valid_text(claimed_digest)){return 3;}if(!bytes::equal(digest(known),claimed_digest)){return 0;}return if(bytes::equal(canonical(known),claimed_canonical)){1}else{2};}
pub fn u32 compare(ref Identity left,ref Identity right){return compare_record(left,digest(right),canonical(right));}
pub fn Result<Identity,u32> root(ref workspace::Workspace input,ref mut usize meter){let local=workspace::local_catalog(input);let m=catalog::package(local,0);let v::Vec<bytes::Buffer> features=v::empty<bytes::Buffer>();return compute(manifest::package_value(m,b"uid"),catalog::digest(local,0),manifest::package_value(m,b"version"),ref features,meter);}
pub fn Result<Identity,u32> dependency(ref workspace::Workspace input,usize at,ref mut usize meter){
    if(at>=workspace::edge_count(input)){return .err(6);}let e=workspace::edge(input,at);let local=workspace::local_catalog(input);let m=catalog::package(local,e.target);var v::Vec<bytes::Buffer> features=v::empty<bytes::Buffer>();var usize id=workspace::feature_first(input,at);
    while(id!=0){let ref u8[] value=workspace::feature(input,at,id);if(!work::charge(meter,value.len+1)){return .err(6);}let bytes::Buffer owned=switch(bytes::copy(value)){case .ok(value):{value}case .err(_):{return .err(5);}};if(!v::push<bytes::Buffer>(ref mut features,owned).is_ok){return .err(5);}id=workspace::next_feature(input,at,id);}
    return compute(manifest::package_value(m,b"uid"),catalog::digest(local,e.target),manifest::package_value(m,b"version"),ref features,meter);
}
