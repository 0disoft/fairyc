module native_ui;
unsafe extern C fn ptr mut void fairy_ui_child(ptr mut void window,u32 id);
unsafe extern C fn i32 fairy_ui_text(ptr mut void control,ptr u8 text,usize length);
unsafe extern C fn usize fairy_ui_read(ptr mut void control,ptr mut u8 output,usize capacity);
unsafe extern C fn i32 fairy_ui_enable(ptr mut void control,u32 enabled);
unsafe extern C fn i32 fairy_ui_focus(ptr mut void control);
unsafe extern C fn i32 fairy_ui_move(ptr mut void control,i32 x,i32 y,i32 width,i32 height);
unsafe extern C fn i32 fairy_ui_extent(ptr mut void window,u32 axis);
pub fn ptr mut void child(ptr mut void window,u32 id){return unsafe{fairy_ui_child(window,id)};}
pub fn bool text(ptr mut void control,ref u8[] value){return unsafe{fairy_ui_text(control,value as ptr u8,value.len)}==0;}
pub fn Result<usize,u32> read(ptr mut void control,ref mut u8[] output){let size=unsafe{fairy_ui_read(control,output as ptr mut u8,output.len)};return if(size<=output.len){.ok(size)}else{.err(1)};}
pub fn bool enable(ptr mut void control,bool enabled){return unsafe{fairy_ui_enable(control,if(enabled){1}else{0})}==0;}
pub fn bool focus(ptr mut void control){return unsafe{fairy_ui_focus(control)}==0;}
pub fn bool move(ptr mut void control,i32 x,i32 y,i32 width,i32 height){return unsafe{fairy_ui_move(control,x,y,width,height)}==0;}
pub fn i32 extent(ptr mut void window,u32 axis){return unsafe{fairy_ui_extent(window,axis)};}
