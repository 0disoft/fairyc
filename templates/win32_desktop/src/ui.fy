module ui;
unsafe extern C fn i32 desktop_system_dpi();
unsafe extern C fn ptr mut void desktop_open(i32 width,i32 height);
unsafe extern C fn i32 desktop_control(ptr mut void window,u32 kind,u32 id);
unsafe extern C fn i32 desktop_text(ptr mut void window,u32 id,ptr u8 text,usize length);
unsafe extern C fn usize desktop_read(ptr mut void window,u32 id,ptr mut u8 output,usize capacity);
unsafe extern C fn i32 desktop_name(ptr mut void window,u32 id,ptr u8 text,usize length);
unsafe extern C fn i32 desktop_announce(ptr mut void window,u32 id);
unsafe extern C fn i32 desktop_poll(ptr mut void window);
unsafe extern C fn i32 desktop_field(ptr mut void window,u32 axis);
unsafe extern C fn i32 desktop_extent(ptr mut void window,u32 axis);
unsafe extern C fn i32 desktop_apply_dpi(ptr mut void window);
unsafe extern C fn i32 desktop_frame(ptr mut void window,i32 width,i32 height,i32 font_height);
unsafe extern C fn i32 desktop_move(ptr mut void window,u32 id,i32 x,i32 y,i32 width,i32 height);
unsafe extern C fn i32 desktop_redraw(ptr mut void window);
unsafe extern C fn i32 desktop_focus_next(ptr mut void window,u32 reverse);
unsafe extern C fn void desktop_show(ptr mut void window);
unsafe extern C fn void desktop_destroy(ptr mut void window);

pub fn i32 scale(i32 value,i32 dpi){return (value*dpi+48)/96;}
pub fn ptr mut void open(){let dpi=unsafe{desktop_system_dpi()};return if(dpi<96 || dpi>960){null}else{unsafe{desktop_open(scale(560,dpi),scale(300,dpi))}};}
pub fn bool text(ptr mut void window,u32 id,ref u8[] value){return unsafe{desktop_text(window,id,value as ptr u8,value.len)}==0;}
fn bool name(ptr mut void window,u32 id,ref u8[] value){return unsafe{desktop_name(window,id,value as ptr u8,value.len)}==0;}
pub fn usize read(ptr mut void window,ref mut u8[] output){return unsafe{desktop_read(window,102,output as ptr mut u8,output.len)};}
pub fn i32 poll(ptr mut void window){return unsafe{desktop_poll(window)};}
pub fn i32 field(ptr mut void window,u32 axis){return unsafe{desktop_field(window,axis)};}
pub fn bool focus(ptr mut void window,bool reverse){return unsafe{desktop_focus_next(window,if(reverse){1}else{0})}==0;}
pub fn bool announce(ptr mut void window){return unsafe{desktop_announce(window,105)}==0;}
pub fn void show(ptr mut void window){unsafe{desktop_show(window)};}
pub fn void destroy(ptr mut void window){unsafe{desktop_destroy(window)};}
pub fn bool layout(ptr mut void window,i32 event){
    if(event==5 && unsafe{desktop_apply_dpi(window)}!=0){return false;}
    let width=unsafe{desktop_extent(window,0)};let height=unsafe{desktop_extent(window,1)};let dpi=unsafe{desktop_extent(window,2)};
    if(width<0 || height<0 || dpi<96 || dpi>960){return false;}if(width==0 || height==0){return true;}
    let w=if(width>scale(640,dpi)){scale(640,dpi)}else{width};let h=if(height>scale(300,dpi)){scale(300,dpi)}else{height};
    let x=(width-w)/2;let y=(height-h)/2;let pad=scale(18,dpi);let button=scale(100,dpi);let gap=scale(10,dpi);
    return unsafe{desktop_frame(window,scale(400,dpi),scale(280,dpi),scale(15,dpi))}==0 &&
        unsafe{desktop_move(window,101,x+pad,y+pad,w-pad*2,scale(24,dpi))}==0 &&
        unsafe{desktop_move(window,102,x+pad,y+scale(50,dpi),w-pad*2-button-gap,scale(32,dpi))}==0 &&
        unsafe{desktop_move(window,103,x+w-pad-button,y+scale(50,dpi),button,scale(32,dpi))}==0 &&
        unsafe{desktop_move(window,104,x+pad,y+scale(102,dpi),w-pad*2,scale(112,dpi))}==0 &&
        unsafe{desktop_move(window,105,x+pad,y+scale(232,dpi),w-pad*2,scale(30,dpi))}==0 && unsafe{desktop_redraw(window)}==0;
}
pub fn bool initialize(ptr mut void window){
    return unsafe{desktop_control(window,3,101)}==0 && unsafe{desktop_control(window,1,102)}==0 &&
        unsafe{desktop_control(window,2,103)}==0 && unsafe{desktop_control(window,4,104)}==0 && unsafe{desktop_control(window,3,105)}==0 &&
        text(window,101,b"입력(&I)") && text(window,103,b"실행(&R)") && text(window,105,b"준비") &&
        name(window,102,b"입력 내용") && name(window,104,b"실행 결과") && name(window,105,b"작업 상태") && layout(window,0);
}
