module tests::linux_fetch_deadline;
import linux_fetch_deadline as deadline;
@link_name("fetch_clock_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("fetch_clock_probe_calls") unsafe extern C fn u32 calls();
@test fn void observes_exact_deadline_and_overflow_boundaries(){
    assert(deadline::duration()==120000 && deadline::remaining_at(17,17)==120000 && deadline::remaining_at(17,16)==0);
    assert(deadline::remaining_at(17,120016)==1 && deadline::remaining_at(17,120017)==0 && deadline::remaining_at(17,120018)==0);
    let usize maximum=18446744073709551615;assert(deadline::remaining_at(maximum-119999,maximum)==1 && deadline::remaining_at(maximum-120000,maximum)==0 && deadline::remaining_at(maximum,0)==0);
}
@test @effects(clock) fn void fails_closed_on_clock_errors_or_reverse_time(){
    unsafe{mode(0);}let started=switch(deadline::start()){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(deadline::remaining(started)==119999 && !deadline::expired(started) && unsafe{calls()}==3);
    unsafe{mode(1);}assert(switch(deadline::start()){case .ok(_):{false}case .err(p):{p==5}} && unsafe{calls()}==1);
    var u32 value=2;while(value<=5){unsafe{mode(value);}let current=switch(deadline::start()){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(deadline::remaining(current)==if(value==5){1}else{0});assert(unsafe{calls()}==2);value+=1;}
}
