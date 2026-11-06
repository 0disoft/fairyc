module linux_remote_fs;
import linux_rooted_file as rooted;
import linux_directory as directory;
import linux_package_directory as lease;
import linux_package_file as file_policy;
import std::bytes as bytes;

// Same private error classes as project_remote_fs: I/O, allocation, bound,
// case/link/type conflict and absent. Caller owns the exclusive fetch lock.
fn u8 folded(u8 c){return if(c>=65 && c<=90){c+32}else{c};}
fn bool equal_fold(ref u8[] a,ref u8[] b){if(a.len!=b.len){return false;}var usize i=0;while(i<a.len){if(folded(a[i])!=folded(b[i])){return false;}i+=1;}return true;}
fn bool leaf(ref u8[] value){
    if(!rooted::valid_leaf(value)){return false;}var usize i=0;while(i<value.len){if(value[i]==92 || value[i]==58){return false;}i+=1;}return true;
}
@effects(filesystem) pub fn Result<usize,u32> kind(ref rooted::Directory root,ref u8[] name){
    if(!leaf(name)){return .err(4);}var usize work=0;let entries=switch(directory::scan_packages(root,ref mut work)){case .ok(p):{p}case .err(_):{return .err(1);}};var usize found=0;var usize i=0;
    while(i<directory::count(ref entries)){let written=directory::name(ref entries,i);if(equal_fold(written,name)){
        if(!bytes::equal(written,name) || found!=0){return .err(4);}let info=directory::metadata(ref entries,i);let kind=info.mode&61440;
        if(kind==40960){return .err(4);}found=if(kind==16384){1}else if(rooted::regular(info)){2}else{3};
    }i+=1;}return .ok(found);
}
@effects(filesystem) pub fn Result<lease::Directory,u32> create(ref rooted::Directory root,ref u8[] name){
    if(!leaf(name)){return .err(4);}let existing=switch(kind(root,name)){case .ok(p):{p}case .err(p):{return .err(p);}};if(existing!=0){return .err(4);}
    return switch(lease::create(root,name)){case .ok(p):{.ok(p)}case .err(_):{.err(1)}};
}
@effects(filesystem) pub fn u32 publish(ref mut lease::Directory stage,ref rooted::Directory root,ref u8[] name){
    let existing=switch(kind(root,name)){case .ok(p):{p}case .err(p):{return p;}};if(existing!=0){return 4;}
    return if(lease::publish(stage,root,name).is_ok){0}else{1};
}
@effects(filesystem) pub fn u32 erase(ref mut lease::Directory stage){return if(lease::erase(stage).is_ok){0}else{1};}
@effects(filesystem) fn Result<rooted::Directory,u32> child(ref rooted::Directory root,ref u8[] name,bool create_missing){
    let existing=switch(kind(root,name)){case .ok(p):{p}case .err(p):{return .err(p);}};
    if(existing==0 && create_missing){let created=switch(create(root,name)){case .ok(p):{p}case .err(p):{return .err(p);}};return switch(lease::duplicate(ref created)){case .ok(p):{.ok(p)}case .err(_):{.err(1)}};}
    if(existing!=1){return .err(if(existing==0){5}else{4});}
    return switch(rooted::directory(root,name)){case .ok(p):{.ok(p)}case .err(_):{.err(1)}};
}
fn bool valid_path(ref u8[] path){
    if(path.len==0 || path.len>4095 || path[0]==47 || path[path.len-1]==47){return false;}
    var usize start=0;var usize depth=0;var usize i=0;
    while(i<=path.len){if(i==path.len || path[i]==47){depth+=1;if(depth>32 || !leaf(path[start..i])){return false;}start=i+1;}i+=1;}return true;
}
@effects(filesystem) pub fn Result<rooted::Directory,u32> descend(ref rooted::Directory root,ref u8[] path,bool create_missing){
    if(!valid_path(path)){return .err(4);}var usize end=0;while(end<path.len && path[end]!=47){end+=1;}
    var current=switch(child(root,path[..end],create_missing)){case .ok(p):{p}case .err(p):{return .err(p);}};var usize start=end+1;
    while(start<path.len){end=start;while(end<path.len && path[end]!=47){end+=1;}let next=switch(child(ref current,path[start..end],create_missing)){case .ok(p):{p}case .err(p):{return .err(p);}};current=next;start=end+1;}
    return .ok(current);
}
pub fn usize split(ref u8[] path){var usize i=path.len;while(i>0){i-=1;if(path[i]==47){return i+1;}}return 0;}
fn bool same(rooted::Metadata a,rooted::Metadata b){return a.device==b.device && a.identity==b.identity && a.size==b.size && a.modified_seconds==b.modified_seconds && a.modified_nanos==b.modified_nanos && a.mode==b.mode;}
@effects(filesystem) fn Result<bytes::Buffer,u32> read_leaf(ref rooted::Directory root,ref u8[] name,usize maximum,bool observed,rooted::Metadata expected){
    let existing=switch(kind(root,name)){case .ok(p):{p}case .err(p):{return .err(p);}};if(existing!=2){return .err(if(existing==0){5}else{4});}
    var file=switch(rooted::open_read(root,name)){case .ok(p):{p}case .err(_):{return .err(1);}};
    let before=switch(file_policy::observe(ref file)){case .ok(p):{p}case .err(p):{return .err(if(p.kind==6 || p.kind==5){4}else{1});}};
    if(observed && !same(expected,before)){return .err(4);}if(before.size>maximum){return .err(3);}
    var output=bytes::empty();if(!bytes::resize_zeroed(ref mut output,before.size+1).is_ok){return .err(2);}var usize used=0;
    while(used<=before.size){let usize count=if(before.size+1-used>65536){65536}else{before.size+1-used};let copied=switch(rooted::read(ref mut file,bytes::view_mut(ref mut output)[used..used+count])){case .ok(p):{p}case .err(_):{return .err(1);}};if(copied==0){break;}used+=copied;}
    if(used!=before.size){return .err(3);}let after=switch(file_policy::observe(ref file)){case .ok(p):{p}case .err(_):{return .err(4);}};if(!same(before,after)){return .err(4);}
    if(!rooted::close(file).is_ok){return .err(1);}discard bytes::truncate(ref mut output,used);return .ok(output);
}
@effects(filesystem) pub fn Result<bytes::Buffer,u32> read(ref rooted::Directory root,ref u8[] path,usize maximum){
    if(!valid_path(path)){return .err(4);}if(maximum>16777216){return .err(3);}let usize at=split(path);
    let unused=rooted::Metadata{.device=0,.identity=0,.size=0,.modified_seconds=0,.modified_nanos=0,.mode=0};
    if(at==0){return read_leaf(root,path,maximum,false,unused);}let parent=switch(descend(root,path[..at-1],false)){case .ok(p):{p}case .err(p):{return .err(p);}};return read_leaf(ref parent,path[at..],maximum,false,unused);
}
// Reuse verification binds the held file to its earlier detached scan fact.
// A different generation, even with identical bytes, fails before any read.
// This is an observation check, not isolation against hostile namespace edits.
@effects(filesystem) pub fn Result<bytes::Buffer,u32> read_observed(ref rooted::Directory root,ref u8[] name,rooted::Metadata expected,usize maximum){
    if(!leaf(name) || !rooted::regular(expected) || expected.identity==0 || expected.modified_nanos>=1000000000){return .err(4);}
    if(maximum>16777216 || expected.size>maximum){return .err(3);}
    return read_leaf(root,name,maximum,true,expected);
}
@effects(filesystem) fn u32 write_leaf(ref rooted::Directory root,ref u8[] name,ref u8[] data){
    let existing=switch(kind(root,name)){case .ok(p):{p}case .err(p):{return p;}};if(existing!=0){return 4;}
    var file=switch(rooted::create(root)){case .ok(p):{p}case .err(_):{return 1;}};
    if(!rooted::write_all(ref mut file,data).is_ok){return 1;}
    switch(rooted::publish_new(root,ref mut file,name)){case .ok(_):{}case .err(p):{return if(rooted::collision(p)){4}else{1};}};
    return if(rooted::close(file).is_ok){0}else{1};
}
@effects(filesystem) pub fn u32 write(ref rooted::Directory root,ref u8[] path,ref u8[] data){
    if(!valid_path(path)){return 4;}if(data.len>8388608){return 3;}let usize at=split(path);
    if(at==0){return write_leaf(root,path,data);}let parent=switch(descend(root,path[..at-1],true)){case .ok(p):{p}case .err(p):{return p;}};return write_leaf(ref parent,path[at..],data);
}
// Caller supplies only its unpublished private stage; existing vendor trees
// never enter this path. Each entry and recursion edge consumes caller bounds.
@effects(filesystem) pub fn u32 clear(ref rooted::Directory root,usize depth,ref mut usize remaining){
    if(depth>32){return 3;}var usize work=0;let entries=switch(directory::scan_packages(root,ref mut work)){case .ok(p):{p}case .err(_):{return 1;}};var usize i=0;
    while(i<directory::count(ref entries)){if(*remaining==0){return 3;};*remaining-=1;let name=directory::name(ref entries,i);let info=directory::metadata(ref entries,i);
        if((info.mode&61440)==16384){var nested=switch(lease::edit(root,name)){case .ok(p):{p}case .err(_):{return 1;}};let code=clear(lease::directory(ref nested),depth+1,remaining);if(code!=0){return code;}if(!lease::erase(ref mut nested).is_ok){return 1;}}
        else{if(!rooted::regular(info)){return 4;}switch(file_policy::erase_if(root,name,info)){case .ok(done):{if(!done){return 1;}}case .err(_):{return 1;}};}i+=1;
    }return 0;
}
