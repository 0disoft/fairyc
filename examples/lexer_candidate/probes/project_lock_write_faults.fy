module main;
import project_lock_write as store;
import project_source_file as source;
import std::bytes as bytes;
import std::process as process;

@effects(filesystem)
fn i32 main(){
    var u8[4096] path=[0;4096];let length=switch(process::arg(1,ref mut path)){case .ok(value):{value}case .err(_):{return 10;}};let ref u8[] buffer=ref path;let root=buffer[..length];
    var u8[64] mode_buffer=[0;64];let count=switch(process::arg(2,ref mut mode_buffer)){case .ok(value):{value}case .err(_):{return 11;}};let ref u8[] modes=ref mode_buffer;let mode=modes[..count];
    var usize meter=0;let payload=switch(source::read(root,b"input.json",8388608,ref mut meter)){case .ok(value):{value}case .err(_):{return 12;}};
    let result=store::write(root,payload,ref mut meter);
    if(bytes::equal(mode,b"ok")){assert(result.is_ok);return 0;}
    let p=switch(result){case .ok(_):{return 13;}case .err(value):{value}};
    assert(p.code==1);
    if(bytes::equal(mode,b"first-write")){assert(p.stage==6 && p.os_code==112 && p.completed==0);}
    else if(bytes::equal(mode,b"partial-write")){assert(p.stage==6 && p.os_code==112 && p.completed==2);}
    else if(bytes::equal(mode,b"sync")){assert(p.stage==7 && p.os_code==29);}
    else if(bytes::equal(mode,b"close")){assert(p.stage==8 && p.os_code==6);}
    else if(bytes::equal(mode,b"replace")){assert(p.stage==9 && p.os_code==112);}
    else if(bytes::equal(mode,b"secondary")){assert(p.stage==6 && p.os_code==112 && p.close_code==1 && p.close_os_code==6);}
    else if(bytes::equal(mode,b"cleanup")){assert(p.stage==7 && p.os_code==29 && p.cleanup_code==1 && p.cleanup_os_code==5);}
    else{return 14;}
    if(!bytes::equal(mode,b"secondary")){assert(p.close_code==0);}
    if(!bytes::equal(mode,b"cleanup")){assert(p.cleanup_code==0);}
    return 0;
}
