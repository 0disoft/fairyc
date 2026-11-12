module reader;

@repr(C) pub struct Observation {
    u32 phase;u32 error;u32 kind;u32 directory;usize count;usize size;
    u32 volume_low;u32 volume_high;u32 file_low_0;u32 file_low_1;u32 file_high_0;u32 file_high_1;
}
@effects(filesystem) unsafe extern C fn ptr mut void hash_reader_open(ptr u8 path,usize length);
@effects(filesystem) unsafe extern C fn void hash_reader_snapshot(ptr mut void context,ptr mut void output);
@effects(filesystem) unsafe extern C fn i32 hash_reader_copy(ptr mut void context,ptr mut u8 output,usize capacity);
@effects(filesystem) unsafe extern C fn i32 hash_reader_next(ptr mut void context);
@effects(filesystem,clock) unsafe extern C fn void hash_reader_wait(ptr mut void context);
@effects(filesystem) unsafe extern C fn void hash_reader_drop(ptr mut void context);
@effects(filesystem) unsafe extern C fn i32 hash_reader_available();

@effects(filesystem) pub resource Reader {
    ptr mut void handle;
    cleanup { unsafe{hash_reader_drop(this.handle)}; }
}
@effects(filesystem) pub fn Result<Reader,u32> open(ref u8[] path){
    if(unsafe{hash_reader_available()}==0){return .err(9);}
    let handle=unsafe{hash_reader_open(path as ptr u8,path.len)};
    return if(handle==null){.err(5)}else{.ok(Reader{.handle})};
}
@effects(filesystem) pub fn Observation observe(ref Reader value){
    var Observation output=Observation{.phase=0,.error=0,.kind=0,.directory=0,.count=0,.size=0,
        .volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0};
    unsafe{hash_reader_snapshot(value.handle,ref mut output as ptr mut void)};
    return output;
}
@effects(filesystem) pub fn bool copy(ref Reader value,ref mut u8[] output){return unsafe{hash_reader_copy(value.handle,output as ptr mut u8,output.len)}==0;}
@effects(filesystem) pub fn bool next(ref Reader value){return unsafe{hash_reader_next(value.handle)}==0;}
@effects(filesystem,clock) pub fn void wait(ref Reader value){unsafe{hash_reader_wait(value.handle)};}
