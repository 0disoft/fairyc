module ui;
import window_metrics as metrics;
unsafe extern C fn i32 desktop_system_dpi();
unsafe extern C fn ptr mut void desktop_open(i32 width,i32 height);
unsafe extern C fn i32 desktop_control(ptr mut void window,u32 kind,u32 id);
unsafe extern C fn i32 desktop_text(ptr mut void window,u32 id,ptr u8 text,usize length);
unsafe extern C fn usize desktop_read(ptr mut void window,u32 id,ptr mut u8 output,usize capacity);
unsafe extern C fn i32 desktop_name(ptr mut void window,u32 id,ptr u8 text,usize length);
unsafe extern C fn i32 desktop_announce(ptr mut void window,u32 id);
unsafe extern C fn i32 desktop_live(ptr mut void window,u32 id,u32 setting,u32 notify);
unsafe extern C fn i32 desktop_poll(ptr mut void window);
unsafe extern C fn i32 desktop_poll_now(ptr mut void window);
unsafe extern C fn i32 desktop_enable(ptr mut void window,u32 id,u32 enabled);
unsafe extern C fn i32 desktop_focus_to(ptr mut void window,u32 id);
unsafe extern C fn i32 desktop_progress(ptr mut void window,usize percent);
unsafe extern C fn usize desktop_choose(ptr mut void window,ptr mut u8 output,usize capacity);
unsafe extern C fn usize desktop_choose_save(ptr mut void window,ptr mut u8 output,usize capacity);
unsafe extern C fn i32 desktop_field(ptr mut void window,u32 axis);
unsafe extern C fn i32 desktop_extent(ptr mut void window,u32 axis);
unsafe extern C fn i32 desktop_apply_dpi(ptr mut void window);
unsafe extern C fn i32 desktop_frame(ptr mut void window,i32 width,i32 height,i32 font_height);
unsafe extern C fn i32 desktop_move(ptr mut void window,u32 id,i32 x,i32 y,i32 width,i32 height);
unsafe extern C fn i32 desktop_redraw(ptr mut void window);
unsafe extern C fn i32 desktop_focus_next(ptr mut void window,u32 reverse);
unsafe extern C fn void desktop_show(ptr mut void window);
unsafe extern C fn void desktop_destroy(ptr mut void window);

pub fn i32 scale(i32 value,i32 dpi){return metrics::scale(value,dpi);}
pub fn ptr mut void open(){let dpi=unsafe{desktop_system_dpi()};return if(!metrics::valid_dpi(dpi)){null}else{unsafe{desktop_open(scale(560,dpi),scale(350,dpi))}};}
pub fn bool text(ptr mut void window,u32 id,ref u8[] value){return unsafe{desktop_text(window,id,value as ptr u8,value.len)}==0;}
fn bool name(ptr mut void window,u32 id,ref u8[] value){return unsafe{desktop_name(window,id,value as ptr u8,value.len)}==0;}
pub fn usize read(ptr mut void window,ref mut u8[] output){return unsafe{desktop_read(window,102,output as ptr mut u8,output.len)};}
pub fn i32 poll(ptr mut void window){return unsafe{desktop_poll(window)};}
pub fn i32 poll_now(ptr mut void window){return unsafe{desktop_poll_now(window)};}
pub fn usize choose(ptr mut void window,ref mut u8[] output){return unsafe{desktop_choose(window,output as ptr mut u8,output.len)};}
pub fn usize choose_save(ptr mut void window,ref mut u8[] output){return unsafe{desktop_choose_save(window,output as ptr mut u8,output.len)};}
pub fn usize result(ptr mut void window,ref mut u8[] output){return unsafe{desktop_read(window,104,output as ptr mut u8,output.len)};}
pub fn bool cancel_enabled(ptr mut void window,bool active){return unsafe{desktop_enable(window,107,if(active){1}else{0})}==0;}
pub fn bool position(ptr mut void window,usize percent){return percent<=100 && unsafe{desktop_progress(window,percent)}==0;}
pub fn bool working(ptr mut void window,bool active){
    return (!active || unsafe{desktop_live(window,105,0,0)}==0) && unsafe{desktop_enable(window,102,if(active){0}else{1})}==0 &&
        unsafe{desktop_enable(window,103,if(active){0}else{1})}==0 && unsafe{desktop_enable(window,106,if(active){0}else{1})}==0 &&
        unsafe{desktop_enable(window,109,if(active){0}else{1})}==0 &&
        cancel_enabled(window,active) && unsafe{desktop_focus_to(window,if(active){107}else{102})}==0;
}
pub fn i32 field(ptr mut void window,u32 axis){return unsafe{desktop_field(window,axis)};}
pub fn bool focus(ptr mut void window,bool reverse){return unsafe{desktop_focus_next(window,if(reverse){1}else{0})}==0;}
pub fn bool announce(ptr mut void window){return unsafe{desktop_announce(window,105)}==0;}
pub fn void show(ptr mut void window){unsafe{desktop_show(window)};}
pub fn void destroy(ptr mut void window){unsafe{desktop_destroy(window)};}
pub fn bool layout(ptr mut void window,i32 event){
    if(event==5 && unsafe{desktop_apply_dpi(window)}!=0){return false;}
    let width=unsafe{desktop_extent(window,0)};let height=unsafe{desktop_extent(window,1)};let dpi=unsafe{desktop_extent(window,2)};
    if(width<0 || height<0 || !metrics::valid_dpi(dpi)){return false;}if(width==0 || height==0){return true;}
    let w=metrics::cap(width,640,dpi);let h=metrics::cap(height,350,dpi);
    let x=metrics::center(width,w);let y=metrics::center(height,h);let pad=scale(18,dpi);let button=scale(100,dpi);let gap=scale(10,dpi);
    return unsafe{desktop_frame(window,scale(400,dpi),scale(330,dpi),scale(15,dpi))}==0 &&
        unsafe{desktop_move(window,101,x+pad,y+pad,w-pad*2,scale(24,dpi))}==0 &&
        unsafe{desktop_move(window,102,x+pad,y+scale(50,dpi),w-pad*2-button-gap,scale(32,dpi))}==0 &&
        unsafe{desktop_move(window,103,x+w-pad-button,y+scale(50,dpi),button,scale(32,dpi))}==0 &&
        unsafe{desktop_move(window,104,x+pad,y+scale(102,dpi),w-pad*2,scale(112,dpi))}==0 &&
        unsafe{desktop_move(window,108,x+pad,y+scale(220,dpi),w-pad*2,scale(5,dpi))}==0 &&
        unsafe{desktop_move(window,106,x+pad,y+scale(238,dpi),scale(120,dpi),scale(32,dpi))}==0 &&
        unsafe{desktop_move(window,109,x+pad+scale(130,dpi),y+scale(238,dpi),scale(100,dpi),scale(32,dpi))}==0 &&
        unsafe{desktop_move(window,107,x+w-pad-button,y+scale(238,dpi),button,scale(32,dpi))}==0 &&
        unsafe{desktop_move(window,105,x+pad,y+scale(282,dpi),w-pad*2,scale(30,dpi))}==0 && unsafe{desktop_redraw(window)}==0;
}
pub fn bool initialize(ptr mut void window){
    return unsafe{desktop_control(window,3,101)}==0 && unsafe{desktop_control(window,1,102)}==0 &&
        unsafe{desktop_control(window,2,103)}==0 && unsafe{desktop_control(window,4,104)}==0 && unsafe{desktop_control(window,3,105)}==0 &&
        unsafe{desktop_control(window,2,106)}==0 && unsafe{desktop_control(window,2,107)}==0 && unsafe{desktop_control(window,5,108)}==0 && unsafe{desktop_control(window,2,109)}==0 &&
        text(window,101,b"입력(&I)") && text(window,103,b"실행(&R)") && text(window,105,b"준비") &&
        text(window,106,b"파일 읽기(&O)") && text(window,107,b"취소(&X)") && text(window,109,b"저장(&S)") && unsafe{desktop_enable(window,107,0)}==0 &&
        name(window,102,b"입력 내용") && name(window,104,b"실행 결과") && name(window,105,b"작업 상태") && layout(window,0);
}
