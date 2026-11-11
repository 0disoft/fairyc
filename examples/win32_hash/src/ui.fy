module ui;
import std::bytes as bytes;
unsafe extern C fn i32 hash_ui_system_dpi();
unsafe extern C fn ptr mut void hash_ui_open(i32 width,i32 height);
unsafe extern C fn i32 hash_ui_poll(ptr mut void window,i32 wait);
unsafe extern C fn i32 hash_ui_text(ptr mut void window,u32 id,ptr u8 text,usize length);
unsafe extern C fn i32 hash_ui_enable(ptr mut void window,u32 id,u32 enabled);
unsafe extern C fn i32 hash_ui_extent(ptr mut void window,u32 axis);
unsafe extern C fn i32 hash_ui_move(ptr mut void window,u32 id,i32 x,i32 y,i32 width,i32 height);
unsafe extern C fn i32 hash_ui_layout(ptr mut void window,i32 width,i32 height,i32 font);
unsafe extern C fn i32 hash_ui_accept_dpi(ptr mut void window);
unsafe extern C fn i32 hash_ui_tone(ptr mut void window,u32 tone);
unsafe extern C fn i32 hash_ui_copy(ptr mut void window,ptr u8 text,usize length);
unsafe extern C fn usize hash_ui_dialog(ptr mut void window,u32 save,ptr u8 initial,usize length,ptr mut u8 output,usize capacity);
unsafe extern C fn void hash_ui_show(ptr mut void window);
unsafe extern C fn void hash_ui_destroy(ptr mut void window);

pub fn i32 scale(i32 value,i32 dpi){return (value*dpi+48)/96;}
pub fn ptr mut void open(){let dpi=unsafe{hash_ui_system_dpi()};if(dpi<96 || dpi>960){return null;}return unsafe{hash_ui_open(scale(600,dpi),scale(280,dpi))};}
pub fn i32 poll(ptr mut void window,i32 wait){return unsafe{hash_ui_poll(window,wait)};}
pub fn bool text(ptr mut void window,u32 id,ref u8[] value){return unsafe{hash_ui_text(window,id,value as ptr u8,value.len)}==0;}
pub fn bool enable(ptr mut void window,u32 id,bool enabled){return unsafe{hash_ui_enable(window,id,if(enabled){1}else{0})}==0;}
pub fn usize choose(ptr mut void window,bool save,ref mut u8[] output){let initial=if(save){b"sha256.json"}else{b""};return unsafe{hash_ui_dialog(window,if(save){1}else{0},initial as ptr u8,initial.len,output as ptr mut u8,output.len)};}
pub fn void show(ptr mut void window){unsafe{hash_ui_show(window)};}
pub fn void destroy(ptr mut void window){unsafe{hash_ui_destroy(window)};}
pub fn bool status(ptr mut void window,ref u8[] value,u32 tone){
    return text(window,205,value) && unsafe{hash_ui_tone(window,tone)}==0;
}
pub fn bool copy(ptr mut void window,ref u8[] value){
    return unsafe{hash_ui_copy(window,value as ptr u8,value.len)}==0;
}
pub fn bool selected(ptr mut void window,ref u8[] path){
    var usize start=0;
    let length=path.len;
    for(var usize i=0;i<length;i+=1){if(path[i]==47 || path[i]==92){start=i+1;}}
    return text(window,206,path[start..]) && text(window,201,path);
}
pub fn bool count(ptr mut void window,usize length){
    var bytes::Buffer label=bytes::empty();
    return bytes::append_usize(ref mut label,length).is_ok && bytes::append(ref mut label,b" 바이트").is_ok &&
        text(window,209,bytes::view(ref label));
}
pub fn bool digest(ptr mut void window,ref u8[] hex){
    if(hex.len!=64){return false;}
    var bytes::Buffer display=bytes::empty();
    return bytes::append(ref mut display,hex[..32]).is_ok && bytes::append(ref mut display,b"\r\n").is_ok &&
        bytes::append(ref mut display,hex[32..]).is_ok && text(window,203,bytes::view(ref display));
}
pub fn bool layout(ptr mut void window,i32 event){
    if(event==5 && unsafe{hash_ui_accept_dpi(window)}!=0){return false;}
    let dpi=unsafe{hash_ui_extent(window,2)};let width=unsafe{hash_ui_extent(window,0)};
    let client_height=unsafe{hash_ui_extent(window,1)};
    if(dpi<96 || dpi>960 || width<0 || client_height<0){return false;}
    if(width==0 || client_height==0){return true;}
    let form_width=if(width>scale(720,dpi)){scale(720,dpi)}else{width};
    let form_height=if(client_height>scale(280,dpi)){scale(280,dpi)}else{client_height};
    let origin_x=(width-form_width)/2;let origin_y=(client_height-form_height)/2;
    let inset=scale(18,dpi);let button=scale(110,dpi);let copy_width=scale(76,dpi);let gap=scale(10,dpi);
    let height=scale(32,dpi);let footer=origin_y+form_height-inset-height;let file_x=origin_x+scale(62,dpi);
    let left=origin_x+inset;let right=origin_x+form_width-inset;
    let file_width=right-file_x-button-gap;
    return unsafe{hash_ui_layout(window,scale(520,dpi),scale(276,dpi),scale(14,dpi))}==0 &&
        unsafe{hash_ui_move(window,211,left,origin_y+scale(24,dpi),scale(30,dpi),scale(38,dpi))}==0 &&
        unsafe{hash_ui_move(window,206,file_x,origin_y+scale(18,dpi),file_width,scale(24,dpi))}==0 &&
        unsafe{hash_ui_move(window,201,file_x,origin_y+scale(45,dpi),file_width,scale(22,dpi))}==0 &&
        unsafe{hash_ui_move(window,209,file_x,origin_y+scale(70,dpi),file_width,scale(20,dpi))}==0 &&
        unsafe{hash_ui_move(window,202,right-button,origin_y+scale(29,dpi),button,height)}==0 &&
        unsafe{hash_ui_move(window,210,left,origin_y+scale(101,dpi),right-left,scale(1,dpi))}==0 &&
        unsafe{hash_ui_move(window,207,left,origin_y+scale(114,dpi),right-left,scale(22,dpi))}==0 &&
        unsafe{hash_ui_move(window,203,left,origin_y+scale(142,dpi),right-left,scale(70,dpi))}==0 &&
        unsafe{hash_ui_move(window,208,right-button-gap-copy_width,footer,copy_width,height)}==0 &&
        unsafe{hash_ui_move(window,204,right-button,footer,button,height)}==0 &&
        unsafe{hash_ui_move(window,205,left,footer-scale(3,dpi),right-left-button-copy_width-gap*2,scale(40,dpi))}==0;
}
