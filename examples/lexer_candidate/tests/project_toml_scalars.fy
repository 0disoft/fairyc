module tests::project_toml_scalars;
import project_toml_scalars as atoms;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@test
fn void bounds_integer_float_calendar_and_token_consumption(){
    assert(atoms::kind(b"9223372036854775807")==6 && atoms::kind(b"-9223372036854775808")==6);
    assert(atoms::kind(b"9223372036854775808")==0 && atoms::kind(b"-9223372036854775809")==0);
    assert(atoms::kind(b"0x7fffffffffffffff")==6 && atoms::kind(b"0x8000000000000000")==0 && atoms::kind(b"-0xff")==0);
    assert(atoms::kind(b"1.7976931348623157e308")==7 && atoms::kind(b"1.7976931348623159e308")==0);
    assert(atoms::kind(b"0e99999999999999999999")==7 && atoms::kind(b"1e-99999999999999999999")==7);
    assert(atoms::kind(b"2000-02-29")==8 && atoms::kind(b"1900-02-29")==0 && atoms::kind(b"2000-02-30")==0);
    assert(atoms::kind(b"07:32")==8 && atoms::kind(b"24:00")==0 && atoms::kind(b"07:32:60")==0);
    let atoms::Token number=atoms::read(b"--1_000,",2);assert(number.kind==6 && number.end==7);
    let atoms::Token date=atoms::read(b"zz2000-02-29 07:32:12.123Z   #x",2);assert(date.kind==8 && bytes::equal(b"zz2000-02-29 07:32:12.123Z   #x"[2..date.end],b"2000-02-29 07:32:12.123Z"));
    assert(atoms::read(b"1",2).kind==0 && atoms::read(b"1",2).end==1);
}

@test
@effects(filesystem,console)
fn void matches_pinned_go_scalar_kinds(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;
    let input=switch(files::manifest(path[..length])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};let ref u8[] data=bytes::view(ref input);assert(data.len>=9 && bytes::equal(data[..9],b"FYATOM01\n"));var usize at=9;var usize count=0;
    while(at<data.len){assert(data.len-at>=3);let usize length=(data[at] as usize)+(data[at+1] as usize)*256;let u32 expected=data[at+2] as u32;at+=3;assert(length<=data.len-at);
        let u32 actual=atoms::kind(data[at..at+length]);if(actual!=expected){discard io::write_usize(count);discard io::write(" atom mismatch ");discard io::write_u32(actual);discard io::write("\n");}assert(actual==expected);at+=length;count+=1;
    }assert(count>0);
}
