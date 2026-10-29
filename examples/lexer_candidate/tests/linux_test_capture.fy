module tests::linux_test_capture;
import linux_test_capture as test_capture;
import linux_run_capture as capture;
import std::bytes as bytes;
import std::vec as v;
@link_name("run_capture_mode") unsafe extern C fn void mode(u32 value);
@link_name("run_capture_drops") unsafe extern C fn u32 drops();
fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let p=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(args,p).is_ok);}
@effects(filesystem,environment,console,clock) @test fn void forwards_stdin_modes_and_owns_failed_open(){
 var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"");add(ref mut args,b"a b");
 var u32 i=0;while(i<3){unsafe{mode(0);};let result=switch(test_capture::run_for(b"app",ref args,b"/tmp",25,i,if(i==2){b"/tmp/replay"}else{b""})){case .ok(p):{p}case .err(_):{assert(false);return;}};
  assert(capture::kind(ref result)==1 && capture::value(ref result)==127 && bytes::equal(capture::stdout(ref result),b"o\0") && unsafe{drops()}==1);i+=1;
 }
 unsafe{mode(0);};assert(!test_capture::run_for(b"app",ref args,b"/tmp",25,3,b"").is_ok && unsafe{drops()}==0);
 assert(!test_capture::run_for(b"app",ref args,b"/tmp",60001,1,b"").is_ok && unsafe{drops()}==0);
 unsafe{mode(11);};assert(!test_capture::run_for(b"app",ref args,b"/tmp",25,1,b"").is_ok && unsafe{drops()}==1);
}
