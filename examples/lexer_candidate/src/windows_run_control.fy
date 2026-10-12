module windows_run_control;
import windows_process as child;
import windows_command as command;
import std::bytes as bytes;
import std::vec as v;

@effects(filesystem) @link_name("fairyc_private_windows_pipe_v1") unsafe extern C fn i32 create_pipe(ptr mut void reader,ptr mut void writer);
@effects(filesystem) @link_name("fairyc_private_windows_inherit_v1") unsafe extern C fn i32 inherit(ptr mut void handle,u32 flags);
@effects(filesystem) @link_name("fairyc_private_windows_peek_v1") unsafe extern C fn i32 peek(ptr mut void handle,ptr mut u32 available);
@effects(filesystem) @link_name("fairyc_private_windows_read_v1") unsafe extern C fn i32 read(ptr mut void handle,ptr mut u8 data,u32 size,ptr mut u32 count);
@effects(filesystem) @link_name("fairyc_private_windows_close_v1") unsafe extern C fn i32 close(ptr mut void handle);
@link_name("fairyc_private_windows_handle_value_v1") unsafe extern C fn usize handle_value(ptr mut void handle);
@effects(environment) @link_name("fairyc_private_windows_environment_v1") unsafe extern C fn ptr mut u8 environment();
@effects(environment) @link_name("fairyc_private_windows_free_environment_v1") unsafe extern C fn i32 free_environment(ptr mut u8 data);
@link_name("fairyc_private_windows_environment_word_v1") unsafe extern C fn u32 environment_word(ptr mut u8 data,usize index);
@link_name("fairyc_private_windows_compare_name_v1") unsafe extern C fn i32 compare_name(ptr u8 left,usize left_size,ptr u8 right,usize right_size);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();

@effects(filesystem) resource Pipe{ptr mut void reader;ptr mut void writer;cleanup{unsafe{if(this.writer!=null){discard close(this.writer);}if(this.reader!=null){discard close(this.reader);}};}}
@effects(environment) resource Environment{ptr mut u8 raw;cleanup{if(this.raw!=null){unsafe{discard free_environment(this.raw);};}}}
pub struct Outcome{u32 status;u32 trap;u32 pid;}
// 9 private channel/environment infrastructure, 10 invalid control receipt;
// the process library's 1..8 errors retain their original native code and PID.
fn child::Problem fail(u32 kind,u32 os_code,u32 pid){return child::Problem{.kind,.os_code,.pid};}
pub fn ref u8[] category(u32 trap){let ref u8[] names=b"boundsinteger-overflowdivision-by-zerosigned-division-overflowinvalid-shiftassertion-failedunreachable";return if(trap==1){names[..6]}else if(trap==2){names[6..22]}else if(trap==3){names[22..38]}else if(trap==4){names[38..62]}else if(trap==5){names[62..75]}else if(trap==6){names[75..91]}else if(trap==7){names[91..102]}else{names[..0]};}
pub fn Result<Outcome,u32> classify(u32 exit_code,ref u8[] receipt,u32 pid){
    if(receipt.len==0){return .ok(Outcome{.status=exit_code,.trap=0,.pid});}
    if(receipt.len!=1 || exit_code<129 || exit_code>135 || (receipt[0] as u32)!=exit_code-128){return .err(10);}
    return .ok(Outcome{.status=5,.trap=canonical(exit_code-128),.pid});
}
// The private candidate C emitter's established tags are overflow=1,
// division=2, signed division=3, shift=4, assertion=5, bounds=6, unreachable=7.
// Only authenticated receipts are normalized to public category numbering.
pub fn u32 canonical(u32 raw){return if(raw==6){1}else if(raw>=1 && raw<=5){raw+1}else if(raw==7){7}else{0};}
fn u32 word(ref u8[] data,usize index){return (data[index*2] as u32)+(data[index*2+1] as u32)*256;}

@effects(environment)
pub fn Result<bytes::Buffer,child::Problem> snapshot(){
    let owner=Environment{.raw=unsafe{environment()}};if(owner.raw==null){return .err(fail(9,unsafe{last_error()},0));}
    var usize count=0;var u32 previous=1;
    while(count<524288){let value=unsafe{environment_word(owner.raw,count)};count+=1;if(value==0 && previous==0){let view=unsafe{byte_slice::from_raw_parts(ref owner,owner.raw,count*2)};return switch(bytes::copy(view)){case .ok(p):{.ok(p)}case .err(_):{.err(fail(2,0,0))}};}previous=value;}
    return .err(fail(9,0,0));
}
fn bool append_entry(ref mut bytes::Buffer output,ref bytes::Buffer entry){return bytes::append(output,bytes::view(entry)).is_ok;}
// Preserve the inherited sorted UTF-16 block, replacing the private name with
// case-insensitive Windows ordinal comparison. The caller's environment is never
// modified. The private environment snapshot/output allowance is one MiB.
pub fn Result<bytes::Buffer,child::Problem> with_handle(ref u8[] original,usize value){
    if(original.len<4 || original.len>1048576 || original.len%2!=0 || word(original,original.len/2-1)!=0 || word(original,original.len/2-2)!=0){return .err(fail(9,0,0));}
    let key=switch(command::path(b"FAIRYC_TRAP_CONTROL_HANDLE_V1")){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0));}};
    var bytes::Buffer text=bytes::empty();if(!bytes::append(ref mut text,b"FAIRYC_TRAP_CONTROL_HANDLE_V1=").is_ok || !bytes::append_usize(ref mut text,value).is_ok){return .err(fail(2,0,0));}
    let entry=switch(command::path(bytes::view(ref text))){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0));}};var bytes::Buffer output=bytes::empty();var usize at=0;var bool inserted=false;
    while(at<original.len/2 && word(original,at)!=0){let start=at;var usize separator=0;
        while(at<original.len/2 && word(original,at)!=0){if(at>start && separator==0 && word(original,at)==61){separator=at;}at+=1;}
        if(at>=original.len/2 || separator==0){return .err(fail(9,0,0));}
        let comparison=unsafe{compare_name(original[start*2..separator*2] as ptr u8,separator-start,bytes::view(ref key) as ptr u8,bytes::len(ref key)/2-1)};
        if(comparison==0){return .err(fail(9,unsafe{last_error()},0));}
        if(comparison==3 && !inserted){if(!append_entry(ref mut output,ref entry)){return .err(fail(2,0,0));}inserted=true;}
        at+=1;if(comparison!=2 && !bytes::append(ref mut output,original[start*2..at*2]).is_ok){return .err(fail(2,0,0));}
    }
    // Empty blocks have two NULs; nonempty blocks have one final NUL after the
    // last entry's terminator. Reject hidden entries after the first double NUL.
    if(at!=original.len/2-1 && !(at==0 && original.len==4)){return .err(fail(9,0,0));}
    if(!inserted && !append_entry(ref mut output,ref entry) || !bytes::append_byte(ref mut output,0).is_ok || !bytes::append_byte(ref mut output,0).is_ok){return .err(fail(2,0,0));}
    if(bytes::len(ref output)>1048576){return .err(fail(9,0,0));}return .ok(output);
}
@effects(filesystem,console,environment)
pub fn Result<Outcome,child::Problem> run(ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref u8[] directory,u32 timeout){
    var ptr mut void reader=null;var ptr mut void writer=null;let created=unsafe{create_pipe(ref mut reader as ptr mut void,ref mut writer as ptr mut void)};let pipe=Pipe{.reader,.writer};if(created==0){return .err(fail(9,unsafe{last_error()},0));}
    if(unsafe{inherit(pipe.writer,1)}==0){return .err(fail(9,unsafe{last_error()},0));}
    let inherited=switch(snapshot()){case .ok(p):{p}case .err(e):{return .err(e);}};
    let block=switch(with_handle(bytes::view(ref inherited),unsafe{handle_value(pipe.writer)})){case .ok(p):{p}case .err(e):{return .err(e);}};
    let completed=switch(child::run_environment(program,arguments,directory,timeout,bytes::view(ref block))){case .ok(p):{p}case .err(e):{return .err(e);}};
    // The child job has closed before inspection. Never wait on pipe EOF:
    // descendants may inherit a writer, and an oversized receipt is a failure.
    var u32 available=0;if(unsafe{peek(pipe.reader,ref mut available as ptr mut u32)}==0){return .err(fail(9,unsafe{last_error()},completed.pid));}
    if(available>1){return .err(fail(10,0,completed.pid));}var u8[1] data=[0;1];var u32 copied=0;
    if(available==1){let ref mut u8[] target=ref mut data;if(unsafe{read(pipe.reader,target as ptr mut u8,1,ref mut copied as ptr mut u32)}==0 || copied!=1){return .err(fail(9,unsafe{last_error()},completed.pid));}}
    let ref u8[] receipt=ref data;return switch(classify(completed.exit_code,receipt[..copied as usize],completed.pid)){case .ok(p):{.ok(p)}case .err(code):{.err(fail(code,0,completed.pid))}};
}
