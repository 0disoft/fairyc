module native_msvc_artifacts;
import std::bytes as bytes;
import native_pe as pe;
import native_coff as coff;
import native_pe_unwind as unwind;
import std::fs as fs;
import std::text as text;

pub struct Problem{u32 stage;u32 status;u32 os_code;}
fn Problem fail(u32 stage,u32 status,u32 os_code){return Problem{.stage,.status,.os_code};}
@effects(filesystem) pub fn Result<bytes::Buffer,Problem> read(ref u8[] path){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(fail(1,1,0));}};var fs::File file=switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(e):{return .err(fail(1,e.status,e.os_code));}};
    let usize observed=switch(fs::size(ref file)){case .ok(n):{n}case .err(e):{return .err(fail(2,e.status,e.os_code));}};let kind=switch(fs::kind(ref file)){case .ok(k):{k}case .err(e):{return .err(fail(2,e.status,e.os_code));}};
    if(!fs::kind_is_disk(kind) || observed==0 || observed>67108864){return .err(fail(2,1,0));}var bytes::Buffer data=bytes::empty();
    while(true){let usize used=bytes::len(ref data);let usize maximum=if(67108865-used<65536){67108865-used}else{65536};let usize n=switch(fs::read_append(ref mut file,ref mut data,maximum)){case .ok(n):{n}case .err(e):{return .err(fail(3,if(e.file_status!=0){e.file_status}else{e.allocation_status},e.os_code));}};if(bytes::len(ref data)>67108864){return .err(fail(2,1,0));}if(n==0){break;}}
    if(bytes::len(ref data)!=observed){return .err(fail(2,1,0));}switch(fs::close(file)){case .ok(_):{}case .err(e):{return .err(fail(4,e.status,e.os_code));}};return .ok(data);
}
// Match the Go project driver's strict code verifier plus data-only fallback.
pub fn bool object(ref u8[] data){return object_problem(data)==0;}
pub fn u32 object_problem(ref u8[] data){return coff::project_problem(data);}
// Full PE structure; validate unwind when either exception-directory field exists.
// A leaf image with no exception directory remains admitted.
pub fn bool executable(ref u8[] data){
    let h=switch(pe::inspect(data)){case .ok(h):{h}case .err(_):{return false;}};
    if(pe::directory_rva(data,h,3)==0 && pe::directory_size(data,h,3)==0){return true;}
    return switch(unwind::inspect(data)){case .ok(_):{true}case .err(_):{false}};
}
