module windows_rooted_file;
import windows_command as command;
import std::bytes as bytes;

@effects(filesystem) @link_name("fairyc_private_windows_directory_open_v1") unsafe extern C fn usize open_directory(ptr u8 path,usize size);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_open_v1") unsafe extern C fn usize open_file(usize root,ptr u8 leaf,usize size,u32 mode);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_size_v1") unsafe extern C fn i32 file_size(usize file,ptr mut usize size);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_read_v1") unsafe extern C fn i32 file_read(usize file,ptr mut u8 data,usize size,ptr mut usize copied);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_write_v1") unsafe extern C fn i32 file_write(usize file,ptr u8 data,usize size,ptr mut usize written);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_sync_v1") unsafe extern C fn i32 file_sync(usize file);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_delete_v1") unsafe extern C fn i32 file_delete(usize file);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_rename_v1") unsafe extern C fn i32 file_rename(usize file,usize root,ptr u8 leaf,usize size);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_close_v1") unsafe extern C fn i32 close_handle(usize handle);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();

@effects(filesystem) pub resource Directory{usize raw;cleanup{if(this.raw!=0){unsafe{discard close_handle(this.raw);};}}}
// Exclusive staging owns its handle, not a path to reopen on cleanup. A failed
// publication deletes only this created object. Published files survive close.
@effects(filesystem) pub resource File{usize raw;bool staged;cleanup{if(this.raw!=0){unsafe{if(this.staged){discard file_delete(this.raw);}discard close_handle(this.raw);};}}}
// 1 input, 2 path allocation, 3 directory open, 4 file open, 5 size,
// 6 read, 7 write, 8 sync, 9 rename, 10 staged cleanup, 11 consuming close.
pub struct Problem{u32 stage;u32 os_code;usize completed;u32 close_code;}
fn Problem fail(u32 stage,u32 os_code,usize completed){return Problem{.stage,.os_code,.completed,.close_code=0};}
@effects(filesystem) fn Problem native(u32 stage,usize completed){return fail(stage,unsafe{last_error()},completed);}
pub fn bool missing(Problem p){return p.stage==4 && (p.os_code==2 || p.os_code==3);}
pub fn bool collision(Problem p){return p.stage==4 && (p.os_code==80 || p.os_code==183);}
pub fn bool valid_leaf(ref u8[] leaf){if(leaf.len==0 || leaf.len>1020 || bytes::equal(leaf,b".") || bytes::equal(leaf,b"..")){return false;}var usize i=0;while(i<leaf.len){if(leaf[i]==0 || leaf[i]==47 || leaf[i]==92 || leaf[i]==58){return false;}i+=1;}return true;}
fn Result<bytes::Buffer,Problem> wide(ref u8[] input,bool leaf){if(leaf && !valid_leaf(input)){return .err(fail(1,0,0));}let out=switch(command::path(input)){case .ok(p):{p}case .err(e):{return .err(fail(if(e.kind==3){2}else{1},0,e.offset));}};if(leaf && bytes::len(ref out)>512){return .err(fail(1,0,0));}return .ok(out);}
@effects(filesystem) pub fn Result<Directory,Problem> open(ref u8[] path){let encoded=switch(wide(path,false)){case .ok(p):{p}case .err(e):{return .err(e);}};let raw=unsafe{open_directory(bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded))};if(raw==0){return .err(native(3,0));}return .ok(Directory{.raw});}
@effects(filesystem) fn Result<File,Problem> child(ref Directory root,ref u8[] leaf,bool create){let encoded=switch(wide(leaf,true)){case .ok(p):{p}case .err(e):{return .err(e);}};let raw=unsafe{open_file(root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded)-2,if(create){2}else{1})};if(raw==0){return .err(native(4,0));}return .ok(File{.raw,.staged=create});}
@effects(filesystem) pub fn Result<File,Problem> open_read(ref Directory root,ref u8[] leaf){return child(root,leaf,false);}
@effects(filesystem) pub fn Result<File,Problem> create_new(ref Directory root,ref u8[] leaf){return child(root,leaf,true);}
@effects(filesystem) pub fn Result<usize,Problem> size(ref File file){var usize size=0;if(unsafe{file_size(file.raw,ref mut size as ptr mut usize)}==0){return .err(native(5,0));}return .ok(size);}
@effects(filesystem) pub fn Result<usize,Problem> read(ref mut File file,ref mut u8[] data){if(data.len>65536){return .err(fail(1,0,0));}var usize copied=0;if(unsafe{file_read(file.raw,data as ptr mut u8,data.len,ref mut copied as ptr mut usize)}==0){return .err(native(6,copied));}if(copied>data.len){return .err(fail(6,0,copied));}return .ok(copied);}
@effects(filesystem) pub fn Result<usize,Problem> write_all(ref mut File file,ref u8[] data){if(!file.staged){return .err(fail(1,0,0));}var usize written=0;while(written<data.len){let usize count=if(data.len-written>65536){65536}else{data.len-written};var usize copied=0;if(unsafe{file_write(file.raw,data[written..written+count] as ptr u8,count,ref mut copied as ptr mut usize)}==0){return .err(native(7,written+copied));}if(copied==0 || copied>count){return .err(fail(7,0,written));}written+=copied;}return .ok(written);}
@effects(filesystem) pub fn Result<bool,Problem> sync(ref mut File file){return if(unsafe{file_sync(file.raw)}!=0){.ok(true)}else{.err(native(8,0))};}
@effects(filesystem) pub fn Result<bool,Problem> publish(ref Directory root,ref mut File file,ref u8[] leaf){if(!file.staged){return .err(fail(1,0,0));}let encoded=switch(wide(leaf,true)){case .ok(p):{p}case .err(e):{return .err(e);}};if(unsafe{file_rename(file.raw,root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded)-2)}==0){return .err(native(9,0));}file.staged=false;return .ok(true);}
// Consume and clear before close: an uncertain close is never retried by glue.
@effects(filesystem) pub fn Result<bool,Problem> close(File input){var File file=input;var Problem problem=fail(0,0,0);if(file.staged && unsafe{file_delete(file.raw)}==0){problem=native(10,0);}let raw=file.raw;file.raw=0;if(unsafe{close_handle(raw)}==0){let code=unsafe{last_error()};if(problem.stage==0){problem=fail(11,code,0);}else{problem.close_code=code;}}return if(problem.stage==0){.ok(true)}else{.err(problem)};}
