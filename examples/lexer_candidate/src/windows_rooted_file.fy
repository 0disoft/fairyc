module windows_rooted_file;
import windows_command as command;
import std::bytes as bytes;
import std::vec as v;
import windows_host_paths as paths;

@effects(filesystem) @link_name("fairyc_private_windows_directory_open_v1") unsafe extern C fn usize open_directory(ptr u8 path,usize size);
@effects(filesystem) @link_name("fairyc_private_windows_root_directory_open_v1") unsafe extern C fn usize open_child_directory(usize root,ptr u8 leaf,usize size);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_open_v1") unsafe extern C fn usize open_file(usize root,ptr u8 leaf,usize size,u32 mode);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_size_v1") unsafe extern C fn i32 file_size(usize file,ptr mut usize size);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_read_v1") unsafe extern C fn i32 file_read(usize file,ptr mut u8 data,usize size,ptr mut usize copied);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_write_v1") unsafe extern C fn i32 file_write(usize file,ptr u8 data,usize size,ptr mut usize written);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_sync_v1") unsafe extern C fn i32 file_sync(usize file);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_delete_v1") unsafe extern C fn i32 file_delete(usize file);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_rename_v1") unsafe extern C fn i32 file_rename(usize file,usize root,ptr u8 leaf,usize size);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_publish_new_v1") unsafe extern C fn i32 file_publish_new(usize file,usize root,ptr u8 leaf,usize size);
@effects(clock) @link_name("fairyc_private_windows_root_file_time_v1") unsafe extern C fn usize file_time();
@effects(filesystem) @link_name("fairyc_private_windows_root_file_close_v1") unsafe extern C fn i32 close_handle(usize handle);
@effects(filesystem) @link_name("fairyc_private_windows_root_file_info_v1") unsafe extern C fn i32 file_info(usize handle,ptr mut void output);
@effects(filesystem) @link_name("fairyc_private_windows_root_directory_query_v1") unsafe extern C fn i32 directory_query(usize handle,ptr mut u8 output,usize size,u32 restart);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();

@effects(filesystem) pub resource Directory{usize raw;cleanup{if(this.raw!=0){unsafe{discard close_handle(this.raw);};}}}
// Exclusive staging owns its handle, not a path to reopen on cleanup. A failed
// publication deletes only this created object. Published files survive close.
@effects(filesystem) pub resource File{usize raw;bool staged;cleanup{if(this.raw!=0){unsafe{if(this.staged){discard file_delete(this.raw);}discard close_handle(this.raw);};}}}
// 1 input, 2 path allocation, 3 directory open, 4 file open, 5 size,
// 6 read, 7 write, 8 sync, 9 rename, 10 staged cleanup, 11 consuming close.
// 12 metadata/scan, 13 malformed native records, 14 bound/work.
pub struct Problem{u32 stage;u32 os_code;usize completed;u32 close_code;}
fn Problem fail(u32 stage,u32 os_code,usize completed){return Problem{.stage,.os_code,.completed,.close_code=0};}
@effects(filesystem) fn Problem native(u32 stage,usize completed){return fail(stage,unsafe{last_error()},completed);}
pub fn bool missing(Problem p){return p.stage==4 && (p.os_code==2 || p.os_code==3);}
pub fn bool collision(Problem p){return p.stage==4 && (p.os_code==80 || p.os_code==183);}
pub fn bool valid_leaf(ref u8[] leaf){if(leaf.len==0 || leaf.len>1020 || bytes::equal(leaf,b".") || bytes::equal(leaf,b"..")){return false;}var usize i=0;while(i<leaf.len){if(leaf[i]==0 || leaf[i]==47 || leaf[i]==92 || leaf[i]==58){return false;}i+=1;}return true;}
fn Result<bytes::Buffer,Problem> wide(ref u8[] input,bool leaf){if(leaf && !valid_leaf(input)){return .err(fail(1,0,0));}let out=switch(command::path(input)){case .ok(p):{p}case .err(e):{return .err(fail(if(e.kind==3){2}else{1},0,e.offset));}};if(leaf && bytes::len(ref out)>512){return .err(fail(1,0,0));}return .ok(out);}
@effects(filesystem) pub fn Result<Directory,Problem> open(ref u8[] path){let encoded=switch(wide(path,false)){case .ok(p):{p}case .err(e):{return .err(e);}};let raw=unsafe{open_directory(bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded))};if(raw==0){return .err(native(3,0));}return .ok(Directory{.raw});}
@effects(filesystem) pub fn Result<Directory,Problem> directory(ref Directory root,ref u8[] leaf){let encoded=switch(wide(leaf,true)){case .ok(p):{p}case .err(e):{return .err(e);}};let raw=unsafe{open_child_directory(root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded)-2)};if(raw==0){return .err(native(3,0));}return .ok(Directory{.raw});}
@effects(filesystem) fn Result<File,Problem> child(ref Directory root,ref u8[] leaf,bool create){let encoded=switch(wide(leaf,true)){case .ok(p):{p}case .err(e):{return .err(e);}};let raw=unsafe{open_file(root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded)-2,if(create){2}else{1})};if(raw==0){return .err(native(4,0));}return .ok(File{.raw,.staged=create});}
@effects(filesystem) pub fn Result<File,Problem> open_read(ref Directory root,ref u8[] leaf){return child(root,leaf,false);}
@effects(filesystem) pub fn Result<File,Problem> create_new(ref Directory root,ref u8[] leaf){return child(root,leaf,true);}
@effects(filesystem) pub fn Result<usize,Problem> size(ref File file){var usize size=0;if(unsafe{file_size(file.raw,ref mut size as ptr mut usize)}==0){return .err(native(5,0));}return .ok(size);}
@effects(filesystem) pub fn Result<usize,Problem> read(ref mut File file,ref mut u8[] data){if(data.len>65536){return .err(fail(1,0,0));}var usize copied=0;if(unsafe{file_read(file.raw,data as ptr mut u8,data.len,ref mut copied as ptr mut usize)}==0){return .err(native(6,copied));}if(copied>data.len){return .err(fail(6,0,copied));}return .ok(copied);}
@effects(filesystem) pub fn Result<usize,Problem> write_all(ref mut File file,ref u8[] data){if(!file.staged){return .err(fail(1,0,0));}var usize written=0;while(written<data.len){let usize count=if(data.len-written>65536){65536}else{data.len-written};var usize copied=0;if(unsafe{file_write(file.raw,data[written..written+count] as ptr u8,count,ref mut copied as ptr mut usize)}==0){return .err(native(7,written+copied));}if(copied==0 || copied>count){return .err(fail(7,0,written));}written+=copied;}return .ok(written);}
@effects(filesystem) pub fn Result<bool,Problem> sync(ref mut File file){return if(unsafe{file_sync(file.raw)}!=0){.ok(true)}else{.err(native(8,0))};}
@effects(filesystem) pub fn Result<bool,Problem> publish(ref Directory root,ref mut File file,ref u8[] leaf){if(!file.staged){return .err(fail(1,0,0));}let encoded=switch(wide(leaf,true)){case .ok(p):{p}case .err(e):{return .err(e);}};if(unsafe{file_rename(file.raw,root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded)-2)}==0){return .err(native(9,0));}file.staged=false;return .ok(true);}
// Consume and clear before close: an uncertain close is never retried by glue.
@effects(filesystem) pub fn Result<bool,Problem> publish_new(ref Directory root,ref mut File file,ref u8[] leaf){if(!file.staged){return .err(fail(1,0,0));}let encoded=switch(wide(leaf,true)){case .ok(p):{p}case .err(e):{return .err(e);}};if(unsafe{file_publish_new(file.raw,root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded)-2)}==0){return .err(native(9,0));}file.staged=false;return .ok(true);}
// Windows absolute file-time units (100ns since 1601), for retention only.
@effects(clock) pub fn usize now(){return unsafe{file_time()};}
// Consume and clear before close: an uncertain close is never retried by glue.
@effects(filesystem) pub fn Result<bool,Problem> close(File input){var File file=input;var Problem problem=fail(0,0,0);if(file.staged && unsafe{file_delete(file.raw)}==0){problem=native(10,0);}let raw=file.raw;file.raw=0;if(unsafe{close_handle(raw)}==0){let code=unsafe{last_error()};if(problem.stage==0){problem=fail(11,code,0);}else{problem.close_code=code;}}return if(problem.stage==0){.ok(true)}else{.err(problem)};}

@repr(C) pub struct Metadata{usize identity;usize modified;usize size;u32 attributes;}

pub resource Entries{v::Vec<bytes::Buffer> names;v::Vec<usize> facts;cleanup{}}
pub fn usize count(ref Entries p){return v::len<bytes::Buffer>(ref p.names);}

pub fn ref u8[] name(ref Entries p,usize index){return bytes::view(v::at<bytes::Buffer>(ref p.names,index));}
pub fn Metadata metadata(ref Entries p,usize index){assert(index<count(p));let flags=*v::at<usize>(ref p.facts,index*4+3);var u32 attributes=0;var u32 bit=1;var usize mask=1;var usize i=0;while(i<32){if((flags&mask)!=0){attributes|=bit;}i+=1;if(i<32){bit*=2;mask*=2;}}return Metadata{.identity=*v::at<usize>(ref p.facts,index*4),.modified=*v::at<usize>(ref p.facts,index*4+1),.size=*v::at<usize>(ref p.facts,index*4+2),.attributes};}
pub fn bool regular(Metadata p){return (p.attributes&1104)==0;}
fn bool charge(ref mut usize work,usize cost){if(*work>16777216 || cost>16777216-*work){return false;};*work+=cost;return true;}
fn usize little(ref u8[] raw,usize offset,usize width){var usize value=0;var usize i=width;while(i>0){i-=1;value=value*256+(raw[offset+i] as usize);}return value;}
fn u32 attributes(ref u8[] raw,usize at){var u32 value=0;var usize i=4;while(i>0){i-=1;value=value*256+(raw[at+i] as u32);}return value;}
// Validate each complete batch before returning any rows to the caller. OS
// metadata is only a retention hint; it grants no saved-program authority.
fn Result<bool,Problem> batch(ref u8[] raw,ref mut v::Vec<bytes::Buffer> names,ref mut v::Vec<usize> facts,usize maximum,ref mut usize work){var usize offset=0;
    while(true){if(offset>raw.len || raw.len-offset<106){return .err(fail(13,0,offset));}let next=little(raw,offset,4);let length=little(raw,offset+60,4);let size=little(raw,offset+40,8);
        if(length==0 || length>510 || length%2!=0 || length>raw.len-offset-104 || size>9223372036854775807 || next!=0 && (next%8!=0 || next<104+length || next>raw.len-offset)){return .err(fail(13,0,offset));}if(!charge(work,length+105)){return .err(fail(14,0,offset));}
        var encoded=bytes::empty();if(!bytes::append(ref mut encoded,raw[offset+104..offset+104+length]).is_ok || !bytes::append_byte(ref mut encoded,0).is_ok || !bytes::append_byte(ref mut encoded,0).is_ok){return .err(fail(2,0,offset));}
        let name=switch(paths::decode(bytes::view(ref encoded))){case .ok(p):{p}case .err(e):{return .err(fail(if(e.kind==2){2}else{13},0,offset));}};
        if(bytes::equal(bytes::view(ref name),b".") || bytes::equal(bytes::view(ref name),b"..")){if(next==0){return .ok(true);}offset+=next;continue;}
        if(v::len<bytes::Buffer>(names)>=maximum){return .err(fail(14,0,offset));}let info=Metadata{.identity=little(raw,offset+96,8),.modified=little(raw,offset+24,8),.size,.attributes=attributes(raw,offset+56)};if(!v::push<bytes::Buffer>(names,name).is_ok || !v::push<usize>(facts,info.identity).is_ok || !v::push<usize>(facts,info.modified).is_ok || !v::push<usize>(facts,info.size).is_ok || !v::push<usize>(facts,(info.attributes as usize)).is_ok){return .err(fail(2,0,offset));}
        if(next==0){return .ok(true);}offset+=next;
    }return .err(fail(13,0,offset));
}
pub fn Result<Entries,Problem> decode_batch(ref u8[] raw,usize maximum,ref mut usize work){if(raw.len>65536 || maximum==0 || maximum>4096 || !charge(work,1)){return .err(fail(14,0,0));}var v::Vec<bytes::Buffer> names=v::empty<bytes::Buffer>();var v::Vec<usize> facts=v::empty<usize>();switch(batch(raw,ref mut names,ref mut facts,maximum,work)){case .ok(_):{}case .err(e):{return .err(e);}};return .ok(Entries{.names,.facts});}
// A separate bounded scan history restarts the same retained handle. No path
// lookup, FindFirstFile or independent directory reopen occurs here.
@effects(filesystem) pub fn Result<Entries,Problem> scan(ref Directory root,usize maximum,ref mut usize work){if(maximum==0 || maximum>4096 || !charge(work,65537)){return .err(fail(14,0,0));}var raw=bytes::empty();if(!bytes::resize_zeroed(ref mut raw,65536).is_ok){return .err(fail(2,0,0));}var v::Vec<bytes::Buffer> names=v::empty<bytes::Buffer>();var v::Vec<usize> facts=v::empty<usize>();var bool first=true;
    while(true){if(!charge(work,1)){return .err(fail(14,0,0));}if(unsafe{directory_query(root.raw,bytes::view_mut(ref mut raw) as ptr mut u8,65536,if(first){1}else{0})}==0){let code=unsafe{last_error()};if(code==18){return .ok(Entries{.names,.facts});}return .err(fail(12,code,0));}first=false;
        switch(batch(bytes::view(ref raw),ref mut names,ref mut facts,maximum,work)){case .ok(_):{}case .err(e):{return .err(e);}};
    }return .err(fail(12,0,0));
}
@effects(filesystem) pub fn Result<Metadata,Problem> observe(ref File file){var info=Metadata{.identity=0,.modified=0,.size=0,.attributes=0};if(unsafe{file_info(file.raw,ref mut info as ptr mut void)}==0){return .err(native(12,0));}return .ok(info);}
// A deletion-capable open denies delete sharing while comparing the observed
// inode/time/size. A concurrent replacement is preserved as a stale snapshot.
@effects(filesystem) pub fn Result<bool,Problem> erase_if(ref Directory root,ref u8[] leaf,Metadata expected){if(expected.identity==0 || !regular(expected)){return .ok(false);}let encoded=switch(wide(leaf,true)){case .ok(p):{p}case .err(e):{return .err(e);}};let raw=unsafe{open_file(root.raw,bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded)-2,3)};if(raw==0){let e=native(4,0);return if(missing(e)){.ok(false)}else{.err(e)};}let file=File{.raw,.staged=false};let actual=switch(observe(ref file)){case .ok(p):{p}case .err(e):{return .err(e);}};
    if(actual.identity!=expected.identity || actual.modified!=expected.modified || actual.size!=expected.size || actual.attributes!=expected.attributes){switch(close(file)){case .ok(_):{return .ok(false);}case .err(e):{return .err(e);}};}
    var problem=fail(0,0,0);if(unsafe{file_delete(file.raw)}==0){problem=native(10,0);}switch(close(file)){case .ok(_):{}case .err(e):{if(problem.stage==0){return .err(e);}problem.close_code=e.os_code;}};return if(problem.stage==0){.ok(true)}else{.err(problem)};
}
