module main;
import hash_file as hash;
import sha256_stream as sha;
import std::fs as fs;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@effects(filesystem)
fn Result<hash::Digest,u32> compute(ref u8[] path) {
    var file=switch(hash::open(path)){case .ok(v):{v}case .err(e):{return .err(e);}};
    var sha::State state=sha::empty(); var bytes::Buffer buffer=bytes::empty();
    if(!bytes::resize_zeroed(ref mut buffer,65536).is_ok){return .err(5);}
    while(true){let n=switch(hash::read(ref mut file,ref mut state,ref mut buffer)){case .ok(v):{v}case .err(e):{return .err(e);}};if(n==0){break;}}
    let value=switch(hash::finish(ref file,ref state)){case .ok(v):{v}case .err(e):{return .err(e);}};
    if(!fs::close(file).is_ok){return .err(2);}return .ok(value);
}

@effects(console)
fn i32 failure(u32 code) {
    let bool written=io::write_bytes(b"{\"ok\":false,\"errorCode\":").is_ok && io::write_u32(code).is_ok && io::write_bytes(b"}\n").is_ok;return 1;
}

@effects(console,filesystem)
fn i32 main() {
    if(process::arg_count()!=3 && process::arg_count()!=4){return failure(1);}
    var u8[32] mode=[0;32];let m=switch(process::arg(1,ref mut mode)){case .ok(v):{v}case .err(_):{return failure(1);}};
    let ref u8[] mode_view=ref mode;if(!bytes::equal(mode_view[..m],b"--hash")){return failure(1);}
    var bytes::Buffer path=bytes::empty();if(!bytes::resize_zeroed(ref mut path,131072).is_ok){return failure(5);}
    let n=switch(process::arg(2,bytes::view_mut(ref mut path))){case .ok(v):{v}case .err(_):{return failure(1);}};
    let ref u8[] path_view=bytes::view(ref path);
    let value=switch(compute(path_view[..n])){case .ok(v):{v}case .err(e):{return failure(e);}};
    var bytes::Buffer report=bytes::empty();if(!hash::report(path_view[..n],ref value,ref mut report)){return failure(5);}
    if(process::arg_count()==4) {
        var bytes::Buffer target=bytes::empty();if(!bytes::resize_zeroed(ref mut target,131072).is_ok){return failure(5);}
        let count=switch(process::arg(3,bytes::view_mut(ref mut target))){case .ok(v):{v}case .err(_):{return failure(1);}};
        let ref u8[] target_view=bytes::view(ref target);
        switch(hash::save(target_view[..count],path_view[..n],ref value,bytes::view(ref report))){case .ok(_):{}case .err(e):{return failure(e);}};
    }
    return if(io::write_bytes(bytes::view(ref report)).is_ok){0}else{1};
}
