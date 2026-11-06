module module_state_command_options;
import std::bytes as bytes;
import std::vec as v;

// Private tools/dev/fairyc-state native-graph invocation, NOT the main CLI.
// argv excludes the executable. Options copy scalars, never borrowed paths.
pub struct Options { u32 action; bool release; usize store_index; usize source_index; }
pub enum Parsed { Help; Native { Options options; }; }
pub enum InvocationError { Invalid; UnsupportedGoMIR; TooLarge; Nul; }
pub fn i32 exit_code(InvocationError error) {
    return switch(error) {
        case .Invalid: { 2 } case .UnsupportedGoMIR: { 2 }
        case .TooLarge: { 2 } case .Nul: { 2 }
    };
}
// A private admission budget, not a new public host-path length contract.
pub fn usize byte_limit() { return 1048576; }
fn ref u8[] arg(ref v::Vec<bytes::Buffer> argv, usize index) {
    return bytes::view(v::at<bytes::Buffer>(argv,index));
}
// filepath.Ext on Windows: last dot in the final slash/backslash component.
// In particular, .fy itself is accepted; .FY and a.fy/child are not.
fn bool fairy_extension(ref u8[] path) {
    var usize at=path.len;
    while(at>0) {
        at-=1;
        if(path[at]==47 || path[at]==92) { return false; }
        if(path[at]==46) { return bytes::equal(path[at..],b".fy"); }
    }
    return false;
}
pub fn Result<Parsed,InvocationError> parse(ref v::Vec<bytes::Buffer> argv) {
    let count=v::len<bytes::Buffer>(argv);
    if(count!=1 && count!=3 && count!=7) { return .err(.Invalid); }
    // Preflight all lengths before scanning. No allocation, copying, trimming,
    // normalization, filesystem/process access, or mutation of caller buffers.
    var usize total=0; var usize i=0;
    while(i<count) {
        let size=bytes::len(v::at<bytes::Buffer>(argv,i));
        if(size>byte_limit()-total) { return .err(.TooLarge); }
        total+=size; i+=1;
    }
    i=0;
    while(i<count) {
        let raw=arg(argv,i); var usize j=0;
        while(j<raw.len) { if(raw[j]==0) { return .err(.Nul); } j+=1; }
        i+=1;
    }
    if(count==1) {
        return if(bytes::equal(arg(argv,0),b"--help")){.ok(.Help)}else{.err(.Invalid)};
    }
    // Never reinterpret the existing Go-MIR default as native-graph mode.
    if(count==3) { return .err(.UnsupportedGoMIR); }
    let action=arg(argv,0);
    let u32 selected=if(bytes::equal(action,b"init")){1}
        else if(bytes::equal(action,b"execute")){2}
        else if(bytes::equal(action,b"snapshot")){3}else{0};
    if(selected==0 || bytes::len(v::at<bytes::Buffer>(argv,1))==0 || !fairy_extension(arg(argv,2)) ||
       !bytes::equal(arg(argv,3),b"--engine") || !bytes::equal(arg(argv,4),b"native-graph") ||
       !bytes::equal(arg(argv,5),b"--profile")) { return .err(.Invalid); }
    let profile=arg(argv,6); let release=bytes::equal(profile,b"optimized");
    if(!release && !bytes::equal(profile,b"low")) { return .err(.Invalid); }
    return .ok(.Native{.options=Options{.action=selected,.release,.store_index=1,.source_index=2}});
}
