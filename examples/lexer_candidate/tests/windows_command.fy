module tests::windows_command;
import windows_command as command;
import std::bytes as bytes;
import std::vec as v;

fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let copied=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(args,copied).is_ok);}
fn bytes::Buffer built(ref u8[] path,ref v::Vec<bytes::Buffer> args){return switch(command::build(path,args)){case .ok(p):{p}case .err(_):{assert(false);return built(path,args);}};}
@test fn void preserves_unicode_empty_quotes_and_backslashes_in_utf16(){
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"");add(ref mut args,b"a b");add(ref mut args,b"x\\\"y\\");add(ref mut args,b"\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a");let wide=built(b"C:\\tool dir\\fairyc.exe",ref args);
    assert(bytes::equal(bytes::view(ref wide),b"\x22\0C\0:\0\\\0t\0o\0o\0l\0 \0d\0i\0r\0\\\0f\0a\0i\0r\0y\0c\0.\0e\0x\0e\0\x22\0 \0\x22\0\x22\0 \0\x22\0a\0 \0b\0\x22\0 \0\x22\0x\0\\\0\\\0\\\0\x22\0y\0\\\0\\\0\x22\0 \0\x22\0\x5c\xd5\x00\xae\x3e\xd8\xda\xdd\x22\0\0\0"));
    let path=switch(command::path(b"\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref path),b"\x5c\xd5\x00\xae\x3e\xd8\xda\xdd\0\0"));
}
@test fn void rejects_invalid_arguments_and_admits_exact_windows_limit(){
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();assert(!command::build(b"",ref args).is_ok && !command::build(b"x\"y",ref args).is_ok && !command::path(b"x\0y").is_ok && !command::path(b"\xff").is_ok);add(ref mut args,b"x\0y");assert(!command::build(b"x",ref args).is_ok && bytes::equal(bytes::view(v::at<bytes::Buffer>(ref args,0)),b"x\0y"));assert(v::clear<bytes::Buffer>(ref mut args)==1);add(ref mut args,b"\xc0\x80");assert(!command::build(b"x",ref args).is_ok);assert(v::clear<bytes::Buffer>(ref mut args)==1);
    var bytes::Buffer long=bytes::empty();assert(bytes::resize_zeroed(ref mut long,32760).is_ok);var usize i=0;while(i<32760){let view=bytes::view_mut(ref mut long);view[i]=65;i+=1;}add(ref mut args,bytes::view(ref long));let exact=built(b"x",ref args);assert(bytes::len(ref exact)==65534);add(ref mut args,b"");assert(!command::build(b"x",ref args).is_ok && v::len<bytes::Buffer>(ref args)==2);
    assert(v::clear<bytes::Buffer>(ref mut args)==2);i=0;while(i<1023){add(ref mut args,b"");i+=1;}assert(command::build(b"x",ref args).is_ok);add(ref mut args,b"");assert(!command::build(b"x",ref args).is_ok);
}
