module jobs;
import reader as reader;
import progress as progress;

// phase: 0 active, 2 complete, 3 cancelled, 4 failed. No HWND, callback or borrow.
pub struct State {
    usize done;usize total;usize count;usize started;
    u32 phase;u32 error;bool opened;bool ending;reader::Observation first;
}
pub fn State begin(usize started){return State{.done=0,.total=0,.count=0,.started,.phase=0,.error=0,.opened=false,.ending=false,.first=reader::empty()};}
pub fn u32 fail(ref mut State state,u32 code){state.phase=4;state.error=code;return 4;}
pub fn void cancel(ref mut State state){if(state.phase==0){state.phase=3;state.error=8;}}
pub fn usize percent(ref State state){
    if(state.phase==2){return 100;}
    let value=progress::percent(state.done,state.total);return if(value>=100){99}else{value};
}
pub fn usize elapsed(ref State state,usize now){return if(now>=state.started){now-state.started}else{0};}
// Pure protocol policy, also usable in headless tests. Terminal states stay sticky.
pub fn u32 accept(ref mut State state,reader::Observation value){
    if(state.phase!=0){return if(state.phase==2){5}else{if(state.phase==3){8}else{4}};}
    if(value.phase==0){return 0;}
    if(value.phase==4){return fail(state,2);}if(value.phase==6){return fail(state,11);}
    if(value.phase==1){
        if(state.opened){return fail(state,2);}
        if(value.kind!=1 || value.directory!=0){return fail(state,3);}
        state.opened=true;state.first=value;state.total=value.size;return 1;
    }
    if(value.phase==2){
        if(!state.opened || state.ending || value.count==0 || value.count>65536 || state.done>state.total){return fail(state,2);}
        if(value.count>state.total-state.done){return fail(state,11);}
        state.count=value.count;state.done+=value.count;return 2;
    }
    if(value.phase==3){
        if(!state.opened || state.ending || state.done!=state.total || !reader::unchanged(ref state.first,ref value)){return fail(state,11);}
        state.ending=true;return 3;
    }
    if(value.phase==5){
        if(!state.ending || state.done!=state.total || !reader::unchanged(ref state.first,ref value)){return fail(state,11);}
        state.phase=2;return 5;
    }
    return fail(state,2);
}
// 0 pending/metadata, 2 owned chunk ready, 4 error, 5 verified done, 8 cancelled.
// A chunk is copied before acknowledging native storage; the worker can read ahead.
@effects(filesystem)
pub fn u32 poll(ref reader::Reader input,ref mut State state,ref mut u8[] output){
    let action=accept(state,reader::observe(input));
    if(action==2 && !reader::copy(input,output)){return fail(state,2);}
    if((action==1 || action==2 || action==3) && !reader::next(input)){return fail(state,2);}
    return if(action==1 || action==3){0}else{action};
}
