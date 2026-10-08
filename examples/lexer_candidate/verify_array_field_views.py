"""Execute fixed byte-array field borrows and checked subranges."""
from pathlib import Path
import verify_sdk_vector as observer
observer.SUITE = "array-field-views"
observer.SDK = observer.REPO / "internal/sdkstdlib/sources/std/bytes.fy"
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import std::bytes as b;"
observer.RUNTIME_NAMES += ("stage0_buffer.c",)
observer.CASES = {
    "mutable-record-parameter": '''
struct Index{usize[4] slots;}
fn void set(ref mut Index p,usize at,usize value){let ref mut usize[] slots=ref mut p.slots;slots[at]=value;}
fn i32 main(){var Index p=Index{.slots=[0;4]};set(ref mut p,2,7);assert(p.slots[2]==7 && p.slots[0]==0);return 7;}
''',
    "nested-shared-origin": r'''
struct Inner{u8[4] bytes;}struct Packet{Inner inner;usize length;}
fn ref u8[] tail(ref Packet p){return ref p.inner.bytes[1..p.length];}
fn i32 main(){let Packet p=Packet{.inner=Inner{.bytes=[1,2,3,4]},.length=3};
    let ref u8[] full=ref p.inner.bytes;let ref u8[] again=ref p.inner.bytes[..];
    assert(unsafe{full as ptr u8}==unsafe{again as ptr u8});
    let ref u8[] data=tail(ref p);assert(data.len==2 && data[0]==2 && data[1]==3);return 7;}
''',
    "exclusive-original-storage": r'''
struct Inner{u8[4] bytes;}struct Packet{Inner inner;usize length;}
fn i32 main(){var Packet p=Packet{.inner=Inner{.bytes=[1,2,3,4]},.length=4};
    let ref mut u8[] data=ref mut p.inner.bytes[1..];data[0]=7;assert(data.len==3 && data[0]==7);
    assert(p.inner.bytes[1]==7 && p.inner.bytes[0]==1 && p.length==4);return 7;}
''',
    "once-and-empty": r'''
struct S{u8[4] bytes;}
fn usize low(ref mut usize calls){*calls=*calls+1;return 1;}
fn usize high(ref mut usize calls){*calls=*calls+1;return 3;}
fn i32 main(){let S s=S{.bytes=[1,2,3,4]};var usize calls=0;
    let ref u8[] data=ref s.bytes[low(ref mut calls)..high(ref mut calls)];
    assert(calls==2 && data.len==2 && data[0]==2 && data[1]==3);
    let ref u8[] empty=ref s.bytes[4..];assert(empty.len==0);return 7;}
''',
    "named-array-subranges": r'''
fn i32 main(){var u8[4] a=[1,2,3,4];let ref mut u8[] data=ref mut a[1..3];data[0]=7;
    assert(data.len==2 && data[0]==7);assert(a[1]==7);
    let i64[3] nums=[1,4000000000,3];let ref i64[] wide=ref nums[1..];assert(wide.len==2 && wide[0]==4000000000);return 7;}
''',
    "reversed": 'struct S{u8[4] bytes;}fn i32 main(){let S s=S{.bytes=[1,2,3,4]};let ref u8[] data=ref s.bytes[2..1];return 7;}',
    "past-end": 'struct S{u8[4] bytes;}fn i32 main(){let S s=S{.bytes=[1,2,3,4]};let ref u8[] data=ref s.bytes[..5];return 7;}',
}
observer.TRAPS = {"reversed":6,"past-end":6}
observer.REJECT = {
    "shared-record-parameter": ('struct Index{usize[4] slots;}fn void bad(ref Index p){let ref mut usize[] slots=ref mut p.slots;slots[0]=1;}fn i32 main(){return 7;}',"phase 5 code 4"),
    "record-parameter-sibling-held": ('struct Index{usize[4] slots;usize count;}fn void bad(ref mut Index p){let ref mut usize[] slots=ref mut p.slots;p.count=1;discard slots[0];}fn i32 main(){return 7;}',"code 2101"),
    "local-escape": ('struct S{u8[4] bytes;}fn ref u8[] bad(){let S s=S{.bytes=[1,2,3,4]};return ref s.bytes[1..];}fn i32 main(){return 7;}',"code 2102"),
    "sibling-write-held": ('struct S{u8[4] bytes;usize count;}fn i32 main(){var S s=S{.bytes=[1,2,3,4],.count=0};let ref u8[] data=ref s.bytes[1..];s.count=2;return data[0] as i32;}',"code 2101"),
    "replacement-held": ('struct S{u8[4] bytes;}fn i32 main(){var S s=S{.bytes=[1,2,3,4]};let ref u8[] data=ref s.bytes[1..];s=S{.bytes=[0;4]};return data[0] as i32;}',"code 2101"),
    "bound-mutates-owner": ('struct S{u8[4] bytes;usize count;}fn usize bound(ref mut S s){s.count=2;return 2;}fn i32 main(){var S s=S{.bytes=[1,2,3,4],.count=0};let ref u8[] data=ref s.bytes[..bound(ref mut s)];return data[0] as i32;}',"code 2101"),
    "immutable-mutable-borrow": ('struct S{u8[4] bytes;}fn i32 main(){let S s=S{.bytes=[1,2,3,4]};let ref mut u8[] data=ref mut s.bytes[..];return 7;}',"phase 5 code 4"),
    "temporary-field": ('struct S{u8[4] bytes;}fn S make(){return S{.bytes=[1,2,3,4]};}fn i32 main(){let ref u8[] data=ref make().bytes[..];return 7;}',"phase 5 code 16"),
    "wrong-bound": ('struct S{u8[4] bytes;}fn i32 main(){let S s=S{.bytes=[1,2,3,4]};let ref u8[] data=ref s.bytes[true..];return 7;}',"phase 5 code 1"),
}
if __name__ == "__main__":
    observer.main()
