module linux_process;
import linux_command as command;
import std::bytes as bytes;
import std::vec as v;

@effects(filesystem,environment,console) @link_name("fairyc_private_linux_process_start_v1")
unsafe extern C fn u32 start(ptr u8 data,usize size,usize argc,ptr u8 cwd,usize cwd_size,ptr mut u32 pid);
@effects(filesystem) @link_name("fairyc_private_linux_process_wait_v1")
unsafe extern C fn u32 wait(u32 pid,ptr mut u32 kind,ptr mut u32 value);
@effects(filesystem) @link_name("fairyc_private_linux_process_discard_v1")
unsafe extern C fn u32 discard_child(u32 pid);

// Kinds 1/2/3 retain command input/bound/allocation failures; 4 start, 5 wait, 6 ABI.
pub struct Problem { u32 kind; u32 os_code; usize argument; usize offset; }
// Outcome kind 1 is ordinary exit (0..255); kind 2 is a terminating signal.
pub struct Outcome { u32 kind; u32 value; u32 pid; }
@effects(filesystem) resource Running { u32 pid; cleanup { let u32 dropped=unsafe { discard_child(this.pid) }; } }
fn Problem fail(u32 kind,u32 os_code,usize argument,usize offset) { return Problem{.kind,.os_code,.argument,.offset}; }
fn Problem encoded(command::Problem value) { return fail(value.kind,0,value.argument,value.offset); }

// Internal blocking primitive: inherits environment and standard streams.
// Deadline/process-tree policy and public CLI dispatch are separate slices.
@effects(filesystem,environment,console)
pub fn Result<Outcome,Problem> run(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory) {
    let argv=switch(command::build(program,arguments)) { case .ok(p): { p } case .err(e): { return .err(encoded(e)); } };
    let cwd=switch(command::path(directory)) { case .ok(p): { p } case .err(e): { return .err(encoded(e)); } };
    var u32 pid=0;
    let status=unsafe { start(command::view(ref argv) as ptr u8,command::view(ref argv).len,command::count(ref argv),bytes::view(ref cwd) as ptr u8,bytes::len(ref cwd),ref mut pid as ptr mut u32) };
    if(status!=0) { return .err(fail(4,status,0,0)); }
    if(pid==0 || pid>2147483647) { return .err(fail(6,0,0,0)); }
    var Running child=Running{.pid};
    var u32 kind=0; var u32 value=0;
    let collected=unsafe { wait(pid,ref mut kind as ptr mut u32,ref mut value as ptr mut u32) };
    if(collected!=0) { return .err(fail(5,collected,0,0)); }
    child.pid=0;
    if(!(kind==1 && value<=255 || kind==2 && value>0 && value<=64)) { return .err(fail(6,0,0,0)); }
    return .ok(Outcome{.kind,.value,.pid});
}
