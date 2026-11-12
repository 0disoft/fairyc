module writer;
@repr(C) pub struct WriterObservation {
    u32 phase;u32 error;u32 exists;
    u32 volume_low;u32 volume_high;u32 file_low_0;u32 file_low_1;u32 file_high_0;u32 file_high_1;
}
@effects(filesystem) unsafe extern C fn ptr mut void hash_writer_open(ptr u8 path,usize length,ptr u8 data,usize count);
@effects(filesystem) unsafe extern C fn i32 hash_writer_submit(ptr mut void context,u32 operation,ptr u8 path,usize length);
@effects(filesystem) unsafe extern C fn void hash_writer_snapshot(ptr mut void context,ptr mut void output);
@effects(filesystem) unsafe extern C fn void hash_writer_finish(ptr mut void context,u32 cancel);
@effects(filesystem,clock) unsafe extern C fn void hash_writer_wait(ptr mut void context);
@effects(filesystem) unsafe extern C fn void hash_writer_drop(ptr mut void context);
@effects(filesystem) pub resource Writer {
    ptr mut void handle;
    cleanup { unsafe{hash_writer_drop(this.handle)}; }
}
@effects(filesystem) pub fn Result<Writer,u32> open(ref u8[] path,ref u8[] data){
    let handle=unsafe{hash_writer_open(path as ptr u8,path.len,data as ptr u8,data.len)};
    return if(handle==null){.err(5)}else{.ok(Writer{.handle})};
}
@effects(filesystem) pub fn WriterObservation observe(ref Writer value){
    var WriterObservation out=WriterObservation{.phase=0,.error=0,.exists=0,.volume_low=0,.volume_high=0,
        .file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0};
    unsafe{hash_writer_snapshot(value.handle,ref mut out as ptr mut void)};return out;
}
@effects(filesystem) pub fn bool submit(ref Writer value,u32 operation,ref u8[] path){
    return unsafe{hash_writer_submit(value.handle,operation,path as ptr u8,path.len)}==0;
}
@effects(filesystem) pub fn void finish(ref Writer value,bool cancel){unsafe{hash_writer_finish(value.handle,if(cancel){1}else{0})};}
@effects(filesystem,clock) pub fn void wait(ref Writer value){unsafe{hash_writer_wait(value.handle)};}
