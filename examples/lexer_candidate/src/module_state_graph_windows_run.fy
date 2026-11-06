module module_state_graph_windows_run;
import module_state_graph_windows_build as build;
import module_state_graph_transport_request as input;
import module_state_graph_request as request;
import module_state_graph_response as response;
import module_state_graph_response_common as admission;
import windows_run_capture as child;
import windows_host_paths as host;
import windows_msvc_toolchain as paths;
import native_scratch as scratch;
import std::bytes as bytes;
import std::vec as v;
import std::text as text;
import std::fs as fs;

pub struct Options{u32 timeout_ms;}
pub struct Work{usize request_structure;usize request_bytes;usize response_structure;usize response_bytes;}
pub fn Work empty_work(){return Work{.request_structure=0,.request_bytes=0,.response_structure=0,.response_bytes=0};}
// 1 options/work, 2 request, 3 scratch/path, 4 file I/O, 5 child transport,
// 6 process outcome, 7 response admission, 8 successful-run scratch removal.
pub resource Failure{u32 stage;u32 status;u32 os_code;bytes::Buffer diagnostics;cleanup{}}
fn Failure fail(u32 stage,u32 status,u32 os_code){return Failure{.stage,.status,.os_code,.diagnostics=bytes::empty()};}
pub fn u32 stage(ref Failure e){return e.stage;}
pub fn u32 status(ref Failure e){return e.status;}
pub fn u32 os_code(ref Failure e){return e.os_code;}
pub fn ref u8[] diagnostics(ref Failure e){return bytes::view(ref e.diagnostics);}
pub fn bool valid(Options options){return options.timeout_ms>0 && options.timeout_ms<=600000;}
fn Result<bytes::Buffer,Failure> joined(ref u8[] root,ref u8[] leaf){return switch(paths::joined(root,leaf)){case .ok(p):{.ok(p)}case .err(e):{.err(fail(3,e.status,e.os_code))}};}
@effects(filesystem)fn Result<usize,Failure> write(ref u8[] path,ref u8[] content){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(fail(4,1,0));}};
    let native=switch(text::cstring_from_text(name)){case .ok(p):{p}case .err(_):{return .err(fail(4,2,0));}};
    var file=switch(fs::create_new(text::cstring_as_cstr(ref native))){case .ok(p):{p}case .err(e):{return .err(fail(4,e.status,e.os_code));}};
    let size=switch(fs::write_all(ref mut file,content)){case .ok(n):{n}case .err(e):{return .err(fail(4,e.status,e.os_code));}};
    if(size!=content.len){return .err(fail(4,1,0));}
    switch(fs::close(file)){case .ok(_):{}case .err(e):{return .err(fail(4,e.status,e.os_code));}};return .ok(size);
}

// Build owns both the image and the exact Complete-bound transport. No caller
// image/path/response can substitute for that pair. This API grants no CAS.
@effects(filesystem,console,environment,clock)
pub fn Result<response::Admitted,Failure> run(ref build::Built executable,ref request::Snapshot snapshot,request::Limits limits,Options options,ref mut Work work){
    if(!valid(options)){return .err(fail(1,1001,0));}
    if(work.request_structure>16777216 || work.request_bytes>1073741824 || work.response_structure>16777216 || work.response_bytes>1073741824){return .err(fail(1,1002,0));}
    var request_work=request::Work{.structure=work.request_structure,.bytes=work.request_bytes};
    let encoded=input::encode(build::prepared(executable),snapshot,limits,ref mut request_work);
    work.request_structure=request_work.structure;work.request_bytes=request_work.bytes;
    let packet=switch(encoded){case .ok(p):{p}case .err(code):{return .err(fail(2,code,0));}};
    let temporary=switch(host::temporary()){case .ok(p):{p}case .err(e):{return .err(fail(3,e.kind,e.os_code));}};
    let directory=switch(scratch::create(bytes::view(ref temporary))){case .ok(p):{p}case .err(e):{return .err(fail(3,e.status,e.os_code));}};
    let root=scratch::path(ref directory);let image=joined(root,b"graph.exe")?;let replay=joined(root,b"request.bin")?;
    discard write(bytes::view(ref image),build::data(executable))?;discard write(bytes::view(ref replay),input::content(ref packet))?;
    let args=v::empty<bytes::Buffer>();
    let captured=switch(child::run_input_limit(bytes::view(ref image),ref args,root,options.timeout_ms,2,bytes::view(ref replay),67108864)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.kind,e.os_code));}};
    if(child::status(ref captured)!=0 || child::trap(ref captured)!=0 || child::stderr(ref captured).len!=0){
        let diagnostics=switch(bytes::copy(child::stderr(ref captured))){case .ok(p):{p}case .err(_):{return .err(fail(6,1003,0));}};
        return .err(Failure{.stage=6,.status=if(child::trap(ref captured)!=0){child::trap(ref captured)}else{child::status(ref captured)},.os_code=0,.diagnostics});
    }
    var response_work=admission::Work{.structure=work.response_structure,.bytes=work.response_bytes};
    let decoded=response::decode(build::prepared(executable),ref packet,child::stdout(ref captured),ref mut response_work);
    work.response_structure=response_work.structure;work.response_bytes=response_work.bytes;
    let result=switch(decoded){case .ok(p):{p}case .err(code):{return .err(fail(7,code,0));}};
    switch(scratch::close(directory)){case .ok(_):{}case .err(e):{return .err(fail(8,e.status,e.os_code));}};
    return .ok(result);
}
