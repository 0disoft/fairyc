module linux_test_capture;
import linux_run_capture as capture;
import linux_command as command;
import std::bytes as bytes;
import std::vec as v;
@effects(filesystem,environment,console) @link_name("fairyc_private_linux_test_capture_open_v1")
unsafe extern C fn u32 open(ptr u8 data,usize size,usize argc,ptr u8 cwd,usize cwd_size,u32 mode,ptr u8 path,usize path_size,ptr mut ptr mut void output);
@effects(filesystem) @link_name("fairyc_private_linux_capture_drop_v1") unsafe extern C fn u32 drop(ptr mut void raw);
fn capture::Problem fail(u32 kind,u32 code){return capture::Problem{.kind,.os_code=code,.argument=0,.offset=0};}
// 0 inherits caller stdin, 1 supplies EOF, 2 reopens the bounded private replay file.
@effects(filesystem,environment,console,clock)
pub fn Result<capture::Captured,capture::Problem> run_for(ref u8[] program,ref v::Vec<bytes::Buffer> args,ref u8[] cwd,u32 timeout,u32 mode,ref u8[] path){
    if(timeout==0 || timeout>60000 || mode>2){return .err(fail(1,0));}
    let argv=switch(command::build(program,args)){case .ok(p):{p}case .err(_):{return .err(fail(1,0));}};
    let directory=switch(command::path(cwd)){case .ok(p):{p}case .err(_):{return .err(fail(1,0));}};
    var bytes::Buffer replay=bytes::empty();if(mode==2){replay=switch(command::path(path)){case .ok(p):{p}case .err(_):{return .err(fail(1,0));}};}
    var bytes::Buffer out=bytes::empty();var bytes::Buffer err=bytes::empty();
    if(!bytes::resize_zeroed(ref mut out,65536).is_ok || !bytes::resize_zeroed(ref mut err,65536).is_ok){return .err(fail(3,0));}
    var ptr mut void raw=null;
    let code=unsafe{open(command::view(ref argv) as ptr u8,command::view(ref argv).len,command::count(ref argv),bytes::view(ref directory) as ptr u8,bytes::len(ref directory),mode,bytes::view(ref replay) as ptr u8,bytes::len(ref replay),ref mut raw as ptr mut ptr mut void)};
    if(code!=0){unsafe{discard drop(raw);};return .err(fail(4,code));}
    return capture::collect_owned(raw,timeout,out,err);
}
