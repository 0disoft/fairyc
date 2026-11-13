module save_cli;
import save_job as save_job;
import publication as publication;
import reader as reader;
import std::process as process;
import std::bytes as bytes;
import std::io as io;
import std::fs as fs;
@effects(filesystem,clock)
fn Result<i32,u32> protected_save(ref u8[] destination,ref u8[] data,ref u8[] source){
    let guard=switch(reader::protect(source)){case .ok(v):{v}case .err(e):{return .err(e);}};
    let value=reader::sample(ref guard,0);if(value.error!=0 || value.kind!=1 || value.directory!=0){return .err(2);}
    let id=fs::FileId{.volume_low=value.volume_low,.volume_high=value.volume_high,.file_low_0=value.file_low_0,.file_low_1=value.file_low_1,.file_high_0=value.file_high_0,.file_high_1=value.file_high_1};
    return save_job::run(destination,data,null,false,false,publication::begin(id,true));
}
@effects(console,filesystem,clock)
pub fn i32 run(){
    let argc=process::arg_count();if(argc<4 || argc>6){return 1;}
    var bytes::Buffer path=bytes::empty();var bytes::Buffer data=bytes::empty();var bytes::Buffer source=bytes::empty();
    if(!bytes::resize_zeroed(ref mut path,131072).is_ok || !bytes::resize_zeroed(ref mut data,131072).is_ok || !bytes::resize_zeroed(ref mut source,131072).is_ok){return 1;}
    let n=switch(process::arg(2,bytes::view_mut(ref mut path))){case .ok(v):{v}case .err(_):{return 1;}};
    let size=switch(process::arg(3,bytes::view_mut(ref mut data))){case .ok(v):{v}case .err(_):{return 1;}};
    var bool cancel=false;var usize original=0;
    if(argc>4){
        var u8[32] flag=[0;32];let length=switch(process::arg(4,ref mut flag)){case .ok(v):{v}case .err(_):{return 1;}};
        let ref u8[] view=ref flag;
        if(argc==5 && bytes::equal(view[..length],b"--cancel-before-commit")){cancel=true;}
        else{
            if(argc!=6 || !bytes::equal(view[..length],b"--protect-source")){return 1;}
            original=switch(process::arg(5,bytes::view_mut(ref mut source))){case .ok(v):{v}case .err(_):{return 1;}};
            if(original==0){return 1;}
        }
    }
    var u32 error=0;
    let result=if(original>0){protected_save(bytes::view(ref path)[..n],bytes::view(ref data)[..size],bytes::view(ref source)[..original])}
        else{save_job::run(bytes::view(ref path)[..n],bytes::view(ref data)[..size],null,false,cancel,publication::unprotected())};
    switch(result){case .ok(_):{}case .err(e):{error=e;}};
    var bytes::Buffer out=bytes::empty();
    if(!bytes::append(ref mut out,b"{\"ok\":").is_ok || !bytes::append(ref mut out,if(error==0){b"true"}else{b"false"}).is_ok ||
        !bytes::append(ref mut out,b",\"errorCode\":").is_ok || !bytes::append_usize(ref mut out,error as usize).is_ok ||
        !bytes::append(ref mut out,b"}\n").is_ok || !io::write_bytes(bytes::view(ref out)).is_ok){return 1;}
    return if(error==0){0}else{if(error==8){2}else{1}};
}
