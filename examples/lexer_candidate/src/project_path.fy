module project_path;
import project_path_unicode_data as unicode;
import std::bytes as bytes;
import std::text as text;

struct Rune{u32 value;usize next;}
pub fn bool charge(ref mut usize work,usize cost){if(*work>16777216 || cost>16777216-*work){return false;};*work+=cost;return true;}
// Physical source paths are broader than portable manifest-v1 paths. Keep
// UTF-8 names until the language layer checks Stage 0 module identifiers.
pub fn bool source_relative(ref u8[] path){
    if(path.len==0 || path.len>4095 || !text::validate_utf8(path).is_ok){return false;}var usize start=0;var usize index=0;
    while(index<=path.len){if(index==path.len || path[index]==47){let ref u8[] segment=path[start..index];if(segment.len==0 || bytes::equal(segment,b".") || bytes::equal(segment,b"..") || segment[segment.len-1]==46 || segment[segment.len-1]==32){return false;}start=index+1;}else{let u8 value=path[index];if(value<32 || value==127 || value==92 || value==58 || value==34 || value==60 || value==62 || value==124 || value==42 || value==63){return false;}}index+=1;}return true;
}
// Call only after UTF-8 validation. Each access belongs to a complete scalar.
fn Rune read(ref u8[] source,usize at){let u32 first=source[at] as u32;if(first<128){return Rune{.value=first,.next=at+1};}
    if(first<224){return Rune{.value=(first&31)*64+(source[at+1] as u32 & 63),.next=at+2};}
    if(first<240){return Rune{.value=(first&15)*4096+(source[at+1] as u32 & 63)*64+(source[at+2] as u32 & 63),.next=at+3};}
    return Rune{.value=(first&7)*262144+(source[at+1] as u32 & 63)*4096+(source[at+2] as u32 & 63)*64+(source[at+3] as u32 & 63),.next=at+4};
}
pub fn Result<bool,u32> equal_fold(ref u8[] left,ref u8[] right,ref mut usize work){
    if(left.len>16777216 || right.len>16777216 || !charge(work,left.len+right.len+1)){return .err(4);}if(!text::validate_utf8(left).is_ok || !text::validate_utf8(right).is_ok){return .err(2);}
    var usize a=0;var usize b=0;while(a<left.len && b<right.len){let Rune x=read(left,a);let Rune y=read(right,b);if(unicode::fold(x.value)!=unicode::fold(y.value)){return .ok(false);}a=x.next;b=y.next;}return .ok(a==left.len && b==right.len);
}
// The byte lookup avoids narrowing casts; only masked UTF-8 payloads index it.
fn u8 byte(u32 value){let ref u8[] table=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f\x20\x21\x22\x23\x24\x25\x26\x27\x28\x29\x2a\x2b\x2c\x2d\x2e\x2f\x30\x31\x32\x33\x34\x35\x36\x37\x38\x39\x3a\x3b\x3c\x3d\x3e\x3f\x40\x41\x42\x43\x44\x45\x46\x47\x48\x49\x4a\x4b\x4c\x4d\x4e\x4f\x50\x51\x52\x53\x54\x55\x56\x57\x58\x59\x5a\x5b\x5c\x5d\x5e\x5f\x60\x61\x62\x63\x64\x65\x66\x67\x68\x69\x6a\x6b\x6c\x6d\x6e\x6f\x70\x71\x72\x73\x74\x75\x76\x77\x78\x79\x7a\x7b\x7c\x7d\x7e\x7f\x80\x81\x82\x83\x84\x85\x86\x87\x88\x89\x8a\x8b\x8c\x8d\x8e\x8f\x90\x91\x92\x93\x94\x95\x96\x97\x98\x99\x9a\x9b\x9c\x9d\x9e\x9f\xa0\xa1\xa2\xa3\xa4\xa5\xa6\xa7\xa8\xa9\xaa\xab\xac\xad\xae\xaf\xb0\xb1\xb2\xb3\xb4\xb5\xb6\xb7\xb8\xb9\xba\xbb\xbc\xbd\xbe\xbf\xc0\xc1\xc2\xc3\xc4\xc5\xc6\xc7\xc8\xc9\xca\xcb\xcc\xcd\xce\xcf\xd0\xd1\xd2\xd3\xd4\xd5\xd6\xd7\xd8\xd9\xda\xdb\xdc\xdd\xde\xdf\xe0\xe1\xe2\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff";return table[value as usize];}
fn bool append_rune(ref mut bytes::Buffer output,u32 value){var u8[4] data=[0;4];var usize count=1;if(value<128){data[0]=byte(value);}else if(value<2048){count=2;data[0]=byte(192+(value>>6));data[1]=byte(128+(value&63));}else if(value<65536){count=3;data[0]=byte(224+(value>>12));data[1]=byte(128+((value>>6)&63));data[2]=byte(128+(value&63));}else{count=4;data[0]=byte(240+(value>>18));data[1]=byte(128+((value>>12)&63));data[2]=byte(128+((value>>6)&63));data[3]=byte(128+(value&63));}let ref u8[] view=ref data;return bytes::append(output,view[..count]).is_ok;}
pub fn Result<usize,u32> append_lower(ref mut bytes::Buffer output,ref u8[] source,ref mut usize work){
    if(source.len>16777216 || !charge(work,source.len+1)){return .err(4);}if(!text::validate_utf8(source).is_ok){return .err(2);}let usize initial=bytes::len(output);var usize at=0;while(at<source.len){let Rune rune=read(source,at);if(!append_rune(output,unicode::lower(rune.value))){discard bytes::truncate(output,initial);return .err(3);}at=rune.next;}return .ok(bytes::len(output)-initial);
}
