module linux_edit_fs;
import linux_rooted_file as rooted;
import linux_directory as directory;
import project_manifest_values as manifest;
import std::bytes as bytes;

// 1 I/O, 2 allocation, 3 bound/length, 4 path/type, 5 absent,
// 6 changed metadata, 7 native protocol. Stages identify path/lookup/directory/
// open/before/read/after/file-close/directory-close (1 through 9).
pub struct Problem {u32 kind;u32 stage;u32 os_code;usize completed;u32 close_code;u32 directory_close_code;}
pub fn usize maximum(){return 8388608;}
pub fn rooted::Metadata empty_metadata(){return rooted::Metadata{.device=0,.identity=0,.size=0,.modified_seconds=0,.modified_nanos=0,.mode=0};}
fn Problem fail(u32 kind,u32 stage,u32 os_code,usize completed){return Problem{.kind,.stage,.os_code,.completed,.close_code=0,.directory_close_code=0};}
pub fn bool same(rooted::Metadata a,rooted::Metadata b){return a.device==b.device && a.identity==b.identity && a.size==b.size && a.modified_seconds==b.modified_seconds && a.modified_nanos==b.modified_nanos && a.mode==b.mode;}
fn bool valid_path(ref u8[] path){
    if(path.len==0 || path.len>4095 || !manifest::path(path)){return false;}
    var usize start=0;var usize end=0;
    while(end<=path.len){if(end==path.len || path[end]==47){if(end-start>255){return false;}start=end+1;}end+=1;}
    return true;
}
pub fn usize split(ref u8[] path){var usize i=path.len;while(i>0){i-=1;if(path[i]==47){return i+1;}}return 0;}
@effects(filesystem) fn Result<directory::Lookup,Problem> lookup(ref rooted::Directory root,ref u8[] leaf,ref mut directory::EditBudget budget){
    return switch(directory::lookup_edit(root,leaf,budget)){
        case .ok(p):{.ok(p)}
        case .err(e):{var p=fail(if(e.kind==7){3}else if(e.kind==6){7}else if(e.kind==1){4}else{1},2,e.os_code,e.completed);p.close_code=e.close_code;.err(p)}
    };
}
@effects(filesystem) fn Result<rooted::Directory,Problem> child(ref rooted::Directory root,ref u8[] leaf,ref mut directory::EditBudget budget){
    let found=lookup(root,leaf,budget)?;
    if(!found.found){return .err(fail(5,3,0,0));}if((found.info.mode&61440)!=16384){return .err(fail(4,3,0,0));}
    return switch(rooted::directory(root,leaf)){case .ok(p):{.ok(p)}case .err(p):{.err(fail(if(p.stage==8){7}else{1},3,p.os_code,0))}};
}
@effects(filesystem) fn u32 close_directory(rooted::Directory root){return switch(rooted::close_directory(root)){case .ok(_):{0}case .err(p):{p.os_code}};}
@effects(filesystem) fn Result<rooted::Directory,Problem> descend(ref rooted::Directory root,ref u8[] path,ref mut directory::EditBudget budget){
    var usize end=0;while(end<path.len && path[end]!=47){end+=1;}
    var current=child(root,path[..end],budget)?;var usize start=end+1;
    while(start<path.len){
        end=start;while(end<path.len && path[end]!=47){end+=1;}
        let next=child(ref current,path[start..end],budget);let closed=close_directory(current);
        current=switch(next){
            case .ok(p):{if(closed!=0){let later=close_directory(p);var problem=fail(1,9,closed,0);problem.directory_close_code=later;return .err(problem);}p}
            case .err(original):{var p=original;p.directory_close_code=closed;return .err(p);}
        };start=end+1;
    }return .ok(current);
}
@effects(filesystem) fn Result<bytes::Buffer,Problem> read_held(ref mut rooted::File file,rooted::Metadata expected,usize maximum_bytes){
    let before=switch(rooted::observe(file)){case .ok(p):{p}case .err(p):{return .err(fail(if(p.stage==8){7}else{1},5,p.os_code,0));}};
    if(!same(expected,before)){return .err(fail(6,5,0,0));}if(before.size>maximum_bytes){return .err(fail(3,5,0,0));}
    var output=bytes::empty();if(!bytes::resize_zeroed(ref mut output,before.size+1).is_ok){return .err(fail(2,6,0,0));}
    var usize used=0;
    while(used<=before.size){
        let usize count=if(before.size+1-used>65536){65536}else{before.size+1-used};
        let copied=switch(rooted::read(file,bytes::view_mut(ref mut output)[used..used+count])){case .ok(p):{p}case .err(p):{return .err(fail(if(p.stage==8){7}else{1},6,p.os_code,used));}};
        if(copied==0){break;}used+=copied;
    }
    if(used!=before.size){return .err(fail(3,6,0,used));}
    let after=switch(rooted::observe(file)){case .ok(p):{p}case .err(p):{return .err(fail(if(p.stage==8){7}else{1},7,p.os_code,used));}};
    if(!same(before,after)){return .err(fail(6,7,0,used));}
    discard bytes::truncate(ref mut output,used);return .ok(output);
}
@effects(filesystem) fn Result<bytes::Buffer,Problem> leaf_contents(ref rooted::Directory root,ref u8[] leaf,usize maximum_bytes,ref mut directory::EditBudget budget,ref mut rooted::Metadata observed){
    let found=lookup(root,leaf,budget)?;
    if(!found.found){return .err(fail(5,4,0,0));}if(!rooted::regular(found.info)){return .err(fail(4,4,0,0));}
    if(found.info.size>maximum_bytes){return .err(fail(3,5,0,0));}
    var file=switch(rooted::open_read(root,leaf)){case .ok(p):{p}case .err(p):{return .err(fail(if(p.stage==8){7}else{1},4,p.os_code,0));}};
    let result=read_held(ref mut file,found.info,maximum_bytes);
    let u32 closed=switch(rooted::close(file)){case .ok(_):{0}case .err(p):{p.os_code}};
    return switch(result){
        case .ok(p):{if(closed!=0){return .err(fail(1,8,closed,found.info.size));};*observed=found.info;.ok(p)}
        case .err(original):{var p=original;p.close_code=closed;.err(p)}
    };
}
@effects(filesystem) fn Result<bytes::Buffer,Problem> read_path(ref rooted::Directory root,ref u8[] path,usize maximum_bytes,ref mut directory::EditBudget budget,ref mut rooted::Metadata observed){
    let at=split(path);if(at==0){return leaf_contents(root,path,maximum_bytes,budget,observed);}
    let parent=descend(root,path[..at-1],budget)?;
    let result=leaf_contents(ref parent,path[at..],maximum_bytes,budget,observed);let closed=close_directory(parent);
    return switch(result){
        case .ok(p):{if(closed!=0){return .err(fail(1,9,closed,bytes::len(ref p)));};.ok(p)}
        case .err(original):{var p=original;p.directory_close_code=closed;.err(p)}
    };
}
// Entire input validation precedes I/O. Metadata is copied out only on success,
// after every owned file/directory has closed. Bytes move without a second copy.
@effects(filesystem) pub fn Result<bytes::Buffer,Problem> read_with_budget(ref rooted::Directory root,ref u8[] path,usize maximum_bytes,ref mut directory::EditBudget budget,ref mut rooted::Metadata observed){
    if(!valid_path(path)){return .err(fail(4,1,0,0));}if(maximum_bytes>maximum()){return .err(fail(3,1,0,0));}
    var info=empty_metadata();
    return switch(read_path(root,path,maximum_bytes,budget,ref mut info)){case .ok(p):{*observed=info;.ok(p)}case .err(p):{.err(p)}};
}
@effects(filesystem) pub fn Result<bytes::Buffer,Problem> read_observed(ref rooted::Directory root,ref u8[] path,usize maximum_bytes,ref mut rooted::Metadata observed){
    var budget=directory::EditBudget{.entries=0,.work=0};return read_with_budget(root,path,maximum_bytes,ref mut budget,observed);
}
// Compatibility port for the shared body-file and snapshot readers.
@effects(filesystem) pub fn Result<bytes::Buffer,u32> read(ref rooted::Directory root,ref u8[] path,usize maximum_bytes){
    var info=empty_metadata();return switch(read_observed(root,path,maximum_bytes,ref mut info)){case .ok(p):{.ok(p)}case .err(p):{.err(p.kind)}};
}
