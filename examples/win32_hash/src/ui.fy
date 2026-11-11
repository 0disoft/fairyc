module ui;
unsafe extern C fn i32 hash_ui_system_dpi();
unsafe extern C fn ptr mut void hash_ui_open(i32 width,i32 height);
unsafe extern C fn i32 hash_ui_poll(ptr mut void window,i32 wait);
unsafe extern C fn i32 hash_ui_text(ptr mut void window,u32 id,ptr u8 text,usize length);
unsafe extern C fn i32 hash_ui_enable(ptr mut void window,u32 id,u32 enabled);
unsafe extern C fn i32 hash_ui_extent(ptr mut void window,u32 axis);
unsafe extern C fn i32 hash_ui_move(ptr mut void window,u32 id,i32 x,i32 y,i32 width,i32 height);
unsafe extern C fn i32 hash_ui_layout(ptr mut void window,i32 width,i32 height,i32 font);
unsafe extern C fn i32 hash_ui_accept_dpi(ptr mut void window);
unsafe extern C fn usize hash_ui_dialog(ptr mut void window,u32 save,ptr u8 initial,usize length,ptr mut u8 output,usize capacity);
unsafe extern C fn void hash_ui_show(ptr mut void window);
unsafe extern C fn void hash_ui_destroy(ptr mut void window);

pub fn i32 scale(i32 value,i32 dpi){return (value*dpi+48)/96;}
pub fn ptr mut void open(){let dpi=unsafe{hash_ui_system_dpi()};if(dpi<96 || dpi>960){return null;}return unsafe{hash_ui_open(scale(640,dpi),scale(260,dpi))};}
pub fn i32 poll(ptr mut void window,i32 wait){return unsafe{hash_ui_poll(window,wait)};}
pub fn bool text(ptr mut void window,u32 id,ref u8[] value){return unsafe{hash_ui_text(window,id,value as ptr u8,value.len)}==0;}
pub fn bool enable(ptr mut void window,u32 id,bool enabled){return unsafe{hash_ui_enable(window,id,if(enabled){1}else{0})}==0;}
pub fn usize choose(ptr mut void window,bool save,ref mut u8[] output){let initial=if(save){b"sha256.json"}else{b""};return unsafe{hash_ui_dialog(window,if(save){1}else{0},initial as ptr u8,initial.len,output as ptr mut u8,output.len)};}
pub fn void show(ptr mut void window){unsafe{hash_ui_show(window)};}
pub fn void destroy(ptr mut void window){unsafe{hash_ui_destroy(window)};}
pub fn bool layout(ptr mut void window,i32 event){
    if(event==5 && unsafe{hash_ui_accept_dpi(window)}!=0){return false;}
    let dpi=unsafe{hash_ui_extent(window,2)};let width=unsafe{hash_ui_extent(window,0)};
    if(dpi<96 || dpi>960 || width<0){return false;}
    let inset=scale(16,dpi);let button=scale(108,dpi);let gap=scale(12,dpi);let height=scale(28,dpi);
    let input=width-inset*2-button-gap;
    return unsafe{hash_ui_layout(window,scale(480,dpi),scale(240,dpi),scale(13,dpi))}==0 &&
        unsafe{hash_ui_move(window,206,inset,scale(14,dpi),width-inset*2,scale(22,dpi))}==0 &&
        unsafe{hash_ui_move(window,201,inset,scale(40,dpi),input,height)}==0 &&
        unsafe{hash_ui_move(window,202,width-inset-button,scale(40,dpi),button,height)}==0 &&
        unsafe{hash_ui_move(window,207,inset,scale(84,dpi),width-inset*2,scale(22,dpi))}==0 &&
        unsafe{hash_ui_move(window,203,inset,scale(110,dpi),width-inset*2,height)}==0 &&
        unsafe{hash_ui_move(window,204,width-inset-button,scale(158,dpi),button,height)}==0 &&
        unsafe{hash_ui_move(window,205,inset,scale(158,dpi),input,height)}==0;
}
