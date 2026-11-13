module save_job;
import publication as publication;
import writer as writer;
import ui as ui;
import domain as domain;
@effects(filesystem,clock)
pub fn Result<i32,u32> run(ref u8[] destination,ref u8[] data,ptr mut void window,bool gui,bool cancel_before,publication::State initial){
    let output=switch(writer::open(destination,data)){case .ok(v):{v}case .err(e):{return .err(e);}};
    var publication::State state=initial;
    while(true){
        if(gui){
            let event=ui::poll_now(window);
            let key=if(event==6){domain::key_action(ui::field(window,1),ui::field(window,2))}else{0};
            if(event==3 || (event==1 && ui::field(window,0)==107) || key==4 || (key==1 && ui::field(window,3)==107)){publication::cancel(ref mut state);}
            if((key==2 || key==3) && !ui::focus(window,key==3)){publication::abort(ref mut state,5);}
            if((event==4 || event==5) && !ui::layout(window,event)){publication::abort(ref mut state,5);}
        }
        let action=publication::poll(ref output,ref mut state,destination);
        if(action==1){
            if(cancel_before){publication::cancel(ref mut state);}
            else{
                if(gui && (!ui::cancel_enabled(window,false) || !ui::text(window,105,b"저장 마무리 중…"))){publication::abort(ref mut state,5);}
                else{let bool authorized=publication::authorize(ref mut state);}
            }
        }
        if(action==2){return .ok(0);}if(action==4 || action==8){return .err(state.error);}
        writer::wait(ref output);
    }
    return .err(5);
}
