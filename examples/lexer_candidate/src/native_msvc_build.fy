module native_msvc_build;
import windows_msvc_toolchain as tools;
import windows_host_paths as host;
import windows_run_capture as child;
import native_packet_decode as packet;
import native_msvc_artifacts as artifact;
import native_scratch as scratch;
import std::bytes as bytes;
import std::vec as v;
import std::text as text;
import std::fs as fs;

pub struct Options{bool release;u32 timeout_ms;usize stack_reserve;}
pub resource Image{bytes::Buffer data;usize objects;cleanup{}}
// 1 options/bounds, 2 allocation, 3 scratch, 4 file I/O, 5 process transport,
// 6 C compilation, 7 object validation, 8 link, 9 image validation.
pub resource Failure{u32 stage;u32 status;u32 os_code;usize unit;bytes::Buffer diagnostics;cleanup{}}
fn Failure fail(u32 stage,u32 status,u32 os_code,usize unit){return Failure{.stage,.status,.os_code,.unit,.diagnostics=bytes::empty()};}
pub fn ref u8[] data(ref Image image){return bytes::view(ref image.data);}
pub fn usize object_count(ref Image image){return image.objects;}
pub fn u32 stage(ref Failure error){return error.stage;}
pub fn u32 status(ref Failure error){return error.status;}
pub fn u32 os_code(ref Failure error){return error.os_code;}
pub fn usize unit(ref Failure error){return error.unit;}
pub fn ref u8[] diagnostics(ref Failure error){return bytes::view(ref error.diagnostics);}
pub fn bool valid(Options options){return options.timeout_ms>0 && options.timeout_ms<=600000 && (options.stack_reserve==0 || options.stack_reserve>=65536 && options.stack_reserve<=16777216 && options.stack_reserve%65536==0);}
fn Result<bytes::Buffer,Failure> join(ref u8[] root,ref u8[] leaf){return switch(tools::joined(root,leaf)){case .ok(p):{.ok(p)}case .err(e):{.err(fail(if(e.stage==2){2}else{1},e.status,e.os_code,0))}};}
fn Result<usize,Failure> argument(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let text=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0,0));}};return switch(v::push<bytes::Buffer>(args,text)){case .ok(n):{.ok(n)}case .err(_):{.err(fail(2,0,0,0))}};}
fn Result<usize,Failure> prefixed(ref mut v::Vec<bytes::Buffer> args,ref u8[] prefix,ref u8[] value){var bytes::Buffer text=bytes::empty();if(!bytes::append(ref mut text,prefix).is_ok || !bytes::append(ref mut text,value).is_ok){return .err(fail(2,0,0,0));}return switch(v::push<bytes::Buffer>(args,text)){case .ok(n):{.ok(n)}case .err(_):{.err(fail(2,0,0,0))}};}
@effects(filesystem) fn Result<usize,Failure> write(ref u8[] path,ref u8[] data){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(fail(4,1,0,0));}};let native=switch(text::cstring_from_text(name)){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0,0));}};
    var fs::File file=switch(fs::create_new(text::cstring_as_cstr(ref native))){case .ok(p):{p}case .err(e):{return .err(fail(4,e.status,e.os_code,0));}};
    let usize count=switch(fs::write_all(ref mut file,data)){case .ok(n):{n}case .err(e):{return .err(fail(4,e.status,e.os_code,0));}};if(count!=data.len){return .err(fail(4,1,0,0));}switch(fs::close(file)){case .ok(_):{}case .err(e):{return .err(fail(4,e.status,e.os_code,0));}};return .ok(count);
}
@effects(filesystem,console,environment,clock) fn Result<usize,Failure> command(ref tools::Toolchain toolchain,ref v::Vec<bytes::Buffer> args,ref u8[] directory,Options options,u32 phase,usize unit){
    let result=switch(child::run_environment(tools::compiler(toolchain),args,directory,options.timeout_ms,1,b"",tools::block(toolchain))){case .ok(p):{p}case .err(e):{return .err(fail(5,e.kind,e.os_code,unit));}};
    if(child::status(ref result)!=0){var bytes::Buffer diagnostics=bytes::empty();if(!bytes::append(ref mut diagnostics,child::stdout(ref result)).is_ok || !bytes::append(ref mut diagnostics,child::stderr(ref result)).is_ok){return .err(fail(2,0,0,unit));}return .err(Failure{.stage=phase,.status=child::status(ref result),.os_code=0,.unit,.diagnostics});}return .ok(0);
}
fn Result<bytes::Buffer,Failure> filename(usize unit,bool object){var bytes::Buffer name=bytes::empty();if(!bytes::append(ref mut name,b"fy_unit_").is_ok || !bytes::append_usize(ref mut name,unit).is_ok || !bytes::append(ref mut name,if(object){b".obj"}else{b".c"}).is_ok){return .err(fail(2,0,0,unit));}return .ok(name);}
@effects(filesystem,console,environment,clock) pub fn Result<Image,Failure> build(ref tools::Toolchain toolchain,ref packet::Packet packet,ref u8[] runtime,Options options){
    if(!valid(options) || runtime.len==0 || runtime.len>8388608){return .err(fail(1,0,0,0));}var usize total=packet::header(packet).len+packet::literals(packet).len+runtime.len;var usize i=0;
    while(i<packet::unit_count(packet)){let usize size=packet::unit(packet,i).len;if(total>100663296 || size>100663296-total){return .err(fail(1,0,0,i));}total+=size;i+=1;}
    let temporary=switch(host::temporary()){case .ok(p):{p}case .err(e):{return .err(fail(3,e.kind,e.os_code,0));}};let directory=switch(scratch::create(bytes::view(ref temporary))){case .ok(p):{p}case .err(e):{return .err(fail(3,e.status,e.os_code,0));}};let root=scratch::path(ref directory);
    let header_path=join(root,b"fy_project.h")?;let usize header_written=write(bytes::view(ref header_path),packet::header(packet))?;
    var bytes::Buffer response=bytes::empty();let usize count=packet::unit_count(packet)+2;i=0;
    while(i<count){let source_name=if(i==0){b"fy_literals.c"}else if(i==count-1){b"fy_runtime.c"}else{b""};let object_name=if(i==0){b"fy_literals.obj"}else if(i==count-1){b"fy_runtime.obj"}else{b""};
        let own_source=if(i>0 && i<count-1){filename(i-1,false)?}else{bytes::empty()};let own_object=if(i>0 && i<count-1){filename(i-1,true)?}else{bytes::empty()};
        let source=join(root,if(source_name.len==0){bytes::view(ref own_source)}else{source_name})?;let object=join(root,if(object_name.len==0){bytes::view(ref own_object)}else{object_name})?;
        let content=if(i==0){packet::literals(packet)}else if(i==count-1){runtime}else{packet::unit(packet,i-1)};let usize written=write(bytes::view(ref source),content)?;
        var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();let usize a=argument(ref mut args,b"/nologo")?;let usize b=argument(ref mut args,b"/std:c11")?;let usize c=argument(ref mut args,b"/TC")?;let usize d=argument(ref mut args,b"/MT")?;let usize e=argument(ref mut args,b"/Brepro")?;let usize f=argument(ref mut args,if(options.release){b"/O2"}else{b"/Od"})?;let usize g=argument(ref mut args,b"/c")?;let usize h=argument(ref mut args,bytes::view(ref source))?;let usize j=prefixed(ref mut args,b"/Fo:",bytes::view(ref object))?;let usize compiled=command(toolchain,ref args,root,options,6,i)?;
        let inspected=switch(artifact::read(bytes::view(ref object))){case .ok(p):{p}case .err(e):{return .err(fail(7,e.status,e.os_code,i));}};let u32 issue=artifact::object_problem(bytes::view(ref inspected));if(issue!=0){return .err(fail(7,issue,0,i));}
        // Relative generated names are ASCII, quoted, and bounded. This avoids
        // overflowing the Windows command-line limit with many object inputs.
        if(!bytes::append_byte(ref mut response,34).is_ok || !bytes::append(ref mut response,if(object_name.len==0){bytes::view(ref own_object)}else{object_name}).is_ok || !bytes::append(ref mut response,b"\"\n").is_ok){return .err(fail(2,0,0,i));}i+=1;
    }
    let response_path=join(root,b"objects.rsp")?;let usize response_written=write(bytes::view(ref response_path),bytes::view(ref response))?;let output=join(root,b"program.exe")?;
    var v::Vec<bytes::Buffer> link=v::empty<bytes::Buffer>();let usize a=argument(ref mut link,b"/nologo")?;let usize b=argument(ref mut link,b"/MT")?;let usize c=prefixed(ref mut link,b"@",bytes::view(ref response_path))?;let usize d=prefixed(ref mut link,b"/Fe:",bytes::view(ref output))?;let usize e=argument(ref mut link,b"/link")?;let usize f=argument(ref mut link,b"/INCREMENTAL:NO")?;let usize g=argument(ref mut link,b"/Brepro")?;
    if(options.stack_reserve!=0){var bytes::Buffer reserve=bytes::empty();if(!bytes::append_usize(ref mut reserve,options.stack_reserve).is_ok){return .err(fail(2,0,0,0));}let usize stack=prefixed(ref mut link,b"/STACK:",bytes::view(ref reserve))?;}
    let usize linked=command(toolchain,ref link,root,options,8,count)?;let image=switch(artifact::read(bytes::view(ref output))){case .ok(p):{p}case .err(e):{return .err(fail(9,e.status,e.os_code,0));}};if(!artifact::executable(bytes::view(ref image))){return .err(fail(9,1,0,0));}
    return .ok(Image{.data=image,.objects=count});
}
