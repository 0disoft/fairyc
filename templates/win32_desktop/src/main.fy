module main;
import application as app;
import domain as domain;
import std::process as process;
import std::bytes as bytes;
import std::io as io;
@effects(console)
fn i32 main(){
    if(process::arg_count()==1){return app::run();}
    if(process::arg_count()!=3){return 1;}
    var u8[32] mode=[0;32];let count=switch(process::arg(1,ref mut mode)){case .ok(v):{v}case .err(_):{return 1;}};
    let ref u8[] flag=ref mode;if(!bytes::equal(flag[..count],b"--process")){return 1;}
    var bytes::Buffer input=bytes::empty();var bytes::Buffer output=bytes::empty();
    if(!bytes::resize_zeroed(ref mut input,2048).is_ok){return 1;}
    let length=switch(process::arg(2,bytes::view_mut(ref mut input))){case .ok(v):{v}case .err(_):{return 1;}};
    if(!domain::transform(bytes::view(ref input)[..length],ref mut output) || !bytes::append_byte(ref mut output,10).is_ok){return 1;}
    return if(io::write_bytes(bytes::view(ref output)).is_ok){0}else{1};
}
