module linux_capture;
import linux_command as command;
import std::bytes as bytes;
import std::vec as v;

@effects(filesystem,environment,console) @link_name("fairyc_private_linux_capture_open_v1")
unsafe extern C fn u32 open(ptr u8 data,usize size,usize argc,ptr u8 cwd,usize cwd_size,ptr mut ptr mut void output);
@effects(filesystem,clock) @link_name("fairyc_private_linux_capture_collect_v1")
unsafe extern C fn u32 collect(ptr mut void raw,u32 timeout_ms,ptr mut u8 output,usize output_limit,ptr mut usize output_len,ptr mut u8 error,usize error_limit,ptr mut usize error_len,ptr mut u32 kind,ptr mut u32 value,ptr mut u32 pid);
@effects(filesystem) @link_name("fairyc_private_linux_capture_drop_v1")
unsafe extern C fn u32 drop(ptr mut void raw);

pub struct Problem { u32 kind; u32 os_code; usize argument; usize offset; }
// Kind 1=exit, 2=signal, 3=timeout, 4=output limit (value 1 stdout, 2 stderr).
pub resource Captured { u32 kind; u32 value; u32 pid; bytes::Buffer output; bytes::Buffer error; cleanup {} }
@effects(filesystem) resource Session { ptr mut void raw; cleanup { let u32 dropped=unsafe { drop(this.raw) }; } }
fn Problem fail(u32 kind,u32 os_code) { return Problem{.kind,.os_code,.argument=0,.offset=0}; }
fn Problem encoded(command::Problem e) { return Problem{.kind=e.kind,.os_code=0,.argument=e.argument,.offset=e.offset}; }
pub fn u32 kind(ref Captured value) { return value.kind; }
pub fn u32 value(ref Captured captured) { return captured.value; }
pub fn u32 pid(ref Captured value) { return value.pid; }
pub fn ref u8[] stdout(ref Captured value) { return bytes::view(ref value.output); }
pub fn ref u8[] stderr(ref Captured value) { return bytes::view(ref value.error); }

// Per-stream requested storage shares a 1MiB bound. The wait/drain budget starts after launch.
@effects(filesystem,environment,console,clock)
pub fn Result<Captured,Problem> run_for(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout_ms,usize output_limit,usize error_limit) {
    if(timeout_ms==0 || timeout_ms>86400000) { return .err(fail(1,0)); }
    if(output_limit>1048576 || error_limit>1048576-output_limit) { return .err(fail(2,0)); }
    let argv=switch(command::build(program,arguments)) { case .ok(p): { p } case .err(e): { return .err(encoded(e)); } };
    let cwd=switch(command::path(directory)) { case .ok(p): { p } case .err(e): { return .err(encoded(e)); } };
    var bytes::Buffer out=bytes::empty(); var bytes::Buffer err=bytes::empty();
    if(!bytes::resize_zeroed(ref mut out,output_limit).is_ok || !bytes::resize_zeroed(ref mut err,error_limit).is_ok) { return .err(fail(3,0)); }
    var ptr mut void raw=null;
    let opened=unsafe { open(command::view(ref argv) as ptr u8,command::view(ref argv).len,command::count(ref argv),bytes::view(ref cwd) as ptr u8,bytes::len(ref cwd),ref mut raw as ptr mut ptr mut void) };
    var Session session=Session{.raw};
    if(opened!=0) { return .err(fail(4,opened)); }
    if(session.raw==null) { return .err(fail(6,0)); }
    var usize out_len=0; var usize err_len=0;
    var u32 kind=0; var u32 value=0; var u32 pid=0;
    let collected=unsafe { collect(session.raw,timeout_ms,bytes::view_mut(ref mut out) as ptr mut u8,output_limit,ref mut out_len as ptr mut usize,bytes::view_mut(ref mut err) as ptr mut u8,error_limit,ref mut err_len as ptr mut usize,ref mut kind as ptr mut u32,ref mut value as ptr mut u32,ref mut pid as ptr mut u32) };
    if(collected!=0) { return .err(fail(5,collected)); }
    if(pid<=1 || pid>2147483647 || out_len>output_limit || err_len>error_limit || !(kind==1 && value<=255 || kind==2 && value>0 && value<=64 || kind==3 && value==0 || kind==4 && (value==1 || value==2))) { return .err(fail(6,0)); }
    if(!bytes::truncate(ref mut out,out_len).is_ok || !bytes::truncate(ref mut err,err_len).is_ok) { return .err(fail(6,0)); }
    return .ok(Captured{.kind,.value,.pid,.output=out,.error=err});
}
