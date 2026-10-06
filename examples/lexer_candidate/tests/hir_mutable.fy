module tests::hir_mutable;
import tests::hir_nominal as support;
import hir_model as h;
import hir_lower as lower;
import hir_output as output;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

fn u32 problem(ref u8[] source) {
    var bytes::Buffer input=bytes::empty();assert(bytes::append(ref mut input,source).is_ok);
    let stream=switch(scanner::collect(input,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return 0;}};
    let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return 0;}};
    return switch(lower::lower(typing::check(names::resolve(parsed)),0)){case .ok(_):{0}case .err(p):{p.code}};
}
@test
fn void mutable_generations_resume_after_last_child_use() {
    let p=support::retained(b"module main;fn i32 main(){var u8[3] a=[1,2,3];let ref mut u8[] b=ref mut a;let s=b[1..];let t=s[..1];t[0]=8;s[1]=9;b[0]=7;assert(a[0]==7 && a[1]==8 && a[2]==9);return 0;}");
    var usize borrow=0;var usize slices=0;var usize i=1;
    while(i<=h::expression_count(ref p)){let e=h::expression_at(ref p,i);let u32 kind=h::expression_code(e.kind);if(kind==16){assert(h::type_code(e.type)==11 && e.boolean);borrow+=1;}if(kind==17){assert(h::type_code(e.type)==11);slices+=1;}i+=1;}
    assert(borrow==1 && slices==2);var bytes::Buffer out=bytes::empty();assert(output::render(ref p,ref mut out)==0);
}
@test
fn void exclusive_parent_sibling_and_call_conflicts_are_rejected() {
    assert(problem(b"module main;fn i32 main(){var u8[2] a=[1,2];let ref mut u8[] b=ref mut a;let s=b[..1];b[1]=7;s[0]=8;return 0;}")==2101);
    assert(problem(b"module main;fn i32 main(){var u8[2] a=[1,2];let ref mut u8[] b=ref mut a;let x=b[..1];let y=b[1..];x[0]=7;y[0]=8;return 0;}")==2101);
    assert(problem(b"module main;fn i32 main(){var u8[1] a=[1];let ref u8[] c=ref a;let ref mut u8[] b=ref mut a;b[0]=2;discard c[0];return 0;}")==2101);
    assert(problem(b"module main;fn void fill(ref mut u8[] b,i32 n){discard n;b[0]=8;}fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;fill(b,{b[0]=7;0});return 0;}")==2101);
}
@test
fn void mutable_local_origin_cannot_escape() {
    assert(problem(b"module main;fn ref mut u8[] bad(){var u8[1] a=[1];return ref mut a;}fn i32 main(){return 0;}")==2102);
}
