module linux_run_cli;
import linux_run_capture as capture;
import native_run_control as control;
import linux_build_scratch as scratch;
import project_emit as emit;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn i32 failure(ref str text){return if(io::write_stderr(text).is_ok){6}else{4};}
@effects(console) fn bool stdout(ref u8[] data){return switch(io::write_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}};}
@effects(console) fn bool stderr(ref u8[] data){return switch(io::write_stderr_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}};}
fn bool location(ref emit::Plan plan,ref u8[] record,ref mut bytes::Buffer out){
    let key=control::location_source(record);var usize i=0;
    while(i<emit::trap_count(plan)){if(bytes::equal(key,emit::trap_key(plan,i))){return control::append_location(emit::trap_path(plan,i),emit::trap_bytes(plan,i),control::location_start(record),control::location_end(record),out);}i+=1;}return true;
}
pub fn i32 exit_code(u32 status){var u32 remaining=status;var u32 bit=128;var i32 weight=128;var i32 code=0;while(bit!=0){if(remaining>=bit){remaining-=bit;code+=weight;}bit/=2;weight/=2;}return code;}
// Linux transports only the low eight exit bits. Control receipts authenticate traps.
@effects(console) pub fn i32 report(ref capture::Captured result,ref emit::Plan plan){
    if(!stdout(capture::stdout(result)) || !stderr(capture::stderr(result))){return 4;}
    let kind=capture::kind(result);let value=capture::value(result);
    if(kind==3){return failure("fairyc: run failed: program timed out after 60000 ms\n");}
    if(kind==4){return failure("fairyc: run failed: program output exceeds 65536 bytes per stream\n");}
    if(kind==2){if(!io::write_stderr("fairyc: run failed: program terminated by signal ").is_ok || !io::write_stderr_u32(value).is_ok || !io::write_stderr("\n").is_ok){return 4;}return 6;}
    if(kind!=1){return failure("fairyc: run failed: invalid execution result\n");}
    let record=capture::receipt(result);
    let outcome=switch(control::classify(value,record,capture::pid(result))){case .ok(p):{p}case .err(_):{return failure("fairyc: run failed: invalid trap control receipt\n");}};
    if(outcome.trap==0){return exit_code(value);}
    var bytes::Buffer message=bytes::empty();
    if(!bytes::append(ref mut message,b"trap: ").is_ok || !bytes::append(ref mut message,control::category(outcome.trap)).is_ok || !bytes::append_byte(ref mut message,10).is_ok || !location(plan,record,ref mut message) || !stderr(bytes::view(ref message))){return 4;}return 5;
}
@effects(filesystem,console,environment,clock)
pub fn i32 executed(scratch::Directory directory,ref u8[] program,ref v::Vec<bytes::Buffer> arguments,ref emit::Plan plan,ref u8[] root){
    let result=switch(capture::run_for(program,arguments,root,60000,65536,65536)){case .ok(p):{p}case .err(_):{return failure("fairyc: run failed: program execution or trap control failed\n");}};
    if(!scratch::close(directory).is_ok){return failure("fairyc: run failed: private staging cleanup failed\n");}
    return report(ref result,plan);
}
