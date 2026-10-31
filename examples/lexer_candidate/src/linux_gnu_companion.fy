module linux_gnu_companion;
import linux_gnu_build as build;
import linux_runtime_assets as assets;
import linux_companion_input as input;
import linux_host_paths as paths;
import linux_command as command;
import native_packet_decode as packet;
import native_cli_options as arguments;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

pub resource Options{bytes::Buffer input;bytes::Buffer stage;bytes::Buffer root;bytes::Buffer output;bool release;bool trap;bool help;u32 compiler;cleanup{}}
pub fn bool help(ref Options p){return p.help;}
pub fn bool release(ref Options p){return p.release;}
pub fn bool trap(ref Options p){return p.trap;}
pub fn u32 compiler(ref Options p){return p.compiler;}
pub fn ref u8[] stage(ref Options p){return bytes::view(ref p.stage);}
pub fn ref u8[] root(ref Options p){return bytes::view(ref p.root);}
pub fn ref u8[] output(ref Options p){return bytes::view(ref p.output);}
pub fn Result<bytes::Buffer,u32> join(ref u8[] parent,ref u8[] leaf){
    if(parent.len==0){return .err(1);}var bytes::Buffer result=bytes::empty();if(!bytes::append(ref mut result,parent).is_ok){return .err(1);}if(parent[parent.len-1]!=47 && !bytes::append_byte(ref mut result,47).is_ok){return .err(1);}if(!bytes::append(ref mut result,leaf).is_ok || bytes::len(ref result)>4095){return .err(1);}return .ok(result);
}
pub fn Result<bytes::Buffer,u32> asset_path(ref u8[] image){var usize end=0;var usize i=0;while(i<image.len){if(image[i]==47){end=i+1;}i+=1;}if(end==0){return .err(1);}return join(image[..end],b"fairyc-linux-runtime-assets.pack");}
pub fn Result<Options,u32> parse(ref v::Vec<bytes::Buffer> args){
    var bytes::Buffer packet_input=bytes::empty();var bytes::Buffer stage_root=bytes::empty();var bytes::Buffer output_root=bytes::empty();var bytes::Buffer image_output=bytes::empty();var bool release=false;var bool trap=false;var u32 compiler=0;
    if(v::len<bytes::Buffer>(args)==1){let arg=bytes::view(v::at<bytes::Buffer>(args,0));if(bytes::equal(arg,b"--help") || bytes::equal(arg,b"-h")){return .ok(Options{.input=packet_input,.stage=stage_root,.root=output_root,.output=image_output,.release=false,.trap=false,.help=true,.compiler=0});}}
    var usize i=0;var u32 seen=0;while(i<v::len<bytes::Buffer>(args)){let name=bytes::view(v::at<bytes::Buffer>(args,i));let u32 bit=if(bytes::equal(name,b"--packet-input")){1}else if(bytes::equal(name,b"--stage-root")){2}else if(bytes::equal(name,b"--output-root")){4}else if(bytes::equal(name,b"--output")){8}else if(bytes::equal(name,b"--release")){16}else if(bytes::equal(name,b"--run-trap-control")){32}else if(bytes::equal(name,b"--cc")){64}else{0};
        if(bit==0 || (seen&bit)!=0){return .err(1);}seen|=bit;
        if(bit==16){release=true;}else if(bit==32){trap=true;}else{i+=1;if(i==v::len<bytes::Buffer>(args)){return .err(1);}let value=bytes::view(v::at<bytes::Buffer>(args,i));if(!command::path(value).is_ok){return .err(1);}
            if(bit==64){if(bytes::equal(value,b"gcc")){compiler=1;}else if(bytes::equal(value,b"clang")){compiler=2;}else{return .err(1);}}
            else if(bit==1){if(!bytes::append(ref mut packet_input,value).is_ok){return .err(1);}}
            else if(bit==2){if(!bytes::append(ref mut stage_root,value).is_ok){return .err(1);}}
            else if(bit==4){if(!bytes::append(ref mut output_root,value).is_ok){return .err(1);}}
            else{if(!bytes::append(ref mut image_output,value).is_ok){return .err(1);}}
        }i+=1;
    }
    if((seen&15)!=15){return .err(1);}let expected=join(bytes::view(ref stage_root),b"packet.bin")?;
    if(bytes::view(ref stage_root)[0]!=47 || !bytes::equal(bytes::view(ref expected),bytes::view(ref packet_input))){return .err(1);}
    if(trap && (!bytes::equal(bytes::view(ref output_root),bytes::view(ref stage_root)) || !bytes::equal(bytes::view(ref image_output),b"program"))){return .err(1);}return .ok(Options{.input=packet_input,.stage=stage_root,.root=output_root,.output=image_output,.release,.trap,.help=false,.compiler});
}
@effects(console) fn i32 failure(ref str message,i32 code){return if(io::write_stderr("candidate build failed: ").is_ok && io::write_stderr(message).is_ok && io::write_stderr("\n").is_ok){code}else{4};}
@effects(console) fn i32 failed(ref build::Failure e){
    if(!io::write_stderr("candidate build failed: ").is_ok || !io::write_stderr(if(build::stage(e)==6){"compile GNU C"}else if(build::stage(e)==8){"link GNU executable"}else if(build::stage(e)==10){"publish candidate image"}else{"GNU driver"}).is_ok || !io::write_stderr(": stage ").is_ok || !io::write_stderr_u32(build::stage(e)).is_ok || !io::write_stderr(" unit ").is_ok || !io::write_stderr_usize(build::unit(e)).is_ok || !io::write_stderr(" status ").is_ok || !io::write_stderr_u32(build::status(e)).is_ok || !io::write_stderr(" os ").is_ok || !io::write_stderr_u32(build::os_code(e)).is_ok || !io::write_stderr("\n").is_ok || !io::write_stderr_bytes(build::diagnostics(e)).is_ok){return 4;}return 1;
}
@effects(filesystem,console,environment,clock) pub fn i32 run(){
    let args=switch(arguments::read(1)){case .ok(p):{p}case .err(_):{return failure("process argument decoding failed",2);}};
    let selected=switch(parse(ref args)){case .ok(p):{p}case .err(_):{return failure("expected --packet-input, --stage-root, --output-root and --output; optional --release, --run-trap-control, --cc gcc|clang",2);}};
    if(selected.help){return if(io::write("fairyc-candidate-linux-build: --packet-input <stage/packet.bin> --stage-root <stage> --output-root <root> --output <relative> [--release] [--run-trap-control] [--cc gcc|clang]\n").is_ok){0}else{4};}
    let packet_raw=switch(input::read(bytes::view(ref selected.input),100663296)){case .ok(p):{p}case .err(_):{return failure("packet input read failed",1);}};
    let p=switch(packet::decode(packet_raw)){case .ok(p):{p}case .err(_):{return failure("decode candidate output: invalid packet input",1);}};
    let image=switch(paths::executable()){case .ok(p):{p}case .err(_):{return failure("runtime asset executable path query failed",1);}};let location=switch(asset_path(bytes::view(ref image))){case .ok(p):{p}case .err(_):{return failure("runtime asset path query failed",1);}};
    let raw=switch(input::read(bytes::view(ref location),1048576)){case .ok(p):{p}case .err(_):{return failure("Linux runtime asset pack is missing or unreadable",1);}};
    let bundle=switch(assets::load(raw)){case .ok(p):{p}case .err(_):{return failure("Linux runtime asset pack does not match compiler pin",1);}};
    let runtime=switch(assets::source(ref bundle,packet::header(ref p),selected.trap)){case .ok(p):{p}case .err(_):{return failure("select Linux runtime: invalid target or runtime symbol",1);}};
    return switch(build::build_publish(ref p,bytes::view(ref runtime),bytes::view(ref selected.stage),build::Options{.release=selected.release,.timeout_ms=60000,.compiler=selected.compiler},bytes::view(ref selected.root),bytes::view(ref selected.output))){case .ok(_):{0}case .err(e):{failed(ref e)}};
}
