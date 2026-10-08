"""Observe authenticated Vec<u8> views without replacing SDK product logic."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "vector-byte-views"
observer.CASE_OBSERVER = Path(__file__)
observer.CASES = {
    "empty-prefix": observer.FOREIGN + """
fn i32 main(){
    var v::Vec<u8> x=v::empty<u8>();
    let ref u8[] a=v::view<u8>(ref x);let ref u8[] b=a[..];let ref u8[] c=b[0..0];assert(c.len==0 && c.is_empty);
    let ref mut u8[] m=v::view_mut<u8>(ref mut x);let ref mut u8[] n=m[..];assert(n.len==0 && n.is_empty);
    assert(v::len<u8>(ref x)==0 && v::capacity<u8>(ref x)==0 && unsafe{fixture_live()}==0);return 7;
}
""",
    "shared-aliases": observer.FOREIGN + """
fn ref u8[] tail(ref u8[] value){return value[1..];}
fn i32 main(){
    {
        var v::Vec<u8> x=v::empty<u8>();assert(v::push<u8>(ref mut x,11).is_ok && v::push<u8>(ref mut x,23).is_ok && v::push<u8>(ref mut x,37).is_ok);
        let ref u8[] a=v::view<u8>(ref x);let ref u8[] sibling=v::view<u8>(ref x);let ref u8[] b=tail(a);let ref u8[] c=b[1..];
        assert(a.len==3 && !a.is_empty && b.len==2 && c.len==1 && sibling[0]==11 && b[0]==23 && c[0]==37 && v::len<u8>(ref x)==3);
        let usize size=v::capacity<u8>(ref x);assert(v::reserve<u8>(ref mut x,size+1).is_ok);assert(v::push<u8>(ref mut x,41).is_ok);
        let ref u8[] fresh=v::view<u8>(ref x);assert(fresh.len==4 && fresh[0]==11 && fresh[3]==41);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "mutable-subview": observer.FOREIGN + """
fn ref mut u8[] tail(ref mut u8[] value){return value[1..];}
fn i32 main(){
    {
        var v::Vec<u8> x=v::empty<u8>();assert(v::push<u8>(ref mut x,11).is_ok && v::push<u8>(ref mut x,23).is_ok && v::push<u8>(ref mut x,37).is_ok);
        let ref mut u8[] a=v::view_mut<u8>(ref mut x);let ref mut u8[] alias=a;let ref mut u8[] child=tail(alias);
        child[0]=43;child[1]=47;assert(child[0]==43 && child[1]==47);a[0]=41;assert(a[0]==41 && a[1]==43);
        assert(v::push<u8>(ref mut x,53).is_ok);let ref u8[] fresh=v::view<u8>(ref x);assert(fresh.len==4 && fresh[0]==41 && fresh[1]==43 && fresh[2]==47 && fresh[3]==53);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "named-sdk-exports": observer.FOREIGN + """
fn i32 main(){
    {
        var v::Vec<u8> x=v::empty<u8>();assert(v::push<u8>(ref mut x,7).is_ok);
        let ref u8[] a=v::view_u8(ref x);assert(a.len==1 && a[0]==7);
        let ref mut u8[] b=v::view_mut_u8(ref mut x);b[0]=19;assert(b[0]==19);
        assert(*v::at<u8>(ref x,0)==19);assert(v::clear<u8>(ref mut x)==1);
        let ref u8[] empty=v::view_u8(ref x);assert(empty[..].len==0);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "array-view-regression": """
fn i32 main(){var u8[3] x=[11,23,37];let ref mut u8[] a=ref mut x;let ref mut u8[] b=a[1..];b[0]=43;assert(b[0]==43);a[2]=47;assert(a[2]==47);let ref u8[] shared=ref x;assert(shared[1]==43 && shared[2]==47);return 7;}
""",
    "bounds-empty": "fn i32 main(){var v::Vec<u8> x=v::empty<u8>();let ref u8[] a=v::view<u8>(ref x);discard a[0];return 7;}",
    "bounds-prefix": "fn i32 main(){var v::Vec<u8> x=v::empty<u8>();assert(v::reserve<u8>(ref mut x,64).is_ok && v::push<u8>(ref mut x,7).is_ok);let ref u8[] a=v::view<u8>(ref x);discard a[1];return 7;}",
    "bounds-range": "fn i32 main(){var v::Vec<u8> x=v::empty<u8>();assert(v::push<u8>(ref mut x,7).is_ok);let ref mut u8[] a=v::view_mut<u8>(ref mut x);discard a[0..2];return 7;}",
}
observer.TRAPS = {name: 6 for name in observer.CASES if name.startswith("bounds-")}
prefix = "fn i32 main(){var v::Vec<u8> x=v::empty<u8>();assert(v::push<u8>(ref mut x,11).is_ok);"
observer.REJECT = {}
for name, change in {
    "shared-push": "discard v::push<u8>(ref mut x,23);",
    "shared-reserve": "discard v::reserve<u8>(ref mut x,1);",
    "shared-pop": "discard v::pop<u8>(ref mut x);",
    "shared-clear": "discard v::clear<u8>(ref mut x);",
    "shared-move": "let v::Vec<u8> moved=x;",
    "shared-replace": "x=v::empty<u8>();",
    "shared-exclusive-element": "let ref mut u8 b=v::at_mut<u8>(ref mut x,0);*b=23;",
}.items():
    observer.REJECT[name] = (prefix + "let ref u8[] a=v::view<u8>(ref x);" + change + "assert(a[0]==11);return 7;}", "code 2101")
observer.REJECT.update({
    "retained-subview": ("fn ref u8[] alias(ref u8[] a){return a[..];}" + prefix + "let ref u8[] a=v::view<u8>(ref x);let ref u8[] b=alias(a);discard v::push<u8>(ref mut x,23);assert(b[0]==11);return 7;}", "code 2101"),
    "mutable-owner-read": (prefix + "let ref mut u8[] a=v::view_mut<u8>(ref mut x);discard v::len<u8>(ref x);a[0]=23;return 7;}", "code 2101"),
    "mutable-sibling": (prefix + "let ref mut u8[] a=v::view_mut<u8>(ref mut x);let ref mut u8[] b=a[..];let ref mut u8[] c=a[..];b[0]=23;c[0]=37;return 7;}", "code 2101"),
    "child-freezes-parent": (prefix + "let ref mut u8[] a=v::view_mut<u8>(ref mut x);let ref mut u8[] b=a[..];a[0]=23;assert(b[0]==23);return 7;}", "code 2101"),
    "later-argument": ("fn u8 combine(ref u8[] a,usize n){return a[0];}" + prefix + "discard combine(v::view<u8>(ref x),v::clear<u8>(ref mut x));return 7;}", "code 2101"),
    "slice-bound": ("fn usize alter(ref mut v::Vec<u8> x){discard v::clear<u8>(x);return 0;}" + prefix + "let ref u8[] a=v::view<u8>(ref x);discard a[alter(ref mut x)..];return 7;}", "code 2101"),
    "local-escape": ("fn ref u8[] escaped(){var v::Vec<u8> x=v::empty<u8>();return v::view<u8>(ref x);}fn i32 main(){return 7;}", "code 2102"),
    "stolen-shared-storage": (prefix + "discard unsafe{storage_borrow<u8[]>(ref x,null,0)};return 7;}", "code 5102"),
    "stolen-mutable-storage": (prefix + "discard unsafe{storage_borrow_mut<u8[]>(ref mut x,null,0)};return 7;}", "code 5102"),
    "wrong-width": ("fn i32 main(){var v::Vec<i32> x=v::empty<i32>();discard v::view<i32>(ref x);return 7;}", "phase 4 code 3"),
})

if __name__ == "__main__":
    observer.main()
