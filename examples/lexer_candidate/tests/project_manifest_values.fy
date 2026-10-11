module tests::project_manifest_values;
import project_manifest_values as values;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@test
fn void keeps_exact_manifest_names_paths_and_versions(){
    assert(values::name(b"a-1") && !values::name(b"a--1") && !values::name(b"A"));
    assert(values::uid(b"example.fairy-tools") && !values::uid(b"example") && !values::uid(b"example..tools"));
    assert(values::path(b"libs/COM0/src") && !values::path(b"libs/COM1.txt") && !values::path(b"../src") && !values::path(b"libs//src"));
    assert(values::language(b"fairyc-language/0") && !values::language(b"fairyc-language/00"));
    assert(values::version(b"1.2.3-rc.1+build.01") && values::version(b"1.2.3-0x") && values::version(b"1.2.3--"));
    assert(!values::version(b"01.2.3") && !values::version(b"1.2.3-01") && !values::version(b"1.2.3+") && !values::version(b"1.2.3+build..id"));
    assert(values::before(b"lib",b"libs") && values::before(b"a-z",b"a0") && !values::before(b"same",b"same"));
}

fn bool validate(u8 mode,ref u8[] value){
    if(mode==1){return values::name(value);}
    if(mode==2){return values::uid(value);}
    if(mode==3){return values::language(value);}
    if(mode==4){return values::version(value);}
    if(mode==5){return values::segment(value);}
    assert(mode==6);return values::path(value);
}

@test
@effects(filesystem,console)
fn void matches_go_manifest_value_reference(){
    var u8[4096] argument=[0;4096];
    let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};
    let ref u8[] path=ref argument;
    let input=switch(files::manifest(path[..length])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};
    let ref u8[] data=bytes::view(ref input);assert(data.len>=9 && bytes::equal(data[..9],b"FYMFVAL1\n"));
    var usize i=9;var usize count=0;
    while(i<data.len){
        assert(data.len-i>=4);let u8 mode=data[i];let u8 expected=data[i+1];
        let usize size=(data[i+2] as usize)+(data[i+3] as usize)*256;i+=4;
        assert(mode>=1 && mode<=6 && expected<=1 && size<=data.len-i);
        let bool actual=validate(mode,data[i..i+size]);
        if(actual!=(expected==1)){discard io::write_usize(count);discard io::write(" mode ");discard io::write_u32(mode as u32);}
        assert(actual==(expected==1));i+=size;count+=1;
    }assert(count>0);
}
