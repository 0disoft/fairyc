"""Execute authenticated console SDK bodies with exact native process streams."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "sdk-io"
observer.SDK = observer.REPO / "internal/sdkstdlib/sources/std/io.fy"
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import std::io as io;"
observer.RUNTIME_NAMES = ("stage0_windows_console.c",)
observer.PREFIX = ""
observer.SUFFIX = """
void fixture_disable_console(void){
    SetStdHandle(STD_OUTPUT_HANDLE,INVALID_HANDLE_VALUE);
    SetStdHandle(STD_ERROR_HANDLE,INVALID_HANDLE_VALUE);
    SetStdHandle(STD_INPUT_HANDLE,INVALID_HANDLE_VALUE);
}
"""
observer.CASES = {
    "stdout-text-bytes": r'''
@effects(console) fn i32 main(){
    assert(switch(io::write("한글")){case .ok(n):{n==6}case .err(_):{false}});
    assert(switch(io::write_bytes(b"A\0B")){case .ok(n):{n==3}case .err(_):{false}});
    assert(switch(io::write("")){case .ok(n):{n==0}case .err(_):{false}});return 7;
}
''',
    "stdout-scalars": r'''
@effects(console) fn i32 main(){
    assert(switch(io::write_i32(-2147483648)){case .ok(n):{n==11}case .err(_):{false}});
    assert(io::write("|").is_ok && io::write_u32(4294967295).is_ok && io::write("|").is_ok);
    assert(io::write_usize(18446744073709551615).is_ok && io::write("|").is_ok);
    assert(switch(io::write_i64(-9223372036854775808)){case .ok(n):{n==20}case .err(_):{false}});
    assert(io::write("|").is_ok && io::write_i64(9223372036854775807).is_ok && io::write("|").is_ok);
    assert(io::write_i64(0).is_ok && io::write_bool(true).is_ok && io::write_bool(false).is_ok);return 7;
}
''',
    "stderr-all": r'''
@effects(console) fn i32 main(){
    assert(io::write_stderr("한").is_ok && io::write_stderr_bytes(b"A\0B").is_ok);
    assert(io::write_stderr_i32(-2147483648).is_ok && io::write_stderr("|").is_ok);
    assert(io::write_stderr_u32(4294967295).is_ok && io::write_stderr("|").is_ok);
    assert(io::write_stderr_usize(18446744073709551615).is_ok && io::write_stderr("|").is_ok);
    assert(io::write_stderr_i64(-9223372036854775808).is_ok && io::write_stderr_bool(false).is_ok);
    assert(switch(io::write_stderr("")){case .ok(n):{n==0}case .err(_):{false}});return 7;
}
''',
    "stdin-partial-eof": r'''
@effects(console) fn i32 main(){
    var u8[16] data=[0;16];data[9]=99;let ref mut u8[] view=ref mut data;var usize used=0;
    assert(switch(io::read_stdin(view[..0])){case .ok(n):{n==0}case .err(_):{false}});
    while(used<9){let usize n=switch(io::read_stdin(view[used..used+2])){case .ok(n):{n}case .err(_):{assert(false);0}};assert(n>0 && n<=2);used+=n;}
    assert(used==9 && view[0]==237 && view[6]==65 && view[7]==0 && view[8]==66 && view[9]==99);
    assert(switch(io::read_stdin(view[9..])){case .ok(n):{n==0}case .err(_):{false}});
    assert(view[1]==149 && view[2]==156 && view[3]==234 && view[4]==184 && view[5]==128);return 7;
}
''',
}
observer.CASES["unavailable-stream-errors"] = r'''
unsafe extern C fn void fixture_disable_console();
@effects(console) fn i32 main(){
    unsafe{fixture_disable_console();}
    assert(switch(io::write("x")){case .err(e):{e.status==2 && e.written==0}case .ok(_):{false}});
    assert(switch(io::write_stderr_i32(7)){case .err(e):{e.status==2 && e.written==0}case .ok(_):{false}});
    var u8[2] data=[0;2];data[0]=99;data[1]=99;let ref mut u8[] view=ref mut data;
    assert(switch(io::read_stdin(view)){case .err(e):{e.status==2 && e.read==0}case .ok(_):{false}});
    assert(view[0]==99 && view[1]==99);
    assert(switch(io::write("")){case .ok(n):{n==0}case .err(_):{false}});
    assert(switch(io::read_stdin(view[..0])){case .ok(n):{n==0}case .err(_):{false}});return 7;
}
'''
observer.STREAMS = {
    "stdout-text-bytes": ("", "한글A\0B", ""),
    "stdout-scalars": ("", "-2147483648|4294967295|18446744073709551615|-9223372036854775808|9223372036854775807|0truefalse", ""),
    "stderr-all": ("", "", "한A\0B-2147483648|4294967295|18446744073709551615|-9223372036854775808false"),
    "stdin-partial-eof": ("한글A\0B", "", ""),
}
observer.REJECT = {
    "missing-console-effect": ('fn i32 main(){discard io::write("x");return 7;}', "code 7102"),
    "missing-transitive-effect": ('@effects(console)fn void emit(){discard io::write("x");}fn i32 main(){emit();return 7;}', "code 7102"),
    "shared-stdin": ('@effects(console)fn i32 main(){let u8[2] data=[0;2];discard io::read_stdin(ref data);return 7;}', "code 1"),
    "stolen-console-symbol": ('@effects(console)@link_name("fy_std_stdout_write_i32_v0")unsafe extern C fn usize stolen(i32 value);fn i32 main(){return 7;}', "code 5106"),
}
if __name__ == "__main__":
    observer.main()
