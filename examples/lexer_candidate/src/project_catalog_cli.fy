module project_catalog_cli;
import project_catalog_options as options;
import project_catalog_source as source;
import public_cli_data as public;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

@effects(console) fn i32 error(i32 code,ref str message){
    if(!io::write_stderr("fairyc: ").is_ok || !io::write_stderr(message).is_ok || !io::write_stderr("\n").is_ok){return 4;}return code;
}
@effects(filesystem,console,environment) pub fn i32 arguments(){
    let argc=process::arg_count();
    if(argc==4 || argc==5){
        var u8[4096] raw=[0;4096];let action_size=switch(process::arg(3,ref mut raw)){case .ok(p):{p}case .err(_):{0}};
        let ref u8[] action=ref raw;let nested=argc==5 && bytes::equal(action[..action_size],b"list");
        if(argc==4 || nested){let n=switch(process::arg(argc-1,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] flag=ref raw;
            if(bytes::equal(flag[..n],b"--help") || bytes::equal(flag[..n],b"-h") || bytes::equal(flag[..n],b"help")){return if(io::write(public::help()).is_ok){0}else{4};}
        }
    }
    let request=switch(options::arguments()){case .ok(p):{p}case .err(_):{return error(2,"invalid experimental atlas-symbols list arguments");}};
    let result=switch(source::query(b".",ref request)){
        case .ok(p):{p}
        case .err(p):{
            if(p.stage==1){return error(7,"atlas-symbols requires a valid workspace and matching lock policy");}
            if(p.stage==2){return error(3,"atlas-symbols source verification or requested map family rejected");}
            if(p.stage==4){return error(2,"atlas-symbols selector or output budget rejected");}
            return error(4,"atlas-symbols current data preparation failed");
        }
    };
    return if(io::write_bytes(bytes::view(ref result)).is_ok){0}else{error(4,"failed to write standard output")};
}
