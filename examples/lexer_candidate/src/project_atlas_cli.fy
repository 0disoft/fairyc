module project_atlas_cli;
import project_atlas_options as options;
import project_atlas_source as source;
import public_cli_data as public;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
@effects(console) fn i32 error(i32 code,ref str message){if(!io::write_stderr("fairyc: ").is_ok || !io::write_stderr(message).is_ok || !io::write_stderr("\n").is_ok){return 4;}return code;}
@effects(filesystem,console,environment) pub fn i32 arguments(){return arguments_with_help(public::help());}
@effects(filesystem,console,environment) pub fn i32 arguments_with_help(ref str help){if(process::arg_count()==4){var u8[4096] raw=[0;4096];let n=switch(process::arg(3,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] s=ref raw;if(bytes::equal(s[..n],b"--help") || bytes::equal(s[..n],b"-h") || bytes::equal(s[..n],b"help")){return if(io::write(help).is_ok){0}else{4};}}
 let o=switch(options::arguments()){case .ok(p):{p}case .err(_):{return error(2,"invalid experimental atlas-impact arguments");}};let data=switch(source::query(b".",ref o)){case .ok(p):{p}case .err(p):{return error(if(p.stage==4){2}else if(p.stage==3){4}else{7},if(p.stage==1){"atlas-impact requires a valid workspace and current existing fairy.lock"}else if(p.stage==2 && p.code==101){"atlas-impact does not support this declaration form yet"}else if(p.stage==2 && p.code==102){"atlas-impact requires supported declaration signatures"}else if(p.stage==2 && p.code==103){"atlas-impact does not support this call shape yet"}else if(p.stage==2 && p.code==104){"atlas-impact requires Atlas declarations in every package"}else if(p.stage==2){"atlas-impact source verification failed"}else if(p.stage==3){"atlas-impact current data preparation failed"}else{"atlas-impact selector or budget rejected"});}};
 return if(switch(io::write_bytes(bytes::view(ref data))){case .ok(n):{n==bytes::len(ref data)}case .err(_):{false}}){0}else{error(4,"write output failed")};
}
