module application;
import hash_file as hash;
import sha256_stream as sha;
import ui as ui;
import reader as reader;
import std::fs as fs;
import std::bytes as bytes;

pub fn bool error(ptr mut void window,u32 code){
    if(code==9){return ui::status(window,b"이전 파일 읽기가 아직 끝나지 않았어요. 잠시 후 다시 시도하세요.",2);}
    if(code==7){return ui::status(window,b"원본 파일에는 덮어쓸 수 없어요.",2);}
    if(code==6){return ui::status(window,b"저장 실패: 다른 위치를 선택하세요.",2);}
    if(code==5){return ui::status(window,b"메모리가 부족해요.",2);}
    if(code==3){return ui::status(window,b"파일 크기 또는 종류를 확인하세요.",2);}
    if(code==8){return ui::status(window,b"취소됨",0);}
    return ui::status(window,b"파일을 읽을 수 없어요.",2);
}

@effects(filesystem,clock)
pub fn Result<hash::Digest,u32> compute(ref u8[] path,ptr mut void window) {
    let input=switch(reader::open(path)){case .ok(v):{v}case .err(e):{return .err(e);}};
    var usize total=0;var bool opened=false;
    var sha::State state=sha::empty();var bytes::Buffer buffer=bytes::empty();
    if(!bytes::resize_zeroed(ref mut buffer,65536).is_ok){return .err(5);}
    let started=ui::ticks();var usize displayed=started;
    if(!ui::progress(window,0,total,0)){return .err(5);}
    var observed=reader::observe(ref input);
    while(true) {
        let event=ui::poll(window,0);if(event==3 || event==7){return .err(8);}
        if((event==4 || event==5) && !ui::layout(window,event)){return .err(5);}
        observed=reader::observe(ref input);
        if(observed.phase==4){return .err(2);}
        if(observed.phase==1){
            if(observed.kind!=1 || observed.directory!=0){return .err(3);}
            total=observed.size;opened=true;
            if(!reader::next(ref input)){return .err(2);}
        }
        if(observed.phase==2){
            if(!opened || observed.count>65536 || !reader::copy(ref input,bytes::view_mut(ref mut buffer))){return .err(2);}
            let ref u8[] data=bytes::view(ref buffer);
            if(!sha::update(ref mut state,data[..observed.count])){return .err(4);}
            if(!reader::next(ref input)){return .err(2);}
        }
        let now=ui::ticks();
        if(now-displayed>=100 || observed.phase==3) {
            if(!ui::progress(window,state.length,total,now-started)){return .err(5);}displayed=now;
        }
        if(observed.phase==3){break;}
        if(observed.phase==0){reader::wait(ref input);}
    }
    if(!opened || state.length!=total || observed.size!=total){return .err(3);}
    let identity=fs::FileId{.volume_low=observed.volume_low,.volume_high=observed.volume_high,
        .file_low_0=observed.file_low_0,.file_low_1=observed.file_low_1,.file_high_0=observed.file_high_0,.file_high_1=observed.file_high_1};
    return .ok(hash::Digest{.words=sha::finish(ref state),.length=state.length,.identity});
}

@effects(filesystem,clock)
pub fn i32 run() {
    var bytes::Buffer path=bytes::empty();var bytes::Buffer selection=bytes::empty();
    var bytes::Buffer report=bytes::empty();var bytes::Buffer hex=bytes::empty();
    if(!bytes::resize_zeroed(ref mut selection,131072).is_ok){return 1;}
    let window=ui::open();if(window==null){return 1;}
    let bool initialized=ui::text(window,206,b"파일을 선택하세요") && ui::text(window,207,b"SHA-256") &&
        ui::text(window,202,b"파일 열기(&O)") && ui::text(window,204,b"결과 저장(&S)") && ui::text(window,208,b"복사(&C)") &&
        ui::text(window,212,b"취소(&X)") && ui::enable(window,212,false) && ui::visible(window,212,false) &&
        ui::name(window,201,b"파일 경로") && ui::name(window,203,b"SHA-256 해시") && ui::name(window,213,b"전체 진행률") &&
        ui::text(window,209,b"선택한 파일의 해시를 계산해요.") && ui::status(window,b"파일 선택 대기",0) &&
        ui::enable(window,204,false) && ui::enable(window,208,false) && ui::position(window,0) && ui::layout(window,0);
    if(!initialized){ui::destroy(window);return 1;}ui::show(window);
    var bool ready=false;var i32 result=0;
    var hash::Digest last=hash::Digest{.words=[0;8],.length=0,.identity=fs::FileId{.volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0}};
    while(true) {
        let event=ui::poll(window,1);if(event==3){break;}
        if(event==4 || event==5){if(!ui::layout(window,event)){result=1;break;}}
        if(event==1) {
            let chosen=ui::choose(window,false,bytes::view_mut(ref mut selection));
            if(chosen>0 && chosen<131072) {
                ready=false;
                let bool disabled=ui::busy(window,true) && ui::enable(window,204,false) && ui::enable(window,208,false);
                if(!disabled || !bytes::clear(ref mut path).is_ok || !bytes::append(ref mut path,bytes::view(ref selection)[..chosen]).is_ok){result=1;break;}
                let bool shown=ui::selected(window,bytes::view(ref path)) && ui::text(window,203,b"") &&
                    ui::text(window,209,b"") && ui::position(window,0) && ui::status(window,b"해시 계산 중…",0);
                if(!shown){result=1;break;}
                let started=ui::ticks();
                switch(compute(bytes::view(ref path),window)) {
                    case .ok(value): {
                        last=value;
                        if(!bytes::clear(ref mut hex).is_ok || !hash::append_hex(ref last,ref mut hex) ||
                            !hash::report(bytes::view(ref path),ref last,ref mut report) || !ui::digest(window,bytes::view(ref hex)) ||
                            !ui::count(window,last.length) || !ui::position(window,100) ||
                            !ui::elapsed_status(window,b"✓ 계산 완료",ui::ticks()-started,1)){result=1;break;}
                        ready=true;
                    }
                    case .err(code): {
                        let bool shown=if(code==8){ui::elapsed_status(window,b"취소됨",ui::ticks()-started,0)}else{error(window,code)};
                        if(!shown){result=1;break;}
                    }
                };
                let bool enabled=ui::busy(window,false) && ui::enable(window,204,ready) && ui::enable(window,208,ready);
                if(!enabled){result=1;break;}
            }else if(chosen>131072){let bool shown=ui::status(window,b"파일 선택창을 열 수 없어요.",2);}
        }
        if(event==2 && ready) {
            let chosen=ui::choose(window,true,bytes::view_mut(ref mut selection));
            if(chosen>0 && chosen<131072) {
                switch(hash::save(bytes::view(ref selection)[..chosen],bytes::view(ref path),ref last,bytes::view(ref report))) {
                    case .ok(_): {let bool shown=ui::status(window,b"✓ 결과 저장 완료",1);}
                    case .err(code): {let bool shown=error(window,code);}
                };
            }else if(chosen>131072){let bool shown=ui::status(window,b"저장창을 열 수 없어요.",2);}
        }
        if(event==6 && ready) {
            let bool copied=ui::copy(window,bytes::view(ref hex));
            let bool shown=if(copied){ui::status(window,b"✓ 해시 복사 완료",1)}else{ui::status(window,b"복사 실패: 다시 시도하세요.",2)};
        }
    }
    ui::destroy(window);return result;
}
