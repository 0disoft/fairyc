module native_build_cli;
import native_packet_decode as packet;
import native_runtime_assets as assets;
import native_runtime as runtime;
import native_msvc_build as build;
import native_image_publish as publication;
import windows_msvc_toolchain as tools;
import windows_host_paths as host;
import std::bytes as bytes;
import std::process as process;
import std::text as text;
import std::fs as fs;
import std::io as io;

// Private companion entry used by native_cli build/run/test. Runtime C comes
// only from the pinned distribution pack beside this executable.
pub resource Options{bytes::Buffer input;bytes::Buffer output;bytes::Buffer root;bool release;bool trap;cleanup{}}
pub fn ref u8[] input(ref Options p){return bytes::view(ref p.input);}
pub fn ref u8[] output(ref Options p){return bytes::view(ref p.output);}
pub fn ref u8[] root(ref Options p){return bytes::view(ref p.root);}
pub fn bool release(ref Options p){return p.release;}
pub fn bool trap(ref Options p){return p.trap;}
fn Result<bytes::Buffer,u32> arg(usize i){var u8[4096] raw=[0;4096];let n=switch(process::arg(i,ref mut raw)){case .ok(n):{n}case .err(_):{return .err(1);}};let ref u8[] view=ref raw;return switch(bytes::copy(view[..n])){case .ok(p):{.ok(p)}case .err(_):{.err(1)}};}
pub fn Result<Options,u32> parse(){
    var bytes::Buffer input=bytes::empty();var bytes::Buffer output=bytes::empty();var bytes::Buffer root=bytes::empty();var bool release=false;var bool trap=false;var u32 seen=0;var usize i=1;
    while(i<process::arg_count()){let value=arg(i)?;let name=bytes::view(ref value);var u32 bit=0;
        if(bytes::equal(name,b"--packet-input")){bit=1;}else if(bytes::equal(name,b"--output")){bit=2;}else if(bytes::equal(name,b"--output-root")){bit=4;}else if(bytes::equal(name,b"--compile-timeout")){bit=8;}else if(bytes::equal(name,b"--release")){bit=16;}else if(bytes::equal(name,b"--run-trap-control")){bit=32;}else{return .err(1);}
        if((seen&bit)!=0){return .err(1);}seen|=bit;
        if(bit<16){i+=1;if(i==process::arg_count()){return .err(1);}let value=arg(i)?;if(bytes::len(ref value)==0){return .err(1);}
            if(bit==1){if(!bytes::append(ref mut input,bytes::view(ref value)).is_ok){return .err(1);}}else if(bit==2){if(!bytes::append(ref mut output,bytes::view(ref value)).is_ok){return .err(1);}}else if(bit==4){if(!bytes::append(ref mut root,bytes::view(ref value)).is_ok){return .err(1);}}else if(!bytes::equal(bytes::view(ref value),b"1m")){return .err(1);}
        }else if(bit==16){release=true;}else{trap=true;}i+=1;
    }if((seen&3)!=3){return .err(1);}return .ok(Options{.input,.output,.root,.release,.trap});
}
@effects(console) fn i32 failure(ref str message){return if(io::write_stderr("candidate build failed: ").is_ok && io::write_stderr(message).is_ok && io::write_stderr("\n").is_ok){1}else{4};}
@effects(filesystem) fn Result<bytes::Buffer,u32> read_assets(ref u8[] path){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(1);}};var file=switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(_):{return .err(1);}};
    let kind=switch(fs::kind(ref file)){case .ok(k):{k}case .err(_):{return .err(1);}};let size=switch(fs::size(ref file)){case .ok(n):{n}case .err(_):{return .err(1);}};
    if(!fs::kind_is_disk(kind) || size==0 || size>1048576){return .err(1);}var bytes::Buffer data=bytes::empty();
    while(true){let usize used=bytes::len(ref data);let usize remaining=1048577-used;let n=switch(fs::read_append(ref mut file,ref mut data,if(remaining>65536){65536}else{remaining})){case .ok(n):{n}case .err(_):{return .err(1);}};if(bytes::len(ref data)>1048576){return .err(1);}if(n==0){break;}}
    if(bytes::len(ref data)!=size){return .err(1);}if(!fs::close(file).is_ok){return .err(1);}return .ok(data);
}
@effects(console) fn i32 build_failure(ref build::Failure e){
    if(!io::write_stderr("candidate build failed: ").is_ok){return 4;}
    let ref str label=if(build::stage(e)==6){"compile MSVC target C"}else if(build::stage(e)==8){"link MSVC target executable"}else if(build::stage(e)==7){"inspect MSVC target object"}else if(build::stage(e)==9){"inspect MSVC target executable"}else{"native driver failed"};
    if(!io::write_stderr(label).is_ok || !io::write_stderr(": stage ").is_ok || !io::write_stderr_u32(build::stage(e)).is_ok || !io::write_stderr(" status ").is_ok || !io::write_stderr_u32(build::status(e)).is_ok || !io::write_stderr(" os ").is_ok || !io::write_stderr_u32(build::os_code(e)).is_ok || !io::write_stderr("\n").is_ok || !io::write_stderr_bytes(build::diagnostics(e)).is_ok){return 4;}return 1;
}
@effects(filesystem,console,environment,clock) pub fn i32 run(){
    let selected=switch(parse()){case .ok(p):{p}case .err(_):{return failure("expected --packet-input and --output with optional --output-root, --compile-timeout 1m, --release, --run-trap-control");}};
    let p=switch(packet::read(input(ref selected))){case .ok(p):{p}case .err(_):{return failure("decode candidate output: invalid packet input");}};
    let image=switch(host::executable()){case .ok(p):{p}case .err(_):{return failure("runtime asset executable path query failed");}};let location=switch(host::sibling(bytes::view(ref image),b"fairyc-runtime-assets.pack")){case .ok(p):{p}case .err(_):{return failure("runtime asset path query failed");}};
    let raw=switch(read_assets(bytes::view(ref location))){case .ok(p):{p}case .err(_):{return failure("runtime asset pack is missing or unreadable");}};let bundle=switch(assets::load(raw)){case .ok(p):{p}case .err(_):{return failure("runtime asset pack does not match compiler pin");}};
    let source=switch(assets::source(ref bundle,packet::header(ref p),trap(ref selected))){case .ok(p):{p}case .err(e):{
        var bytes::Buffer message=bytes::empty();if(e.kind==2 && runtime::diagnostic(packet::header(ref p),runtime::Problem{.kind=e.symbol_kind,.start=e.start,.end=e.end},ref mut message)){
            if(!io::write_stderr("candidate build failed: select candidate runtime: ").is_ok || !io::write_stderr_bytes(bytes::view(ref message)).is_ok || !io::write_stderr("\n").is_ok){return 4;}return 1;
        }return failure("runtime C assembly failed");
    }};
    let t=switch(tools::resolve()){case .ok(p):{p}case .err(_):{return failure("MSVC toolchain discovery failed");}};
    let result=switch(build::build(ref t,ref p,bytes::view(ref source),build::Options{.release=release(ref selected),.timeout_ms=60000,.stack_reserve=0})){case .ok(p):{p}case .err(e):{return build_failure(ref e);}};
    switch(publication::publish(root(ref selected),output(ref selected),build::data(ref result))){case .ok(_):{}case .err(_):{return failure("publish candidate image failed");}};
    if(!io::write("built: ").is_ok || !io::write_bytes(output(ref selected)).is_ok || !io::write(" (").is_ok || !io::write(if(release(ref selected)){"release"}else{"development"}).is_ok || !io::write(", ").is_ok || !io::write_usize(packet::unit_count(ref p)).is_ok || !io::write(" C units, ").is_ok || !io::write_usize(build::data(ref result).len).is_ok || !io::write(" bytes)\n").is_ok){return 4;}return 0;
}
