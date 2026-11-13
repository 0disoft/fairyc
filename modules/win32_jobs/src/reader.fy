module reader;

@repr(C) pub struct Observation {
    u32 phase;u32 error;u32 kind;u32 directory;usize count;usize size;
    u32 volume_low;u32 volume_high;u32 file_low_0;u32 file_low_1;u32 file_high_0;u32 file_high_1;
    usize created;usize written;usize changed;
}
@effects(filesystem) unsafe extern C fn ptr mut void job_reader_open(ptr u8 path,usize length);
@effects(filesystem) unsafe extern C fn void job_reader_snapshot(ptr mut void context,ptr mut void output);
@effects(filesystem) unsafe extern C fn i32 job_reader_copy(ptr mut void context,ptr mut u8 output,usize capacity);
@effects(filesystem) unsafe extern C fn i32 job_reader_next(ptr mut void context);
@effects(filesystem,clock) unsafe extern C fn void job_reader_wait(ptr mut void context);
@effects(filesystem) unsafe extern C fn void job_reader_drop(ptr mut void context);
@effects(filesystem) unsafe extern C fn i32 job_reader_available();
@effects(clock) unsafe extern C fn usize job_reader_ticks();
@effects(clock) pub fn usize ticks(){return unsafe{job_reader_ticks()};}
pub fn Observation empty(){return Observation{.phase=0,.error=0,.kind=0,.directory=0,.count=0,.size=0,
    .volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0,.created=0,.written=0,.changed=0};}

@effects(filesystem) pub resource Reader {
    ptr mut void handle;
    cleanup { unsafe{job_reader_drop(this.handle)}; }
}
@effects(filesystem) pub fn Result<Reader,u32> open(ref u8[] path){
    if(unsafe{job_reader_available()}==0){return .err(9);}
    let handle=unsafe{job_reader_open(path as ptr u8,path.len)};
    return if(handle==null){.err(5)}else{.ok(Reader{.handle})};
}
@effects(filesystem) pub fn Observation observe(ref Reader value){
    var Observation output=Observation{.phase=0,.error=0,.kind=0,.directory=0,.count=0,.size=0,
        .volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0,.created=0,.written=0,.changed=0};
    unsafe{job_reader_snapshot(value.handle,ref mut output as ptr mut void)};
    return output;
}
@effects(filesystem) pub fn bool copy(ref Reader value,ref mut u8[] output){return unsafe{job_reader_copy(value.handle,output as ptr mut u8,output.len)}==0;}
@effects(filesystem) pub fn bool next(ref Reader value){return unsafe{job_reader_next(value.handle)}==0;}
@effects(filesystem,clock) pub fn void wait(ref Reader value){unsafe{job_reader_wait(value.handle)};}

@effects(filesystem) unsafe extern C fn ptr mut void job_source_open(ptr u8 path,usize length);
@effects(filesystem) unsafe extern C fn void job_source_snapshot(ptr mut void context,ptr mut void output,u32 refresh);
@effects(filesystem) unsafe extern C fn void job_source_drop(ptr mut void context);
@effects(filesystem) pub resource Guard {
    ptr mut void handle;
    cleanup { unsafe{job_source_drop(this.handle)}; }
}
@effects(filesystem) pub fn Result<Guard,u32> protect(ref u8[] path){
    let handle=unsafe{job_source_open(path as ptr u8,path.len)};
    return if(handle==null){.err(5)}else{.ok(Guard{.handle})};
}
@effects(filesystem) pub fn Observation sample(ref Guard value,u32 refresh){
    var Observation output=Observation{.phase=0,.error=0,.kind=0,.directory=0,.count=0,.size=0,
        .volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0,.created=0,.written=0,.changed=0};
    unsafe{job_source_snapshot(value.handle,ref mut output as ptr mut void,refresh)};return output;
}
pub fn bool unchanged(ref Observation first,ref Observation last){
    return first.kind==last.kind && first.directory==last.directory && first.size==last.size &&
        first.volume_low==last.volume_low && first.volume_high==last.volume_high &&
        first.file_low_0==last.file_low_0 && first.file_low_1==last.file_low_1 &&
        first.file_high_0==last.file_high_0 && first.file_high_1==last.file_high_1 &&
        first.created==last.created && first.written==last.written && first.changed==last.changed;
}
