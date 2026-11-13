module application;
import hash_file as hash;
import sha256_stream as sha;
import ui as ui;
import reader as reader;
import jobs as jobs;
import saving as saving;
import comparison as comparison;
import checksum as checksum;
import checksum_ui as checksum_ui;
import std::fs as fs;
import std::bytes as bytes;

pub fn bool error(ptr mut void window,u32 code){
    if(code==13){return ui::status(window,b"체크섬 파일 형식을 확인하세요. 한 항목만 지원해요.",2);}
    if(code==11){return ui::status(window,b"계산 중 원본이 변경됐거나 확인할 수 없어요. 다시 계산하세요.",2);}
    if(code==10){return ui::status(window,b"임시 파일 정리에 실패했어요. 저장 위치를 확인하세요.",2);}
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
    var sha::State state=sha::empty();var bytes::Buffer buffer=bytes::empty();
    if(!bytes::resize_zeroed(ref mut buffer,65536).is_ok){return .err(5);}
    let started=ui::ticks();var usize displayed=started;var jobs::State work=jobs::begin(started);
    if(!ui::progress(window,0,0,0)){return .err(5);}
    while(true) {
        let event=ui::poll(window,0);
        if(event==3 || event==7){jobs::cancel(ref mut work);return .err(work.error);}
        if((event==4 || event==5) && !ui::layout(window,event)){return .err(5);}
        let action=jobs::poll(ref input,ref mut work,bytes::view_mut(ref mut buffer));
        if(action==4){return .err(work.error);}
        if(action==2){
            if(!sha::update(ref mut state,bytes::view(ref buffer)[..work.count])){return .err(4);}
        }
        let now=ui::ticks();
        if(now-displayed>=100 || action==5){
            if(!ui::progress(window,work.done,work.total,jobs::elapsed(ref work,now))){return .err(5);}displayed=now;
        }
        if(action==5){break;}
        if(action==0){reader::wait(ref input);}
    }
    if(state.length!=work.total){return .err(11);}
    let identity=fs::FileId{.volume_low=work.first.volume_low,.volume_high=work.first.volume_high,
        .file_low_0=work.first.file_low_0,.file_low_1=work.first.file_low_1,.file_high_0=work.first.file_high_0,.file_high_1=work.first.file_high_1};
    return .ok(hash::Digest{.words=sha::finish(ref state),.length=state.length,.identity});
}

fn bool comparison_report(ptr mut void window,ref u8[] path,ref hash::Digest value,ref u8[] actual,ref mut bytes::Buffer report){
    if(!hash::report(path,value,report)){return false;}
    var u8[1024] input=[0;1024];var u8[64] expected=[0;64];
    let count=ui::expected(window,ref mut input);if(count>1024){return false;}
    let ref u8[] raw=ref input;
    if(!comparison::normalize(raw[..count],ref mut expected)){return true;}
    return comparison::append_report(report,ref expected,bytes::equal(actual,ref expected));
}

@effects(filesystem,clock)
pub fn i32 run(ref u8[] initial) {
    var bytes::Buffer path=bytes::empty();var bytes::Buffer selection=bytes::empty();
    var bytes::Buffer report=bytes::empty();var bytes::Buffer hex=bytes::empty();
    if(!bytes::resize_zeroed(ref mut selection,131072).is_ok){return 1;}
    if(initial.len>=131072){return 1;}
    let length=initial.len;
    for(var usize i=0;i<length;i+=1){bytes::view_mut(ref mut selection)[i]=initial[i];}
    var bool startup=length>0;
    let window=ui::open();if(window==null){return 1;}
    let bool initialized=ui::text(window,206,b"파일을 선택하세요") && ui::text(window,207,b"SHA-256") &&
        ui::text(window,202,b"파일 열기(&O)") && ui::text(window,204,b"결과 저장(&S)") && ui::text(window,208,b"복사(&C)") &&
        ui::text(window,212,b"취소(&X)") && ui::enable(window,212,false) && ui::visible(window,212,false) &&
        ui::text(window,215,b"예상 SHA-256(&E)") && ui::name(window,214,b"비교할 예상 SHA-256") &&
        ui::text(window,217,b"체크섬 열기(&L)") &&
        ui::compare(window,b"",false) &&
        ui::name(window,201,b"파일 경로") && ui::name(window,203,b"SHA-256 해시") && ui::name(window,213,b"전체 진행률") &&
        ui::text(window,209,b"선택한 파일의 해시를 계산해요.") && ui::status(window,b"파일 선택 대기",0) &&
        ui::enable(window,204,false) && ui::enable(window,208,false) && ui::position(window,0) && ui::layout(window,0);
    if(!initialized){ui::destroy(window);return 1;}ui::show(window);
    var bool ready=false;var i32 result=0;
    var hash::Digest last=hash::Digest{.words=[0;8],.length=0,.identity=fs::FileId{.volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0}};
    while(true) {
        let event=if(startup){12}else{ui::poll(window,1)};if(event==3){break;}
        if(event==4 || event==5){if(!ui::layout(window,event)){result=1;break;}}
        if(event==11 && !ui::compare(window,bytes::view(ref hex),ready)){result=1;break;}
        if(event==14){
            let chosen=ui::choose_checksum(window,bytes::view_mut(ref mut selection));
            if(chosen>0 && chosen<131072){
                if(!ui::busy(window,true) || !ui::enable(window,204,false) || !ui::enable(window,208,false) || !ui::status(window,b"체크섬 읽는 중…",0)){result=1;break;}
                var u8[64] expected=[0;64];
                switch(checksum_ui::load(bytes::view(ref selection)[..chosen],window,ref mut expected)){
                    case .ok(_):{if(!ui::text(window,214,ref expected) || !ui::status(window,b"✓ 예상 체크섬 불러옴",1)){result=1;break;}}
                    case .err(code):{if(!error(window,code)){result=1;break;}}
                };
                if(!ui::busy(window,false) || !ui::enable(window,204,ready) || !ui::enable(window,208,ready) || !ui::compare(window,bytes::view(ref hex),ready)){result=1;break;}
            }else if(chosen>131072){let bool shown=ui::status(window,b"체크섬 선택창을 열 수 없어요.",2);}
        }
        if(event==1 || event==12) {
            let chosen=if(startup){startup=false;length}else{
                if(event==12){ui::dropped(window,bytes::view_mut(ref mut selection))}else{ui::choose(window,false,bytes::view_mut(ref mut selection))}
            };
            if(chosen>0 && chosen<131072) {
                ready=false;
                let bool disabled=ui::busy(window,true) && ui::enable(window,204,false) && ui::enable(window,208,false);
                if(!disabled || !bytes::clear(ref mut path).is_ok || !bytes::append(ref mut path,bytes::view(ref selection)[..chosen]).is_ok){result=1;break;}
                let bool shown=ui::selected(window,bytes::view(ref path)) && ui::text(window,203,b"") &&
                    ui::text(window,209,b"") && ui::position(window,0) && ui::status(window,b"해시 계산 중…",0);
                if(!shown || !ui::compare(window,b"",false)){result=1;break;}
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
                if(!enabled || !ui::compare(window,bytes::view(ref hex),ready)){result=1;break;}
            }else if(chosen>131072){let bool shown=ui::status(window,if(event==12){b"파일을 하나씩 놓아주세요. 경로도 확인하세요."}else{b"파일 선택창을 열 수 없어요."},2);}
        }
        if(event==2 && ready) {
            if(!comparison_report(window,bytes::view(ref path),ref last,bytes::view(ref hex),ref mut report)){result=1;break;}
            let chosen=ui::choose(window,true,bytes::view_mut(ref mut selection));
            if(chosen>0 && chosen<131072) {
                if(checksum::extension(bytes::view(ref selection)[..chosen]) && !checksum::format(bytes::view(ref path),ref last,ref mut report)){
                    let bool shown=error(window,13);continue;
                }
                if(!ui::busy(window,true) || !ui::enable(window,204,false) || !ui::enable(window,208,false) ||
                    !ui::status(window,b"결과 저장 중…",0)){result=1;break;}
                switch(saving::save(bytes::view(ref selection)[..chosen],bytes::view(ref path),ref last,bytes::view(ref report),window)) {
                    case .ok(_): {let bool shown=ui::status(window,b"✓ 결과 저장 완료",1);}
                    case .err(code): {let bool shown=if(code==8){ui::status(window,b"저장 취소됨",0)}else{error(window,code)};}
                };
                if(!ui::busy(window,false) || !ui::enable(window,204,true) || !ui::enable(window,208,true)){result=1;break;}
            }else if(chosen>131072){let bool shown=ui::status(window,b"저장창을 열 수 없어요.",2);}
        }
        if(event==6 && ready) {
            let bool copied=ui::copy(window,bytes::view(ref hex));
            let bool shown=if(copied){ui::status(window,b"✓ 해시 복사 완료",1)}else{ui::status(window,b"복사 실패: 다시 시도하세요.",2)};
        }
    }
    ui::destroy(window);return result;
}
