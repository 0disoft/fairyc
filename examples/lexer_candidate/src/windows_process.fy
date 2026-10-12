module windows_process;
import windows_command as command;
import std::bytes as bytes;
import std::vec as v;

@repr(C) struct Startup{u32 size;ptr mut void reserved;ptr mut void desktop;ptr mut void title;u32 x;u32 y;u32 width;u32 height;u32 columns;u32 rows;u32 fill;u32 flags;u32 show_reserved;ptr mut void reserved_bytes;ptr mut void input;ptr mut void output;ptr mut void error;}
@repr(C) struct Information{ptr mut void process;ptr mut void thread;u32 pid;u32 tid;}
@effects(filesystem) @link_name("fairyc_private_windows_job_v1") unsafe extern C fn ptr mut void create_job();
@effects(filesystem) @link_name("fairyc_private_windows_job_info_v1") unsafe extern C fn i32 set_job(ptr mut void job,u32 kind,ptr u8 data,u32 size);
@effects(filesystem,console) @link_name("fairyc_private_windows_create_v1") unsafe extern C fn i32 create(ptr u8 app,ptr mut u8 line,ptr u8 cwd,i32 inherit,u32 flags,ptr mut void startup,ptr mut void information);
@effects(filesystem,console) @link_name("fairyc_private_windows_create_env_v1") unsafe extern C fn i32 create_environment(ptr u8 app,ptr mut u8 line,ptr u8 cwd,i32 inherit,u32 flags,ptr mut void startup,ptr mut void information,ptr u8 environment);
@effects(filesystem) @link_name("fairyc_private_windows_assign_v1") unsafe extern C fn i32 assign(ptr mut void job,ptr mut void process);
@effects(filesystem) @link_name("fairyc_private_windows_resume_v1") unsafe extern C fn u32 resume(ptr mut void thread);
@effects(filesystem) @link_name("fairyc_private_windows_wait_v1") unsafe extern C fn u32 wait(ptr mut void process,u32 timeout);
@effects(filesystem) @link_name("fairyc_private_windows_exit_v1") unsafe extern C fn i32 get_exit(ptr mut void process,ptr mut u32 result);
@effects(filesystem) @link_name("fairyc_private_windows_kill_job_v1") unsafe extern C fn i32 kill_job(ptr mut void job,u32 code);
@effects(filesystem) @link_name("fairyc_private_windows_kill_process_v1") unsafe extern C fn i32 kill_process(ptr mut void process,u32 code);
@effects(filesystem) @link_name("fairyc_private_windows_close_v1") unsafe extern C fn i32 close(ptr mut void handle);
@effects(console) @link_name("fairyc_private_windows_std_handle_v1") unsafe extern C fn ptr mut void std_handle(u32 kind);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();
@link_name("fairyc_private_windows_startup_size_v1") unsafe extern C fn usize startup_size();
@link_name("fairyc_private_windows_job_size_v1") unsafe extern C fn usize job_size();

@effects(filesystem) resource Handle{ptr mut void raw;cleanup{if(this.raw!=null){unsafe{discard close(this.raw);};}}}
pub resource Running{Handle job;Handle process;Handle thread;u32 pid;cleanup{}}
pub struct Handles{ptr mut void input;ptr mut void output;ptr mut void error;}
// 1 input, 2 allocation, 3 native ABI, 4 creation, 5 job setup, 6 resume,
// 7 wait/result, 8 deadline. Native error and the created PID are retained.
pub struct Problem{u32 kind;u32 os_code;u32 pid;}
pub struct Outcome{u32 exit_code;u32 pid;}
fn Problem fail(u32 kind,u32 os_code,u32 pid){return Problem{.kind,.os_code,.pid};}
fn Problem encoded_error(command::Problem p){return fail(if(p.kind==3){2}else{1},0,0);}
@effects(console) pub fn Handles inherited_handles(){return Handles{.input=unsafe{std_handle(4294967286)},.output=unsafe{std_handle(4294967285)},.error=unsafe{std_handle(4294967284)}};}
pub fn u32 pid(ref Running running){return running.pid;}
@effects(filesystem) pub fn void terminate(ref Running running){unsafe{discard kill_job(running.job.raw,1);discard wait(running.process.raw,2000);};}
@effects(filesystem) pub fn Result<bool,Problem> poll(ref Running running,u32 timeout){let waited=unsafe{wait(running.process.raw,timeout)};return if(waited==0){.ok(true)}else if(waited==258){.ok(false)}else{.err(fail(7,unsafe{last_error()},running.pid))};}
@effects(filesystem) pub fn Result<Outcome,Problem> result(ref Running running){switch(poll(running,0)){case .ok(done):{if(!done){return .err(fail(7,0,running.pid));}}case .err(e):{return .err(e);}};var u32 exit_code=0;if(unsafe{get_exit(running.process.raw,ref mut exit_code as ptr mut u32)}==0){return .err(fail(7,unsafe{last_error()},running.pid));}return .ok(Outcome{.exit_code,.pid=running.pid});}
@effects(filesystem,console)
pub fn Result<Outcome,Problem> run(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout){if(timeout==0 || timeout>600000){return .err(fail(1,0,0));}
    return launch(program,arguments,directory,timeout,b"");
}
@effects(filesystem,console)
pub fn Result<Outcome,Problem> run_environment(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout,ref u8[] environment){
    if(environment.len<4 || environment.len>1048576 || environment.len%2!=0 || environment[environment.len-1]!=0 || environment[environment.len-2]!=0 || environment[environment.len-3]!=0 || environment[environment.len-4]!=0){return .err(fail(1,0,0));}
    return launch(program,arguments,directory,timeout,environment);
}
@effects(filesystem,console)
fn Result<Outcome,Problem> launch(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout,ref u8[] environment){if(timeout==0 || timeout>600000){return .err(fail(1,0,0));}
    let running=switch(start(program,arguments,directory,environment,inherited_handles())){case .ok(p):{p}case .err(e):{return .err(e);}};
    switch(poll(ref running,timeout)){case .ok(done):{if(!done){terminate(ref running);return .err(fail(8,0,running.pid));}}case .err(e):{terminate(ref running);return .err(e);}};return result(ref running);
}
// Ownership is published only after suspended launch, job assignment and resume.
// The caller owns all three handles until collection or an early-return cleanup.
@effects(filesystem,console)
pub fn Result<Running,Problem> start(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,ref u8[] environment,Handles handles){
    if(environment.len!=0 && (environment.len<4 || environment.len>1048576 || environment.len%2!=0 || environment[environment.len-1]!=0 || environment[environment.len-2]!=0 || environment[environment.len-3]!=0 || environment[environment.len-4]!=0)){return .err(fail(1,0,0));}
    let app=switch(command::path(program)){case .ok(p):{p}case .err(e):{return .err(encoded_error(e));}};var bytes::Buffer line=switch(command::build(program,arguments)){case .ok(p):{p}case .err(e):{return .err(encoded_error(e));}};let cwd=switch(command::path(directory)){case .ok(p):{p}case .err(e):{return .err(encoded_error(e));}};
    if(unsafe{startup_size()!=104 || job_size()!=144}){return .err(fail(3,0,0));}var bytes::Buffer limits=bytes::empty();if(!bytes::resize_zeroed(ref mut limits,144).is_ok){return .err(fail(2,0,0));}{let data=bytes::view_mut(ref mut limits);data[17]=32;}
    let job=Handle{.raw=unsafe{create_job()}};if(job.raw==null){return .err(fail(5,unsafe{last_error()},0));}if(unsafe{set_job(job.raw,9,bytes::view(ref limits) as ptr u8,144)}==0){return .err(fail(5,unsafe{last_error()},0));}
    var Startup startup=Startup{.size=104,.reserved=null,.desktop=null,.title=null,.x=0,.y=0,.width=0,.height=0,.columns=0,.rows=0,.fill=0,.flags=257,.show_reserved=0,.reserved_bytes=null,.input=handles.input,.output=handles.output,.error=handles.error};
    var Information information=Information{.process=null,.thread=null,.pid=0,.tid=0};let i32 created=if(environment.len==0){unsafe{create(bytes::view(ref app) as ptr u8,bytes::view_mut(ref mut line) as ptr mut u8,bytes::view(ref cwd) as ptr u8,1,134217732,ref mut startup as ptr mut void,ref mut information as ptr mut void)}}else{unsafe{create_environment(bytes::view(ref app) as ptr u8,bytes::view_mut(ref mut line) as ptr mut u8,bytes::view(ref cwd) as ptr u8,1,134218756,ref mut startup as ptr mut void,ref mut information as ptr mut void,environment as ptr u8)}};if(created==0){return .err(fail(4,unsafe{last_error()},0));}
    let process=Handle{.raw=information.process};let thread=Handle{.raw=information.thread};if(unsafe{assign(job.raw,process.raw)}==0){let error=unsafe{last_error()};unsafe{discard kill_process(process.raw,1);discard wait(process.raw,2000);};return .err(fail(5,error,information.pid));}
    if(unsafe{resume(thread.raw)}==4294967295){let error=unsafe{last_error()};unsafe{discard kill_job(job.raw,1);discard wait(process.raw,2000);};return .err(fail(6,error,information.pid));}
    return .ok(Running{.job,.process,.thread,.pid=information.pid});
}
