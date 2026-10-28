module linux_host_paths;
import std::bytes as bytes;
import std::text as text;

@effects(filesystem,environment) @link_name("fairyc_private_linux_host_path_v1")
unsafe extern C fn u32 query(u32 kind,ptr mut u8 data,u32 capacity,ptr mut u32 length);

// 1 empty/NUL/non-UTF-8 path, 2 allocation, 3 native errno, 4 private path bound.
pub struct Problem { u32 kind; u32 os_code; usize offset; }
fn Problem fail(u32 kind,u32 os_code,usize offset) { return Problem{.kind,.os_code,.offset}; }

pub fn Result<bytes::Buffer,Problem> decode(ref u8[] path) {
    if(path.len>4095) { return .err(fail(4,0,path.len)); }
    if(path.len==0 || !text::validate_utf8(path).is_ok) { return .err(fail(1,0,0)); }
    var usize at=0;
    while(at<path.len) { if(path[at]==0) { return .err(fail(1,0,at)); } at+=1; }
    var bytes::Buffer out=bytes::empty();
    if(!bytes::append(ref mut out,path).is_ok) { return .err(fail(2,0,0)); }
    return .ok(out);
}

@effects(filesystem,environment) fn Result<bytes::Buffer,Problem> read(u32 kind) {
    var bytes::Buffer data=bytes::empty();
    if(!bytes::resize_zeroed(ref mut data,4096).is_ok) { return .err(fail(2,0,0)); }
    var u32 length=0;
    let status=unsafe { query(kind,bytes::view_mut(ref mut data) as ptr mut u8,4096,ref mut length as ptr mut u32) };
    if(status!=0) { return .err(fail(3,status,0)); }
    let count=length as usize;
    if(count>=4096) { return .err(fail(4,0,count)); }
    if(bytes::view(ref data)[count]!=0) { return .err(fail(1,0,count)); }
    if(kind==2 && count==0) { return decode(b"/tmp"); }
    return decode(bytes::view(ref data)[..count]);
}

@effects(filesystem,environment) pub fn Result<bytes::Buffer,Problem> current() { return read(1); }
@effects(filesystem,environment) pub fn Result<bytes::Buffer,Problem> temporary() { return read(2); }
@effects(filesystem,environment) pub fn Result<bytes::Buffer,Problem> executable() { return read(3); }
