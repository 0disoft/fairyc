"""Execute original std::bytes Buffer bodies and target runtime bytes."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "sdk-buffer"
observer.SDK = observer.REPO / "internal/sdkstdlib/sources/std/bytes.fy"
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import std::bytes as b;"
observer.RUNTIME_NAMES += ("stage0_buffer.c",)
observer.CASES = {
    "buffer-decimal-formatting": observer.FOREIGN + """
fn i32 main(){
    {
        var b::Buffer x=b::empty();
        assert(b::append_i32(ref mut x,-2147483648).is_ok && b::append_byte(ref mut x,44).is_ok);
        assert(b::append_i64(ref mut x,-9223372036854775808).is_ok && b::append_byte(ref mut x,44).is_ok);
        assert(b::append_u32(ref mut x,4294967295).is_ok && b::append_byte(ref mut x,44).is_ok);
        assert(b::append_usize(ref mut x,18446744073709551615).is_ok);
        assert(b::equal(b::view(ref x),b"-2147483648,-9223372036854775808,4294967295,18446744073709551615"));
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "buffer-lifecycle": observer.FOREIGN + """
fn i32 main(){
    {
        var b::Buffer x=b::empty();assert(b::is_empty(ref x) && b::capacity(ref x)==0 && b::view(ref x)[..].is_empty);
        assert(b::append(ref mut x,b"ab").is_ok && b::append_byte(ref mut x,99).is_ok);
        assert(b::equal(b::view(ref x),b"abc"));let usize capacity=b::capacity(ref x);
        assert(b::reserve(ref mut x,capacity+1).is_ok && b::equal(b::view(ref x),b"abc"));
        assert(b::resize_zeroed(ref mut x,6).is_ok);let ref u8[] a=b::view(ref x);assert(a.len==6 && a[0]==97 && a[3]==0 && a[5]==0);
        let ref mut u8[] m=b::view_mut(ref mut x);let ref mut u8[] child=m[1..3];child[0]=120;assert(child[0]==120);m[2]=121;assert(m[2]==121);
        assert(b::truncate(ref mut x,3).is_ok && b::equal(b::view(ref x),b"axy"));
        assert(b::clear(ref mut x).is_ok && b::len(ref x)==0);assert(b::append_bool(ref mut x,true).is_ok);
        assert(b::equal(b::view(ref x),b"true"));
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "buffer-copy-and-move": observer.FOREIGN + """
fn b::Buffer copied(){return switch(b::copy(b"a\\0z")){case .ok(x):{x}case .err(_):{assert(false);b::empty()}};}
fn i32 main(){
    {
        let b::Buffer x=copied();assert(b::equal(b::view(ref x),b"a\\0z"));let b::Buffer moved=x;
        assert(b::len(ref moved)==3 && b::view(ref moved)[1]==0);
        var b::Buffer reserved=switch(b::with_capacity(17)){case .ok(x):{x}case .err(_):{assert(false);b::empty()}};
        assert(b::capacity(ref reserved)>=17 && b::is_empty(ref reserved));assert(b::append(ref mut reserved,b::view(ref moved)).is_ok);
        assert(b::equal(b::view(ref reserved),b"a\\0z"));
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "buffer-allocation-failure": observer.FOREIGN + """
fn i32 main(){
    {
        unsafe{fixture_fail_next();}assert(!b::with_capacity(17).is_ok);assert(unsafe{fixture_live()}==0);
        var b::Buffer x=b::empty();assert(b::append(ref mut x,b"hello").is_ok);let usize capacity=b::capacity(ref x);
        unsafe{fixture_fail_next();}assert(!b::resize_zeroed(ref mut x,capacity+1).is_ok);
        assert(b::len(ref x)==5 && b::capacity(ref x)==capacity && b::equal(b::view(ref x),b"hello"));
        assert(!b::truncate(ref mut x,6).is_ok && b::equal(b::view(ref x),b"hello"));
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
}
observer.TRAPS = {}
observer.REJECT = {
    "retained-buffer-view": ("fn i32 main(){var b::Buffer x=b::empty();let ref u8[] a=b::view(ref x);discard b::clear(ref mut x);discard a.len;return 7;}", "code 2101"),
    "retained-buffer-move": ("fn i32 main(){let b::Buffer x=b::empty();let ref u8[] a=b::view(ref x);let b::Buffer y=x;discard a.len;return 7;}", "code 2101"),
    "shared-buffer-store": ("fn i32 main(){var b::Buffer x=b::empty();let ref u8[] a=b::view(ref x);a[0]=7;return 7;}", "phase 5 code 4"),
    "buffer-local-escape": ("fn ref u8[] escaped(){let b::Buffer x=b::empty();return b::view(ref x);}fn i32 main(){return 7;}", "code 2102"),
    "stolen-buffer-symbol": ('@link_name("fy_std_buffer_drop_v0")unsafe extern C fn u32 stolen(ptr mut void p);fn i32 main(){return 7;}', "code 5106"),
}

if __name__ == "__main__":
    observer.main()
