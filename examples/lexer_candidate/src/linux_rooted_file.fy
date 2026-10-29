module linux_rooted_file;
import linux_command as command;
import std::bytes as bytes;
import std::text as text;

@effects(filesystem) @link_name("fairyc_private_linux_directory_open_v1") unsafe extern C fn u32 open_directory(ptr u8 path,usize size,ptr mut u32 output);
@effects(filesystem) @link_name("fairyc_private_linux_root_directory_open_v1") unsafe extern C fn u32 open_child_directory(u32 root,ptr u8 leaf,usize size,ptr mut u32 output);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_open_v1") unsafe extern C fn u32 open_file(u32 root,ptr u8 leaf,usize size,ptr mut u32 output);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_info_v1") unsafe extern C fn u32 file_info(u32 file,ptr mut void output);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_read_v1") unsafe extern C fn u32 file_read(u32 file,ptr mut u8 data,usize size,ptr mut usize copied);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_close_v1") unsafe extern C fn u32 close_handle(u32 handle);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_create_v1") unsafe extern C fn u32 create_file(u32 root,ptr mut u32 output);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_write_v1") unsafe extern C fn u32 file_write(u32 file,ptr u8 data,usize size,ptr mut usize copied);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_sync_v1") unsafe extern C fn u32 file_sync(u32 file);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_publish_new_v1") unsafe extern C fn u32 file_publish_new(u32 file,u32 root,ptr u8 leaf,usize size);

// raw is fd+1; zero means no owned descriptor, including after consuming close.
@effects(filesystem) pub resource Directory { u32 raw; cleanup { unsafe { discard close_handle(this.raw); }; } }
@effects(filesystem) pub resource File { u32 raw; bool staged; cleanup { unsafe { discard close_handle(this.raw); }; } }
// Borrowed transport for other private host modules; ownership stays with Directory.
pub fn u32 directory_handle(ref Directory root) { return root.raw; }
// Stages: 1 input/bound, 2 allocation, 3 directory, 4 file, 5 metadata, 6 read, 7 close, 8 native ABI.
// 9 anonymous creation, 10 write (completed prefix), 11 sync, 12 publication.
pub struct Problem { u32 stage; u32 os_code; usize completed; }
pub struct Metadata { usize device; usize identity; usize size; i64 modified_seconds; u32 modified_nanos; u32 mode; }
fn Problem fail(u32 stage,u32 os_code,usize completed) { return Problem{.stage,.os_code,.completed}; }
pub fn bool missing(Problem p) { return (p.stage==3 || p.stage==4) && p.os_code==2; }
pub fn bool collision(Problem p) { return p.stage==12 && p.os_code==17; }
pub fn bool regular(Metadata p) { return (p.mode&61440)==32768; }
pub fn bool valid_leaf(ref u8[] value) {
    if(value.len==0 || value.len>255 || bytes::equal(value,b".") || bytes::equal(value,b"..") || !text::validate_utf8(value).is_ok) { return false; }
    var usize at=0; while(at<value.len) { if(value[at]==0 || value[at]==47) { return false; } at+=1; } return true;
}
fn Result<bytes::Buffer,Problem> path(ref u8[] value,bool leaf) {
    if(leaf && !valid_leaf(value)) { return .err(fail(1,0,0)); }
    if(!leaf && value.len>1 && value[value.len-1]==47) { return .err(fail(1,0,0)); }
    return switch(command::path(value)) { case .ok(p): { .ok(p) } case .err(e): { .err(fail(if(e.kind==3){2}else{1},0,e.offset)) } };
}
@effects(filesystem) pub fn Result<Directory,Problem> open(ref u8[] value) {
    let encoded=switch(path(value,false)) { case .ok(p): { p } case .err(e): { return .err(e); } };
    var u32 raw=0; let code=unsafe { open_directory(bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref mut raw as ptr mut u32) };
    if(code!=0) { return .err(fail(3,code,0)); } if(raw==0 || raw>2147483648) { return .err(fail(8,0,0)); } return .ok(Directory{.raw});
}
@effects(filesystem) pub fn Result<Directory,Problem> directory(ref Directory root,ref u8[] leaf) {
    let encoded=switch(path(leaf,true)) { case .ok(p): { p } case .err(e): { return .err(e); } };
    var u32 raw=0; let code=unsafe { open_child_directory(root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref mut raw as ptr mut u32) };
    if(code!=0) { return .err(fail(3,code,0)); } if(raw==0 || raw>2147483648) { return .err(fail(8,0,0)); } return .ok(Directory{.raw});
}
@effects(filesystem) pub fn Result<File,Problem> open_read(ref Directory root,ref u8[] leaf) {
    let encoded=switch(path(leaf,true)) { case .ok(p): { p } case .err(e): { return .err(e); } };
    var u32 raw=0; let code=unsafe { open_file(root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref mut raw as ptr mut u32) };
    if(code!=0) { return .err(fail(4,code,0)); } if(raw==0 || raw>2147483648) { return .err(fail(8,0,0)); } return .ok(File{.raw,.staged=false});
}
@effects(filesystem) pub fn Result<Metadata,Problem> observe(ref File file) {
    var info=Metadata{.device=0,.identity=0,.size=0,.modified_seconds=0,.modified_nanos=0,.mode=0};
    let code=unsafe { file_info(file.raw,ref mut info as ptr mut void) }; if(code!=0) { return .err(fail(5,code,0)); }
    if(!regular(info) || info.modified_nanos>=1000000000) { return .err(fail(8,0,0)); } return .ok(info);
}
@effects(filesystem) pub fn Result<usize,Problem> read(ref mut File file,ref mut u8[] data) {
    if(data.len>65536) { return .err(fail(1,0,0)); } var usize copied=0;
    let code=unsafe { file_read(file.raw,data as ptr mut u8,data.len,ref mut copied as ptr mut usize) };
    if(code!=0) { return .err(fail(6,code,0)); } if(copied>data.len) { return .err(fail(8,0,0)); } return .ok(copied);
}
@effects(filesystem) pub fn Result<bool,Problem> close(File input) {
    var File file=input;
    let raw=file.raw; file.raw=0; let code=unsafe { close_handle(raw) }; return if(code==0){.ok(true)}else{.err(fail(7,code,0))};
}
@effects(filesystem) pub fn Result<bool,Problem> close_directory(Directory input) {
    var Directory root=input;
    let raw=root.raw; root.raw=0; let code=unsafe { close_handle(raw) }; return if(code==0){.ok(true)}else{.err(fail(7,code,0))};
}
// Unsupported O_TMPFILE filesystems fail rather than introducing named cleanup races.
@effects(filesystem) pub fn Result<File,Problem> create(ref Directory root) {
    var u32 raw=0; let code=unsafe { create_file(root.raw,ref mut raw as ptr mut u32) };
    if(code!=0) { return .err(fail(9,code,0)); } if(raw==0 || raw>2147483648) { return .err(fail(8,0,0)); } return .ok(File{.raw,.staged=true});
}
@effects(filesystem) pub fn Result<usize,Problem> write_all(ref mut File file,ref u8[] data) {
    if(!file.staged) { return .err(fail(1,0,0)); }
    var usize written=0;
    while(written<data.len) {
        let usize count=if(data.len-written>65536){65536}else{data.len-written};var usize copied=0;
        let code=unsafe { file_write(file.raw,data[written..written+count] as ptr u8,count,ref mut copied as ptr mut usize) };
        if(code!=0) { return .err(fail(10,code,written)); }
        if(copied==0 || copied>count) { return .err(fail(8,0,written)); } written+=copied;
    } return .ok(written);
}
@effects(filesystem) pub fn Result<bool,Problem> sync(ref File file) {
    let code=unsafe { file_sync(file.raw) }; return if(code==0){.ok(true)}else{.err(fail(11,code,0))};
}
// File data is synced before an exclusive link. No parent-directory sync guarantee.
@effects(filesystem) pub fn Result<bool,Problem> publish_new(ref Directory root,ref mut File file,ref u8[] leaf) {
    if(!file.staged) { return .err(fail(1,0,0)); }
    let encoded=switch(path(leaf,true)) { case .ok(p): { p } case .err(e): { return .err(e); } };
    switch(sync(file)) { case .ok(_): {} case .err(e): { return .err(e); } };
    let code=unsafe { file_publish_new(file.raw,root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded)) };
    if(code!=0) { return .err(fail(12,code,0)); } file.staged=false; return .ok(true);
}
