module tests::result_handling;
import tests::result_types as fixture;
import tests::hir_nominal as owned;
import type_check as typing;
import hir_model as h;
import hir_output as output;
import mir_lower as lower;
import mir_verify as verify;
import mir_model as m;
import c_emit as c;
import std::bytes as bytes;

@test
fn void bare_result_call_has_exact_obligation_span() {
    let t=typing::check(fixture::resolved(b"module main;fn Result<i32,bool> make(){return .ok(1);}fn void f(){make();}"));
    let p=typing::problem(ref t);assert(p.code==42);
    assert(bytes::equal(typing::source_view(ref t)[p.start..p.end],b"make()"));
}
@test
fn void branch_and_zero_iteration_obligations_survive_joins() {
    let branch=typing::check(fixture::resolved(b"module main;fn void f(bool b,Result<i32,bool> r){if(b){discard r;}}"));
    assert(typing::problem(ref branch).code==42);
    let loop=typing::check(fixture::resolved(b"module main;fn void f(bool b,Result<i32,bool> r){while(b){discard r;break;}}"));
    assert(typing::problem(ref loop).code==42);
    let handled=typing::check(fixture::resolved(b"module main;fn void f(bool b,Result<i32,bool> r){if(b){discard r;return;}assert(r.is_ok);}"));
    assert(typing::problem(ref handled).code==0);
}
@test
fn void propagation_checks_other_pending_obligations_on_error_exit() {
    let t=typing::check(fixture::resolved(b"module main;fn Result<i32,bool> f(Result<i32,bool> r){let Result<i32,bool> other=.ok(1);let n=r?;discard other;return .ok(n);}"));
    let p=typing::problem(ref t);assert(p.code==42);
    assert(bytes::equal(typing::source_view(ref t)[p.start..p.end],b".ok(1)"));
}
@test
fn void owned_is_ok_survives_type_owner_and_lowers_to_verified_tag_test() {
    let p=owned::retained(b"module main;fn bool f(Result<i32,bool> r){return r.is_ok;}fn i32 main(){assert(f(.ok(1)));assert(!f(.err(true)));return 7;}");
    var usize found=0;var usize i=1;
    while(i<=h::expression_count(ref p)){let e=h::expression_at(ref p,i);if(h::expression_code(e.kind)==23){assert(h::type_code(e.type)==6 && e.left!=0);found+=1;}i+=1;}
    assert(found==1);var bytes::Buffer text=bytes::empty();assert(output::render(ref p,ref mut text)==0);
    let ir=switch(lower::lower(p)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(verify::check(ref ir).code==0);found=0;i=1;
    while(i<=m::operation_count(ref ir)){if(m::operation_at(ref ir,i).kind==22){found+=1;}i+=1;}assert(found==1);
    var bytes::Buffer generated=bytes::empty();assert(c::render(ref ir,ref mut generated)==0);
}
