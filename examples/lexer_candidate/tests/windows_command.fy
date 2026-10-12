module tests::windows_command;
import windows_command as command;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import file_source as files;

fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let copied=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(args,copied).is_ok);}
fn bytes::Buffer built(ref u8[] path,ref v::Vec<bytes::Buffer> args){return switch(command::build(path,args)){case .ok(p):{p}case .err(_):{assert(false);return built(path,args);}};}
@test fn void preserves_unicode_empty_quotes_and_backslashes_in_utf16(){
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"");add(ref mut args,b"a b");add(ref mut args,b"x\\\"y\\");add(ref mut args,b"\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a");let wide=built(b"C:\\tool dir\\fairyc.exe",ref args);
    assert(bytes::equal(bytes::view(ref wide),b"\x22\0C\0:\0\\\0t\0o\0o\0l\0 \0d\0i\0r\0\\\0f\0a\0i\0r\0y\0c\0.\0e\0x\0e\0\x22\0 \0\x22\0\x22\0 \0\x22\0a\0 \0b\0\x22\0 \0x\0\\\0\\\0\\\0\x22\0y\0\\\0 \0\x5c\xd5\x00\xae\x3e\xd8\xda\xdd\0\0"));
    let path=switch(command::path(b"\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref path),b"\x5c\xd5\x00\xae\x3e\xd8\xda\xdd\0\0"));
}
@test fn void rejects_invalid_arguments_and_admits_exact_windows_limit(){
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();assert(!command::build(b"",ref args).is_ok && !command::build(b"x\"y",ref args).is_ok && !command::path(b"x\0y").is_ok && !command::path(b"\xff").is_ok);add(ref mut args,b"x\0y");assert(!command::build(b"x",ref args).is_ok && bytes::equal(bytes::view(v::at<bytes::Buffer>(ref args,0)),b"x\0y"));assert(v::clear<bytes::Buffer>(ref mut args)==1);add(ref mut args,b"\xc0\x80");assert(!command::build(b"x",ref args).is_ok);assert(v::clear<bytes::Buffer>(ref mut args)==1);
    var bytes::Buffer long=bytes::empty();assert(bytes::resize_zeroed(ref mut long,32764).is_ok);var usize i=0;while(i<32764){let view=bytes::view_mut(ref mut long);view[i]=65;i+=1;}add(ref mut args,bytes::view(ref long));let exact=built(b"x",ref args);assert(bytes::len(ref exact)==65534);add(ref mut args,b"");assert(!command::build(b"x",ref args).is_ok && v::len<bytes::Buffer>(ref args)==2);
    assert(v::clear<bytes::Buffer>(ref mut args)==2);i=0;while(i<16381){add(ref mut args,b"x");i+=1;}add(ref mut args,b"xx");let many=built(b"x",ref args);assert(bytes::len(ref many)==65534);add(ref mut args,b"x");assert(!command::build(b"x",ref args).is_ok);
}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=4);let value=(data[*at] as usize)+(data[*at+1] as usize)*256+(data[*at+2] as usize)*65536+(data[*at+3] as usize)*16777216;*at=*at+4;return value;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let size=number(data,at);let start=*at;assert(start<=data.len && size<=data.len-start);*at=start+size;return data[start..start+size];}
@test @effects(filesystem) fn void matches_go_escape_arg_command_lines_exactly(){
    var u8[4096] path=[0;4096];let size=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let ref u8[] filename=ref path;let fixture=switch(files::manifest(filename[..size])){case .ok(p):{p}case .err(_):{assert(false);return;}};let data=bytes::view(ref fixture);assert(data.len>=11 && bytes::equal(data[..7],b"FWCMD1\n"));var usize at=7;let cases=number(data,ref mut at);assert(cases>0 && cases<=128);var usize i=0;
    while(i<cases){let program=field(data,ref mut at);let count=number(data,ref mut at);assert(count<=128);var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();var usize j=0;while(j<count){add(ref mut args,field(data,ref mut at));j+=1;}let expected=field(data,ref mut at);let actual=built(program,ref args);assert(bytes::equal(bytes::view(ref actual),expected));i+=1;}assert(at==data.len);
}
