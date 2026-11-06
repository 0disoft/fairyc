module project_fetch_cli;
import project_remote_offline as offline;
import project_remote_online as online;
import project_fetch_output as output;
import public_cli_data as public;
import std::bytes as bytes;
import std::process as process;

fn bool help(ref u8[] value){return bytes::equal(value,b"--help") || bytes::equal(value,b"-h") || bytes::equal(value,b"help");}
@effects(console) fn i32 invocation(ref str message){return output::invocation(message,public::help());}
@effects(console) fn i32 show_help(){return output::show_help(public::help());}
@effects(console) fn i32 unknown(ref u8[] name){return output::unknown(name,public::help());}
@effects(console) fn i32 success(offline::Outcome p){return output::success(p.packages,p.downloaded,p.cache_hits,p.installed,p.reused);}
@effects(filesystem,console,clock) pub fn i32 run(ref u8[] root){return switch(offline::restore(root)){case .ok(result):{success(result)}case .err(p):{output::failure(p.stage,p.code,p.cleanup_code)}};}
// Called after dispatch of argv[1] == experimental. Validate all syntax before
// touching the workspace. Only an explicit online invocation enables transport.
@effects(filesystem,console,network,clock) pub fn i32 arguments(){
    let usize argc=process::arg_count();if(argc<3){return invocation("an experimental command is required");}
    var u8[4096] raw=[0;4096];let usize n=switch(process::arg(2,ref mut raw)){case .ok(value):{value}case .err(_):{return invocation("argument exceeds the supported byte limit");}};let ref u8[] command=ref raw;
    if(argc==3 && help(command[..n])){return show_help();}
    var u8[4096] raw_option=[0;4096];var usize length=0;if(argc>=4){length=switch(process::arg(3,ref mut raw_option)){case .ok(value):{value}case .err(_):{return invocation("experimental fetch-packages accepts only optional --offline");}};}
    let ref u8[] option=ref raw_option;if(argc==4 && help(option[..length])){return show_help();}
    if(!bytes::equal(command[..n],b"fetch-packages")){return unknown(command[..n]);}
    if(argc==3){return switch(online::fetch(b".")){case .ok(result):{success(result)}case .err(p):{output::failure(p.stage,p.code,p.cleanup_code)}};}
    if(argc!=4 || !bytes::equal(option[..length],b"--offline")){return invocation("experimental fetch-packages accepts only optional --offline");}
    return run(b".");
}
