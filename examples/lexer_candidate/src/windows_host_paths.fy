module windows_host_paths;
import std::bytes as bytes;

@effects(filesystem) @link_name("fairyc_private_windows_current_path_v1") unsafe extern C fn u32 current_path(ptr mut u8 data,u32 capacity);
@effects(environment) @link_name("fairyc_private_windows_temp_path_v1") unsafe extern C fn u32 temp_path(ptr mut u8 data,u32 capacity);
@link_name("fairyc_private_windows_image_path_v1") unsafe extern C fn u32 image_path(ptr mut u8 data,u32 capacity);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();
// 1 malformed UTF-16/path, 2 allocation, 3 native path query, 4 host path bound.
pub struct Problem{u32 kind;u32 os_code;usize offset;}
fn Problem fail(u32 kind,u32 os_code,usize offset){return Problem{.kind,.os_code,.offset};}
fn u32 word(ref u8[] data,usize at){return (data[at*2] as u32)+(data[at*2+1] as u32)*256;}
fn u8 octet(u32 value){let data=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (data[((value/16)%16) as usize]<<4)|data[(value%16) as usize];}
fn bool scalar(ref mut bytes::Buffer out,u32 value){var u8[4] data=[0;4];var usize count=1;if(value<128){data[0]=octet(value);}else if(value<2048){count=2;data[0]=octet(192+value/64);data[1]=octet(128+value%64);}else if(value<65536){count=3;data[0]=octet(224+value/4096);data[1]=octet(128+(value/64)%64);data[2]=octet(128+value%64);}else{count=4;data[0]=octet(240+value/262144);data[1]=octet(128+(value/4096)%64);data[2]=octet(128+(value/64)%64);data[3]=octet(128+value%64);}let ref u8[] view=ref data;return bytes::append(out,view[..count]).is_ok;}
pub fn Result<bytes::Buffer,Problem> decode(ref u8[] data){
    if(data.len<2 || data.len>65536 || data.len%2!=0 || word(data,data.len/2-1)!=0){return .err(fail(1,0,0));}var bytes::Buffer out=bytes::empty();var usize at=0;let count=data.len/2-1;
    while(at<count){var u32 point=word(data,at);if(point==0){return .err(fail(1,0,at));}at+=1;if(point>=55296 && point<=56319){if(at==count){return .err(fail(1,0,at-1));}let low=word(data,at);if(low<56320 || low>57343){return .err(fail(1,0,at));}point=65536+(point-55296)*1024+low-56320;at+=1;}else if(point>=56320 && point<=57343){return .err(fail(1,0,at-1));}if(!scalar(ref mut out,point)){return .err(fail(2,0,at));}}return .ok(out);
}
@effects(filesystem,environment) fn Result<bytes::Buffer,Problem> read(u32 kind){var bytes::Buffer wide=bytes::empty();if(!bytes::resize_zeroed(ref mut wide,65536).is_ok){return .err(fail(2,0,0));}let count=unsafe{if(kind==1){current_path(bytes::view_mut(ref mut wide) as ptr mut u8,32768)}else if(kind==2){temp_path(bytes::view_mut(ref mut wide) as ptr mut u8,32768)}else{image_path(bytes::view_mut(ref mut wide) as ptr mut u8,32768)}};if(count==0){return .err(fail(3,unsafe{last_error()},0));}if(count>=32768){return .err(fail(4,0,count as usize));}return decode(bytes::view(ref wide)[..((count as usize)+1)*2]);}
@effects(filesystem,environment) pub fn Result<bytes::Buffer,Problem> current(){return read(1);}
@effects(filesystem,environment) pub fn Result<bytes::Buffer,Problem> temporary(){return read(2);}
@effects(filesystem,environment) pub fn Result<bytes::Buffer,Problem> executable(){return read(3);}
pub fn Result<bytes::Buffer,Problem> sibling(ref u8[] image,ref u8[] leaf){var usize last=0;var usize at=0;while(at<image.len){if(image[at]==47 || image[at]==92){last=at+1;}at+=1;}if(last==0 || last==image.len || leaf.len==0){return .err(fail(1,0,last));}var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,image[..last]).is_ok || !bytes::append(ref mut out,leaf).is_ok){return .err(fail(2,0,last));}return .ok(out);}
@effects(filesystem,environment) pub fn Result<bytes::Buffer,Problem> driver(){let image=switch(executable()){case .ok(p):{p}case .err(e):{return .err(e);}};return sibling(bytes::view(ref image),b"fairyc-candidate-build.exe");}
