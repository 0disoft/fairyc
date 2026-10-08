"""Observe owner-bound scalar Vec element borrows through real SDK ASTs."""
import verify_sdk_vector as observer
from pathlib import Path

observer.SUITE = "vector-borrows"
observer.CASE_OBSERVER = Path(__file__)
observer.CASES = {
    "shared-last-use": observer.FOREIGN + """
fn ref i32 forwarded(ref i32 value){return value;}
fn i32 main(){
    {
        var v::Vec<i32> x=v::empty<i32>();
        assert(v::push<i32>(ref mut x,17).is_ok && v::push<i32>(ref mut x,29).is_ok);
        let ref i32 first=v::at<i32>(ref x,0);
        let ref i32 alias=forwarded(first);
        let ref i32 second=v::at<i32>(ref x,1);
        assert(v::len<i32>(ref x)==2 && *alias==17 && *second==29);
        let usize grown=v::capacity<i32>(ref x)+1;
        assert(v::reserve<i32>(ref mut x,grown).is_ok);
        assert(v::push<i32>(ref mut x,41).is_ok);
        assert(*v::at<i32>(ref x,0)==17 && *v::at<i32>(ref x,2)==41);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "exclusive-resume": observer.FOREIGN + """
fn ref i32 forwarded(ref i32 value){return value;}
fn i32 main(){
    {
        var v::Vec<i32> x=v::empty<i32>();assert(v::push<i32>(ref mut x,11).is_ok);
        let ref mut i32 element=v::at_mut<i32>(ref mut x,0);
        let ref mut i32 alias=element;
        let ref i32 child=forwarded(alias);assert(*child==11);
        *alias=37;assert(*element==37);
        assert(v::push<i32>(ref mut x,43).is_ok);
        assert(switch(v::pop<i32>(ref mut x)){case .some(n):{n==43}case .none:{false}});
        assert(*v::at<i32>(ref x,0)==37);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "scalar-elements": observer.FOREIGN + """
fn i32 main(){
    {
        var v::Vec<u8> a=v::empty<u8>();assert(v::push<u8>(ref mut a,255).is_ok);
        assert(*v::at<u8>(ref a,0)==255);let ref mut u8 aa=v::at_mut<u8>(ref mut a,0);*aa=7;assert(*aa==7);
        var v::Vec<i64> b=v::empty<i64>();assert(v::push<i64>(ref mut b,-9223372036854775808).is_ok);
        assert(*v::at<i64>(ref b,0)==-9223372036854775808);let ref mut i64 bb=v::at_mut<i64>(ref mut b,0);*bb=9223372036854775807;assert(*bb==9223372036854775807);
        var v::Vec<u32> c=v::empty<u32>();assert(v::push<u32>(ref mut c,4294967295).is_ok);
        assert(*v::at<u32>(ref c,0)==4294967295);let ref mut u32 cc=v::at_mut<u32>(ref mut c,0);*cc=19;assert(*cc==19);
        var v::Vec<usize> d=v::empty<usize>();assert(v::push<usize>(ref mut d,18446744073709551615).is_ok);
        assert(*v::at<usize>(ref d,0)==18446744073709551615);let ref mut usize dd=v::at_mut<usize>(ref mut d,0);*dd=23;assert(*dd==23);
        var v::Vec<bool> e=v::empty<bool>();assert(v::push<bool>(ref mut e,true).is_ok);
        assert(*v::at<bool>(ref e,0));let ref mut bool ee=v::at_mut<bool>(ref mut e,0);*ee=false;assert(!*ee);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "independent-owners": observer.FOREIGN + """
fn i32 main(){
    {
        var v::Vec<i32> a=v::empty<i32>();var v::Vec<i32> b=v::empty<i32>();
        assert(v::push<i32>(ref mut a,11).is_ok && v::push<i32>(ref mut b,13).is_ok);
        let ref mut i32 aa=v::at_mut<i32>(ref mut a,0);let ref mut i32 bb=v::at_mut<i32>(ref mut b,0);
        *aa=19;*bb=23;assert(*aa==19 && *bb==23);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "bounds-empty": "fn i32 main(){var v::Vec<i32> x=v::empty<i32>();discard *v::at<i32>(ref x,0);return 7;}",
    "bounds-length": "fn i32 main(){var v::Vec<i32> x=v::empty<i32>();assert(v::push<i32>(ref mut x,11).is_ok);discard *v::at_mut<i32>(ref mut x,1);return 7;}",
    "bounds-max": "fn i32 main(){var v::Vec<i64> x=v::empty<i64>();assert(v::push<i64>(ref mut x,11).is_ok);discard *v::at<i64>(ref x,18446744073709551615);return 7;}",
}
# SDK at() asserts its initialized-prefix bound before computing a raw address.
observer.TRAPS = {name: 5 for name in observer.CASES if name.startswith("bounds-")}
prefix = "fn i32 main(){var v::Vec<i32> x=v::empty<i32>();assert(v::push<i32>(ref mut x,11).is_ok);"
observer.REJECT = {}
for name, change in {
    "shared-push": "discard v::push<i32>(ref mut x,23);",
    "shared-reserve": "discard v::reserve<i32>(ref mut x,64);",
    "shared-pop": "discard v::pop<i32>(ref mut x);",
    "shared-clear": "discard v::clear<i32>(ref mut x);",
    "shared-replace": "x=v::empty<i32>();",
    "shared-move": "let v::Vec<i32> moved=x;",
    "shared-exclusive": "let ref mut i32 other=v::at_mut<i32>(ref mut x,0);*other=3;",
}.items():
    observer.REJECT[name] = (prefix + "let ref i32 a=v::at<i32>(ref x,0);" + change + "assert(*a==11);return 7;}", "code 2101")
observer.REJECT.update({
    "exclusive-owner-read": (prefix + "let ref mut i32 a=v::at_mut<i32>(ref mut x,0);discard v::len<i32>(ref x);*a=3;return 7;}", "code 2101"),
    "exclusive-second": (prefix + "let ref mut i32 a=v::at_mut<i32>(ref mut x,0);let ref mut i32 b=v::at_mut<i32>(ref mut x,0);*a=3;*b=5;return 7;}", "code 2101"),
    "child-freezes-parent": ("fn ref i32 forwarded(ref i32 value){return value;}" + prefix + "let ref mut i32 a=v::at_mut<i32>(ref mut x,0);let ref i32 child=forwarded(a);*a=3;assert(*child==3);return 7;}", "code 2101"),
    "escaped-owner": ("fn ref i32 escaped(){var v::Vec<i32> x=v::empty<i32>();assert(v::push<i32>(ref mut x,11).is_ok);return v::at<i32>(ref x,0);}fn i32 main(){return 7;}", "code 2102"),
    "stolen-borrow": (prefix + "discard unsafe{storage_borrow<i32>(ref x,null)};return 7;}", "code 5102"),
    "stolen-address": ("fn i32 main(){discard unsafe{storage_element_address<i32>(null,0)};return 7;}", "code 5102"),
    "retained-call-alias": ("fn ref i32 forwarded(ref i32 value){return value;}" + prefix + "let ref i32 a=v::at<i32>(ref x,0);let ref i32 alias=forwarded(a);discard v::push<i32>(ref mut x,23);assert(*alias==11);return 7;}", "code 2101"),
    "loop-reentry": (prefix + "let ref i32 a=v::at<i32>(ref x,0);var usize i=0;while(i<2){assert(*a==11);discard v::push<i32>(ref mut x,23);i+=1;}return 7;}", "code 2101"),
    "later-argument": ("fn i32 combined(ref i32 a,usize count){return *a;}" + prefix + "discard combined(v::at<i32>(ref x,0),v::clear<i32>(ref mut x));return 7;}", "code 2101"),
})

if __name__ == "__main__":
    observer.main()
