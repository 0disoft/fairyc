"""Execute local mixed provenance while retaining every possible owner."""
from pathlib import Path
import verify_sdk_vector as observer
observer.SUITE="mixed-origins"
observer.SDK=observer.REPO/"internal/sdkstdlib/sources/std/bytes.fy"
observer.DEPENDENCIES=[observer.REPO/"internal/sdkstdlib/sources/optional/std/vec.fy"]
observer.PREAMBLE="module main;import std::bytes as b;import std::vec as v;"
observer.CASE_OBSERVER=Path(__file__)
observer.RUNTIME_NAMES+=("stage0_buffer.c",)
observer.CASES={
"input-or-cooked-vector":observer.FOREIGN+r'''
fn ref u8[] identity(ref u8[] data){return data;}
fn usize count(ref u8[] input,bool cooked){
    var v::Vec<b::Buffer> values=v::empty<b::Buffer>();
    let b::Buffer text=switch(b::copy(b"\x03\x04")){case .ok(x):{x}case .err(_):{return 0;}};
    assert(v::push<b::Buffer>(ref mut values,text).is_ok);
    let ref u8[] payload=if(cooked){b::view(v::at<b::Buffer>(ref values,0))}else{input};
    let ref u8[] alias=identity(payload);var usize sum=0;
    for(let u8 byte in alias){sum+=byte as usize;}return sum;
}
fn i32 main(){assert(count(b"\x01\x02",false)==3 && count(b"\x01\x02",true)==7);assert(unsafe{fixture_live()}==0);return 7;}
''',
"tagged-and-static":r'''
fn i32 run(bool first){var u8[2] a=[2,3];var u8[2] b=[4,5];
    let ref u8[] av=ref a;let ref u8[] bv=ref b;
    let Option<ref u8[]> selected=if(first){.some(av)}else{.some(bv)};
    let ref u8[] data=switch(selected){case .some(x):{x}case .none:{b"\x00"}};
    assert(data.len==2 && data[0]==if(first){2}else{4});
    a[0]=9;b[0]=9;let ref u8[] after=if(first){ref a}else{b"\x09"};assert(after[0]==9);return 7;
}fn i32 main(){assert(run(true)==7 && run(false)==7);return 7;}
''',
"exclusive-selection":r'''
fn i32 edit(ref mut u8[] a,ref mut u8[] b,bool first){
    let ref mut u8[] selected=if(first){a}else{b};selected[0]=7;assert(selected[0]==7);return 7;
}
fn i32 main(){var u8[1] a=[1];var u8[1] b=[2];assert(edit(ref mut a,ref mut b,true)==7 && a[0]==7 && b[0]==2);
    assert(edit(ref mut a,ref mut b,false)==7 && b[0]==7);return 7;}
''',
"nested-and-generic":r'''
fn T pick<T>(T marker,ref u8[] a,ref u8[] b,ref u8[] c,bool first,bool second){
    let ref u8[] selected=if(first){if(second){a}else{b}}else{c};assert(selected[0]>=1 && selected[0]<=3);return marker;
}fn i32 main(){let ref u8[] a=b"\x01";let ref u8[] b=b"\x02";let ref u8[] c=b"\x03";
    assert(pick<i32>(7,a,b,c,true,true)==7 && pick<bool>(true,a,b,c,true,false) && pick<i32>(7,a,b,c,false,false)==7);return 7;}
''',
}
observer.REJECT={
"left-owner-write":('fn i32 main(){var u8[1] a=[1];var u8[1] b=[2];let ref u8[] p=if(true){ref a}else{ref b};a[0]=9;return p[0] as i32;}',"code 2101"),
"right-owner-write":('fn i32 main(){var u8[1] a=[1];var u8[1] b=[2];let ref u8[] p=if(true){ref a}else{ref b};b[0]=9;return p[0] as i32;}',"code 2101"),
"resource-move-held":('fn i32 main(){let b::Buffer a=b::empty();let b::Buffer other=b::empty();let ref u8[] p=if(true){b::view(ref a)}else{b::view(ref other)};let b::Buffer moved=other;discard p.len;return 7;}',"code 2101"),
"inner-owner-escape":('fn i32 main(){let u8[1] a=[1];let ref u8[] p={let u8[1] b=[2];if(true){ref a}else{ref b}};return p[0] as i32;}',"code 2102"),
"ambiguous-input-return":('fn ref u8[] bad(ref u8[] a,ref u8[] b,bool first){return if(first){a}else{b};}fn i32 main(){return 7;}',"code 2103"),
"mixed-local-return":('fn ref u8[] bad(ref u8[] a,bool first){let u8[1] b=[2];return if(first){a}else{ref b};}fn i32 main(){return 7;}',"code 2103"),
"exclusive-left-parent":('fn i32 bad(ref mut u8[] a,ref mut u8[] b,bool first){let ref mut u8[] selected=if(first){a}else{b};a[0]=9;return selected[0] as i32;}fn i32 main(){return 7;}',"code 2101"),
"exclusive-right-parent":('fn i32 bad(ref mut u8[] a,ref mut u8[] b,bool first){let ref mut u8[] selected=if(first){a}else{b};b[0]=9;return selected[0] as i32;}fn i32 main(){return 7;}',"code 2101"),
}
if __name__=="__main__":observer.main()
