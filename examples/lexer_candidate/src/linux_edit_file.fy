module linux_edit_file;
import linux_rooted_file as rooted;
import linux_command as command;
import std::bytes as bytes;

@effects(filesystem) @link_name("fairyc_private_linux_edit_file_prepare_v1") unsafe extern C fn u32 native_prepare(u32 file,u32 mode);
@effects(filesystem) @link_name("fairyc_private_linux_edit_file_publish_v1") unsafe extern C fn u32 native_publish(u32 file,u32 stage,u32 root,ptr u8 leaf,usize size,ptr void metadata,ptr mut u32 cleanup_code);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_close_v1") unsafe extern C fn u32 native_close(u32 file);

// 0 writable, 1 prepared, 2 published, 3 failed/uncertain. Only close is allowed
// after a publication attempt or a native write failure. The cleared-handle
// transfer keeps exactly one close owner, including during ordinary error exits.
@effects(filesystem) pub resource File{u32 raw;rooted::Metadata original;usize written;u32 phase;cleanup{unsafe{discard native_close(this.raw);};}}
pub struct Problem{u32 stage;u32 os_code;u32 cleanup_code;usize completed;bool published;}
fn Problem fail(u32 stage,u32 code,usize completed){return Problem{.stage,.os_code=code,.cleanup_code=0,.completed,.published=false};}
pub fn usize maximum(){return 8388608;}
pub fn bool writable(rooted::Metadata info){return rooted::regular(info) && (info.mode&128)!=0 && info.modified_nanos<1000000000;}
pub fn bool published(ref File file){return file.phase==2;}
@effects(filesystem) pub fn Result<File,Problem> create(ref rooted::Directory root,rooted::Metadata original){
    if(!writable(original)){return .err(fail(5,0,0));}
    let owned=switch(rooted::create(root)){case .ok(p):{p}case .err(e):{return .err(fail(3,e.os_code,e.completed));}};
    return .ok(File{.raw=rooted::into_handle(owned),.original,.written=0,.phase=0});
}
@effects(filesystem) pub fn Result<usize,Problem> write_all(ref mut File file,ref u8[] data){
    if(file.phase>=2 || file.written>maximum() || data.len>maximum()-file.written){return .err(fail(1,0,0));}
    file.phase=0;
    let raw=file.raw;file.raw=0;
    var owned=unsafe{rooted::from_owned_staging_handle(raw)};
    let outcome=rooted::write_all(ref mut owned,data);
    file.raw=rooted::into_handle(owned);
    let n=switch(outcome){case .ok(p):{p}case .err(e):{file.phase=3;return .err(fail(4,e.os_code,e.completed));}};
    if(n!=data.len){file.phase=3;return .err(fail(9,0,n));}file.written+=n;return .ok(n);
}
@effects(filesystem) pub fn Result<bool,Problem> prepare(ref mut File file){
    if(file.phase!=0){return .err(fail(5,0,0));}
    // Freeze before crossing the native boundary; even a failed chmod/sync
    // consumes this preparation attempt. A fresh file is required to retry.
    file.phase=3;
    let original=file.original;let code=unsafe{native_prepare(file.raw,original.mode&511)};
    if(code!=0){return .err(fail(6,code,0));}file.phase=1;return .ok(true);
}
// The caller owns the exclusive edit lock and a private stage directory.
// Metadata is the immutable snapshot retained by successful preparation.
@effects(filesystem) pub fn Result<bool,Problem> publish(ref mut File file,ref rooted::Directory stage,ref rooted::Directory root,ref u8[] leaf){
    if(file.phase!=1 || !rooted::valid_leaf(leaf)){return .err(fail(1,0,0));}
    let encoded=switch(command::path(leaf)){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0));}};
    let expected=file.original;var u32 cleanup_code=0;file.phase=3;
    let code=unsafe{native_publish(file.raw,rooted::directory_handle(stage),rooted::directory_handle(root),bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref expected as ptr void,ref mut cleanup_code as ptr mut u32)};
    if(code==0){file.phase=2;if(cleanup_code==0){return .ok(true);}}
    return .err(Problem{.stage=if(code==0){9}else{7},.os_code=code,.cleanup_code,.completed=0,.published=code==0});
}
@effects(filesystem) pub fn Result<bool,Problem> close(File input){
    var File file=input;let was_published=published(ref file);let raw=file.raw;file.raw=0;
    let code=unsafe{native_close(raw)};
    return if(code==0){.ok(true)}else{.err(Problem{.stage=8,.os_code=code,.cleanup_code=0,.completed=0,.published=was_published})};
}
