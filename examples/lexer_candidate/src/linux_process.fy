module linux_process;
import linux_command as command;
import std::bytes as bytes;
import std::vec as v;

@effects(filesystem,environment,console) @link_name("fairyc_private_linux_process_start_v1")
unsafe extern C fn u32 start(ptr u8 data,usize size,usize argc,ptr u8 cwd,usize cwd_size,ptr mut u32 pid);
@effects(filesystem) @link_name("fairyc_private_linux_process_wait_v1")
unsafe extern C fn u32 wait(u32 pid,ptr mut u32 kind,ptr mut u32 value);
@effects(filesystem) @link_name("fairyc_private_linux_process_wait_for_v1")
unsafe extern C fn u32 wait_for(u32 pid,u32 timeout_ms,ptr mut u32 kind,ptr mut u32 value);
@effects(filesystem) @link_name("fairyc_private_linux_process_discard_v1")
unsafe extern C fn u32 discard_child(u32 pid);

// Kinds 1/2/3 retain command input/bound/allocation failures; 4 start, 5 wait, 6 ABI.
pub struct Problem { u32 kind; u32 os_code; usize argument; usize offset; }
// Outcome 1 is exit (0..255), 2 is signal, 3 is timeout (value zero).
pub struct Outcome { u32 kind; u32 value; u32 pid; }
@effects(filesystem) resource Running { u32 pid; cleanup { let u32 dropped=unsafe { discard_child(this.pid) }; } }
fn Problem fail(u32 kind,u32 os_code,usize argument,usize offset) { return Problem{.kind,.os_code,.argument,.offset}; }
fn Problem encoded(command::Problem value) { return fail(value.kind,0,value.argument,value.offset); }

// Internal blocking primitive: inherits environment and standard streams.
// Each child owns a process group. Public CLI dispatch remains separate.
@effects(filesystem,environment,console)
pub fn Result<Outcome,Problem> run(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory) {
    return execute(program,arguments,directory,0);
}
// Bounds post-start exit observation. Launch and cleanup may take additional time.
@effects(filesystem,environment,console)
pub fn Result<Outcome,Problem> run_for(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout_ms) {
    if(timeout_ms==0 || timeout_ms>86400000) { return .err(fail(1,0,0,0)); }
    return execute(program,arguments,directory,timeout_ms);
}
@effects(filesystem,environment,console)
fn Result<Outcome,Problem> execute(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout_ms) {
    let argv=switch(command::build(program,arguments)) { case .ok(p): { p } case .err(e): { return .err(encoded(e)); } };
    let cwd=switch(command::path(directory)) { case .ok(p): { p } case .err(e): { return .err(encoded(e)); } };
    var u32 pid=0;
    let status=unsafe { start(command::view(ref argv) as ptr u8,command::view(ref argv).len,command::count(ref argv),bytes::view(ref cwd) as ptr u8,bytes::len(ref cwd),ref mut pid as ptr mut u32) };
    if(status!=0) { return .err(fail(4,status,0,0)); }
    if(pid<=1 || pid>2147483647) { return .err(fail(6,0,0,0)); }
    var Running child=Running{.pid};
    var u32 kind=0; var u32 value=0;
    let collected=if(timeout_ms==0) { unsafe { wait(pid,ref mut kind as ptr mut u32,ref mut value as ptr mut u32) } } else { unsafe { wait_for(pid,timeout_ms,ref mut kind as ptr mut u32,ref mut value as ptr mut u32) } };
    if(collected!=0) { return .err(fail(5,collected,0,0)); }
    child.pid=0;
    if(!(kind==1 && value<=255 || kind==2 && value>0 && value<=64 || timeout_ms!=0 && kind==3 && value==0)) { return .err(fail(6,0,0,0)); }
    return .ok(Outcome{.kind,.value,.pid});
}
