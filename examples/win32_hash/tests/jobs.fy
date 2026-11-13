module tests::jobs;
import jobs as jobs;
import reader as reader;
@test fn void completion_cancel_and_source_validation(){
    var reader::Observation value=reader::empty();value.phase=1;value.kind=1;value.size=4;
    var jobs::State state=jobs::begin(100);
    assert(jobs::accept(ref mut state,value)==1);
    value.phase=2;value.count=4;assert(jobs::accept(ref mut state,value)==2);
    assert(jobs::percent(ref state)==99);
    value.phase=3;assert(jobs::accept(ref mut state,value)==3);
    value.phase=5;assert(jobs::accept(ref mut state,value)==5);
    assert(jobs::percent(ref state)==100);jobs::cancel(ref mut state);assert(state.phase==2);
    state=jobs::begin(100);jobs::cancel(ref mut state);assert(jobs::accept(ref mut state,value)==8);assert(state.done==0);
    assert(jobs::elapsed(ref state,150)==50);assert(jobs::elapsed(ref state,99)==0);
    state=jobs::begin(0);value.phase=1;value.count=0;assert(jobs::accept(ref mut state,value)==1);
    value.phase=2;value.count=4;assert(jobs::accept(ref mut state,value)==2);
    value.phase=3;value.changed=1;assert(jobs::accept(ref mut state,value)==4);assert(state.error==11);
    value.changed=0;value.phase=5;assert(jobs::accept(ref mut state,value)==4);
}
