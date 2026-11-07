module module_state_graph_windows_build;
import module_state_graph_transport as transport;
import module_state_graph_native_assets as runtime_assets;
import native_msvc_build as msvc;
import native_msvc_artifacts as artifact;
import native_scratch as scratch;
import windows_msvc_toolchain as tools;
import windows_host_paths as host;
import windows_run_capture as child;
import windows_cancel as cancellation;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;
import std::text as text;
import std::fs as fs;

// No public constructor, mutable projection, path, Packet, or raw-C admission.
pub resource Built{transport::Prepared prepared;bytes::Buffer image;usize objects;cleanup{}}
pub resource Failure{u32 stage;u32 status;u32 os_code;usize unit;bytes::Buffer diagnostics;cleanup{}}
pub fn ref transport::Prepared prepared(ref Built built){return ref built.prepared;}
pub fn ref u8[] data(ref Built built){return bytes::view(ref built.image);}
pub fn usize object_count(ref Built built){return built.objects;}
pub fn u32 stage(ref Failure error){return error.stage;}
pub fn u32 status(ref Failure error){return error.status;}
pub fn u32 os_code(ref Failure error){return error.os_code;}
pub fn usize unit(ref Failure error){return error.unit;}
pub fn ref u8[] diagnostics(ref Failure error){return bytes::view(ref error.diagnostics);}
// Same stages as native_msvc_build; 10 reports explicit scratch cleanup failure.
// 11 is cancellation (1009) or token observation failure (1010 + native cause).
// Units 0/1 are entry/runtime, 2 is the link/image/cleanup boundary.
fn Failure fail(u32 stage,u32 status,u32 os_code,usize unit){return Failure{.stage,.status,.os_code,.unit,.diagnostics=bytes::empty()};}
fn Failure preparation(u32 code,usize unit){return fail(if(code==1003){2}else{1},code,0,unit);}
@effects(filesystem) fn Result<bool,Failure> checkpoint(ref cancellation::Token token,usize unit){
    return switch(cancellation::requested(token)){case .ok(stop):{if(stop){.err(fail(11,1009,0,unit))}else{.ok(false)}}case .err(e):{.err(fail(11,1010,e.os_code,unit))}};
}
fn Result<bytes::Buffer,Failure> join(ref u8[] root,ref u8[] leaf,usize unit){return switch(tools::joined(root,leaf)){case .ok(p):{.ok(p)}case .err(e):{.err(fail(if(e.stage==2){2}else{1},e.status,e.os_code,unit))}};}
fn Result<usize,Failure> argument(ref mut v::Vec<bytes::Buffer> args,ref u8[] value,usize unit){
    let data=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0,unit));}};
    return switch(v::push<bytes::Buffer>(args,data)){case .ok(n):{.ok(n)}case .err(_):{.err(fail(2,0,0,unit))}};
}
@effects(filesystem) fn Result<usize,Failure> write(ref u8[] path,ref u8[] data,usize unit){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(fail(4,1,0,unit));}};
    let native=switch(text::cstring_from_text(name)){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0,unit));}};
    var fs::File file=switch(fs::create_new(text::cstring_as_cstr(ref native))){case .ok(p):{p}case .err(e):{return .err(fail(4,e.status,e.os_code,unit));}};
    let count=switch(fs::write_all(ref mut file,data)){case .ok(n):{n}case .err(e):{return .err(fail(4,e.status,e.os_code,unit));}};
    if(count!=data.len){return .err(fail(4,1,0,unit));}
    switch(fs::close(file)){case .ok(_):{}case .err(e):{return .err(fail(4,e.status,e.os_code,unit));}};return .ok(count);
}
fn u8 path_byte(u8 c){return if(c==92){47}else if(c>=65 && c<=90){c+32}else{c};}
// MSVC can print an absolute working directory even with relative inputs.
// Match Windows case and separator variants before publishing diagnostics.
fn Result<usize,Failure> diagnostic(ref mut bytes::Buffer out,ref u8[] raw,ref u8[] root,usize unit){
    var usize i=0;while(i<raw.len){var bool same=root.len!=0 && root.len<=raw.len-i;var usize j=0;
        while(same && j<root.len){if(path_byte(raw[i+j])!=path_byte(root[j])){same=false;}j+=1;}
        if(same){if(!bytes::append(out,b"<temporary>").is_ok){return .err(fail(2,0,0,unit));}i+=root.len;}
        else{if(!bytes::append_byte(out,raw[i]).is_ok){return .err(fail(2,0,0,unit));}i+=1;}
    }return .ok(raw.len);
}
@effects(filesystem,console,environment,clock) fn Result<usize,Failure> command(ref tools::Toolchain toolchain,ref v::Vec<bytes::Buffer> args,ref u8[] root,msvc::Options options,u32 phase,usize unit,ref cancellation::Token token){
    // Ordinary shared 64KiB capture, EOF stdin, unchanged caller timeout.
    let result=switch(child::run_environment_cancel(tools::compiler(toolchain),args,root,options.timeout_ms,1,b"",tools::block(toolchain),65536,token)){case .ok(p):{p}case .err(e):{
        return .err(if(e.kind==12){fail(11,1009,e.os_code,unit)}else if(e.kind==13){fail(11,1010,e.os_code,unit)}else{fail(5,e.kind,e.os_code,unit)});
    }};
    if(child::status(ref result)!=0){var output=bytes::empty();let a=diagnostic(ref mut output,child::stdout(ref result),root,unit)?;let b=diagnostic(ref mut output,child::stderr(ref result),root,unit)?;
        return .err(Failure{.stage=phase,.status=child::status(ref result),.os_code=0,.unit,.diagnostics=output});}
    discard checkpoint(token,unit)?;return .ok(0);
}
@effects(filesystem,console,environment,clock) fn Result<bytes::Buffer,Failure> build_directory(ref tools::Toolchain toolchain,ref u8[] source,ref u8[] runtime,ref u8[] root,msvc::Options options,ref cancellation::Token token){
    var usize i=0;while(i<2){let source_name=if(i==0){b"unit.c"}else{b"runtime.c"};let object_name=if(i==0){b"unit.obj"}else{b"runtime.obj"};
        discard checkpoint(token,i)?;
        let source_path=join(root,source_name,i)?;let object_path=join(root,object_name,i)?;let written=write(bytes::view(ref source_path),if(i==0){source}else{runtime},i)?;
        var args=v::empty<bytes::Buffer>();let a=argument(ref mut args,b"/nologo",i)?;let b=argument(ref mut args,b"/std:c11",i)?;let c=argument(ref mut args,b"/TC",i)?;
        let d=argument(ref mut args,b"/W4",i)?;let e=argument(ref mut args,b"/WX",i)?;let f=argument(ref mut args,b"/MT",i)?;let g=argument(ref mut args,b"/Brepro",i)?;
        let h=argument(ref mut args,if(options.release){b"/O2"}else{b"/Od"},i)?;let j=argument(ref mut args,b"/c",i)?;let k=argument(ref mut args,source_name,i)?;
        let l=argument(ref mut args,if(i==0){b"/Fo:unit.obj"}else{b"/Fo:runtime.obj"},i)?;let compiled=command(toolchain,ref args,root,options,6,i,token)?;
        let object=switch(artifact::read(bytes::view(ref object_path))){case .ok(p):{p}case .err(e):{return .err(fail(7,e.status,e.os_code,i));}};
        let issue=artifact::object_problem(bytes::view(ref object));if(issue!=0){return .err(fail(7,issue,0,i));}i+=1;
    }
    discard checkpoint(token,2)?;
    var link=v::empty<bytes::Buffer>();let a=argument(ref mut link,b"/nologo",2)?;let b=argument(ref mut link,b"/MT",2)?;let c=argument(ref mut link,b"unit.obj",2)?;let d=argument(ref mut link,b"runtime.obj",2)?;
    let e=argument(ref mut link,b"/Fe:program.exe",2)?;let f=argument(ref mut link,b"/link",2)?;let g=argument(ref mut link,b"/INCREMENTAL:NO",2)?;let h=argument(ref mut link,b"/Brepro",2)?;
    if(options.stack_reserve!=0){var reserve=bytes::empty();if(!bytes::append(ref mut reserve,b"/STACK:").is_ok || !bytes::append_usize(ref mut reserve,options.stack_reserve).is_ok){return .err(fail(2,0,0,2));}let stack=argument(ref mut link,bytes::view(ref reserve),2)?;}
    let linked=command(toolchain,ref link,root,options,8,2,token)?;let path=join(root,b"program.exe",2)?;
    let image=switch(artifact::read(bytes::view(ref path))){case .ok(p):{p}case .err(e):{return .err(fail(9,e.status,e.os_code,2));}};
    if(!artifact::executable(bytes::view(ref image))){return .err(fail(9,1,0,2));}discard checkpoint(token,2)?;return .ok(image);
}
@effects(filesystem,console,environment,clock) pub fn Result<Built,Failure> build(ref tools::Toolchain toolchain,transport::Prepared prepared,ref runtime_assets::Bundle runtime,msvc::Options options,ref mut usize work){
    let token=cancellation::none();return build_cancel(toolchain,prepared,runtime,options,ref token,work);
}
@effects(filesystem,console,environment,clock) pub fn Result<Built,Failure> build_cancel(ref tools::Toolchain toolchain,transport::Prepared prepared,ref runtime_assets::Bundle runtime,msvc::Options options,ref cancellation::Token token,ref mut usize work){
    // All options and both owned source copies are admitted before host I/O.
    if(!msvc::valid(options)){return .err(fail(1,0,0,0));}
    if(!budget::charge(work,0)){return .err(preparation(1002,0));}
    let raw=transport::source(ref prepared);if(raw.len==0 || raw.len>16777216 || raw.len+1>16777216-*work){return .err(preparation(1002,0));}
    // Refuse a pre-signaled valid build without charging or copying either source.
    discard checkpoint(token,0)?;
    if(!budget::charge(work,raw.len+1)){return .err(preparation(1002,0));}
    let source=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return .err(preparation(1003,0));}};
    discard checkpoint(token,1)?;
    let support=switch(runtime_assets::runtime(runtime,work)){case .ok(p):{p}case .err(code):{return .err(preparation(code,1));}};
    discard checkpoint(token,0)?;
    let temporary=switch(host::temporary()){case .ok(p):{p}case .err(e):{return .err(fail(3,e.kind,e.os_code,0));}};
    discard checkpoint(token,0)?;
    let directory=switch(scratch::create(bytes::view(ref temporary))){case .ok(p):{p}case .err(e):{return .err(fail(3,e.status,e.os_code,0));}};
    let result=build_directory(toolchain,bytes::view(ref source),bytes::view(ref support),scratch::path(ref directory),options,token);
    let image=switch(result){case .ok(image):{image}case .err(error):{
        // Cleanup failure must not replace the precise original stage/diagnostic.
        discard scratch::close(directory);return .err(error);
    }};
    switch(scratch::close(directory)){case .ok(_):{}case .err(e):{return .err(fail(10,e.status,e.os_code,2));}};
    discard checkpoint(token,2)?;
    return .ok(Built{.prepared,.image,.objects=2});
}
