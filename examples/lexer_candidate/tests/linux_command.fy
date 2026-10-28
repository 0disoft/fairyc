module tests::linux_command;
import linux_command as command;
import std::bytes as bytes;
import std::vec as v;

fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value) {
    let copied=switch(bytes::copy(value)) { case .ok(p): { p } case .err(_): { assert(false);return; } };
    assert(v::push<bytes::Buffer>(args,copied).is_ok);
}
fn command::Arguments built(ref u8[] program,ref v::Vec<bytes::Buffer> args) {
    return switch(command::build(program,args)) {
        case .ok(p): { p } case .err(_): { assert(false);return built(program,args); }
    };
}
fn void rejected(ref u8[] program,ref v::Vec<bytes::Buffer> args,u32 kind,usize argument,usize offset) {
    switch(command::build(program,args)) {
        case .ok(_): { assert(false); }
        case .err(e): { assert(e.kind==kind && e.argument==argument && e.offset==offset); }
    };
}

@test fn void preserves_exact_bytes_empty_arguments_and_owned_storage() {
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();
    add(ref mut args,b""); add(ref mut args,b"a b"); add(ref mut args,b"x\\\"y\\");
    add(ref mut args,b"한 😀"); add(ref mut args,b"line\n;$HOME*"); add(ref mut args,b"");
    let packed=built(b"./tool '\"\\name",ref args);
    assert(command::count(ref packed)==7 && v::clear<bytes::Buffer>(ref mut args)==6);
    assert(bytes::equal(command::view(ref packed),b"./tool '\"\\name\0\0a b\0x\\\"y\\\0한 😀\0line\n;$HOME*\0\0"));
    let path=switch(command::path(b"relative path/한")) { case .ok(p): { p } case .err(_): { assert(false);return; } };
    assert(bytes::equal(bytes::view(ref path),b"relative path/한\0"));
    let alone=built(b"/bin/tool",ref args);
    assert(command::count(ref alone)==1 && bytes::equal(command::view(ref alone),b"/bin/tool\0"));
}

@test fn void rejects_invalid_paths_and_arguments_without_changing_inputs() {
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();
    rejected(b"",ref args,1,0,0); rejected(b"x\0y",ref args,1,0,1);
    rejected(b"\xff",ref args,1,0,0); rejected(b"\xe2\x82",ref args,1,0,0);
    assert(!command::path(b"").is_ok && !command::path(b"x\0y").is_ok && !command::path(b"\xc0\x80").is_ok);
    add(ref mut args,b"ok"); add(ref mut args,b"x\0y");
    rejected(b"x",ref args,1,2,1);
    assert(v::len<bytes::Buffer>(ref args)==2 && bytes::equal(bytes::view(v::at<bytes::Buffer>(ref args,1)),b"x\0y"));
    assert(v::clear<bytes::Buffer>(ref mut args)==2); add(ref mut args,b"\xed\xa0\x80");
    rejected(b"x",ref args,1,1,0); assert(v::clear<bytes::Buffer>(ref mut args)==1);
    var u8[4096] large=[0;4096];
    { var usize at=0; while(at<4096) { large[at]=65; at+=1; } }
    let ref u8[] path=ref large;
    let exact=built(path[..4095],ref args); assert(command::view(ref exact).len==4096);
    rejected(path,ref args,2,0,4096); assert(!command::path(path).is_ok);
}

@test fn void accounts_for_pointer_table_at_exact_byte_and_empty_argument_limits() {
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();
    var bytes::Buffer value=bytes::empty();
    let length=command::byte_limit()-27;
    assert(bytes::resize_zeroed(ref mut value,length).is_ok);
    { let view=bytes::view_mut(ref mut value); var usize at=0; while(at<length) { view[at]=65; at+=1; } }
    assert(v::push<bytes::Buffer>(ref mut args,value).is_ok);
    let exact=built(b"x",ref args);
    assert(command::count(ref exact)==2 && command::view(ref exact).len+24==command::byte_limit());
    add(ref mut args,b""); rejected(b"x",ref args,2,2,0); assert(v::clear<bytes::Buffer>(ref mut args)==2);
    let maximum=(command::byte_limit()-18)/9;
    var usize at=0;
    while(at<maximum) { assert(v::push<bytes::Buffer>(ref mut args,bytes::empty()).is_ok); at+=1; }
    let many=built(b"x",ref args);
    assert(command::count(ref many)==maximum+1 && command::view(ref many).len==maximum+2);
    assert(v::push<bytes::Buffer>(ref mut args,bytes::empty()).is_ok);
    rejected(b"x",ref args,2,0,0); assert(v::len<bytes::Buffer>(ref args)==maximum+1);
}
