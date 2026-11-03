module project_fetch_cli;
import project_remote_offline as offline;
import project_json_string as quoted;
import public_cli_data as public;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

fn bool help(ref u8[] value){return bytes::equal(value,b"--help") || bytes::equal(value,b"-h") || bytes::equal(value,b"help");}
@effects(console) fn i32 invocation(ref str message){if(!io::write_stderr("fairyc: ").is_ok || !io::write_stderr(message).is_ok || !io::write_stderr("\n\n").is_ok || !io::write_stderr(public::help()).is_ok){return 4;}return 2;}
@effects(console) fn i32 output_error(){discard io::write_stderr("fairyc: write output failed\n");return 4;}
@effects(console) fn i32 show_help(){return if(io::write(public::help()).is_ok){0}else{output_error()};}
@effects(console) fn i32 unknown(ref u8[] name){var usize meter=0;let text=switch(quoted::quote_diagnostic(name,ref mut meter)){case .ok(value):{value}case .err(_):{return 4;}};
    if(!io::write_stderr("fairyc: unknown experimental command ").is_ok || !io::write_stderr_bytes(bytes::view(ref text)).is_ok || !io::write_stderr("\n\n").is_ok || !io::write_stderr(public::help()).is_ok){return 4;}return 2;
}
// Error details are candidate-owned, bounded classifications, not Go/OS error
// strings. The command's prefix, result line and exit statuses are shared.
fn ref str reason(offline::Problem p){
    if(p.stage==1){return "cannot read source plan or root manifest";}
    if(p.stage==2){return "invalid source plan, manifest or workspace binding";}
    if(p.stage==3){return "cannot acquire exclusive package fetch lock";}
    if(p.stage==4){if(p.code==5){return "offline source cache miss";}if(p.code==3){return "source cache byte or entry limit exceeded";}if(p.code==4){return "source cache path is noncanonical, linked or not a regular file";}return "source cache read failed";}
    if(p.stage==5){return "source archive SHA-256 mismatch";}
    if(p.stage==6){if(p.code==3){return "source ZIP file, byte or work limit exceeded";}return "invalid source ZIP or package identity";}
    if(p.stage==7){if(p.code==4){return "existing package conflicts with pinned archive";}return "cannot publish or reuse source package";}
    if(p.stage==8){return "package fetch lock cleanup failed";}
    if(p.stage==9){return "package fetch deadline exceeded";}
    return "source package preparation failed";
}
@effects(console) fn i32 failure(offline::Problem problem){
    if(!io::write_stderr("package fetch failed: ").is_ok || !io::write_stderr(reason(problem)).is_ok){return 4;}
    if(problem.cleanup_code!=0 && !io::write_stderr("; cleanup failed; inspect leftover fetch lock or staging directory").is_ok){return 4;}
    return if(io::write_stderr("\n").is_ok){7}else{4};
}
@effects(console) fn i32 success(offline::Outcome result){var bytes::Buffer text=bytes::empty();
    if(!bytes::append(ref mut text,b"packages fetched: ").is_ok || !bytes::append_usize(ref mut text,result.packages).is_ok || !bytes::append(ref mut text,b" package(s), 0 downloaded, ").is_ok || !bytes::append_usize(ref mut text,result.cache_hits).is_ok || !bytes::append(ref mut text,b" cache hit(s), ").is_ok || !bytes::append_usize(ref mut text,result.installed).is_ok || !bytes::append(ref mut text,b" installed, ").is_ok || !bytes::append_usize(ref mut text,result.reused).is_ok || !bytes::append(ref mut text,b" reused\n").is_ok){return output_error();}
    return if(switch(io::write_bytes(bytes::view(ref text))){case .ok(n):{n==bytes::len(ref text)}case .err(_):{false}}){0}else{output_error()};
}
@effects(filesystem,console,clock) pub fn i32 run(ref u8[] root){return switch(offline::restore(root)){case .ok(result):{success(result)}case .err(p):{failure(p)}};}
// Called after dispatch of argv[1] == experimental. Validate all syntax before
// touching the workspace. Online preparation remains a separate capability.
@effects(filesystem,console,clock) pub fn i32 arguments(){
    let usize argc=process::arg_count();if(argc<3){return invocation("an experimental command is required");}
    var u8[4096] raw=[0;4096];let usize n=switch(process::arg(2,ref mut raw)){case .ok(value):{value}case .err(_):{return invocation("argument exceeds the supported byte limit");}};let ref u8[] command=ref raw;
    if(argc==3 && help(command[..n])){return show_help();}
    var u8[4096] raw_option=[0;4096];var usize length=0;if(argc>=4){length=switch(process::arg(3,ref mut raw_option)){case .ok(value):{value}case .err(_):{return invocation("experimental fetch-packages accepts only optional --offline");}};}
    let ref u8[] option=ref raw_option;if(argc==4 && help(option[..length])){return show_help();}
    if(!bytes::equal(command[..n],b"fetch-packages")){return unknown(command[..n]);}
    if(argc==3){return invocation("experimental fetch-packages requires --offline in this compiler");}
    if(argc!=4 || !bytes::equal(option[..length],b"--offline")){return invocation("experimental fetch-packages accepts only optional --offline");}
    return run(b".");
}
