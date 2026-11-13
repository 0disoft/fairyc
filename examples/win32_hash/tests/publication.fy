module tests::publication;
import publication as publication;
import writer as writer;
@test fn void commit_cancel_collision_and_cleanup(){
    var publication::State state=publication::unprotected();
    var writer::WriterObservation value=writer::WriterObservation{.phase=1,.error=0,.exists=0,.volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0};
    state.waiting=true;publication::accept(ref mut state,value);assert(state.stage==1);
    state.waiting=true;value.error=80;publication::accept(ref mut state,value);assert(state.stage==1 && state.attempt==1);
    value.error=0;for(var usize i=0;i<4;i+=1){state.waiting=true;publication::accept(ref mut state,value);}
    assert(state.stage==5);assert(publication::authorize(ref mut state));publication::cancel(ref mut state);assert(state.stage==6);
    state.waiting=true;publication::accept(ref mut state,value);assert(state.stage==7);
    value.phase=2;publication::accept(ref mut state,value);assert(publication::outcome(ref state)==2);
    publication::cancel(ref mut state);assert(publication::outcome(ref state)==2);
    state=publication::unprotected();publication::cancel(ref mut state);publication::accept(ref mut state,value);assert(publication::outcome(ref state)==8);
    state=publication::unprotected();state.protect=true;state.waiting=true;value.phase=1;value.exists=1;
    publication::accept(ref mut state,value);assert(state.error==7 && state.stage==7);
    value.phase=2;value.error=5;publication::accept(ref mut state,value);assert(publication::outcome(ref state)==4 && state.error==10);
}
