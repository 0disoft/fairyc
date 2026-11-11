module main;
import hash_file as hash;
import application as app;
import std::fs as fs;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@effects(console)
fn i32 failure(u32 code) {
    let bool written=io::write_bytes(b"{\"ok\":false,\"errorCode\":").is_ok && io::write_u32(code).is_ok && io::write_bytes(b"}\n").is_ok;return 1;
}

@effects(console,filesystem,clock)
fn i32 main() {
    if(process::arg_count()==1){return app::run();}
    if(process::arg_count()!=3 && process::arg_count()!=4){return failure(1);}
    var u8[32] mode=[0;32];let m=switch(process::arg(1,ref mut mode)){case .ok(v):{v}case .err(_):{return failure(1);}};
    let ref u8[] mode_view=ref mode;if(!bytes::equal(mode_view[..m],b"--hash")){return failure(1);}
    var bytes::Buffer path=bytes::empty();if(!bytes::resize_zeroed(ref mut path,131072).is_ok){return failure(5);}
    let n=switch(process::arg(2,bytes::view_mut(ref mut path))){case .ok(v):{v}case .err(_):{return failure(1);}};
    let ref u8[] path_view=bytes::view(ref path);
    let value=switch(app::compute(path_view[..n],null)){case .ok(v):{v}case .err(e):{return failure(e);}};
    var bytes::Buffer report=bytes::empty();if(!hash::report(path_view[..n],ref value,ref mut report)){return failure(5);}
    if(process::arg_count()==4) {
        var bytes::Buffer target=bytes::empty();if(!bytes::resize_zeroed(ref mut target,131072).is_ok){return failure(5);}
        let count=switch(process::arg(3,bytes::view_mut(ref mut target))){case .ok(v):{v}case .err(_):{return failure(1);}};
        let ref u8[] target_view=bytes::view(ref target);
        switch(hash::save(target_view[..count],path_view[..n],ref value,bytes::view(ref report))){case .ok(_):{}case .err(e):{return failure(e);}};
    }
    return if(io::write_bytes(bytes::view(ref report)).is_ok){0}else{1};
}
