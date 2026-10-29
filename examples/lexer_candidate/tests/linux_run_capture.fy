module tests::linux_run_capture;
import linux_run_capture as capture;
import std::bytes as bytes;
import std::vec as v;
@link_name("run_capture_mode") unsafe extern C fn void mode(u32 value);
@link_name("run_capture_drops") unsafe extern C fn u32 drops();
fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let p=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(args,p).is_ok);}
@effects(filesystem,environment,console,clock) @test fn void captures_exit_signal_limits_and_authenticated_receipts(){
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"");add(ref mut args,b"a b");
    var u32 i=0;while(i<6){unsafe{mode(i);};let result=switch(capture::run_for(b"app",ref args,b"/tmp",25,3,3)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(capture::pid(ref result)==4321 && bytes::equal(capture::stdout(ref result),b"o\0") && bytes::equal(capture::stderr(ref result),b"\xffe"));
        if(i==0){assert(capture::kind(ref result)==1 && capture::value(ref result)==127 && capture::receipt(ref result).len==0);}
        if(i==1){assert(capture::kind(ref result)==1 && capture::value(ref result)==133 && bytes::equal(capture::receipt(ref result),b"\x05"));}
        if(i==2){assert(capture::kind(ref result)==1 && capture::value(ref result)==133 && capture::receipt(ref result).len==0);}
        if(i==3){assert(capture::kind(ref result)==2 && capture::value(ref result)==9);}
        if(i==4){assert(capture::kind(ref result)==3 && capture::value(ref result)==0);}
        if(i==5){assert(capture::kind(ref result)==4 && capture::value(ref result)==1);}
        assert(unsafe{drops()}==1);i+=1;
    }
    while(i<12){unsafe{mode(i);};assert(!capture::run_for(b"app",ref args,b"/tmp",25,3,3).is_ok);assert(unsafe{drops()}==1);i+=1;}
}
