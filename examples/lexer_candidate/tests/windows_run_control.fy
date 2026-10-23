module tests::windows_run_control;
import windows_run_control as control;
import windows_command as command;
import windows_process as child;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

@effects(filesystem) @link_name("fairyc_private_windows_handle_count_v1") unsafe extern C fn i32 handle_count(ptr mut u32 count);
fn void add(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let data=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(args,data).is_ok);}
fn bytes::Buffer joined(ref u8[] root,ref u8[] file){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,root).is_ok && bytes::append_byte(ref mut out,47).is_ok && bytes::append(ref mut out,file).is_ok);return out;}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] data=[0;4096];let size=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref data;return switch(files::manifest(view[..size])){case .ok(p):{p}case .err(_):{assert(false);return root();}};}
@effects(filesystem) fn u32 handles(){var u32 count=0;assert(unsafe{handle_count(ref mut count as ptr mut u32)}!=0);return count;}
@effects(console) fn void require(bool ok,u32 step){if(!ok){discard io::write_stderr_u32(step);assert(false);}}
@effects(console) fn void failure(child::Problem p){discard io::write_stderr_u32(p.kind);discard io::write_stderr_u32(p.os_code);assert(false);}
fn bytes::Buffer block(ref v::Vec<bytes::Buffer> entries){var bytes::Buffer result=bytes::empty();var usize i=0;while(i<v::len<bytes::Buffer>(entries)){let entry=switch(command::path(bytes::view(v::at<bytes::Buffer>(entries,i)))){case .ok(p):{p}case .err(_):{assert(false);return result;}};assert(bytes::append(ref mut result,bytes::view(ref entry)).is_ok);i+=1;}assert(bytes::append_byte(ref mut result,0).is_ok && bytes::append_byte(ref mut result,0).is_ok);return result;}

@test fn void preserves_all_normal_exit_bits_and_requires_exact_trap_receipts(){
    var u32 code=1;while(code<=7){var u8[1] receipt=[0;1];receipt[0]=if(code==1){1}else if(code==2){2}else if(code==3){3}else if(code==4){4}else if(code==5){5}else if(code==6){6}else{7};let ref u8[] view=ref receipt;
        let normal=switch(control::classify(128+code,b"",42)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(normal.status==128+code && normal.trap==0 && normal.pid==42);
        let trapped=switch(control::classify(128+code,view,42)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(trapped.status==5 && trapped.trap==(if(code==6){1}else if(code==7){7}else{code+1}) && trapped.pid==42);code+=1;
    }
    assert(bytes::equal(control::category(1),b"bounds") && bytes::equal(control::category(2),b"integer-overflow") && bytes::equal(control::category(3),b"division-by-zero") && bytes::equal(control::category(4),b"signed-division-overflow") && bytes::equal(control::category(5),b"invalid-shift") && bytes::equal(control::category(6),b"assertion-failed") && bytes::equal(control::category(7),b"unreachable") && control::category(0).len==0 && control::category(8).len==0);
    let full=switch(control::classify(4294967295,b"",7)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(full.status==4294967295 && full.trap==0);
    assert(!control::classify(42,b"\x03",1).is_ok && !control::classify(131,b"\x02",1).is_ok && !control::classify(131,b"\x03\x03",1).is_ok && !control::classify(128,b"\0",1).is_ok && !control::classify(136,b"\x08",1).is_ok);
}
@test fn void validates_extended_trap_location_records(){
    let record=b"\x05\x01\x0a\0\x01\0\0\0\0\0\0\0\x09\0\0\0\0\0\0\0p1/main.fy";
    let accepted=switch(control::classify(133,record,42)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(accepted.trap==6 && accepted.status==5 && accepted.pid==42);
    assert(bytes::equal(control::location_source(record),b"p1/main.fy") && control::location_start(record)==1 && control::location_end(record)==9);
    assert(control::location_source(b"\x05").len==0 && control::location_start(b"\x05")==0 && control::location_end(b"")==0);
    var usize n=2;while(n<record.len){assert(!control::classify(133,record[..n],42).is_ok);n+=1;}
    var bytes::Buffer data=bytes::empty();assert(bytes::append(ref mut data,record[..20]).is_ok);while(bytes::len(ref data)<277){assert(bytes::append_byte(ref mut data,97).is_ok);}
    bytes::view_mut(ref mut data)[2]=0;bytes::view_mut(ref mut data)[3]=1;assert(control::classify(133,bytes::view(ref data)[..276],42).is_ok);
    assert(!control::classify(133,bytes::view(ref data),42).is_ok && !control::classify(132,record,42).is_ok);
    bytes::view_mut(ref mut data)[1]=2;assert(!control::classify(133,bytes::view(ref data)[..276],42).is_ok);bytes::view_mut(ref mut data)[1]=1;
    bytes::view_mut(ref mut data)[20]=0;assert(!control::classify(133,bytes::view(ref data)[..276],42).is_ok);bytes::view_mut(ref mut data)[20]=255;assert(!control::classify(133,bytes::view(ref data)[..276],42).is_ok);bytes::view_mut(ref mut data)[20]=97;
    bytes::view_mut(ref mut data)[4]=10;assert(!control::classify(133,bytes::view(ref data)[..276],42).is_ok);
    var usize i=4;while(i<20){bytes::view_mut(ref mut data)[i]=255;i+=1;}assert(control::classify(133,bytes::view(ref data)[..276],42).is_ok);
    assert(control::location_start(bytes::view(ref data)[..276])==18446744073709551615 && control::location_end(bytes::view(ref data)[..276])==18446744073709551615);
}
@test fn void replaces_case_insensitive_control_name_and_preserves_environment_bytes(){
    var v::Vec<bytes::Buffer> entries=v::empty<bytes::Buffer>();add(ref mut entries,b"=C:=C:\\where");add(ref mut entries,b"AAA=kept");add(ref mut entries,b"fairyc_trap_control_handle_v1=old");add(ref mut entries,b"FAIRYC_TRAP_CONTROL_HANDLE_V1=duplicate");add(ref mut entries,b"ZED=\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a");let original=block(ref entries);
    let actual=switch(control::with_handle(bytes::view(ref original),123)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::clear<bytes::Buffer>(ref mut entries)==5);add(ref mut entries,b"=C:=C:\\where");add(ref mut entries,b"AAA=kept");add(ref mut entries,b"FAIRYC_TRAP_CONTROL_HANDLE_V1=123");add(ref mut entries,b"ZED=\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a");let expected=block(ref entries);assert(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)));
    let empty=switch(control::with_handle(b"\0\0\0\0",123)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::clear<bytes::Buffer>(ref mut entries)==4);add(ref mut entries,b"FAIRYC_TRAP_CONTROL_HANDLE_V1=123");let sole=block(ref entries);assert(bytes::equal(bytes::view(ref empty),bytes::view(ref sole)));
    assert(!control::with_handle(b"\0\0\0",1).is_ok && !control::with_handle(b"A\0\0\0\0\0",1).is_ok && !control::with_handle(b"\0\0\0\0A\0\0\0\0\0",1).is_ok);
}
@effects(filesystem,console,environment) fn control::Outcome invoke(ref u8[] program,ref u8[] directory,ref v::Vec<bytes::Buffer> args){return switch(control::run(program,args,directory,2000)){case .ok(p):{p}case .err(e):{failure(e);return invoke(program,directory,args);}};}
@test @effects(filesystem,console,environment) fn void isolates_native_control_receipts_and_actual_generated_fairy_traps(){
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"control");add(ref mut args,b"131");let warm=invoke(bytes::view(ref program),bytes::view(ref directory),ref args);require(warm.status==131 && warm.trap==0,1);let before=handles();var u32 code=1;
    while(code<=7){assert(v::clear<bytes::Buffer>(ref mut args)==2 || v::len<bytes::Buffer>(ref args)==0);add(ref mut args,b"control");var bytes::Buffer status=bytes::empty();assert(bytes::append_u32(ref mut status,128+code).is_ok);add(ref mut args,bytes::view(ref status));let normal=invoke(bytes::view(ref program),bytes::view(ref directory),ref args);require(normal.status==128+code && normal.trap==0,2);var bytes::Buffer receipt=bytes::empty();assert(bytes::append_u32(ref mut receipt,code).is_ok);add(ref mut args,bytes::view(ref receipt));let trap=invoke(bytes::view(ref program),bytes::view(ref directory),ref args);require(trap.status==5 && trap.trap==(if(code==6){1}else if(code==7){7}else{code+1}),3);assert(v::clear<bytes::Buffer>(ref mut args)==3);code+=1;}
    add(ref mut args,b"control");add(ref mut args,b"4294967295");let full=invoke(bytes::view(ref program),bytes::view(ref directory),ref args);require(full.status==4294967295 && full.trap==0,4);assert(v::clear<bytes::Buffer>(ref mut args)==2);
    add(ref mut args,b"control");add(ref mut args,b"131");add(ref mut args,b"2");switch(control::run(bytes::view(ref program),ref args,bytes::view(ref directory),2000)){case .ok(_):{assert(false);}case .err(e):{require(e.kind==10 && e.pid!=0,5);}};assert(v::clear<bytes::Buffer>(ref mut args)==3);add(ref mut args,b"control");add(ref mut args,b"131");add(ref mut args,b"3");add(ref mut args,b"3");switch(control::run(bytes::view(ref program),ref args,bytes::view(ref directory),2000)){case .ok(_):{assert(false);}case .err(e):{require(e.kind==10 && e.pid!=0,5);}};assert(v::clear<bytes::Buffer>(ref mut args)==4);
    let actual=joined(bytes::view(ref directory),b"trap.exe");let trapped=invoke(bytes::view(ref actual),bytes::view(ref directory),ref args);require(trapped.status==5 && trapped.trap==6,7);let ordinary=joined(bytes::view(ref directory),b"normal.exe");let returned=invoke(bytes::view(ref ordinary),bytes::view(ref directory),ref args);require(returned.status==134 && returned.trap==0,8);require(handles()==before,9);
}
@test @effects(filesystem,console,environment) fn void forwards_binary_streams_and_preserves_the_parent_environment(){
    let before=switch(control::snapshot()){case .ok(p):{p}case .err(e):{failure(e);return;}};
    let directory=root();let program=joined(bytes::view(ref directory),b"receiver.exe");var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();add(ref mut args,b"stdio");let completed=invoke(bytes::view(ref program),bytes::view(ref directory),ref args);require(completed.status==41 && completed.trap==0,10);
    let after=switch(control::snapshot()){case .ok(p):{p}case .err(e):{failure(e);return;}};require(bytes::equal(bytes::view(ref before),bytes::view(ref after)),11);
}
