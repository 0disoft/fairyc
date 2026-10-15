module project_source_file;
import project_directory as directory;
import project_path as paths;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;

// 1 I/O, 2 path syntax/case, 3 per-file byte bound, 4 cumulative byte bound,
// 5 work/path bound, 6 allocation, 7 UTF-8, 8 reparse, 9 nonregular,
// 10 changed opened identity. Offsets refer to the decoded relative path.
pub struct Error{u32 code;u32 os_code;usize start;usize end;}
fn Error fail(u32 code,u32 os_code,usize start,usize end){return Error{.code,.os_code,.start,.end};}
fn u32 directory_code(u32 code){return if(code==3 || code==2){2}else if(code==4){8}else if(code==7){6}else if(code==8){5}else if(code==9){7}else if(code==5){9}else{1};}
@effects(filesystem)
fn Result<bytes::Buffer,Error> observe(ref u8[] root,ref u8[] relative,ref mut usize work){
    if(root.len==0 || root.len>4095 || relative.len>4095){return .err(fail(5,0,0,relative.len));}if(!text::validate_utf8(root).is_ok){return .err(fail(7,0,0,0));}
    if(!paths::source_relative(relative)){return .err(fail(2,0,0,relative.len));}var usize leaf=0;var usize index=0;while(index<relative.len){if(relative[index]==47){leaf=index+1;}index+=1;}
    let ref str expected=switch(text::validate_utf8(relative[leaf..])){case .ok(value):{value}case .err(_):{return .err(fail(7,0,leaf,relative.len));}};
    var bytes::Buffer parent=if(leaf>0){switch(directory::resolve_source(root,relative[..leaf-1],work)){case .ok(value):{value}case .err(e):{return .err(fail(directory_code(e.code),e.os_code,e.start,e.end));}}}else{var bytes::Buffer copy=bytes::empty();if(!bytes::append(ref mut copy,root).is_ok){return .err(fail(6,0,0,leaf));}copy};
    if(bytes::len(ref parent)==0 || bytes::len(ref parent)+relative.len-leaf+1>4095 || !paths::charge(work,bytes::len(ref parent)+relative.len-leaf+1)){return .err(fail(5,0,leaf,relative.len));}
    let ref str path=switch(text::validate_utf8(bytes::view(ref parent))){case .ok(value):{value}case .err(_):{return .err(fail(7,0,0,leaf));}};
    var fs::ReadDir stream=switch(fs::read_dir_utf8(path)){case .ok(value):{value}case .err(e):{return .err(fail(if(e.os_code==1920){8}else{1},e.os_code,0,leaf));}};
    var u8[4096] name=[0;4096];var usize matches=0;var bool exact=false;var bool reparse=false;var bool regular=false;
    while(true){let fs::DirEntry entry=switch(fs::next_dir(ref mut stream,ref mut name)){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,leaf,relative.len));}};if(!entry.has_entry){break;}if(entry.name_length>4096){return .err(fail(5,0,leaf,relative.len));}let ref u8[] view=ref name;
        let ref str actual=switch(text::validate_utf8(view[..entry.name_length])){case .ok(value):{value}case .err(_):{return .err(fail(7,0,leaf,relative.len));}};
        let bool equal=switch(paths::equal_fold_text(actual,expected,work)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==4){5}else{7},0,leaf,relative.len));}};
        if(equal){matches+=1;exact=bytes::equal(view[..entry.name_length],relative[leaf..]);reparse=entry.is_reparse;regular=fs::path_kind_is_file(entry.kind);}
    }
    switch(fs::close_dir(stream)){case .ok(_):{}case .err(e):{return .err(fail(1,e.os_code,leaf,relative.len));}};
    if(matches!=1 || !exact){return .err(fail(2,0,leaf,relative.len));}if(reparse){return .err(fail(8,0,leaf,relative.len));}if(!regular){return .err(fail(9,0,leaf,relative.len));}
    let usize length=bytes::len(ref parent);let bool separator=length>0 && bytes::view(ref parent)[length-1]!=47 && bytes::view(ref parent)[length-1]!=92;
    if(separator && !bytes::append_byte(ref mut parent,47).is_ok || !bytes::append(ref mut parent,relative[leaf..]).is_ok){return .err(fail(6,0,leaf,relative.len));}return .ok(parent);
}
@effects(filesystem)
pub fn Result<bytes::Buffer,Error> read(ref u8[] root,ref u8[] relative,usize remaining,ref mut usize work){
    let bytes::Buffer absolute=switch(observe(root,relative,work)){case .ok(value):{value}case .err(e):{return .err(e);}};
    let ref str path=switch(text::validate_utf8(bytes::view(ref absolute))){case .ok(value):{value}case .err(_):{return .err(fail(7,0,0,relative.len));}};
    var fs::File file=switch(fs::open_read_utf8(path)){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};
    let fs::FileKind kind=switch(fs::kind(ref file)){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};if(!fs::kind_is_disk(kind)){return .err(fail(9,0,0,relative.len));}
    let fs::FileId identity=switch(fs::file_id(ref file)){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};
    let usize size=switch(fs::size(ref file)){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};
    if(size>8388608){return .err(fail(3,0,0,relative.len));}if(size>remaining){return .err(fail(4,0,0,relative.len));}
    let bytes::Buffer repeated=switch(observe(root,relative,work)){case .ok(value):{value}case .err(e):{return .err(e);}};if(!bytes::equal(bytes::view(ref absolute),bytes::view(ref repeated))){return .err(fail(10,0,0,relative.len));}
    let fs::File probe=switch(fs::open_read_utf8(path)){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};
    let fs::FileId after=switch(fs::file_id(ref probe)){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};if(!fs::file_id_equal(ref identity,ref after)){return .err(fail(10,0,0,relative.len));}
    switch(fs::close(probe)){case .ok(_):{}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};
    var bytes::Buffer input=bytes::empty();var u8[4096] chunk=[0;4096];let usize maximum=if(remaining<8388608){remaining}else{8388608};
    while(true){if(!paths::charge(work,1)){return .err(fail(5,0,0,relative.len));}let usize used=bytes::len(ref input);let usize capacity=if(maximum-used<4096){maximum-used+1}else{4096};let ref mut u8[] writable=ref mut chunk;
        let usize count=switch(fs::read(ref mut file,writable[..capacity])){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};
        if(count>maximum-used){return .err(fail(if(maximum==8388608){3}else{4},0,0,relative.len));}if(count==0){break;}let ref u8[] view=ref chunk;if(!bytes::append(ref mut input,view[..count]).is_ok){return .err(fail(6,0,0,relative.len));}
    }
    switch(fs::close(file)){case .ok(_):{}case .err(e):{return .err(fail(1,e.os_code,0,relative.len));}};return .ok(input);
}
