module tests::windows_process;
import windows_process as child;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

@effects(filesystem) @link_name("fairyc_private_windows_open_v1") unsafe extern C fn ptr mut void open_process(u32 access,i32 inherit,u32 pid);
@effects(filesystem) @link_name("fairyc_private_windows_wait_v1") unsafe extern C fn u32 wait_process(ptr mut void handle,u32 timeout);
@effects(filesystem) @link_name("fairyc_private_windows_close_v1") unsafe extern C fn i32 close_handle(ptr mut void handle);
@effects(filesystem) @link_name("fairyc_private_windows_handle_count_v1") unsafe extern C fn i32 handle_count(ptr mut u32 count);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();

@effects(console) fn void failure(child::Problem p){discard io::write_stderr_u32(p.kind);discard io::write_stderr_u32(p.os_code);assert(false);}
@effects(console) fn void require(bool ok,u32 step){if(!ok){discard io::write_stderr_u32(step);assert(false);}}
fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let data=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(args,data).is_ok);}
fn bytes::Buffer joined(ref u8[] root,ref u8[] file){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,root).is_ok && bytes::append_byte(ref mut out,47).is_ok && bytes::append(ref mut out,file).is_ok);return out;}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] data=[0;4096];let size=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref data;return switch(files::manifest(view[..size])){case .ok(p):{p}case .err(_):{assert(false);return root();}};}
@effects(filesystem) fn u32 handles(){var u32 count=0;assert(unsafe{handle_count(ref mut count as ptr mut u32)}!=0);return count;}
@effects(filesystem) fn bool exited(u32 pid){let raw=unsafe{open_process(1048576,0,pid)};if(raw==null){return unsafe{last_error()}==87;}let done=unsafe{wait_process(raw,2000)}==0;assert(unsafe{close_handle(raw)}!=0);return done;}
@test @effects(filesystem,console) fn void forwards_special_arguments_and_preserves_full_exit_bits_without_handle_leaks(){
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"args");add(ref mut args,b"");add(ref mut args,b"a b");add(ref mut args,b"x\\\"y\\");add(ref mut args,b"\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a");add(ref mut args,b"--flag");
    let first=switch(child::run(bytes::view(ref program),ref args,bytes::view(ref directory),2000)){case .ok(p):{p}case .err(e):{failure(e);return;}};require(first.exit_code==37 && first.pid!=0,1);let before=handles();var usize i=0;
    while(i<8){let result=switch(child::run(bytes::view(ref program),ref args,bytes::view(ref directory),2000)){case .ok(p):{p}case .err(e):{failure(e);return;}};require(result.exit_code==37,2);i+=1;}require(handles()==before,3);
    require(v::clear<bytes::Buffer>(ref mut args)==6,7);add(ref mut args,b"exit");add(ref mut args,b"2147483690");let full=switch(child::run(bytes::view(ref program),ref args,bytes::view(ref directory),2000)){case .ok(p):{p}case .err(e):{failure(e);return;}};require(full.exit_code==2147483690,4);
    // Windows may lazily initialize process-wide state on the first failure.
    // Measure repeated failures against their own warmed baseline.
    require(!child::run(b"missing.exe",ref args,bytes::view(ref directory),2000).is_ok && !child::run(bytes::view(ref program),ref args,bytes::view(ref directory),0).is_ok && !child::run(bytes::view(ref program),ref args,bytes::view(ref directory),600001).is_ok,5);let failure_before=handles();i=0;
    while(i<8){switch(child::run(b"missing.exe",ref args,bytes::view(ref directory),2000)){case .ok(_):{assert(false);}case .err(e):{require(e.kind==4 && e.os_code!=0 && e.pid==0,8);}};i+=1;}require(handles()==failure_before,6);
}
@test @effects(filesystem,console) fn void deadline_terminates_the_observed_live_parent_and_child(){
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");let report=joined(bytes::view(ref directory),b"child.pid");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"tree");add(ref mut args,bytes::view(ref report));let before=handles();
    switch(child::run(bytes::view(ref program),ref args,bytes::view(ref directory),500)){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==8 && e.os_code==0 && e.pid!=0 && exited(e.pid));}};
    let data=switch(files::manifest(bytes::view(ref report))){case .ok(p):{p}case .err(_):{assert(false);return;}};let view=bytes::view(ref data);var u32 pid=0;var usize i=0;while(i<view.len && view[i]!=32){assert(view[i]>=48 && view[i]<=57);pid=pid*10+(view[i] as u32)-48;i+=1;}assert(i<view.len && bytes::equal(view[i..],b" 259") && pid!=0 && exited(pid));assert(handles()==before);
}
@test @effects(filesystem,console) fn void forwards_binary_stdin_and_exact_stdout_stderr(){
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"stdio");
    let result=switch(child::run(bytes::view(ref program),ref args,bytes::view(ref directory),2000)){case .ok(p):{p}case .err(e):{failure(e);return;}};require(result.exit_code==41 && result.pid!=0,9);
}
