module file_job;
import jobs as jobs;
import reader as reader;
import progress as progress;
import domain as domain;
import ui as ui;
import publication as publication;
import std::fs as fs;
import std::bytes as bytes;

@effects(clock)
fn bool present(ptr mut void window,ref jobs::State state){
    var bytes::Buffer out=bytes::empty();
    return bytes::append(ref mut out,if(state.phase==2){b"완료 · "}else{b""}).is_ok &&
        bytes::append_usize(ref mut out,jobs::percent(state)).is_ok && bytes::append(ref mut out,b"% · ").is_ok &&
        bytes::append_usize(ref mut out,state.done).is_ok && bytes::append(ref mut out,b" / ").is_ok &&
        bytes::append_usize(ref mut out,state.total).is_ok && bytes::append(ref mut out,b" 바이트 · ").is_ok &&
        progress::elapsed(ref mut out,jobs::elapsed(state,reader::ticks())) &&
        ui::text(window,105,bytes::view(ref out)) && ui::position(window,jobs::percent(state));
}
@effects(filesystem,clock)
pub fn Result<i32,u32> run(ref u8[] path,ptr mut void window,ref mut publication::State protection){
    let input=switch(reader::open(path)){case .ok(v):{v}case .err(e):{return .err(e);}};
    var jobs::State state=jobs::begin(reader::ticks());var usize displayed=state.started;var usize lines=0;
    var bytes::Buffer buffer=bytes::empty();if(!bytes::resize_zeroed(ref mut buffer,65536).is_ok){return .err(5);}
    if(!present(window,ref state)){return .err(5);}
    while(true){
        let event=ui::poll_now(window);
        let key=if(event==6){domain::key_action(ui::field(window,1),ui::field(window,2))}else{0};
        if(event==3 || (event==1 && ui::field(window,0)==107) || key==4 || (key==1 && ui::field(window,3)==107)){
            jobs::cancel(ref mut state);return .err(8);
        }
        if((key==2 || key==3) && !ui::focus(window,key==3)){return .err(5);}
        if((event==4 || event==5) && !ui::layout(window,event)){return .err(5);}
        let action=jobs::poll(ref input,ref mut state,bytes::view_mut(ref mut buffer));
        if(action==4){return .err(state.error);}
        if(action==2){lines+=domain::newlines(bytes::view(ref buffer)[..state.count]);}
        let now=reader::ticks();
        if(now-displayed>=100 || action==5){if(!present(window,ref state)){return .err(5);}displayed=now;}
        if(action==5){break;}if(action==0){reader::wait(ref input);}
    }
    var bytes::Buffer out=bytes::empty();
    if(!bytes::append(ref mut out,b"읽은 크기: ").is_ok || !bytes::append_usize(ref mut out,state.done).is_ok ||
        !bytes::append(ref mut out,b" 바이트\r\nLF 줄바꿈: ").is_ok || !bytes::append_usize(ref mut out,lines).is_ok ||
        !ui::text(window,104,bytes::view(ref out)) || !ui::announce(window)){return .err(5);}
    protection.identity=fs::FileId{.volume_low=state.first.volume_low,.volume_high=state.first.volume_high,.file_low_0=state.first.file_low_0,.file_low_1=state.first.file_low_1,.file_high_0=state.first.file_high_0,.file_high_1=state.first.file_high_1};protection.protect=true;
    return .ok(0);
}
