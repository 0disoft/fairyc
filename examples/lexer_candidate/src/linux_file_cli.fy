module linux_file_cli;
import linux_cli_file as files;
import file_frontend_cli as frontend;
import public_cli_data as public;
import project_check_cli as checking;
import linux_build_cli as building;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

pub fn ref str help() { return "usage: lexer-candidate-linux --help | --version | targets\n       lexer-candidate-linux check [--locked]\n       lexer-candidate-linux build [--release] [--locked]\n       lexer-candidate-linux run [--release] [--locked] [-- <program args>]\n       lexer-candidate-linux --tokens-file <path>\n       lexer-candidate-linux --declarations-file <path>\n       lexer-candidate-linux --check-names-file <path>\n       lexer-candidate-linux --bindings-file <path>\n       lexer-candidate-linux --check-types-file <path>\n       lexer-candidate-linux --hir-file <path>\n       lexer-candidate-linux --mir-file <path>\nNo arguments: private binary stdin protocol.\nLinux workspace test/new, cache options and C emission are not connected yet.\n"; }
pub fn bool source_mode(ref u8[] flag) {
    return bytes::equal(flag,b"--tokens-file") || bytes::equal(flag,b"--declarations-file") ||
        bytes::equal(flag,b"--check-names-file") || bytes::equal(flag,b"--bindings-file") ||
        bytes::equal(flag,b"--check-types-file") || bytes::equal(flag,b"--hir-file") || bytes::equal(flag,b"--mir-file");
}
// 1 help, 2 version, 3 target registry, 4 source, 5 unsupported workspace, 6 check, 7 build, 8 run, 0 invocation.
pub fn u32 command(ref u8[] flag,usize argc) {
    if(argc==2) {
        if(bytes::equal(flag,b"--help") || bytes::equal(flag,b"-h") || bytes::equal(flag,b"help")) { return 1; }
        if(bytes::equal(flag,b"--version")) { return 2; }
        if(bytes::equal(flag,b"targets")) { return 3; }
    }
    if(argc==3 && source_mode(flag)) { return 4; }
    if(bytes::equal(flag,b"check")) { return 6; }
    if(bytes::equal(flag,b"build")) { return 7; }
    if(bytes::equal(flag,b"run")) { return 8; }
    if(bytes::equal(flag,b"test") || bytes::equal(flag,b"new")) { return 5; }
    return 0;
}
@effects(console) fn i32 failure(ref str message,i32 code) { return if(io::write_stderr(message).is_ok){code}else{4}; }
@effects(filesystem,console,environment,clock)
pub fn i32 run() {
    var u8[4096] raw=[0;4096];
    let n=switch(process::arg(1,ref mut raw)) { case .ok(p):{p} case .err(_):{return failure(help(),2);} };
    let ref u8[] data=ref raw;
    let mode=command(data[..n],process::arg_count());
    if(mode==0) { return failure(help(),2); }
    if(mode==6) { return check_arguments(); }
    if(mode==7) { return build_arguments(false); }
    if(mode==8) { return build_arguments(true); }
    if(mode==5) { return failure("fairyc: unsupported subset capability: Linux workspace CLI is not connected\n",3); }
    if(mode!=4) { return if(io::write(if(mode==1){help()}else if(mode==2){public::version()}else{public::targets()}).is_ok){0}else{4}; }
    var u8[4096] path=[0;4096];
    let length=switch(process::arg(2,ref mut path)) { case .ok(p):{p} case .err(_):{return failure("error: file path limit (4095 UTF-8 bytes)\n",2);} };
    let ref u8[] spelling=ref path;
    let input=switch(files::load(spelling[..length])) {
        case .ok(p):{p}
        case .err(code):{return failure(if(code==4){"error: source limit (65536 bytes)\n"}else if(code==3){"error: buffer allocation failed\n"}else{"error: input file read failed\n"},2);}
    };
    return frontend::run(input,spelling[..length],frontend::mode(data[..n]));
}

@effects(filesystem,console)
fn i32 check_arguments() {
    let argc=process::arg_count();
    if(argc==2) { return checking::run(b".",false); }
    if(argc!=3) { return failure("fairyc: command \"check\" accepts only optional --locked\n",2); }
    var u8[64] raw=[0;64];
    let n=switch(process::arg(2,ref mut raw)) { case .ok(p):{p} case .err(_):{return failure("fairyc: command \"check\" accepts only optional --locked\n",2);} };
    let ref u8[] value=ref raw;
    if(bytes::equal(value[..n],b"--help") || bytes::equal(value[..n],b"-h")) { return if(io::write(help()).is_ok){0}else{4}; }
    if(!bytes::equal(value[..n],b"--locked")) { return failure("fairyc: command \"check\" accepts only optional --locked\n",2); }
    return checking::run(b".",true);
}
@effects(filesystem,console,environment,clock) fn i32 build_arguments(bool execute){
    if(process::arg_count()==3){var u8[64] raw=[0;64];let n=switch(process::arg(2,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] value=ref raw;if(bytes::equal(value[..n],b"--help") || bytes::equal(value[..n],b"-h")){return if(io::write(help()).is_ok){0}else{4};}}
    return building::run(execute);
}
