module tests::project_directory;
import project_directory as directory;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
@effects(filesystem)
fn void rejected(ref u8[] root,ref u8[] path,u32 code,usize start,usize end){var usize work=0;switch(directory::resolve(root,path,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==code && e.start==start && e.end==end);}};}
@test
@effects(filesystem,console)
fn void resolves_exact_case_and_rejects_observed_path_violations(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;let input=switch(files::manifest(path[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref u8[] root=bytes::view(ref input);var usize work=0;
    switch(directory::resolve(root,b"Source/Nested",ref mut work)){case .ok(result):{var bytes::Buffer expected=bytes::empty();assert(bytes::append(ref mut expected,root).is_ok && bytes::append(ref mut expected,b"/Source/Nested").is_ok);assert(bytes::equal(bytes::view(ref result),bytes::view(ref expected)) && work>0);}case .err(_):{assert(false);}};
    rejected(root,b"source",3,0,6);rejected(root,b"Source/nested",3,7,13);rejected(root,b"Missing",3,0,7);rejected(root,b"ordinary.txt",5,0,12);rejected(root,b"k",3,0,1);rejected(root,b"Link",4,0,4);rejected(root,b"Source/../Nested",2,0,16);rejected(root,b"Source\\Nested",2,0,13);
    var bytes::Buffer link=bytes::empty();assert(bytes::append(ref mut link,root).is_ok && bytes::append(ref mut link,b"/Link").is_ok);rejected(bytes::view(ref link),b"Nested",4,0,6);
    rejected(b"\xff",b"Source",9,0,0);work=16777216;switch(directory::resolve(root,b"Source",ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==8);}};
}
