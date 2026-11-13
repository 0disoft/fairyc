module main;
import hash_file as hash;
import application as app;
import comparison as comparison;
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
    if(process::arg_count()==1){return app::run(b"");}
    let argc=process::arg_count();
    if(argc==2){
        var bytes::Buffer initial=bytes::empty();if(!bytes::resize_zeroed(ref mut initial,131072).is_ok){return failure(5);}
        let count=switch(process::arg(1,bytes::view_mut(ref mut initial))){case .ok(v):{v}case .err(_):{return failure(1);}};
        let ref u8[] path=bytes::view(ref initial);
        if(count==0 || (count>=2 && bytes::equal(path[..2],b"--"))){return failure(1);}
        return app::run(path[..count]);
    }
    if(argc<3 || argc>5){return failure(1);}
    var u8[32] mode=[0;32];let m=switch(process::arg(1,ref mut mode)){case .ok(v):{v}case .err(_):{return failure(1);}};
    let ref u8[] mode_view=ref mode;
    let compare=bytes::equal(mode_view[..m],b"--compare");
    if((compare && argc!=4 && argc!=5) || (!compare && (!bytes::equal(mode_view[..m],b"--hash") || argc>4))){return failure(1);}
    var u8[64] expected=[0;64];
    if(compare){
        var u8[1024] raw=[0;1024];
        let count=switch(process::arg(3,ref mut raw)){case .ok(v):{v}case .err(_):{return failure(12);}};
        let ref u8[] input=ref raw;
        if(!comparison::normalize(input[..count],ref mut expected)){return failure(12);}
    }
    var bytes::Buffer path=bytes::empty();if(!bytes::resize_zeroed(ref mut path,131072).is_ok){return failure(5);}
    let n=switch(process::arg(2,bytes::view_mut(ref mut path))){case .ok(v):{v}case .err(_):{return failure(1);}};
    let ref u8[] path_view=bytes::view(ref path);
    let value=switch(hash::compute(path_view[..n])){case .ok(v):{v}case .err(e):{return failure(e);}};
    var bytes::Buffer report=bytes::empty();if(!hash::report(path_view[..n],ref value,ref mut report)){return failure(5);}
    var bool matches=false;
    if(compare){
        var bytes::Buffer actual=bytes::empty();
        if(!hash::append_hex(ref value,ref mut actual)){return failure(5);}
        matches=bytes::equal(bytes::view(ref actual),ref expected);
        if(!comparison::append_report(ref mut report,ref expected,matches)){return failure(5);}
    }
    let usize destination=if(compare){4}else{3};
    if(argc==destination+1) {
        var bytes::Buffer target=bytes::empty();if(!bytes::resize_zeroed(ref mut target,131072).is_ok){return failure(5);}
        let count=switch(process::arg(destination,bytes::view_mut(ref mut target))){case .ok(v):{v}case .err(_):{return failure(1);}};
        let ref u8[] target_view=bytes::view(ref target);
        switch(hash::save(target_view[..count],path_view[..n],ref value,bytes::view(ref report))){case .ok(_):{}case .err(e):{return failure(e);}};
    }
    if(!io::write_bytes(bytes::view(ref report)).is_ok){return 1;}
    return if(compare && !matches){2}else{0};
}
