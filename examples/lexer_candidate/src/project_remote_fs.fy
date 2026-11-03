module project_remote_fs;
import windows_rooted_file as host;

import std::bytes as bytes;


// Private errors: 1 I/O, 2 allocation, 3 byte/work/entry bound,
// 4 case/link/type rejection, 5 absent. Never create on a read/miss path.
fn u8 folded(u8 c){return if(c>=65 && c<=90){c+32}else{c};}
fn bool equal_fold(ref u8[] a,ref u8[] b){if(a.len!=b.len){return false;}var usize i=0;while(i<a.len){if(folded(a[i])!=folded(b[i])){return false;}i+=1;}return true;}
@effects(filesystem) pub fn Result<usize,u32> kind(ref host::Directory parent,ref u8[] leaf){
    var usize work=0;let entries=switch(host::scan_packages(parent,ref mut work)){case .ok(value):{value}case .err(_):{return .err(1);}};var usize i=0;var usize found=0;
    while(i<host::count(ref entries)){let ref u8[] name=host::name(ref entries,i);if(equal_fold(name,leaf)){if(!bytes::equal(name,leaf) || found!=0){return .err(4);}let info=host::metadata(ref entries,i);if((info.attributes&1088)!=0){return .err(4);}found=if((info.attributes&16)!=0){1}else if(host::regular(info)){2}else{3};}i+=1;}return .ok(found);
}
@effects(filesystem) pub fn Result<host::Directory,u32> create(ref host::Directory parent,ref u8[] leaf){return switch(host::create_directory(parent,leaf)){case .ok(value):{.ok(value)}case .err(_):{.err(1)}};}
@effects(filesystem) pub fn Result<host::Directory,u32> edit(ref host::Directory parent,ref u8[] leaf){return switch(host::edit_directory(parent,leaf)){case .ok(value):{.ok(value)}case .err(_):{.err(1)}};}
@effects(filesystem) pub fn u32 publish(ref host::Directory stage,ref host::Directory parent,ref u8[] leaf){return if(host::publish_directory(stage,parent,leaf).is_ok){0}else{1};}
@effects(filesystem) pub fn u32 erase(ref host::Directory directory){return if(host::erase_directory(directory)){0}else{1};}
@effects(filesystem) fn Result<host::Directory,u32> child(ref host::Directory root,ref u8[] leaf,bool create_missing){
    if(!host::valid_leaf(leaf)){return .err(4);}let usize k=switch(kind(root,leaf)){case .ok(value):{value}case .err(e):{return .err(e);}};
    if(k==0 && create_missing){return create(root,leaf);}if(k!=1){return .err(if(k==0){5}else{4});}return switch(host::directory(root,leaf)){case .ok(value):{.ok(value)}case .err(_):{.err(1)}};
}
@effects(filesystem) pub fn Result<host::Directory,u32> descend(ref host::Directory root,ref u8[] path,bool create_missing){
    if(path.len==0){return .err(4);}var usize end=0;while(end<path.len && path[end]!=47){end+=1;}
    var current=switch(child(root,path[..end],create_missing)){case .ok(value):{value}case .err(e):{return .err(e);}};var usize start=end+1;
    while(start<path.len){end=start;while(end<path.len && path[end]!=47){end+=1;}let next=switch(child(ref current,path[start..end],create_missing)){case .ok(value):{value}case .err(e):{return .err(e);}};current=next;start=end+1;}
    return .ok(current);
}
pub fn usize split(ref u8[] path){var usize i=path.len;while(i>0){i-=1;if(path[i]==47){return i+1;}}return 0;}
@effects(filesystem) pub fn Result<bytes::Buffer,u32> read(ref host::Directory root,ref u8[] path,usize maximum){
    let usize at=split(path);if(at==0){return read_leaf(root,path,maximum);}let trail=switch(descend(root,path[..at-1],false)){case .ok(value):{value}case .err(e):{return .err(e);}};return read_leaf(ref trail,path[at..],maximum);
}
@effects(filesystem) fn Result<bytes::Buffer,u32> read_leaf(ref host::Directory current,ref u8[] leaf,usize maximum){let usize k=switch(kind(current,leaf)){case .ok(value):{value}case .err(e):{return .err(e);}};if(k!=2){return .err(if(k==0){5}else{4});}
    var file=switch(host::open_read(current,leaf)){case .ok(value):{value}case .err(_):{return .err(1);}};let usize size=switch(host::size(ref file)){case .ok(value):{value}case .err(_):{return .err(1);}};if(size>maximum){return .err(3);}var bytes::Buffer output=bytes::empty();if(!bytes::resize_zeroed(ref mut output,size+1).is_ok){return .err(2);}var usize used=0;
    while(used<=size){let usize n=if(size+1-used>65536){65536}else{size+1-used};let usize copied=switch(host::read(ref mut file,bytes::view_mut(ref mut output)[used..used+n])){case .ok(value):{value}case .err(_):{return .err(1);}};if(copied==0){break;}used+=copied;}
    if(used!=size){return .err(3);}if(!host::close(file).is_ok){return .err(1);}discard bytes::truncate(ref mut output,size);return .ok(output);
}
@effects(filesystem) pub fn u32 write(ref host::Directory root,ref u8[] path,ref u8[] data){
    let usize at=split(path);if(at==0){return write_leaf(root,path,data);}let trail=switch(descend(root,path[..at-1],true)){case .ok(value):{value}case .err(e):{return e;}};return write_leaf(ref trail,path[at..],data);
}
@effects(filesystem) fn u32 write_leaf(ref host::Directory current,ref u8[] leaf,ref u8[] data){var file=switch(host::create_new(current,leaf)){case .ok(value):{value}case .err(_):{return 1;}};
    if(!host::write_all(ref mut file,data).is_ok || !host::sync(ref mut file).is_ok){return 1;}host::keep_in_stage(ref mut file);return if(host::close(file).is_ok){0}else{1};
}
// Only call on a newly created private stage, under the exclusive fetch lock.
// Existing vendor trees never enter this deletion path. Depth/entries bound it.
@effects(filesystem) pub fn u32 clear(ref host::Directory root,usize depth,ref mut usize remaining){
    if(depth>32){return 3;}var usize work=0;let entries=switch(host::scan_packages(root,ref mut work)){case .ok(value):{value}case .err(_):{return 1;}};var usize i=0;while(i<host::count(ref entries)){if(*remaining==0){return 3;};*remaining-=1;let ref u8[] leaf=host::name(ref entries,i);let info=host::metadata(ref entries,i);if((info.attributes&1088)!=0){return 4;}
        if((info.attributes&16)!=0){let child=switch(edit(root,leaf)){case .ok(value):{value}case .err(e):{return e;}};let u32 code=clear(ref child,depth+1,remaining);if(code!=0){return code;}if(erase(ref child)!=0){return 1;}}
        else{switch(host::erase_if(root,leaf,info)){case .ok(done):{if(!done){return 1;}}case .err(_):{return 1;}};}i+=1;
    }return 0;
}
