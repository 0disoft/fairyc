module main;
import application as app;
import domain as domain;
import scan_cli as scan_cli;
import save_cli as save_cli;
import std::num as num;
import std::process as process;
import std::bytes as bytes;
import std::io as io;
@effects(console,filesystem,clock)
fn i32 main(){
    if(process::arg_count()==1){return app::run();}
    let argc=process::arg_count();if(argc<3){return 1;}
    var u8[32] mode=[0;32];let count=switch(process::arg(1,ref mut mode)){case .ok(v):{v}case .err(_):{return 1;}};
    let ref u8[] flag=ref mode;
    if(bytes::equal(flag[..count],b"--save")){return save_cli::run();}
    if(argc!=3 && argc!=5){return 1;}
    if(bytes::equal(flag[..count],b"--scan")){
        var bytes::Buffer path=bytes::empty();if(!bytes::resize_zeroed(ref mut path,131072).is_ok){return 1;}
        let length=switch(process::arg(2,bytes::view_mut(ref mut path))){case .ok(v):{v}case .err(_):{return 1;}};
        var usize after=0;
        if(argc==5){
            var u8[32] option=[0;32];var u8[32] number=[0;32];
            let n=switch(process::arg(3,ref mut option)){case .ok(v):{v}case .err(_):{return 1;}};
            let ref u8[] view=ref option;if(!bytes::equal(view[..n],b"--cancel-after")){return 1;}
            let size=switch(process::arg(4,ref mut number)){case .ok(v):{v}case .err(_):{return 1;}};
            let ref u8[] raw=ref number;let parsed=switch(num::parse_i64_decimal(raw[..size])){case .ok(v):{v}case .err(_):{return 1;}};
            after=switch(num::usize_from_i64(parsed)){case .ok(v):{v}case .err(_):{return 1;}};
        }
        return scan_cli::run(bytes::view(ref path)[..length],argc==5,after);
    }
    if(argc!=3 || !bytes::equal(flag[..count],b"--process")){return 1;}
    var bytes::Buffer input=bytes::empty();var bytes::Buffer output=bytes::empty();
    if(!bytes::resize_zeroed(ref mut input,2048).is_ok){return 1;}
    let length=switch(process::arg(2,bytes::view_mut(ref mut input))){case .ok(v):{v}case .err(_):{return 1;}};
    if(!domain::transform(bytes::view(ref input)[..length],ref mut output) || !bytes::append_byte(ref mut output,10).is_ok){return 1;}
    return if(io::write_bytes(bytes::view(ref output)).is_ok){0}else{1};
}
