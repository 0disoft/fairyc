module tests::project_toml_strings;
import project_toml_strings as strings;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@test
fn void decodes_tokens_keeps_cursors_and_rolls_back_errors(){
    var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,b"base").is_ok);
    let usize end=switch(strings::append(b"zz\"cat\",",2,false,ref mut output)){case .ok(n):{n}case .err(_):{assert(false);0}};
    assert(end==7 && bytes::equal(bytes::view(ref output),b"basecat"));
    assert(bytes::truncate(ref mut output,4).is_ok);
    assert(!strings::append(b"\"ab\\q\"",0,true,ref mut output).is_ok);
    assert(bytes::equal(bytes::view(ref output),b"base"));
    assert(!strings::append(b"\"ab\\U00110000\"",0,true,ref mut output).is_ok);
    assert(bytes::equal(bytes::view(ref output),b"base"));
    assert(!strings::append(b"\"ab\xff\"",0,true,ref mut output).is_ok);
    assert(bytes::equal(bytes::view(ref output),b"base"));
    assert(!strings::append(b"'''key'''",0,false,ref mut output).is_ok);
    assert(bytes::equal(bytes::view(ref output),b"base"));
    assert(strings::append(b"\"\\xFF\\e\\u0000\"",0,true,ref mut output).is_ok);
    assert(bytes::equal(bytes::view(ref output),b"base\xc3\xbf\x1b\x00"));
    assert(bytes::truncate(ref mut output,4).is_ok);
    let u32 invalid_cursor=switch(strings::append(b"''",3,true,ref mut output)){case .ok(_):{0}case .err(e):{assert(e.start==2 && e.end==2);e.code}};
    assert(invalid_cursor==1 && bytes::equal(bytes::view(ref output),b"base"));
    var bytes::Buffer large=bytes::empty();assert(bytes::resize_zeroed(ref mut large,1048575).is_ok);
    let u32 full=switch(strings::append(b"\"\\xFF\"",0,true,ref mut large)){case .ok(_):{0}case .err(e):{e.code}};
    assert(full==4 && bytes::len(ref large)==1048575 && bytes::view(ref large)[1048574]==0);
    assert(bytes::resize_zeroed(ref mut large,1048577).is_ok);
    let u32 source_limit=switch(strings::append(bytes::view(ref large),0,true,ref mut output)){case .ok(_):{0}case .err(e):{e.code}};
    assert(source_limit==4 && bytes::equal(bytes::view(ref output),b"base"));
}

@test
@effects(filesystem,console)
fn void matches_pinned_go_toml_string_reference(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};
    let ref u8[] path=ref argument;
    let input=switch(files::manifest(path[..length])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};
    let ref u8[] data=bytes::view(ref input);assert(data.len>=9 && bytes::equal(data[..9],b"FYTSTR01\n"));
    var usize i=9;var usize count=0;
    while(i<data.len){
        assert(data.len-i>=6);let bool key=data[i]==1;let bool expected=data[i+1]==1;
        assert(data[i]<=1 && data[i+1]<=1);
        let usize size=(data[i+2] as usize)+(data[i+3] as usize)*256;
        let usize cooked=(data[i+4] as usize)+(data[i+5] as usize)*256;i+=6;
        assert(size<=data.len-i && cooked<=data.len-i-size);
        var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,b"base").is_ok);
        let usize end=switch(strings::append(data[i..i+size],0,!key,ref mut output)){
            case .ok(n):{n}
            case .err(_):{assert(bytes::equal(bytes::view(ref output),b"base"));0}
        };
        let bool accepted=end!=0 && end==size;
        if(accepted!=expected){discard io::write_usize(count);discard io::write(" string mismatch\n");}
        assert(accepted==expected);
        if(accepted){assert(bytes::equal(bytes::view(ref output)[4..],data[i+size..i+size+cooked]));}
        i+=size+cooked;count+=1;
    }assert(count>0);
}
