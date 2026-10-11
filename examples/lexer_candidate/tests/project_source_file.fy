module tests::project_source_file;
import project_source_file as source;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::text as text;
import std::fs as fs;
@effects(filesystem)
fn void rejected(ref u8[] root,ref u8[] path,usize remaining,u32 code){var usize work=0;switch(source::read(root,path,remaining,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==code && e.start<=e.end && e.end<=path.len);}};}
@test
@effects(filesystem,console)
fn void retains_exact_bytes_and_checks_file_boundaries(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};
    let ref u8[] argument_path=ref argument;let fixture=switch(files::manifest(argument_path[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref u8[] root=bytes::view(ref fixture);var usize work=0;
    switch(source::read(root,b"Good/a.fy",10,ref mut work)){case .ok(value):{assert(bytes::equal(bytes::view(ref value),b"module a;\n") && work>0);}case .err(_):{assert(false);}};
    work=0;switch(source::read(root,b"Reads/binary.fy",3,ref mut work)){case .ok(value):{assert(bytes::equal(bytes::view(ref value),b"\x00\xff\r"));}case .err(_):{assert(false);}};
    work=0;switch(source::read(root,b"Reads/large.fy",65537,ref mut work)){case .ok(value):{assert(bytes::len(ref value)==65537 && bytes::view(ref value)[0]==97 && bytes::view(ref value)[65536]==122);}case .err(_):{assert(false);}};
    work=0;switch(source::read(root,b"Reads/maximum.fy",8388608,ref mut work)){case .ok(value):{assert(bytes::len(ref value)==8388608 && bytes::view(ref value)[0]==0 && bytes::view(ref value)[8388607]==0);}case .err(_):{assert(false);}};
    work=0;switch(source::read(root,b"Reads/empty.fy",0,ref mut work)){case .ok(value):{assert(bytes::len(ref value)==0);}case .err(_):{assert(false);}};
    rejected(root,b"Reads/too-large.fy",67108864,3);rejected(root,b"Good/a.fy",9,4);rejected(root,b"Good/A.fy",10,2);rejected(root,b"Good/missing.fy",10,2);rejected(root,b"Good/nested",10,9);
    rejected(root,b"Reparse/linked",10,8);rejected(root,b"Reparse/linked/a.fy",10,8);rejected(root,b"Good/../a.fy",10,2);rejected(b"\xff",b"Good/a.fy",10,7);
    work=16777216;switch(source::read(root,b"Good/a.fy",10,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==5);}};
    // Discriminates the mutable six-field C output bridge: two different files
    // must not accidentally compare equal because an output stayed zeroed.
    var bytes::Buffer first=bytes::empty();var bytes::Buffer second=bytes::empty();assert(bytes::append(ref mut first,root).is_ok && bytes::append(ref mut first,b"/Good/a.fy").is_ok && bytes::append(ref mut second,root).is_ok && bytes::append(ref mut second,b"/Good/z.fy").is_ok);
    let ref str first_path=switch(text::validate_utf8(bytes::view(ref first))){case .ok(value):{value}case .err(_):{assert(false);return;}};let ref str second_path=switch(text::validate_utf8(bytes::view(ref second))){case .ok(value):{value}case .err(_):{assert(false);return;}};
    let fs::File a=switch(fs::open_read_utf8(first_path)){case .ok(value):{value}case .err(_):{assert(false);return;}};let fs::File b=switch(fs::open_read_utf8(second_path)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    let fs::FileId a_id=switch(fs::file_id(ref a)){case .ok(value):{value}case .err(_):{assert(false);return;}};let fs::FileId repeated=switch(fs::file_id(ref a)){case .ok(value):{value}case .err(_):{assert(false);return;}};let fs::FileId b_id=switch(fs::file_id(ref b)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(fs::file_id_equal(ref a_id,ref repeated) && !fs::file_id_equal(ref a_id,ref b_id));assert(fs::close(a).is_ok);assert(fs::close(b).is_ok);
}
