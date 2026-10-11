module tests::project_source_root;
import project_source_root as roots;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer document(ref u8[] source){
    var bytes::Buffer result=bytes::empty();
    assert(bytes::append(ref mut result,b"schema='fairyc.manifest/v1'\n[package]\nuid='example.directory'\nname='directory'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource=").is_ok);
    assert(bytes::append(ref mut result,source).is_ok);return result;
}
@effects(filesystem)
fn void rejected(ref u8[] root,ref u8[] source,u32 reason,usize start,usize end){
    let input=document(source);var usize work=0;
    switch(roots::load(root,bytes::view(ref input),ref mut work)){
        case .ok(_):{assert(false);}
        case .err(e):{assert(e.phase==2 && e.reason==reason && e.field==7 && e.start==start && e.end==end);}
    };
}
@test
@effects(filesystem,console)
fn void binds_decoded_source_to_owned_directory(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};
    let ref u8[] argument_path=ref argument;let fixture=switch(files::manifest(argument_path[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};
    let ref u8[] root=bytes::view(ref fixture);
    // The filesystem uses S, while the TOML source deliberately spells it as an escape.
    var bytes::Buffer input=document(b"\"\\u0053ource/Nested\"\n");var usize work=0;
    let loaded=switch(roots::load(root,bytes::view(ref input),ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(bytes::resize_zeroed(ref mut input,0).is_ok);
    var bytes::Buffer expected=bytes::empty();assert(bytes::append(ref mut expected,root).is_ok && bytes::append(ref mut expected,b"/Source/Nested").is_ok);
    assert(bytes::equal(roots::path(ref loaded),bytes::view(ref expected)));
    assert(bytes::equal(roots::package_value(ref loaded,b"source"),b"Source/Nested") && bytes::equal(roots::package_value(ref loaded,b"uid"),b"example.directory") && work>0);
    rejected(root,b"'source'",3,0,6);rejected(root,b"'Source/nested'",3,7,13);rejected(root,b"'Missing'",3,0,7);
    rejected(root,b"'ordinary.txt'",5,0,12);rejected(root,b"'k'",3,0,1);rejected(root,b"'Link'",4,0,4);
}
@test
@effects(filesystem)
fn void rejects_manifest_before_directory_and_preserves_error_phase(){
    var usize work=0;
    switch(roots::load(b"",b"",ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==1 && e.reason==3 && e.field==1 && e.start==0 && e.end==0 && e.os_code==0 && work==0);}};
    let invalid=document(b"'Source/../Nested'");switch(roots::load(b"",bytes::view(ref invalid),ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==1 && e.reason==8 && e.field==7 && e.start<e.end && e.end<=bytes::len(ref invalid) && work==0);}};
    let valid=document(b"'Source'");work=16777216;
    switch(roots::load(b"root",bytes::view(ref valid),ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==2 && e.reason==8 && e.field==7 && e.start==0 && e.end==6);}};
}
