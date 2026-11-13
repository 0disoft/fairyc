module publication;
import writer as writer;
import std::bytes as bytes;
import std::fs as fs;

// 0 check, 1 create, 2 write, 3 flush, 4 recheck, 5 authorize,
// 6 commit, 7 cleanup, 8 terminal. No UI or product callback is retained.
pub struct State { u32 stage;u32 error;usize attempt;bool waiting;bool protect;fs::FileId identity; }
pub fn State begin(fs::FileId identity,bool protect){return State{.stage=0,.error=0,.attempt=0,.waiting=false,.protect,.identity};}
pub fn State unprotected(){return begin(fs::FileId{.volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0},false);}
pub fn void abort(ref mut State state,u32 error){if(state.stage<6){state.error=error;state.stage=7;state.waiting=false;}}
pub fn void cancel(ref mut State state){abort(state,8);}
pub fn bool authorize(ref mut State state){if(state.stage!=5){return false;}state.stage=6;return true;}
pub fn i32 outcome(ref State state){return if(state.stage!=8){0}else{if(state.error==0){2}else{if(state.error==8){8}else{4}}};}
pub fn void accept(ref mut State state,writer::WriterObservation observed){
    if(state.stage==8 || observed.phase==0){return;}
    if(state.stage==7){if(observed.phase==2){if(observed.error!=0){state.error=10;}state.stage=8;}return;}
    if(!state.waiting || observed.phase!=1){return;}state.waiting=false;
    if(observed.error!=0){
        if(state.stage==1 && (observed.error==80 || observed.error==183) && state.attempt<31){state.attempt+=1;return;}
        state.error=6;state.stage=7;return;
    }
    if(state.protect && (state.stage==0 || state.stage==4) && observed.exists!=0){
        let id=fs::FileId{.volume_low=observed.volume_low,.volume_high=observed.volume_high,
            .file_low_0=observed.file_low_0,.file_low_1=observed.file_low_1,.file_high_0=observed.file_high_0,.file_high_1=observed.file_high_1};
        if(fs::file_id_equal(ref id,ref state.identity)){state.error=7;state.stage=7;return;}
    }
    state.stage+=1;
}
// 0 pending, 1 awaiting explicit commit, 2 completed, 4 failed, 8 cancelled.
@effects(filesystem)
pub fn i32 poll(ref writer::Writer output,ref mut State state,ref u8[] destination){
    if(state.stage==8){return outcome(state);}if(state.stage==5){return 1;}
    if(state.stage==7){
        if(!state.waiting){writer::finish(output,state.error==8);state.waiting=true;}
        accept(state,writer::observe(output));return outcome(state);
    }
    if(!state.waiting){
        let u32 kind=if(state.stage==0 || state.stage==4){1}else{if(state.stage==1){2}else{if(state.stage==2){3}else{if(state.stage==3){4}else{5}}}};
        var bytes::Buffer temporary=bytes::empty();
        if(kind==2 && (!bytes::append(ref mut temporary,destination).is_ok || !bytes::append(ref mut temporary,b".fairy-save-").is_ok ||
            !bytes::append_usize(ref mut temporary,state.attempt).is_ok || !bytes::append(ref mut temporary,b".tmp").is_ok)){abort(state,5);return 0;}
        if(!writer::submit(output,kind,bytes::view(ref temporary))){state.error=5;state.stage=7;state.waiting=false;return 0;}
        state.waiting=true;
    }
    accept(state,writer::observe(output));return if(state.stage==5){1}else{0};
}
