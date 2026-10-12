module tests::windows_run_capture;
import windows_run_capture as capture;
import windows_process as child;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;
import std::fs as fs;
import std::text as text;

@effects(filesystem) @link_name("fairyc_private_windows_open_v1") unsafe extern C fn ptr mut void open_process(u32 access,i32 inherit,u32 pid);
@effects(filesystem) @link_name("fairyc_private_windows_wait_v1") unsafe extern C fn u32 wait_process(ptr mut void handle,u32 timeout);
@effects(filesystem) @link_name("fairyc_private_windows_close_v1") unsafe extern C fn i32 close_handle(ptr mut void handle);
@effects(filesystem) @link_name("fairyc_private_windows_handle_count_v1") unsafe extern C fn i32 handle_count(ptr mut u32 count);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();
@effects(console) fn void require(bool ok,u32 step){if(!ok){discard io::write_stderr_u32(step);assert(false);}}
@effects(console) fn void failure(child::Problem e){discard io::write_stderr_u32(e.kind);discard io::write_stderr_u32(e.os_code);assert(false);}
fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let copied=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(args,copied).is_ok);}
fn bytes::Buffer joined(ref u8[] root,ref u8[] file){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,root).is_ok && bytes::append_byte(ref mut out,47).is_ok && bytes::append(ref mut out,file).is_ok);return out;}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] data=[0;4096];let size=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref data;return switch(files::manifest(view[..size])){case .ok(p):{p}case .err(_):{assert(false);return root();}};}
@effects(filesystem) fn u32 handles(){var u32 count=0;assert(unsafe{handle_count(ref mut count as ptr mut u32)}!=0);return count;}
@effects(filesystem) fn bool exited(u32 pid){let raw=unsafe{open_process(1048576,0,pid)};if(raw==null){return unsafe{last_error()}==87;}let done=unsafe{wait_process(raw,2000)}==0;assert(unsafe{close_handle(raw)}!=0);return done;}
@effects(filesystem) fn void reset(ref u8[] path){let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return;}};let file=switch(fs::create_truncate_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(fs::close(file).is_ok);}
@effects(filesystem) fn bool done(ref u8[] path){let data=switch(files::manifest(path)){case .ok(p):{p}case .err(_):{return false;}};return bytes::equal(bytes::view(ref data),b"done");}
@effects(filesystem,console,environment,clock) fn capture::Captured invoke(ref u8[] program,ref v::Vec<bytes::Buffer> args,ref u8[] directory){return switch(capture::run(program,args,directory,2000)){case .ok(p):{p}case .err(e):{failure(e);return invoke(program,args,directory);}};}
fn void pattern(ref u8[] data,ref u8[] expected){var usize i=0;while(i<data.len){assert(data[i]==expected[i%expected.len]);i+=1;}}

@test fn void preserves_signed_main_exit_bits(){assert(capture::exit_bits(0)==0 && capture::exit_bits(5)==5 && capture::exit_bits(2147483647)==2147483647 && capture::exit_bits(2147483648)==-2147483648 && capture::exit_bits(2147483690)==-2147483606 && capture::exit_bits(4294967295)==-1);}
@test @effects(filesystem,console,environment,clock) fn void retains_the_exact_shared_output_limit_and_drains_overflow(){
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");let report=joined(bytes::view(ref directory),b"capture-bulk.done");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"bulk");add(ref mut args,b"32768");add(ref mut args,b"32768");add(ref mut args,bytes::view(ref report));reset(bytes::view(ref report));
    let exact=invoke(bytes::view(ref program),ref args,bytes::view(ref directory));require(capture::status(ref exact)==42 && capture::trap(ref exact)==0 && capture::stdout(ref exact).len==32768 && capture::stderr(ref exact).len==32768 && done(bytes::view(ref report)),1);pattern(capture::stdout(ref exact),b"\0\x01\x02\xff");pattern(capture::stderr(ref exact),b"\xff\0A\n");let before=handles();
    assert(v::clear<bytes::Buffer>(ref mut args)==4);add(ref mut args,b"bulk");add(ref mut args,b"32768");add(ref mut args,b"32769");add(ref mut args,bytes::view(ref report));reset(bytes::view(ref report));switch(capture::run(bytes::view(ref program),ref args,bytes::view(ref directory),2000)){case .ok(_):{assert(false);}case .err(e):{require(e.kind==11 && e.pid!=0 && exited(e.pid) && done(bytes::view(ref report)),2);}};require(handles()==before,3);
}
@test @effects(filesystem,console,environment,clock) fn void separates_traps_and_drains_an_oversized_control_pipe(){
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");let report=joined(bytes::view(ref directory),b"capture-control.done");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();let trap=joined(bytes::view(ref directory),b"trap.exe");let trapped=invoke(bytes::view(ref trap),ref args,bytes::view(ref directory));require(capture::status(ref trapped)==5 && capture::trap(ref trapped)==6 && capture::stdout(ref trapped).len==0 && capture::stderr(ref trapped).len==0,4);
    let normal=joined(bytes::view(ref directory),b"normal.exe");let result=invoke(bytes::view(ref normal),ref args,bytes::view(ref directory));require(capture::status(ref result)==134 && capture::trap(ref result)==0,5);
    add(ref mut args,b"exit");add(ref mut args,b"4294967295");let full=invoke(bytes::view(ref program),ref args,bytes::view(ref directory));require(capture::status(ref full)==4294967295 && capture::exit_bits(capture::status(ref full))==-1,6);assert(v::clear<bytes::Buffer>(ref mut args)==2);
    add(ref mut args,b"flood-control");add(ref mut args,bytes::view(ref report));reset(bytes::view(ref report));switch(capture::run(bytes::view(ref program),ref args,bytes::view(ref directory),2000)){case .ok(_):{assert(false);}case .err(e):{require(e.kind==10 && e.pid!=0 && exited(e.pid) && done(bytes::view(ref report)),7);}};
}
@test @effects(filesystem,console,environment,clock) fn void deadline_terminates_the_observed_live_parent_and_child(){
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");let report=joined(bytes::view(ref directory),b"capture-child.pid");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"tree");add(ref mut args,bytes::view(ref report));reset(bytes::view(ref report));let before=handles();
    switch(capture::run(bytes::view(ref program),ref args,bytes::view(ref directory),500)){case .ok(_):{assert(false);}case .err(e):{require(e.kind==8 && e.os_code==0 && e.pid!=0 && exited(e.pid),8);}};
    let data=switch(files::manifest(bytes::view(ref report))){case .ok(p):{p}case .err(_):{assert(false);return;}};let view=bytes::view(ref data);var u32 pid=0;var usize i=0;while(i<view.len && view[i]!=32){assert(view[i]>=48 && view[i]<=57);pid=pid*10+(view[i] as u32)-48;i+=1;}require(i<view.len && bytes::equal(view[i..],b" 259") && pid!=0 && exited(pid) && handles()==before,9);
}
@test @effects(filesystem,console,environment,clock) fn void captures_binary_stdin_and_streams_before_forwarding(){
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"stdio");let result=invoke(bytes::view(ref program),ref args,bytes::view(ref directory));require(capture::status(ref result)==41 && capture::trap(ref result)==0 && bytes::equal(capture::stdout(ref result),b"child stdout\n") && bytes::equal(capture::stderr(ref result),b"child stderr\n"),10);assert(io::write_bytes(capture::stdout(ref result)).is_ok && io::write_stderr_bytes(capture::stderr(ref result)).is_ok);
}
