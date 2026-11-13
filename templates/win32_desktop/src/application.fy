module application;
import domain as domain;
import ui as ui;
import file_job as file_job;
import std::bytes as bytes;
fn bool execute(ptr mut void window){
    var u8[2048] input=[0;2048];let count=ui::read(window,ref mut input);if(count>2048){return false;}
    var bytes::Buffer output=bytes::empty();let ref u8[] value=ref input;
    return domain::transform(value[..count],ref mut output) && ui::text(window,104,bytes::view(ref output)) &&
        ui::text(window,105,b"완료") && ui::announce(window);
}
@effects(filesystem,clock)
fn bool read_file(ptr mut void window){
    var bytes::Buffer path=bytes::empty();if(!bytes::resize_zeroed(ref mut path,131072).is_ok){return false;}
    let count=ui::choose(window,bytes::view_mut(ref mut path));if(count==0){return true;}
    if(count>=131072){return ui::text(window,105,b"파일을 선택할 수 없어요.");}
    if(!ui::working(window,true) || !ui::text(window,104,b"")){return false;}
    switch(file_job::run(bytes::view(ref path)[..count],window)){
        case .ok(_):{}
        case .err(code):{
            if(!ui::text(window,105,if(code==8){b"취소됨"}else{if(code==9){b"이전 작업이 아직 끝나지 않았어요."}else{b"파일을 읽을 수 없어요."}}) || !ui::announce(window)){return false;}
        }
    };
    return ui::working(window,false);
}
@effects(filesystem,clock)
pub fn i32 run(){
    let window=ui::open();if(window==null){return 1;}
    if(!ui::initialize(window)){ui::destroy(window);return 1;}ui::show(window);
    var i32 result=0;
    while(true){
        let event=ui::poll(window);if(event==3){break;}
        if((event==4 || event==5) && !ui::layout(window,event)){result=1;break;}
        var bool apply=event==1 && ui::field(window,0)==103;
        var bool file=event==1 && ui::field(window,0)==106;
        if(event==6){
            let action=domain::key_action(ui::field(window,1),ui::field(window,2));
            if(action==4){break;}
            if((action==2 || action==3) && !ui::focus(window,action==3)){result=1;break;}
            if(action==1){if(ui::field(window,3)==106){file=true;}else{apply=true;}}
        }
        // EN_CHANGE (event 2) is available for your validation/preview policy.
        if(apply && !execute(window)){result=1;break;}
        if(file && !read_file(window)){result=1;break;}
    }
    ui::destroy(window);return result;
}
