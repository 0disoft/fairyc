module tests::linux_capture;
import linux_capture as capture;
import std::bytes as bytes;
import std::vec as v;
@link_name("capture_mode") unsafe extern C fn void mode(u32 value);
@link_name("capture_counts") unsafe extern C fn u32 counts();

@test @effects(filesystem,environment,console,clock)
fn void owns_binary_output_and_classifies_limits_failures_and_timeout() {
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();
    let a=switch(bytes::copy(b"")) { case .ok(p): { p } case .err(_): { assert(false);return; } };
    let b=switch(bytes::copy(b"a b")) { case .ok(p): { p } case .err(_): { assert(false);return; } };
    assert(v::push<bytes::Buffer>(ref mut args,a).is_ok);
    assert(v::push<bytes::Buffer>(ref mut args,b).is_ok);
    unsafe { mode(0); }
    assert(!capture::run_for(b"app",ref args,b"/tmp",0,3,3).is_ok);
    assert(!capture::run_for(b"app",ref args,b"/tmp",86400001,3,3).is_ok);
    assert(!capture::run_for(b"app",ref args,b"/tmp",25,1048576,1).is_ok);
    assert(!capture::run_for(b"",ref args,b"/tmp",25,3,3).is_ok);
    switch(capture::run_for(b"app",ref args,b"/t\0mp",25,3,3)) {
        case .ok(_): { assert(false); } case .err(e): { assert(e.kind==1 && e.offset==2); }
    };
    assert(unsafe { counts() }==0);
    var u32 scenario=0;
    while(scenario<15) {
        unsafe { mode(scenario); }
        switch(capture::run_for(b"app",ref args,b"/tmp",25,3,3)) {
            case .ok(value): {
                assert(scenario<5 && capture::pid(ref value)==4321);
                assert(capture::kind(ref value)==if(scenario<3){scenario+1}else{4});
                assert(capture::value(ref value)==if(scenario==0){127}else if(scenario==1){9}else if(scenario==2){0}else{scenario-2});
                assert(capture::stdout(ref value).len==if(scenario==3){3}else{2});
                assert(capture::stderr(ref value).len==if(scenario==4){3}else{2});
                assert(capture::stdout(ref value)[1]==0 && capture::stderr(ref value)[0]==255);
                assert(unsafe { counts() }==111);
            }
            case .err(e): {
                assert(scenario>=5);
                assert(e.kind==if(scenario==5){4}else if(scenario==6){5}else{6});
                if(scenario==5) { assert(e.os_code==2); }
                if(scenario==6) { assert(e.os_code==5); }
            }
        };
        assert(unsafe { counts() }==if(scenario==5 || scenario==10){100}else{111});
        assert(v::len<bytes::Buffer>(ref args)==2 && bytes::equal(bytes::view(v::at<bytes::Buffer>(ref args,1)),b"a b"));
        scenario+=1;
    }
    unsafe { mode(15); }
    switch(capture::run_for(b"app",ref args,b"/tmp",25,0,0)) {
        case .ok(value): { assert(capture::stdout(ref value).len==0 && capture::stderr(ref value).len==0 && capture::kind(ref value)==1 && capture::value(ref value)==0); }
        case .err(_): { assert(false); }
    };
    assert(unsafe { counts() }==111);
}
