module linux_new_cli;
import linux_starter as host;
import project_starter as starter;
import project_sdk_bundle as sdk;
import project_manifest_values as values;
import native_cli_options as arguments;
import public_cli_data as public;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;
import std::io as io;
// 1 arity/flag, 2 portable name, 3 UID. No filesystem access precedes validation.
pub fn u32 problem(ref v::Vec<bytes::Buffer> args){
    if(v::len<bytes::Buffer>(args)!=3 || !bytes::equal(bytes::view(v::at<bytes::Buffer>(args,1)),b"--uid")){return 1;}
    let name=bytes::view(v::at<bytes::Buffer>(args,0));if(!values::name(name) || !values::segment(name)){return 2;}
    return if(values::uid(bytes::view(v::at<bytes::Buffer>(args,2)))){0}else{3};
}
@effects(console) fn bool stderr(ref u8[] data){return switch(io::write_stderr_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}};}
@effects(console) pub fn i32 diagnostic(host::Problem problem,ref u8[] name){
    var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,b"fairyc: project creation failed: ").is_ok){return 4;}
    if(problem.kind==1){if(!bytes::append(ref mut out,b"project parent directory is unavailable\n").is_ok){return 4;}}
    else{
        let prefix=if(problem.kind==2){b"project path \""}else if(problem.kind==3){b"permission denied while creating \""}else if(problem.kind==5){b"project creation failed and \""}else{b"could not create \""};
        if(!bytes::append(ref mut out,prefix).is_ok || !bytes::append(ref mut out,name).is_ok){return 4;}
        if(problem.kind!=5 && problem.slot!=0 && (!bytes::append_byte(ref mut out,47).is_ok || !bytes::append(ref mut out,starter::leaf(problem.slot)).is_ok)){return 4;}
        if(!bytes::append(ref mut out,if(problem.kind==2){b"\" already exists\n"}else if(problem.kind==5){b"\" could not be rolled back\n"}else{b"\"\n"}).is_ok){return 4;}
    }
    return if(stderr(bytes::view(ref out))){if(problem.kind==8){4}else{7}}else{4};
}
@effects(console) fn i32 invocation(u32 kind){
    let message=if(kind==2){b"command \"new\": package name must use portable lowercase kebab-case"}else if(kind==3){b"command \"new\": package UID must use at least two lowercase dot-separated segments"}else{b"command \"new\" expects <package-name> --uid <package-uid>"};
    return if(stderr(b"fairyc: ") && stderr(message) && stderr(b"\n\n") && stderr(text::as_bytes(public::help()))){2}else{4};
}
@effects(filesystem,console) pub fn i32 run(){
    let args=switch(arguments::read(2)){case .ok(p):{p}case .err(e):{return if(stderr(b"fairyc: process argument decoding or allocation failed\n")){arguments::exit_code(e)}else{4};}};
    let error=problem(ref args);if(error!=0){return invocation(error);}
    let name=bytes::view(v::at<bytes::Buffer>(ref args,0));let uid=bytes::view(v::at<bytes::Buffer>(ref args,2));
    switch(host::create(name,uid,sdk::language())){case .ok(_):{}case .err(e):{return diagnostic(e,name);}};
    var bytes::Buffer message=bytes::empty();if(!bytes::append(ref mut message,b"created Fairy C package \"").is_ok || !bytes::append(ref mut message,name).is_ok || !bytes::append(ref mut message,b"\" at ").is_ok || !bytes::append(ref mut message,name).is_ok || !bytes::append_byte(ref mut message,10).is_ok){return 4;}
    return if(switch(io::write_bytes(bytes::view(ref message))){case .ok(n):{n==bytes::len(ref message)}case .err(_):{false}}){0}else{4};
}
