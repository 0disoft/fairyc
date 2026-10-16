module project_source_catalog;
import project_path as paths;
import project_manifest_values as rules;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;
import std::fs as fs;

struct Span{usize start;usize end;}
struct File{usize start;usize end;usize lower_start;usize lower_end;usize size;}
pub resource Catalog{bytes::Buffer names;v::Vec<File> files;v::Vec<usize> order;cleanup{}}
// 1 directory I/O, 2 reparse, 3 nonregular .fy, 4 lower-case collision,
// 5 empty source root, 6 work/path/file bound, 7 allocation, 8 UTF-8.
pub resource Error{u32 code;u32 os_code;bytes::Buffer path;cleanup{}}
pub fn u32 error_code(ref Error error){return error.code;}
pub fn u32 error_os_code(ref Error error){return error.os_code;}
pub fn ref u8[] error_path(ref Error error){return bytes::view(ref error.path);}
fn Error fail(u32 code,u32 os_code,ref u8[] path){var bytes::Buffer copy=bytes::empty();if(!bytes::append(ref mut copy,path).is_ok){return Error{.code=7,.os_code=0,.path=copy};}return Error{.code,.os_code,.path=copy};}
pub fn usize count(ref Catalog catalog){return v::len<File>(ref catalog.files);}
pub fn ref u8[] path(ref Catalog catalog,usize index){if(index>=count(catalog)){return bytes::view(ref catalog.names)[..0];}let File file=*v::at<File>(ref catalog.files,*v::at<usize>(ref catalog.order,index));return bytes::view(ref catalog.names)[file.start..file.end];}
pub fn usize observed_size(ref Catalog catalog,usize index){return if(index<count(catalog)){v::at<File>(ref catalog.files,*v::at<usize>(ref catalog.order,index)).size}else{0};}
fn ref u8[] key(ref bytes::Buffer names,ref v::Vec<File> files,usize index,bool lower){let File file=*v::at<File>(files,index);return if(lower){bytes::view(names)[file.lower_start..file.lower_end]}else{bytes::view(names)[file.start..file.end]};}
fn void put(ref mut v::Vec<usize> values,usize index,usize value){let ref mut usize slot=v::at_mut<usize>(values,index);*slot=value;}
fn u32 order(ref bytes::Buffer names,ref v::Vec<File> files,bool lower,ref mut v::Vec<usize> output,ref mut usize work){
    let usize count=v::len<File>(files);var v::Vec<usize> scratch=v::empty<usize>();var usize index=0;
    while(index<count){if(!v::push<usize>(output,index).is_ok || !v::push<usize>(ref mut scratch,0).is_ok){return 7;}index+=1;}
    var usize width=1;while(width<count){var usize start=0;while(start<count){let usize middle=if(width<count-start){start+width}else{count};let usize end=if(width<count-middle){middle+width}else{count};var usize left=start;var usize right=middle;var usize out=start;
        while(out<end){var bool take_left=right==end;if(left<middle && right<end){let ref u8[] a=key(names,files,*v::at<usize>(output,left),lower);let ref u8[] b=key(names,files,*v::at<usize>(output,right),lower);if(!paths::charge(work,a.len+b.len+1)){return 6;}take_left=rules::before(a,b);}
            put(ref mut scratch,out,if(take_left){*v::at<usize>(output,left)}else{*v::at<usize>(output,right)});if(take_left){left+=1;}else{right+=1;}out+=1;
        }start=end;
    }index=0;while(index<count){put(output,index,*v::at<usize>(ref scratch,index));index+=1;}width*=2;}return 0;
}
fn bool append_path(ref mut bytes::Buffer output,ref u8[] base,ref u8[] name){return bytes::append(output,base).is_ok && (base.len==0 || bytes::append_byte(output,47).is_ok) && bytes::append(output,name).is_ok;}
fn bool source_name(ref u8[] name){return name.len>=3 && bytes::equal(name[name.len-3..],b".fy");}

// The file counter is cumulative across packages and commits only on success.
// Sizes are enumeration observations; later byte reads must recheck their budget.
@effects(filesystem)
pub fn Result<Catalog,Error> collect(ref u8[] root,ref mut usize files,ref mut usize work){return collect_optional(root,false,files,work);}
@effects(filesystem)
pub fn Result<Catalog,Error> collect_optional(ref u8[] root,bool optional,ref mut usize files,ref mut usize work){
    if(root.len==0 || root.len>4095 || *files>4096){return .err(fail(6,0,root));}if(!text::validate_utf8(root).is_ok){return .err(fail(8,0,root));}
    var bytes::Buffer names=bytes::empty();var v::Vec<File> collected=v::empty<File>();
    var v::Vec<Span> directories=v::empty<Span>();if(!v::push<Span>(ref mut directories,Span{.start=0,.end=0}).is_ok){return .err(fail(7,0,root));}var usize directory=0;
    while(directory<v::len<Span>(ref directories)){
        let Span prefix=*v::at<Span>(ref directories,directory);var bytes::Buffer absolute=bytes::empty();if(!bytes::append(ref mut absolute,root).is_ok){return .err(fail(7,0,root));}
        if(prefix.end>prefix.start && (!bytes::append_byte(ref mut absolute,47).is_ok || !bytes::append(ref mut absolute,bytes::view(ref names)[prefix.start..prefix.end]).is_ok)){return .err(fail(7,0,root));}
        if(bytes::len(ref absolute)>4095 || !paths::charge(work,bytes::len(ref absolute)+32)){return .err(fail(6,0,bytes::view(ref absolute)));}
        let ref str directory_path=switch(text::validate_utf8(bytes::view(ref absolute))){case .ok(value):{value}case .err(_):{return .err(fail(8,0,bytes::view(ref absolute)));}};
        var fs::ReadDir stream=switch(fs::read_dir_utf8(directory_path)){case .ok(value):{value}case .err(e):{return .err(fail(if(e.os_code==1920){2}else{1},e.os_code,bytes::view(ref absolute)));}};
        var u8[4096] name=[0;4096];
        while(true){
            let fs::DirEntry entry=switch(fs::next_dir(ref mut stream,ref mut name)){case .ok(value):{value}case .err(e):{return .err(fail(1,e.os_code,bytes::view(ref absolute)));}};if(!entry.has_entry){break;}
            if(entry.name_length==0 || entry.name_length>4096 || !paths::charge(work,entry.name_length+32)){return .err(fail(6,0,bytes::view(ref absolute)));}
            let ref u8[] name_bytes=ref name;let ref u8[] spelling=name_bytes[..entry.name_length];
            var bytes::Buffer relative=bytes::empty();if(!append_path(ref mut relative,bytes::view(ref names)[prefix.start..prefix.end],spelling)){return .err(fail(7,0,bytes::view(ref absolute)));}
            if(bytes::len(ref relative)+root.len+1>4095){return .err(fail(6,0,bytes::view(ref relative)));}
            if(!text::validate_utf8(bytes::view(ref relative)).is_ok){return .err(fail(8,0,bytes::view(ref relative)));}
            if(entry.is_reparse){return .err(fail(2,0,bytes::view(ref relative)));}
            let bool is_directory=fs::path_kind_is_directory(entry.kind);if(!is_directory && !source_name(spelling)){continue;}
            if(!is_directory && !fs::path_kind_is_file(entry.kind)){return .err(fail(3,0,bytes::view(ref relative)));}
            let usize start=bytes::len(ref names);if(!paths::charge(work,bytes::len(ref relative)+1)){return .err(fail(6,0,bytes::view(ref relative)));}if(!bytes::append(ref mut names,bytes::view(ref relative)).is_ok){return .err(fail(7,0,bytes::view(ref relative)));}let usize end=bytes::len(ref names);
            if(is_directory){if(!v::push<Span>(ref mut directories,Span{.start,.end}).is_ok){return .err(fail(7,0,bytes::view(ref relative)));}continue;}
            if(v::len<File>(ref collected)>=4096-*files){return .err(fail(6,0,bytes::view(ref relative)));}
            let usize lower_start=bytes::len(ref names);switch(paths::append_lower(ref mut names,bytes::view(ref relative),work)){case .ok(_):{}case .err(code):{return .err(fail(if(code==3){7}else{if(code==4){6}else{8}},0,bytes::view(ref relative)));}};
            if(!v::push<File>(ref mut collected,File{.start,.end,.lower_start,.lower_end=bytes::len(ref names),.size=entry.size}).is_ok){return .err(fail(7,0,bytes::view(ref relative)));}
        }
        switch(fs::close_dir(stream)){case .ok(_):{}case .err(e):{return .err(fail(1,e.os_code,bytes::view(ref absolute)));}};directory+=1;
    }
    if(!optional && v::len<File>(ref collected)==0){return .err(fail(5,0,root));}
    var v::Vec<usize> folded=v::empty<usize>();let u32 folded_code=order(ref names,ref collected,true,ref mut folded,work);if(folded_code!=0){return .err(fail(folded_code,0,root));}
    var usize index=1;while(index<v::len<File>(ref collected)){let usize before=*v::at<usize>(ref folded,index-1);let usize after=*v::at<usize>(ref folded,index);let ref u8[] a=key(ref names,ref collected,before,true);let ref u8[] b=key(ref names,ref collected,after,true);if(!paths::charge(work,a.len+b.len+1)){return .err(fail(6,0,root));}if(bytes::equal(a,b)){return .err(fail(4,0,key(ref names,ref collected,after,false)));}index+=1;}
    var v::Vec<usize> sorted=v::empty<usize>();let u32 sorted_code=order(ref names,ref collected,false,ref mut sorted,work);if(sorted_code!=0){return .err(fail(sorted_code,0,root));};*files+=v::len<File>(ref collected);return .ok(Catalog{.names,.files=collected,.order=sorted});
}
