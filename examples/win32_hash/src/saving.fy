module saving;
import writer as writer;
import hash_file as hash;
import ui as ui;
import std::bytes as bytes;
import std::fs as fs;

// Event/cancel/close policy stays in Fairy. Native code executes one file call
// at a time and never receives a Fairy borrow or a window handle.
@effects(filesystem,clock)
fn u32 await(ref writer::Writer output,ptr mut void window,bool cancellable,bool finished){
    while(true){
        let event=ui::poll(window,0);
        if((event==3 || event==7) && cancellable){return 8;}
        if(event==4 || event==5){if(!ui::layout(window,event)){return 5;}}
        if(event==3){let bool laid=ui::layout(window,0);}
        let observed=writer::observe(output);
        if((finished && observed.phase==2) || (!finished && observed.phase==1)){return 0;}
        writer::wait(output);
    }
    return 5;
}
@effects(filesystem,clock)
fn u32 operation(ref writer::Writer output,ptr mut void window,u32 kind,ref u8[] path){
    if(!writer::submit(output,kind,path)){return 5;}
    return await(output,window,kind!=5,false);
}
@effects(filesystem,clock)
fn u32 check(ref writer::Writer output,ptr mut void window,ref hash::Digest digest){
    let code=operation(output,window,1,b"");if(code!=0){return code;}
    let observed=writer::observe(output);
    if(observed.error!=0){return 6;}
    if(observed.exists!=0){
        let id=fs::FileId{.volume_low=observed.volume_low,.volume_high=observed.volume_high,
            .file_low_0=observed.file_low_0,.file_low_1=observed.file_low_1,.file_high_0=observed.file_high_0,.file_high_1=observed.file_high_1};
        if(fs::file_id_equal(ref id,ref digest.identity)){return 7;}
    }
    return 0;
}
@effects(filesystem,clock)
fn u32 prepared(ref writer::Writer output,ptr mut void window,ref u8[] destination,ref hash::Digest digest){
    let initial=check(output,window,digest);if(initial!=0){return initial;}
    var bytes::Buffer temporary=bytes::empty();var usize index=0;var bool created=false;
    while(index<32){
        if(!bytes::clear(ref mut temporary).is_ok || !bytes::append(ref mut temporary,destination).is_ok ||
            !bytes::append(ref mut temporary,b".fairy-hash-").is_ok || !bytes::append_usize(ref mut temporary,index).is_ok ||
            !bytes::append(ref mut temporary,b".tmp").is_ok){return 5;}
        let code=operation(output,window,2,bytes::view(ref temporary));if(code!=0){return code;}
        let observed=writer::observe(output);
        if(observed.error==0){created=true;break;}
        if(observed.error!=80 && observed.error!=183){return 6;}
        index+=1;
    }
    if(!created){return 6;}
    for(var u32 kind=3;kind<=4;kind+=1){
        let code=operation(output,window,kind,b"");if(code!=0){return code;}
        if(writer::observe(output).error!=0){return 6;}
    }
    let final_check=check(output,window,digest);if(final_check!=0){return final_check;}
    // Commit is the point of no cancellation. Keep pumping on Close until the
    // complete old/new file and owned temporary cleanup have a known outcome.
    let event=ui::poll(window,0);if(event==3 || event==7){return 8;}
    if(!ui::enable(window,212,false) || !ui::status(window,b"저장 마무리 중…",0)){return 5;}
    let code=operation(output,window,5,b"");if(code!=0){return code;}
    return if(writer::observe(output).error==0){0}else{6};
}
@effects(filesystem,clock)
pub fn Result<i32,u32> save(ref u8[] destination,ref u8[] source,ref hash::Digest digest,ref u8[] data,ptr mut void window){
    if(hash::same_path(source,destination)){return .err(7);}
    let output=switch(writer::open(destination,data)){case .ok(v):{v}case .err(e):{return .err(e);}};
    let code=prepared(ref output,window,destination,digest);
    let bool shown=ui::enable(window,212,false) && ui::status(window,if(code==8){b"저장 취소·정리 중…"}else{b"저장 마무리 중…"},0);
    writer::finish(ref output,code==8);
    let waited=await(ref output,window,false,true);
    if(writer::observe(ref output).error!=0){return .err(10);}
    if(code!=0){return .err(code);}return .ok(0);
}
