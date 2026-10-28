module tests::linux_process;
import linux_process as process;
import std::bytes as bytes;
import std::vec as v;
@link_name("probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("probe_counts") unsafe extern C fn u32 counts();
fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value) {
    let copied=switch(bytes::copy(value)) { case .ok(p): { p } case .err(_): { assert(false);return; } };
    assert(v::push<bytes::Buffer>(args,copied).is_ok);
}

@test @effects(filesystem,environment,console)
fn void separates_launch_failures_exit_status_signals_and_cleanup() {
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>(); add(ref mut args,b""); add(ref mut args,b"a b");
    var u32 scenario=0;
    while(scenario<9) {
        unsafe { mode(scenario); }
        switch(process::run(b"app",ref args,b"/tmp")) {
            case .ok(value): {
                assert(scenario<3 && value.pid==4321);
                assert(value.kind==if(scenario==2){2}else{1});
                assert(value.value==if(scenario==0){127}else if(scenario==1){41}else{15});
                assert(unsafe { counts() }==110);
            }
            case .err(e): {
                assert(scenario>=3);
                assert(e.kind==if(scenario<5){4}else if(scenario==5){5}else{6});
                if(scenario<5) { assert(e.os_code==if(scenario==3){2}else{13}); }
                if(scenario==5) { assert(e.os_code==22); }
                assert(unsafe { counts() }==if(scenario<5 || scenario==6){100}else if(scenario==5){111}else{110});
            }
        };
        assert(v::len<bytes::Buffer>(ref args)==2 && bytes::equal(bytes::view(v::at<bytes::Buffer>(ref args,1)),b"a b"));
        scenario+=1;
    }
    unsafe { mode(0); }
    assert(!process::run(b"",ref args,b"/tmp").is_ok);
    switch(process::run(b"app",ref args,b"/t\0mp")) {
        case .ok(_): { assert(false); } case .err(e): { assert(e.kind==1 && e.offset==2); }
    };
    assert(unsafe { counts() }==0);
}
