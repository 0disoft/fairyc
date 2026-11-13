module scan_cli;
import jobs as jobs;
import reader as reader;
import domain as domain;
import std::bytes as bytes;
import std::io as io;

@effects(console,clock)
fn bool emit(ref jobs::State state,usize lines,bool final){
    var bytes::Buffer out=bytes::empty();
    return bytes::append(ref mut out,if(final){b"{\"event\":\"result\",\"ok\":"}else{b"{\"event\":\"progress\",\"ok\":"}).is_ok &&
        bytes::append(ref mut out,if(state.phase==2){b"true"}else{b"false"}).is_ok &&
        bytes::append(ref mut out,b",\"cancelled\":").is_ok && bytes::append(ref mut out,if(state.phase==3){b"true"}else{b"false"}).is_ok &&
        bytes::append(ref mut out,b",\"done\":").is_ok && bytes::append_usize(ref mut out,state.done).is_ok &&
        bytes::append(ref mut out,b",\"total\":").is_ok && bytes::append_usize(ref mut out,state.total).is_ok &&
        bytes::append(ref mut out,b",\"percent\":").is_ok && bytes::append_usize(ref mut out,jobs::percent(state)).is_ok &&
        bytes::append(ref mut out,b",\"elapsedMs\":").is_ok && bytes::append_usize(ref mut out,jobs::elapsed(state,reader::ticks())).is_ok &&
        bytes::append(ref mut out,b",\"lines\":").is_ok && bytes::append_usize(ref mut out,lines).is_ok &&
        bytes::append(ref mut out,b",\"errorCode\":").is_ok && bytes::append_usize(ref mut out,state.error as usize).is_ok &&
        bytes::append(ref mut out,b"}\n").is_ok && io::write_bytes(bytes::view(ref out)).is_ok;
}
@effects(console,filesystem,clock)
pub fn i32 run(ref u8[] path,bool cancellable,usize after){
    var jobs::State state=jobs::begin(reader::ticks());
    let input=switch(reader::open(path)){
        case .ok(v):{v}
        case .err(code):{let failed=jobs::fail(ref mut state,code);let printed=emit(ref state,0,true);return 1;}
    };
    var bytes::Buffer buffer=bytes::empty();if(!bytes::resize_zeroed(ref mut buffer,65536).is_ok){return 1;}
    var usize lines=0;var usize last_percent=101;var usize last_total=0;var usize displayed=state.started;
    while(true){
        if(cancellable && state.done>=after){
            jobs::cancel(ref mut state);return if(emit(ref state,lines,true)){2}else{1};
        }
        let action=jobs::poll(ref input,ref mut state,bytes::view_mut(ref mut buffer));
        if(action==2){lines+=domain::newlines(bytes::view(ref buffer)[..state.count]);}
        if(action==4 || action==5){
            if(!emit(ref state,lines,true)){return 1;}return if(action==5){0}else{1};
        }
        let now=reader::ticks();let percent=jobs::percent(ref state);
        if(percent!=last_percent || state.total!=last_total || now-displayed>=100){
            if(!emit(ref state,lines,false)){return 1;}last_percent=percent;last_total=state.total;displayed=now;
        }
        if(action==0){reader::wait(ref input);}
    }
    return 1;
}
