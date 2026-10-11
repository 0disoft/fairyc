module tests::project_source_catalog;
import project_source_catalog as catalog;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer child(ref u8[] root,ref u8[] name){var bytes::Buffer path=bytes::empty();assert(bytes::append(ref mut path,root).is_ok && bytes::append_byte(ref mut path,47).is_ok && bytes::append(ref mut path,name).is_ok);return path;}
@effects(filesystem)
fn void rejected(ref u8[] root,ref u8[] name,u32 code){let path=child(root,name);var usize count=3;var usize work=0;switch(catalog::collect(bytes::view(ref path),ref mut count,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(catalog::error_code(ref e)==code && catalog::error_path(ref e).len>0 && count==3);}};}
@test
@effects(filesystem,console)
fn void collects_sorted_exact_paths_and_rejects_invalid_trees(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};
    let ref u8[] argument_path=ref argument;let fixture=switch(files::manifest(argument_path[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref u8[] root=bytes::view(ref fixture);
    var bytes::Buffer path=child(root,b"Good");var usize count=5;var usize work=0;
    let loaded=switch(catalog::collect(bytes::view(ref path),ref mut count,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(bytes::resize_zeroed(ref mut path,0).is_ok);assert(count==8 && work>0 && catalog::count(ref loaded)==3);
    assert(bytes::equal(catalog::path(ref loaded,0),b"a.fy") && bytes::equal(catalog::path(ref loaded,1),b"nested/z.fy") && bytes::equal(catalog::path(ref loaded,2),b"z.fy"));
    assert(catalog::observed_size(ref loaded,0)==10 && catalog::observed_size(ref loaded,1)==18 && catalog::observed_size(ref loaded,2)==10 && bytes::equal(catalog::path(ref loaded,3),b"") && catalog::observed_size(ref loaded,3)==0);
    rejected(root,b"Empty",5);rejected(root,b"Collision",4);rejected(root,b"Reparse",2);rejected(root,b"IgnoredReparse",2);rejected(root,b"Good/a.fy",1);
    let limited=child(root,b"Good");count=4095;work=0;
    switch(catalog::collect(bytes::view(ref limited),ref mut count,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(catalog::error_code(ref e)==6 && count==4095);}};
    count=0;work=16777216;switch(catalog::collect(bytes::view(ref limited),ref mut count,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(catalog::error_code(ref e)==6 && count==0);}};
    work=0;switch(catalog::collect(b"\xff",ref mut count,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(catalog::error_code(ref e)==8 && count==0);}};
}
