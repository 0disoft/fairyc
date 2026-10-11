module tests::project_manifest;
import project_manifest as manifest;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

struct Cursor{usize offset;}
fn usize number(ref u8[] data,ref mut Cursor cursor){assert(data.len-cursor.offset>=2);let usize n=(data[cursor.offset] as usize)+(data[cursor.offset+1] as usize)*256;cursor.offset+=2;return n;}
fn ref u8[] string(ref u8[] data,ref mut Cursor cursor){let usize size=number(data,cursor);let usize start=cursor.offset;assert(size<=data.len-start);cursor.offset+=size;return data[start..start+size];}
fn void inspect(ref manifest::Manifest value,ref u8[] data,ref mut Cursor cursor){
    assert(bytes::equal(manifest::package_value(value,b"uid"),string(data,cursor)));
    assert(bytes::equal(manifest::package_value(value,b"name"),string(data,cursor)));
    assert(bytes::equal(manifest::package_value(value,b"version"),string(data,cursor)));
    assert(bytes::equal(manifest::package_value(value,b"language"),string(data,cursor)));
    assert(bytes::equal(manifest::package_value(value,b"source"),string(data,cursor)));
    assert(cursor.offset<data.len && data[cursor.offset]<=1);assert(manifest::workspace_present(value)==(data[cursor.offset]==1));cursor.offset+=1;
    let usize members=number(data,cursor);var usize member=manifest::member_first(value);var usize i=0;
    while(i<members){assert(member!=0 && bytes::equal(manifest::value(value,member),string(data,cursor)));member=manifest::next_value(value,member);i+=1;}assert(member==0);
    let usize dependencies=number(data,cursor);assert(manifest::dependency_count(value)==dependencies);i=0;
    while(i<dependencies){assert(bytes::equal(manifest::dependency_name(value,i),string(data,cursor)));assert(bytes::equal(manifest::dependency_value(value,i,b"path"),string(data,cursor)));assert(bytes::equal(manifest::dependency_value(value,i,b"version"),string(data,cursor)));
        let usize features=number(data,cursor);var usize feature=manifest::feature_first(value,i);var usize j=0;
        while(j<features){assert(feature!=0 && bytes::equal(manifest::value(value,feature),string(data,cursor)));feature=manifest::next_value(value,feature);j+=1;}assert(feature==0);i+=1;
    }
}

@test
fn void keeps_owned_manifest_values_and_missing_workspace(){
    let ref u8[] source=b"schema='fairyc.manifest/v1'\n[package]\nuid='example.app'\nname='example-app'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n";
    switch(manifest::parse(source)){case .ok(value):{assert(bytes::equal(manifest::package_value(ref value,b"uid"),b"example.app") && !manifest::workspace_present(ref value) && manifest::member_first(ref value)==0 && manifest::dependency_count(ref value)==0);assert(bytes::equal(manifest::dependency_name(ref value,1),b"") && manifest::feature_first(ref value,1)==0);}case .err(_):{assert(false);}};
    switch(manifest::parse(b"")){case .ok(_):{assert(false);}case .err(e):{assert(e.reason==3 && e.field==1 && bytes::equal(manifest::field(e.field),b"schema") && e.start==0 && e.end==0);}};
}

@test
@effects(filesystem,console)
fn void matches_go_manifest_documents(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;
    let input=switch(files::manifest(path[..length])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};let ref u8[] data=bytes::view(ref input);
    assert(data.len>=9 && bytes::equal(data[..9],b"FYMNF001\n"));var Cursor cursor=Cursor{.offset=9};var usize count=0;
    while(cursor.offset<data.len){let usize size=number(data,ref mut cursor);assert(data.len-cursor.offset>=2);let bool expected=data[cursor.offset]==1;let u32 reason=data[cursor.offset+1] as u32;assert(data[cursor.offset]<=1);cursor.offset+=2;
        assert(size<=data.len-cursor.offset);let usize start=cursor.offset;cursor.offset+=size;let ref u8[] source=data[start..start+size];
        switch(manifest::parse(source)){case .ok(value):{if(!expected){discard io::write_usize(count);discard io::write(" accepted manifest mismatch\n");}assert(expected);inspect(ref value,data,ref mut cursor);}case .err(e):{if(expected || e.reason!=reason){discard io::write_usize(count);discard io::write(" manifest reason mismatch ");discard io::write_u32(e.reason);discard io::write("\n");}assert(!expected && e.reason==reason && e.start<=e.end && e.end<=source.len);}};
        count+=1;
    }assert(count>0);
}
