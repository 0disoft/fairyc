module windows_run_capture;
import windows_process as child;
import windows_command as command;
import windows_run_control as control;
import std::bytes as bytes;
import std::vec as v;

@effects(filesystem) @link_name("fairyc_private_windows_pipe_v1") unsafe extern C fn i32 create_pipe(ptr mut void reader,ptr mut void writer);
@effects(filesystem) @link_name("fairyc_private_windows_inherit_v1") unsafe extern C fn i32 inherit(ptr mut void handle,u32 flags);
@effects(filesystem) @link_name("fairyc_private_windows_peek_v1") unsafe extern C fn i32 peek(ptr mut void handle,ptr mut u32 available);
@effects(filesystem) @link_name("fairyc_private_windows_read_v1") unsafe extern C fn i32 read(ptr mut void handle,ptr mut u8 data,u32 size,ptr mut u32 count);
@effects(filesystem) @link_name("fairyc_private_windows_close_v1") unsafe extern C fn i32 close(ptr mut void handle);
@effects(filesystem) @link_name("fairyc_private_windows_input_file_v1") unsafe extern C fn ptr mut void input_file(ptr u8 path);
@link_name("fairyc_private_windows_handle_value_v1") unsafe extern C fn usize handle_value(ptr mut void handle);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();
@effects(clock) @link_name("fairyc_private_windows_ticks_v1") unsafe extern C fn usize ticks();

@effects(filesystem) resource Pipe{ptr mut void reader;ptr mut void writer;cleanup{unsafe{if(this.writer!=null){discard close(this.writer);}if(this.reader!=null){discard close(this.reader);}};}}
pub resource Captured{u32 status;u32 trap;u32 pid;bytes::Buffer output;bytes::Buffer error;cleanup{}}
@effects(filesystem) resource Input{ptr mut void raw;bool owned;cleanup{if(this.owned && this.raw!=null){unsafe{discard close(this.raw);};}}}
// 0 inherits caller input, 1 supplies EOF, 2 reopens a private replay file.
@effects(filesystem,console) fn Result<Input,child::Problem> input(u32 mode,ref u8[] path){
    if(mode==0){return .ok(Input{.raw=child::inherited_handles().input,.owned=false});}
    if(mode==1){var ptr mut void reader=null;var ptr mut void writer=null;let made=unsafe{create_pipe(ref mut reader as ptr mut void,ref mut writer as ptr mut void)};let p=Input{.raw=reader,.owned=true};if(made==0){return .err(fail(9,unsafe{last_error()},0));}if(unsafe{close(writer)}==0 || unsafe{inherit(p.raw,1)}==0){return .err(fail(9,unsafe{last_error()},0));}return .ok(p);}
    if(mode!=2){return .err(fail(1,0,0));}let encoded=switch(command::path(path)){case .ok(p):{p}case .err(_):{return .err(fail(1,0,0));}};let p=Input{.raw=unsafe{input_file(bytes::view(ref encoded) as ptr u8)},.owned=true};if(p.raw==null){return .err(fail(9,unsafe{last_error()},0));}return .ok(p);
}
pub fn u32 status(ref Captured result){return result.status;}
pub fn u32 trap(ref Captured result){return result.trap;}
pub fn u32 pid(ref Captured result){return result.pid;}
pub fn ref u8[] stdout(ref Captured result){return bytes::view(ref result.output);}
pub fn ref u8[] stderr(ref Captured result){return bytes::view(ref result.error);}
// Preserve Windows bits through the signed i32 main ABI without a narrowing cast.
fn i32 positive(u32 magnitude){var u32 remaining=magnitude;var i32 out=0;var i32 weight=1073741824;var u32 bit=1073741824;while(bit!=0){if(remaining>=bit){remaining-=bit;out+=weight;}bit/=2;weight/=2;}return out;}
pub fn i32 exit_bits(u32 status){return if(status<=2147483647){positive(status)}else{-2147483648+positive(status-2147483648)};}
fn child::Problem fail(u32 kind,u32 os_code,u32 pid){return child::Problem{.kind,.os_code,.pid};}
@effects(filesystem) fn Result<Pipe,child::Problem> pipe(){
    var ptr mut void reader=null;var ptr mut void writer=null;let made=unsafe{create_pipe(ref mut reader as ptr mut void,ref mut writer as ptr mut void)};let result=Pipe{.reader,.writer};if(made==0){return .err(fail(9,unsafe{last_error()},0));}
    if(unsafe{inherit(result.writer,1)}==0){return .err(fail(9,unsafe{last_error()},0));}return .ok(result);
}
// Drain one bounded chunk per stream per round, even after the retention limit.
// Discarding excess matches the existing runner: the child can finish its writes
// before the caller reports output failure. stdout/stderr share one 64KiB ledger.
@effects(filesystem) fn Result<bool,child::Problem> pump(ref Pipe input,ref mut bytes::Buffer output,ref mut bytes::Buffer scratch,ref mut usize retained,usize limit,ref mut bool truncated,u32 pid){
    var u32 available=0;if(unsafe{peek(input.reader,ref mut available as ptr mut u32)}==0){return .err(fail(9,unsafe{last_error()},pid));}if(available==0){return .ok(false);}
    let u32 count=if(available>4096){4096}else{available};var u32 copied=0;if(unsafe{read(input.reader,bytes::view_mut(scratch) as ptr mut u8,count,ref mut copied as ptr mut u32)}==0 || copied==0 || copied>count){return .err(fail(9,unsafe{last_error()},pid));}
    let keep=if((copied as usize)>limit-*retained){limit-*retained}else{copied as usize};if(keep!=copied as usize){*truncated=true;}
    if(keep!=0 && !bytes::append(output,bytes::view(scratch)[..keep]).is_ok){return .err(fail(2,0,pid));};*retained=*retained+keep;return .ok(true);
}
@effects(filesystem,console,environment,clock)
pub fn Result<Captured,child::Problem> run(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout){
    return run_input(program,arguments,directory,timeout,0,b"");
}
@effects(filesystem,console,environment,clock)
pub fn Result<Captured,child::Problem> run_input(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout,u32 input_mode,ref u8[] input_path){
    if(timeout==0 || timeout>600000){return .err(fail(1,0,0));}
    let output=switch(pipe()){case .ok(p):{p}case .err(e):{return .err(e);}};let error=switch(pipe()){case .ok(p):{p}case .err(e):{return .err(e);}};let receipt=switch(pipe()){case .ok(p):{p}case .err(e):{return .err(e);}};
    let inherited=switch(control::snapshot()){case .ok(p):{p}case .err(e):{return .err(e);}};let environment=switch(control::with_handle(bytes::view(ref inherited),unsafe{handle_value(receipt.writer)})){case .ok(p):{p}case .err(e):{return .err(e);}};
    var bytes::Buffer scratch=bytes::empty();if(!bytes::resize_zeroed(ref mut scratch,4096).is_ok){return .err(fail(2,0,0));}var bytes::Buffer out=bytes::empty();var bytes::Buffer err=bytes::empty();var bytes::Buffer signal=bytes::empty();var usize retained=0;var usize signal_retained=0;var bool truncated=false;var bool signal_truncated=false;
    let standard=switch(input(input_mode,input_path)){case .ok(p):{p}case .err(e):{return .err(e);}};let started=unsafe{ticks()};let running=switch(child::start(program,arguments,directory,bytes::view(ref environment),child::Handles{.input=standard.raw,.output=output.writer,.error=error.writer})){case .ok(p):{p}case .err(e):{return .err(e);}};
    let id=child::pid(ref running);var bool stopped=false;var child::Outcome completed=child::Outcome{.exit_code=0,.pid=id};var usize collected=0;
    while(true){
        let a=switch(pump(ref output,ref mut out,ref mut scratch,ref mut retained,65536,ref mut truncated,id)){case .ok(p):{p}case .err(e):{child::terminate(ref running);return .err(e);}};
        let b=switch(pump(ref error,ref mut err,ref mut scratch,ref mut retained,65536,ref mut truncated,id)){case .ok(p):{p}case .err(e):{child::terminate(ref running);return .err(e);}};
        let c=switch(pump(ref receipt,ref mut signal,ref mut scratch,ref mut signal_retained,2,ref mut signal_truncated,id)){case .ok(p):{p}case .err(e):{child::terminate(ref running);return .err(e);}};
        let now=unsafe{ticks()};if(!stopped && (now<started || now-started>=timeout as usize)){child::terminate(ref running);return .err(fail(8,0,id));}
        if(stopped){if(!a && !b && !c){break;}if(now<collected || now-collected>=2000){return .err(fail(9,0,id));}}
        else{switch(child::poll(ref running,1)){case .ok(done):{if(done){completed=switch(child::result(ref running)){case .ok(p):{p}case .err(e):{return .err(e);}};child::terminate(ref running);stopped=true;collected=unsafe{ticks()};}}case .err(e):{child::terminate(ref running);return .err(e);}};}
    }
    if(truncated){return .err(fail(11,0,id));}if(signal_truncated){return .err(fail(10,0,id));}
    let classified=switch(control::classify(completed.exit_code,bytes::view(ref signal),id)){case .ok(p):{p}case .err(code):{return .err(fail(code,0,id));}};
    return .ok(Captured{.status=classified.status,.trap=classified.trap,.pid=id,.output=out,.error=err});
}
