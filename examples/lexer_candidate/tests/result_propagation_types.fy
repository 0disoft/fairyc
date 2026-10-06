module tests::result_propagation_types;
import tests::result_types as fixture;
import name_resolution as names;
import syntax as syntax;
import type_check as typing;
import tagged_types as tagged;
import std::bytes as bytes;

@test
fn void propagation_records_success_type_without_requiring_same_success_return() {
    let typed=typing::check(fixture::resolved(b"module main;fn Result<bool,bool> f(Result<i64,bool> r){let i64 n=r?;return .ok(n>0);}"));
    assert(typing::problem(ref typed).code==0);let ref syntax::Parsed p=names::parsed(typing::resolved(ref typed));
    var usize i=1;var usize count=0;
    while(i<=syntax::expression_count(p)){let e=syntax::expression_at(p,i);if(e.kind==30){
        assert(typing::expression_type(ref typed,i)==2 && typing::expression_type(ref typed,e.left)==tagged::make_result(2,6));count+=1;
    }i+=1;}assert(count==1);
}
fn void rejected(ref u8[] source,u32 code,u32 expected,u32 actual) {
    let typed=typing::check(fixture::resolved(source));let p=typing::problem(ref typed);
    assert(p.code==code && p.expected==expected && p.actual==actual);
    assert(p.end==p.start+1 && bytes::equal(typing::source_view(ref typed)[p.start..p.end],b"?"));
}
@test
fn void propagation_errors_point_to_question_and_preserve_ordered_error_identity() {
    rejected(b"module main;fn Result<i32,bool> f(i32 r){let n=r?;return .ok(n);}",40,0,1);
    rejected(b"module main;fn Result<i32,bool> f(Option<i32> r){let n=r?;return .ok(n);}",40,0,196609);
    rejected(b"module main;fn i32 f(Result<i32,bool> r){return r?;}",41,0,6);
    rejected(b"module main;fn Result<i32,i32> f(Result<i32,bool> r){let n=r?;return .ok(n);}",41,1,6);
    rejected(b"module main;fn Result<i32,i64> f(Result<i32,i32> r){let n=r?;return .ok(n);}",41,2,1);
    rejected(b"module main;fn Result<bool,i32> f(Result<i32,bool> r){let n=r?;return .ok(n>0);}",41,1,6);
}
@test
fn void propagation_is_allowed_in_function_body_but_not_classic_for_header() {
    let typed=typing::check(fixture::resolved(b"module main;fn void log(i32 n){}fn Result<i32,bool> f(Result<i32,bool> r){defer log(r?);var i32 n=0;while(n<1){n=r?;}return .ok(n);}"));
    assert(typing::problem(ref typed).code==0);
    let header=typing::check(fixture::resolved(b"module main;fn Result<i32,bool> f(Result<i32,bool> r){for(let i32 n=r?;n<1;n+=1){}return .ok(0);}"));
    assert(typing::problem(ref header).code==14);
}
