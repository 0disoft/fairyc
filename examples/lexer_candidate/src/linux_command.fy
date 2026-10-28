module linux_command;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Private amd64 transport budget: strings including NULs plus argv pointers,
// including the final null pointer. This is not an OS exec admission guarantee.
pub fn usize byte_limit() { return 1048576; }
// 1 invalid UTF-8/NUL/empty path, 2 private bound, 3 allocation.
// Argument zero is argv[0]; subsequent arguments are numbered from one.
pub struct Problem { u32 kind; usize argument; usize offset; }
pub resource Arguments { bytes::Buffer storage; usize argc; cleanup { } }
fn Problem fail(u32 kind,usize argument,usize offset) { return Problem{.kind,.argument,.offset}; }
pub fn usize count(ref Arguments value) { return value.argc; }
pub fn ref u8[] view(ref Arguments value) { return bytes::view(ref value.storage); }

fn Result<usize,Problem> valid(ref u8[] value,usize argument) {
    if(!text::validate_utf8(value).is_ok) { return .err(fail(1,argument,0)); }
    var usize at=0;
    while(at<value.len) { if(value[at]==0) { return .err(fail(1,argument,at)); } at+=1; }
    return .ok(value.len);
}
fn Result<usize,Problem> valid_path(ref u8[] value) {
    if(value.len==0) { return .err(fail(1,0,0)); }
    if(value.len>4095) { return .err(fail(2,0,value.len)); }
    return valid(value,0);
}
fn bool append(ref mut bytes::Buffer out,ref u8[] value) {
    return bytes::append(out,value).is_ok && bytes::append_byte(out,0).is_ok;
}

// Executable/cwd byte transport. Preserve Linux path characters verbatim.
pub fn Result<bytes::Buffer,Problem> path(ref u8[] value) {
    switch(valid_path(value)) { case .ok(_): { } case .err(e): { return .err(e); } };
    var bytes::Buffer out=switch(bytes::with_capacity(value.len+1)) {
        case .ok(p): { p } case .err(_): { return .err(fail(3,0,0)); }
    };
    if(!append(ref mut out,value)) { return .err(fail(3,0,0)); }
    return .ok(out);
}

// Preflight all input before allocating the output. argv[0] is the executable;
// empty arguments contribute one NUL and are distinguished by the explicit count.
pub fn Result<Arguments,Problem> build(ref u8[] program,ref v::Vec<bytes::Buffer> arguments) {
    switch(valid_path(program)) { case .ok(_): { } case .err(e): { return .err(e); } };
    var usize encoded=program.len+1;
    var usize charged=encoded+16;
    let n=v::len<bytes::Buffer>(arguments);
    if(n>(byte_limit()-charged)/9) { return .err(fail(2,0,0)); }
    var usize i=0;
    while(i<n) {
        let value=bytes::view(v::at<bytes::Buffer>(arguments,i));
        if(charged>byte_limit()-9 || value.len>byte_limit()-charged-9) { return .err(fail(2,i+1,0)); }
        switch(valid(value,i+1)) { case .ok(_): { } case .err(e): { return .err(e); } };
        encoded+=value.len+1; charged+=value.len+9; i+=1;
    }
    var bytes::Buffer out=switch(bytes::with_capacity(encoded)) {
        case .ok(p): { p } case .err(_): { return .err(fail(3,0,0)); }
    };
    if(!append(ref mut out,program)) { return .err(fail(3,0,0)); }
    i=0;
    while(i<n) {
        if(!append(ref mut out,bytes::view(v::at<bytes::Buffer>(arguments,i)))) { return .err(fail(3,i+1,0)); }
        i+=1;
    }
    return .ok(Arguments{.storage=out,.argc=n+1});
}
