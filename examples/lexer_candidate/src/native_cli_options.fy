module native_cli_options;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;

pub resource Options{bool locked;bool release;v::Vec<bytes::Buffer> arguments;cleanup{}}
// 1 unknown/compiler argument, 2 duplicate flag, 3 program NUL, 4 argv decode,
// 5 allocation or inconsistent argv size. Parse indices refer to the input
// vector; reader indices refer to the native process argv.
pub struct Problem{u32 kind;usize index;u32 flag;u32 status;usize needed;}
fn Problem fail(u32 kind,usize index,u32 flag,u32 status,usize needed){return Problem{.kind,.index,.flag,.status,.needed};}
pub fn bool locked(ref Options options){return options.locked;}
pub fn bool release(ref Options options){return options.release;}
pub fn ref v::Vec<bytes::Buffer> arguments(ref Options options){return ref options.arguments;}
pub fn i32 exit_code(Problem p){return if(p.kind==5){4}else{2};}
fn bool nul(ref u8[] data){var usize i=0;while(i<data.len){if(data[i]==0){return true;}i+=1;}return false;}
pub fn Result<Options,Problem> parse(bool run,ref v::Vec<bytes::Buffer> input){
    let count=v::len<bytes::Buffer>(input);var usize limit=count;var usize first=count;var usize i=0;
    // Go splits and rejects NULs before parsing compiler flags. Preserve that
    // precedence even when an earlier compiler argument is also invalid.
    if(run){while(i<count){if(bytes::equal(bytes::view(v::at<bytes::Buffer>(input,i)),b"--")){limit=i;first=i+1;break;}i+=1;}
        i=first;while(i<count){if(nul(bytes::view(v::at<bytes::Buffer>(input,i)))){return .err(fail(3,i,0,0,0));}i+=1;}}
    var bool locked=false;var bool release=false;i=0;
    while(i<limit){let arg=bytes::view(v::at<bytes::Buffer>(input,i));if(bytes::equal(arg,b"--locked")){if(locked){return .err(fail(2,i,1,0,0));}locked=true;}
        else if(bytes::equal(arg,b"--release")){if(release){return .err(fail(2,i,2,0,0));}release=true;}
        else{return .err(fail(1,i,0,0,0));}i+=1;}
    var v::Vec<bytes::Buffer> arguments=v::empty<bytes::Buffer>();i=first;while(i<count){let copied=switch(bytes::copy(bytes::view(v::at<bytes::Buffer>(input,i)))){case .ok(p):{p}case .err(_):{return .err(fail(5,i,0,0,0));}};if(!v::push<bytes::Buffer>(ref mut arguments,copied).is_ok){return .err(fail(5,i,0,0,0));}i+=1;}
    return .ok(Options{.locked,.release,.arguments});
}
pub fn Result<v::Vec<bytes::Buffer>,Problem> read(usize first){
    let count=process::arg_count();if(first>count){return .err(fail(4,first,0,1,0));}var v::Vec<bytes::Buffer> input=v::empty<bytes::Buffer>();var bytes::Buffer scratch=bytes::empty();if(!bytes::resize_zeroed(ref mut scratch,128).is_ok){return .err(fail(5,first,0,0,128));}var usize i=first;
    while(i<count){let size=switch(process::arg(i,bytes::view_mut(ref mut scratch))){case .ok(n):{n}case .err(e):{
        if(e.status!=3){return .err(fail(4,i,0,e.status,e.needed));}if(e.needed<=bytes::len(ref scratch)){return .err(fail(5,i,0,e.status,e.needed));}if(!bytes::resize_zeroed(ref mut scratch,e.needed).is_ok){return .err(fail(5,i,0,e.status,e.needed));}
        switch(process::arg(i,bytes::view_mut(ref mut scratch))){case .ok(n):{n}case .err(e):{return .err(fail(4,i,0,e.status,e.needed));}}
    }};if(size>bytes::len(ref scratch)){return .err(fail(5,i,0,0,size));}let copied=switch(bytes::copy(bytes::view(ref scratch)[..size])){case .ok(p):{p}case .err(_):{return .err(fail(5,i,0,0,size));}};if(!v::push<bytes::Buffer>(ref mut input,copied).is_ok){return .err(fail(5,i,0,0,size));}i+=1;}
    return .ok(input);
}
pub fn Result<Options,Problem> from_process(bool run){let input=switch(read(2)){case .ok(p):{p}case .err(e):{return .err(e);}};return parse(run,ref input);}
pub fn bool diagnostic(bool run,Problem p,ref mut bytes::Buffer output){
    if(p.kind==4){return bytes::append(output,b"fairyc: process argument decoding failed\n").is_ok;}if(p.kind==5){return bytes::append(output,b"fairyc: internal argument processing failed\n").is_ok;}
    if(!bytes::append(output,b"fairyc: command \"").is_ok || !bytes::append(output,if(run){b"run"}else{b"build"}).is_ok){return false;}
    if(p.kind==1){return bytes::append(output,b"\" accepts only optional --release and --locked\n").is_ok;}
    if(p.kind==2){return bytes::append(output,b"\" received ").is_ok && bytes::append(output,if(p.flag==1){b"--locked"}else{b"--release"}).is_ok && bytes::append(output,b" more than once\n").is_ok;}
    return bytes::append(output,b"\" rejects program arguments containing a NUL byte\n").is_ok;
}
