module tests::resource_dead;
import tests::result_types as fixture;
import tests::hir_nominal as hir;
import type_check as typing;
import hir_model as h;
import std::bytes as bytes;

@test
fn void consumed_root_reuse_has_exact_use_and_transfer_spans() {
    let source=b"module main;resource R{i32 n;cleanup{}}fn void take(R x){}fn i32 main(){let R donor=R{.n=7};take(donor);return donor.n;}";
    let t=typing::check(fixture::resolved(source));let problem=typing::problem(ref t);
    assert(problem.code==45 && problem.move_start<problem.start && problem.move_end<problem.end);
    assert(bytes::equal(source[problem.start..problem.end],b"donor") && bytes::equal(source[problem.move_start..problem.move_end],b"donor"));
    assert(bytes::equal(source[problem.move_start-5..problem.move_start],b"take("));
}

@test
fn void fresh_replacement_shadowing_and_unreachable_uses_do_not_reuse_donors() {
    let p=hir::retained(b"module main;resource R{i32 n;cleanup{}}fn void take(R x){}fn i32 main(){var R r=R{.n=1};take(r);r=R{.n=2};{let R r=R{.n=3};take(r);};return r.n;take(r);}");
    assert(h::function_count(ref p)==2);
    let source=b"module main;resource R{cleanup{}}fn void take(R a,R b){}fn void main(){let R r=R{};take(r,r);}";
    let t=typing::check(fixture::resolved(source));let problem=typing::problem(ref t);
    assert(problem.code==45 && problem.start==problem.move_start+2);
}

@test
fn void consumed_tagged_payloads_and_inspection_cannot_revive_an_owner() {
    let source=b"module main;resource R{cleanup{}}fn Result<R,i32> bad(Result<R,i32> x){let R r=x?;return x;}";
    let t=typing::check(fixture::resolved(source));let problem=typing::problem(ref t);
    assert(problem.code==45 && bytes::equal(source[problem.move_start..problem.move_end],b"x") && bytes::equal(source[problem.start..problem.end],b"x"));
    let inspected=b"module main;resource R{cleanup{}}fn bool bad(Result<R,i32> x){let bool ok=x.is_ok;return x.is_ok;}";
    let checked=typing::check(fixture::resolved(inspected));assert(typing::problem(ref checked).code==45);
}
