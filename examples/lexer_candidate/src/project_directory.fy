module project_directory;
import project_manifest_values as rules;
import project_path as paths;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;

// 1 path/open, 2 invalid portable path, 3 noncanonical case, 4 reparse,
// 5 non-directory, 6 I/O, 7 allocation, 8 bound, 9 UTF-8.
pub struct Problem{u32 code;u32 os_code;usize start;usize end;}
fn Problem fail(u32 code,u32 os_code,usize start,usize end){return Problem{.code,.os_code,.start,.end};}
// These codes originate in the Windows SDK directory adapter's prefix check.
fn u32 open_code(fs::FileError error){if(error.os_code==1920){return 4;}if(error.os_code==8 || error.os_code==14){return 7;}if(error.status==6){return 8;}if(error.status==5){return 9;}if(error.os_code==267){return 5;}return 1;}
pub fn ref str reason(u32 code){if(code==3){return "noncanonical-path-case";}if(code==4){return "symlink-not-supported";}if(code==8){return "source-limit-exceeded";}if(code==7){return "allocation-failed";}return "invalid-workspace-path";}
@effects(filesystem)
fn Result<bytes::Buffer,Problem> walk(ref u8[] root,ref u8[] relative,bool source,ref mut usize work){
    if(root.len==0 || root.len>4095 || relative.len>4095){return .err(fail(8,0,0,relative.len));}if(!(if(source){paths::source_relative(relative)}else{rules::path(relative)})){return .err(fail(2,0,0,relative.len));}if(!text::validate_utf8(root).is_ok){return .err(fail(9,0,0,0));}if(!paths::charge(work,root.len+relative.len+1)){return .err(fail(8,0,0,relative.len));}
    var bytes::Buffer current=bytes::empty();if(!bytes::append(ref mut current,root).is_ok){return .err(fail(7,0,0,0));}var usize start=0;
    while(start<relative.len){var usize end=start;while(end<relative.len && relative[end]!=47){end+=1;}let ref u8[] segment=relative[start..end];
        let ref str directory=switch(text::validate_utf8(bytes::view(ref current))){case .ok(value):{value}case .err(_):{return .err(fail(9,0,start,end));}};var fs::ReadDir stream=switch(fs::read_dir_utf8(directory)){case .ok(value):{value}case .err(e):{return .err(fail(open_code(e),e.os_code,start,end));}};
        var u8[4096] name=[0;4096];var usize matches=0;var bool exact=false;var bool reparse=false;var bool is_directory=false;
        while(true){let fs::DirEntry entry=switch(fs::next_dir(ref mut stream,ref mut name)){case .ok(value):{value}case .err(e):{return .err(fail(6,e.os_code,start,end));}};if(!entry.has_entry){break;}if(entry.name_length>4096){return .err(fail(8,0,start,end));}let ref u8[] view=ref name;
            let bool equal=switch(paths::equal_fold(view[..entry.name_length],segment,work)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==4){8}else{9},0,start,end));}};
            if(equal){matches+=1;exact=bytes::equal(view[..entry.name_length],segment);reparse=entry.is_reparse;is_directory=fs::path_kind_is_directory(entry.kind);}
        }switch(fs::close_dir(stream)){case .ok(_):{}case .err(e):{return .err(fail(6,e.os_code,start,end));}};
        if(matches!=1 || !exact){return .err(fail(3,0,start,end));}if(reparse){return .err(fail(4,0,start,end));}if(!is_directory){return .err(fail(5,0,start,end));}
        let usize length=bytes::len(ref current);let bool separator=length>0 && bytes::view(ref current)[length-1]!=47 && bytes::view(ref current)[length-1]!=92;
        if(length+segment.len+if(separator){1}else{0}>4095){return .err(fail(8,0,start,end));}if(separator && !bytes::append_byte(ref mut current,47).is_ok || !bytes::append(ref mut current,segment).is_ok){return .err(fail(7,0,start,end));}start=end+1;
    }return .ok(current);
}
@effects(filesystem)
pub fn Result<bytes::Buffer,Problem> resolve(ref u8[] root,ref u8[] relative,ref mut usize work){return walk(root,relative,false,work);}
@effects(filesystem)
pub fn Result<bytes::Buffer,Problem> resolve_source(ref u8[] root,ref u8[] relative,ref mut usize work){return walk(root,relative,true,work);}
