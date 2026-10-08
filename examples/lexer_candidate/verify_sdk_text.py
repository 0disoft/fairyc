"""Execute original String/CString/UTF-8 SDK bodies and target runtime."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "sdk-text"
observer.SDK = observer.REPO / "internal/sdkstdlib/sources/std/text.fy"
observer.DEPENDENCIES = (observer.REPO / "internal/sdkstdlib/sources/std/bytes.fy",)
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import std::text as t;import std::bytes as b;"
observer.RUNTIME_NAMES += ("stage0_buffer.c", "stage0_text_view.c", "stage0_owned_text.c")
observer.CASES = {
    "string-copy-append-move": observer.FOREIGN + r'''
fn t::String copied(ref str value){return switch(t::copy(value)){case .ok(x):{x}case .err(_):{assert(false);return copied("");}};}
fn i32 main(){
    {
        var t::String x=copied("한");assert(t::string_len(ref x)==3 && !t::string_is_empty(ref x));
        assert(t::append(ref mut x,"글").is_ok && b::equal(t::as_bytes(t::string_as_str(ref x)),b"한글"));
        let t::String moved=x;assert(t::string_len(ref moved)==6);
        let t::String empty=copied("");assert(t::string_is_empty(ref empty));
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
''',
    "buffer-string-no-copy": observer.FOREIGN + r'''
fn i32 main(){
    {
        let b::Buffer x=switch(b::copy(b"hello")){case .ok(x):{x}case .err(_):{assert(false);b::empty()}};
        let ptr u8 address=unsafe{b::view(ref x) as ptr u8};let usize live=unsafe{fixture_live()};
        let t::String text=switch(t::from_buffer(x)){case .ok(s):{s}case .err(_):{assert(false);return 0;}};
        assert(unsafe{fixture_live()}==live && unsafe{t::string_as_str(ref text) as ptr u8}==address);
        let b::Buffer roundtrip=t::into_buffer(text);assert(unsafe{fixture_live()}==live);
        assert(unsafe{b::view(ref roundtrip) as ptr u8}==address && b::equal(b::view(ref roundtrip),b"hello"));
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
''',
    "invalid-utf8-donor-return": observer.FOREIGN + r'''
fn i32 main(){
    {
        let b::Buffer x=switch(b::copy(b"a\xFFz")){case .ok(x):{x}case .err(_):{assert(false);b::empty()}};
        let ptr u8 address=unsafe{b::view(ref x) as ptr u8};let usize capacity=b::capacity(ref x);
        let b::Buffer returned=switch(t::from_buffer(x)){case .err(x):{x}case .ok(_):{assert(false);b::empty()}};
        assert(b::capacity(ref returned)==capacity && b::equal(b::view(ref returned),b"a\xFFz"));
        assert(unsafe{b::view(ref returned) as ptr u8}==address);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
''',
    "string-allocation-failure": observer.FOREIGN + r'''
fn i32 main(){
    {
        unsafe{fixture_fail_next();}assert(switch(t::copy("hello")){case .err(e):{t::error_status(e)==1 && t::error_allocation_status(e)!=0}case .ok(_):{false}});
        assert(unsafe{fixture_live()}==0);
        unsafe{fixture_fail_next();}assert(switch(t::cstring_from_text("x")){case .err(e):{t::error_status(e)==1 && t::error_allocation_status(e)!=0}case .ok(_):{false}});
        assert(unsafe{fixture_live()}==0);
        var t::String x=switch(t::copy("hello")){case .ok(x):{x}case .err(_):{assert(false);return 0;}};
        unsafe{fixture_fail_next();}assert(!t::append(ref mut x,"abcdefghijklmnopqrstuvwxyz0123456789").is_ok);
        assert(t::string_len(ref x)==5 && b::equal(t::as_bytes(t::string_as_str(ref x)),b"hello"));
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
''',
    "cstring-nul-contract": observer.FOREIGN + r'''
unsafe extern C fn i32 fixture_cstr(ptr u8 value,usize length);
fn i32 main(){
    {
        let t::CString x=switch(t::cstring_from_text("한")){case .ok(x):{x}case .err(_):{assert(false);return 0;}};
        assert(t::cstring_len(ref x)==3 && !t::cstring_is_empty(ref x));
        assert(unsafe{fixture_cstr(t::cstring_as_cstr(ref x) as ptr u8,3)}==7);
        assert(switch(t::cstring_from_text("a\0z")){case .err(e):{t::error_status(e)==3}case .ok(_):{false}});
        let t::CString empty=switch(t::cstring_from_text("")){case .ok(x):{x}case .err(_):{assert(false);return 0;}};
        assert(t::cstring_is_empty(ref empty) && unsafe{fixture_cstr(t::cstring_as_cstr(ref empty) as ptr u8,0)}==7);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
''',
    "utf8-validation-origin": observer.FOREIGN + r'''
fn Result<ref str,t::TextError> checked(ref u8[] value){return t::validate_utf8(value);}
fn i32 main(){
    {
        let b::Buffer x=switch(b::copy(b"한글")){case .ok(x):{x}case .err(_):{assert(false);b::empty()}};
        let ref str text=switch(checked(b::view(ref x))){case .ok(s):{s}case .err(_):{assert(false);""}};
        assert(text.len==6 && b::equal(t::as_bytes(text),b"한글"));
        assert(switch(checked(b"\xC0\x80")){case .err(e):{t::error_status(e)==2}case .ok(_):{false}});
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
''',
}
observer.SUFFIX += """
int32_t fixture_cstr(const uint8_t *value,uintptr_t length){
    if(value[length]!=0)return 0;
    return length==0 || length==3 && value[0]==237 && value[1]==149 && value[2]==156 ? 7 : 0;
}
"""
observer.TRAPS = {}
observer.REJECT = {
    "retained-string-view": ('fn i32 main(){var t::String x=switch(t::copy("x")){case .ok(s):{s}case .err(_):{return 0;}};let ref str s=t::string_as_str(ref x);discard t::append(ref mut x,"y");discard s.len;return 7;}', "code 2101"),
    "retained-string-transfer": ('fn i32 main(){let t::String x=switch(t::copy("x")){case .ok(s):{s}case .err(_):{return 0;}};let ref str s=t::string_as_str(ref x);let b::Buffer y=t::into_buffer(x);discard s.len;return 7;}', "code 2101"),
    "retained-validated-buffer": ('fn i32 main(){var b::Buffer x=b::empty();let ref str s=switch(t::validate_utf8(b::view(ref x))){case .ok(s):{s}case .err(_):{return 0;}};discard b::clear(ref mut x);discard s.len;return 7;}', "code 2101"),
    "local-string-escape": ('fn ref str bad(){let t::String x=switch(t::copy("x")){case .ok(s):{s}case .err(_):{return "";}};return t::string_as_str(ref x);}fn i32 main(){return 7;}', "code 2102"),
    "local-validated-escape": ('fn ref str bad(){let b::Buffer x=b::empty();return switch(t::validate_utf8(b::view(ref x))){case .ok(s):{s}case .err(_):{""}};}fn i32 main(){return 7;}', "code 2102"),
    "mixed-borrow-join": ('fn ref str bad(ref str a,ref str b,bool first){let Option<ref str> x=if(first){.some(a)}else{.none};return switch(x){case .some(s):{s}case .none:{b}};}fn i32 main(){return 7;}', "code 2103"),
    "stolen-text-symbol": ('@link_name("fy_std_text_utf8_valid_v0")unsafe extern C fn u32 stolen(ptr u8 p,usize length);fn i32 main(){return 7;}', "code 5106"),
}

if __name__ == "__main__":
    observer.main()
