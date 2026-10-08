"""Execute copied byte loops through canonical owned HIR/MIR lowering."""
from pathlib import Path
import verify_sdk_vector as observer
observer.SUITE = "byte-loops"
observer.SDK = observer.REPO / "internal/sdkstdlib/sources/std/bytes.fy"
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import std::bytes as b;"
observer.RUNTIME_NAMES += ("stage0_buffer.c",)
observer.TRAPS = {"reversed-bounds": 6}
observer.CASES = {
    "reversed-bounds": 'fn i32 main(){let ref u8[] data=b"abc";for(let u8 byte in data[2..1]){discard byte;}return 7;}',
    "control-and-shadow": r'''
fn usize sum(ref u8[] data){var usize value=0;for(let u8 byte in data){if(byte==2){continue;}if(byte==4){break;}value+=byte as usize;}return value;}
fn i32 main(){
    let ref u8[] data=b"\x01\x02\x03\x04\x05";assert(sum(data)==4);
    assert(sum(data[..0])==0 && sum(data[2..3])==3);
    let u8 byte=9;var usize count=0;for(let u8 byte in data[..2]){for(let u8 byte in data[..1]){count+=byte as usize;}}assert(byte==9 && count==2);return 7;
}
''',
    "once-and-generic": r'''
fn usize lower(ref mut usize calls){*calls=*calls+1;return 1;}
fn usize upper(ref mut usize calls){*calls=*calls+1;return 3;}
fn T repeat<T>(T marker,ref u8[] data){var usize sum=0;for(let u8 byte in data){sum+=byte as usize;}assert(sum==5);return marker;}
fn i32 main(){let ref u8[] data=b"\x01\x02\x03\x04";var usize calls=0;var usize sum=0;
    for(let u8 byte in data[lower(ref mut calls)..upper(ref mut calls)]){sum+=byte as usize;}assert(calls==2 && sum==5);
    assert(repeat<i32>(7,data[1..3])==7 && repeat<bool>(true,data[1..3]));return 7;
}
''',
    "resource-exits": observer.FOREIGN + r'''
unsafe extern C fn void fixture_drop(i32 value);
resource R{i32 value;cleanup{unsafe{fixture_drop(this.value);}}}
fn i32 early(ref u8[] data){for(let u8 byte in data){let R x=R{.value=byte as i32};return 7;}return 0;}
fn i32 main(){
    {var b::Buffer owned=switch(b::copy(b"\x01\x02\x03")){case .ok(x):{x}case .err(_):{return 0;}};
        let ref u8[] data=b::view(ref owned);
        for(let u8 byte in data){let R x=R{.value=byte as i32};if(byte==1){continue;}if(byte==2){break;}}
        assert(early(data[2..])==7 && b::append_byte(ref mut owned,4).is_ok);
    }
    assert(unsafe{fixture_trace_count()}==3 && unsafe{fixture_trace_at(0)}==1 && unsafe{fixture_trace_at(1)}==2 && unsafe{fixture_trace_at(2)}==3 && unsafe{fixture_live()}==0);return 7;
}
''',
}
observer.SUFFIX += "\nvoid fixture_drop(int32_t value){if(fixture_trace_size>=64)abort();fixture_trace[fixture_trace_size++]=value;}\n"
observer.REJECT = {
    "computed-root": ('fn i32 main(){for(let u8 byte in b"x"){discard byte;}return 7;}', "phase 5 code 11"),
    "mutable-descriptor": ('fn i32 main(){var ref u8[] data=b"x";for(let u8 byte in data){discard byte;}return 7;}', "phase 5 code 11"),
    "immutable-item": ('fn i32 main(){let ref u8[] data=b"x";for(let u8 byte in data){byte=9;}return 7;}', "code 4"),
    "outside-item": ('fn i32 main(){let ref u8[] data=b"x";for(let u8 byte in data){}return byte as i32;}', "phase 4 code 1"),
    "wrong-item": ('fn i32 main(){let ref u8[] data=b"x";for(let i32 byte in data){}return 7;}', "phase 5 code 11"),
    "owner-mutation": ('fn i32 main(){var b::Buffer x=b::empty();let ref u8[] data=b::view(ref x);for(let u8 byte in data){discard b::append_byte(ref mut x,byte);}return 7;}', "code 2101"),
    "owner-move": ('fn i32 main(){let b::Buffer x=b::empty();let ref u8[] data=b::view(ref x);for(let u8 byte in data){let b::Buffer moved=x;}return 7;}', "code 1104"),
}
if __name__ == "__main__":
    observer.main()
