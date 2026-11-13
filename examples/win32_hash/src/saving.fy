module saving;
import writer as writer;
import publication as publication;
import hash_file as hash;
import ui as ui;
@effects(filesystem,clock)
pub fn Result<i32,u32> save(ref u8[] destination,ref u8[] source,ref hash::Digest digest,ref u8[] data,ptr mut void window){
    if(hash::same_path(source,destination)){return .err(7);}
    let output=switch(writer::open(destination,data)){case .ok(v):{v}case .err(e):{return .err(e);}};
    var publication::State state=publication::begin(digest.identity,true);var bool ui_failed=false;var bool finishing=false;
    while(true){
        let event=ui::poll(window,0);
        if(event==3 || event==7){publication::cancel(ref mut state);}
        if((event==4 || event==5) && !ui::layout(window,event)){ui_failed=true;publication::abort(ref mut state,5);}
        if(event==3){let bool laid=ui::layout(window,0);}
        let action=publication::poll(ref output,ref mut state,destination);
        if(action==1){
            if(!ui::enable(window,212,false) || !ui::status(window,b"저장 마무리 중…",0)){publication::abort(ref mut state,5);}
            else{let bool authorized=publication::authorize(ref mut state);}
        }
        if(state.stage==7 && !finishing){
            let bool shown=ui::enable(window,212,false) && ui::status(window,if(state.error==8){b"저장 취소·정리 중…"}else{b"저장 마무리 중…"},0);finishing=true;
        }
        if(action==2){return if(ui_failed){.err(5)}else{.ok(0)};}
        if(action==4 || action==8){return .err(state.error);}
        writer::wait(ref output);
    }
    return .err(5);
}
