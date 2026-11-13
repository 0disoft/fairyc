module application;
import domain as domain;
import ui as ui;
import std::bytes as bytes;
fn bool execute(ptr mut void window){
    var u8[2048] input=[0;2048];let count=ui::read(window,ref mut input);if(count>2048){return false;}
    var bytes::Buffer output=bytes::empty();let ref u8[] value=ref input;
    return domain::transform(value[..count],ref mut output) && ui::text(window,104,bytes::view(ref output)) &&
        ui::text(window,105,b"완료") && ui::announce(window);
}
pub fn i32 run(){
    let window=ui::open();if(window==null){return 1;}
    if(!ui::initialize(window)){ui::destroy(window);return 1;}ui::show(window);
    var i32 result=0;
    while(true){
        let event=ui::poll(window);if(event==3){break;}
        if((event==4 || event==5) && !ui::layout(window,event)){result=1;break;}
        var bool apply=event==1 && ui::field(window,0)==103;
        if(event==6){
            let action=domain::key_action(ui::field(window,1),ui::field(window,2));
            if(action==4){break;}
            if((action==2 || action==3) && !ui::focus(window,action==3)){result=1;break;}
            if(action==1){apply=true;}
        }
        // EN_CHANGE (event 2) is available for your validation/preview policy.
        if(apply && !execute(window)){result=1;break;}
    }
    ui::destroy(window);return result;
}
