module linux_build_scratch;
import linux_command as command;
import std::bytes as bytes;
@effects(filesystem) @link_name("fairyc_private_linux_build_scratch_open_v1") unsafe extern C fn u32 open_raw(ptr u8 root,usize size,ptr mut u8 path,usize capacity,ptr mut usize length,ptr mut usize output);
@effects(filesystem) @link_name("fairyc_private_linux_build_scratch_drop_v1") unsafe extern C fn u32 drop_raw(usize directory);
@effects(filesystem) pub resource Directory { usize raw; bytes::Buffer location; usize length; cleanup {unsafe{discard drop_raw(this.raw);};} }
pub fn ref u8[] path(ref Directory p){return bytes::view(ref p.location)[..p.length];}
@effects(filesystem) pub fn Result<Directory,u32> create(ref u8[] root){
    var usize end=root.len;while(end>1 && root[end-1]==47){end-=1;}
    let encoded=switch(command::path(root[..end])){case .ok(p):{p}case .err(_):{return .err(1);}};
    var bytes::Buffer location=bytes::empty();if(!bytes::resize_zeroed(ref mut location,4096).is_ok){return .err(2);}
    var usize raw=0;var usize length=0;
    let code=unsafe{open_raw(bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),bytes::view_mut(ref mut location) as ptr mut u8,4096,ref mut length as ptr mut usize,ref mut raw as ptr mut usize)};
    var directory=Directory{.raw,.location,.length};
    if(code!=0){return .err(code);}if(raw==0 || length==0 || length>4095){return .err(1);}
    return .ok(directory);
}
@effects(filesystem) pub fn Result<bool,u32> close(Directory input){var directory=input;let raw=directory.raw;directory.raw=0;let code=unsafe{drop_raw(raw)};return if(code==0){.ok(true)}else{.err(code)};}
