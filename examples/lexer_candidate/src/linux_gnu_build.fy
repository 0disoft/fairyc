module linux_gnu_build;
import linux_gnu_host as host;
import linux_gnu_policy as policy;
import linux_elf as elf;
import linux_runtime as selection;
import native_packet_decode as packet;
import std::bytes as bytes;
import std::vec as v;

// Compiler: 0 auto (GCC then Clang), 1 GCC, 2 Clang. Each invocation keeps
// the existing companion deadline; the outer Fairy caller owns the stage/tree.
pub struct Options{bool release;u32 timeout_ms;u32 compiler;}
pub resource Image{bytes::Buffer data;usize objects;cleanup{}}
// 1 bounds/options, 2 allocation, 3 toolchain/stage, 4 file I/O, 5 process,
// 6 compile, 7 object, 8 link, 9 executable, 10 publication.
pub resource Failure{u32 stage;u32 status;u32 os_code;usize unit;bytes::Buffer diagnostics;cleanup{}}
fn Failure fail(u32 stage,u32 status,u32 os_code,usize unit){return Failure{.stage,.status,.os_code,.unit,.diagnostics=bytes::empty()};}
pub fn u32 stage(ref Failure e){return e.stage;}
pub fn u32 status(ref Failure e){return e.status;}
pub fn u32 os_code(ref Failure e){return e.os_code;}
pub fn usize unit(ref Failure e){return e.unit;}
pub fn ref u8[] diagnostics(ref Failure e){return bytes::view(ref e.diagnostics);}
pub fn ref u8[] data(ref Image image){return bytes::view(ref image.data);}
pub fn usize object_count(ref Image image){return image.objects;}
pub fn bool valid(Options options){return policy::timeout(options.timeout_ms) && options.compiler<=2;}
fn Result<usize,Failure> argument(ref mut v::Vec<bytes::Buffer> args,ref u8[] data){let value=switch(bytes::copy(data)){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0,0));}};return switch(v::push<bytes::Buffer>(args,value)){case .ok(n):{.ok(n)}case .err(_):{.err(fail(2,0,0,0))}};}
fn Result<bytes::Buffer,Failure> filename(usize index,bool object){var bytes::Buffer name=bytes::empty();if(!bytes::append(ref mut name,b"fy_part_").is_ok || !bytes::append_usize(ref mut name,index).is_ok || !bytes::append(ref mut name,if(object){b".o"}else{b".c"}).is_ok){return .err(fail(2,0,0,index));}return .ok(name);}
fn Failure outcome(ref host::Captured result,u32 stage,usize unit){var bytes::Buffer diagnostics=bytes::empty();if(!bytes::append(ref mut diagnostics,host::stdout(result)).is_ok || !bytes::append(ref mut diagnostics,host::stderr(result)).is_ok){return fail(2,0,0,unit);}return Failure{.stage,.status=if(host::kind(result)==1){host::value(result)}else{256+host::kind(result)},.os_code=0,.unit,.diagnostics};}
@effects(filesystem,console,environment,clock) fn Result<host::Captured,Failure> command(ref u8[] compiler,ref v::Vec<bytes::Buffer> args,ref u8[] directory,Options options,u32 stage,usize unit){
    let result=switch(host::run(compiler,args,directory,options.timeout_ms)){case .ok(p):{p}case .err(code):{return .err(fail(5,0,code,unit));}};
    if(host::kind(ref result)!=1 || host::value(ref result)!=0){return .err(outcome(ref result,stage,unit));}return .ok(result);
}
@effects(filesystem,console,environment,clock) fn Result<host::Captured,Failure> probe(ref u8[] program,ref u8[] first,ref u8[] second,ref u8[] directory,Options options){
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();let usize a=argument(ref mut args,first)?;if(second.len!=0){let usize b=argument(ref mut args,second)?;}return command(program,ref args,directory,options,3,0);
}
@effects(filesystem,console,environment,clock) fn Result<bytes::Buffer,Failure> resolve(bool clang,ref u8[] libc,ref u8[] directory,Options options){
    let compiler=switch(host::find(if(clang){b"clang"}else{b"gcc"})){case .ok(p):{p}case .err(code):{return .err(fail(3,0,code,0));}};
    let banner=probe(bytes::view(ref compiler),b"--version",b"",directory,options)?;
    let triple=probe(bytes::view(ref compiler),b"-dumpmachine",b"",directory,options)?;
    if(clang){if(!policy::admit(true,libc,host::stdout(ref banner),b"",host::stdout(ref triple)).is_ok){return .err(fail(3,1,0,0));}}
    else{let numeric=probe(bytes::view(ref compiler),b"-dumpfullversion",b"-dumpversion",directory,options)?;if(!policy::admit(false,libc,host::stdout(ref banner),host::stdout(ref numeric),host::stdout(ref triple)).is_ok){return .err(fail(3,1,0,0));}}
    return .ok(compiler);
}
@effects(filesystem,console,environment,clock) fn Result<bytes::Buffer,Failure> toolchain(ref u8[] directory,Options options){
    let getconf=switch(host::find(b"getconf")){case .ok(p):{p}case .err(code):{return .err(fail(3,0,code,0));}};
    let libc=probe(bytes::view(ref getconf),b"GNU_LIBC_VERSION",b"",directory,options)?;if(!policy::libc(host::stdout(ref libc))){return .err(fail(3,1,0,0));}
    if(options.compiler!=0){return resolve(options.compiler==2,host::stdout(ref libc),directory,options);}
    switch(resolve(false,host::stdout(ref libc),directory,options)){case .ok(p):{return .ok(p);}case .err(_):{}};
    return resolve(true,host::stdout(ref libc),directory,options);
}
fn ref u8[] startup(){return b"#include <stdint.h>\nextern int32_t fy_candidate_entry(void);\nextern void fy_std_process_set_args_v0(int, char **);\nint main(int argc, char **argv) {\n    fy_std_process_set_args_v0(argc, argv);\n    return (int)fy_candidate_entry();\n}\n";}
@effects(filesystem) fn Result<bool,Failure> write(ref host::Stage stage,ref u8[] name,ref u8[] content,usize unit){return switch(host::write(stage,name,content)){case .ok(p):{.ok(p)}case .err(code):{.err(fail(4,0,code,unit))}};}
@effects(filesystem) fn Result<bytes::Buffer,Failure> read(ref host::Stage stage,ref u8[] name,u32 phase,usize unit){return switch(host::read(stage,name)){case .ok(p):{.ok(p)}case .err(code):{.err(fail(phase,0,code,unit))}};}
@effects(filesystem,console,environment,clock) pub fn Result<Image,Failure> build(ref packet::Packet input,ref u8[] runtime,ref u8[] directory,Options options){
    if(!valid(options) || runtime.len==0 || runtime.len>8388608){return .err(fail(1,0,0,0));}
    let plan=switch(selection::select(packet::header(input))){case .ok(p):{p}case .err(_):{return .err(fail(1,0,0,0));}};
    var usize total=packet::header(input).len+packet::literals(input).len+runtime.len;var usize i=0;
    while(i<packet::unit_count(input)){let size=packet::unit(input,i).len;if(total>100663296 || size>100663296-total){return .err(fail(1,0,0,i));}total+=size;i+=1;}
    let stage=switch(host::open(directory)){case .ok(p):{p}case .err(code):{return .err(fail(3,0,code,0));}};
    let compiler=toolchain(directory,options)?;let bool header_written=write(ref stage,b"fy_project.h",packet::header(input),0)?;
    let count=packet::unit_count(input)+2+(if(plan.arguments){1}else{0});var bytes::Buffer response=bytes::empty();i=0;
    while(i<count){let source=filename(i,false)?;let object=filename(i,true)?;
        let content=if(i==0){packet::literals(input)}else if(i<=packet::unit_count(input)){packet::unit(input,i-1)}else if(i==packet::unit_count(input)+1){runtime}else{startup()};
        let bool written=write(ref stage,bytes::view(ref source),content,i)?;
        var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();
        let usize a=argument(ref mut args,b"-std=c11")?;let usize b=argument(ref mut args,b"-fno-common")?;let usize c=argument(ref mut args,policy::optimization(options.release))?;let usize d=argument(ref mut args,b"-pipe")?;
        let usize e=argument(ref mut args,b"-c")?;let usize f=argument(ref mut args,bytes::view(ref source))?;let usize g=argument(ref mut args,b"-o")?;let usize h=argument(ref mut args,bytes::view(ref object))?;
        if(plan.arguments){let usize n=argument(ref mut args,b"-DFY_CANDIDATE_NO_MAIN")?;}
        let compiled=command(bytes::view(ref compiler),ref args,directory,options,6,i)?;
        let data=read(ref stage,bytes::view(ref object),7,i)?;if(!elf::object(bytes::view(ref data))){return .err(fail(7,1,0,i));}
        if(!bytes::append(ref mut response,bytes::view(ref object)).is_ok || !bytes::append_byte(ref mut response,10).is_ok){return .err(fail(2,0,0,i));}i+=1;
    }
    let bool response_written=write(ref stage,b"fy_objects.rsp",bytes::view(ref response),0)?;
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();let usize a=argument(ref mut args,b"@fy_objects.rsp")?;let usize b=argument(ref mut args,b"-o")?;let usize c=argument(ref mut args,b"fy_program")?;
    let linked=command(bytes::view(ref compiler),ref args,directory,options,8,count)?;let image=read(ref stage,b"fy_program",9,0)?;
    if(!elf::executable(bytes::view(ref image))){return .err(fail(9,1,0,0));}switch(host::close(stage)){case .ok(_):{}case .err(code):{return .err(fail(3,0,code,0));}};
    return .ok(Image{.data=image,.objects=count});
}
@effects(filesystem,console,environment,clock) pub fn Result<usize,Failure> build_publish(ref packet::Packet input,ref u8[] runtime,ref u8[] directory,Options options,ref u8[] root,ref u8[] output){
    let image=build(input,runtime,directory,options)?;switch(host::publish(root,output,data(ref image))){case .ok(_):{}case .err(code):{return .err(fail(10,0,code,0));}};return .ok(data(ref image).len);
}
