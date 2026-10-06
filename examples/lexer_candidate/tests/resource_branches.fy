module tests::resource_branches;
import tests::result_types as fixture;
import tests::hir_nominal as hir;
import type_check as typing;
import hir_model as h;
import std::bytes as bytes;

@test
fn void ambiguous_join_rejects_even_without_a_subsequent_use() {
    let source=b"module main;resource R{cleanup{}}fn void take(R r){}fn void bad(bool b,R r){if(b){take(r);}}";
    let t=typing::check(fixture::resolved(source));let p=typing::problem(ref t);
    assert(p.code==46 && bytes::equal(source[p.start..p.end],b"if(b){take(r);}"));
    assert(bytes::equal(source[p.move_start..p.move_end],b"r"));
}

@test
fn void exited_paths_and_equal_live_or_dead_states_merge_correctly() {
    let live=hir::retained(b"module main;resource R{cleanup{}}fn void take(R r){}fn R good(bool b,R r){if(b){take(r);return R{};}return r;}");
    assert(h::function_count(ref live)==2);
    let dead=hir::retained(b"module main;resource R{cleanup{}}fn void take(R r){}fn void good(bool b,R r){if(b){take(r);}else{take(r);}}");
    assert(h::function_count(ref dead)==2);
    let restored=hir::retained(b"module main;resource R{cleanup{}}fn void take(R r){}fn R good(bool b,R input){var R r=input;if(b){take(r);r=R{};}return r;}");
    assert(h::function_count(ref restored)==2);
}

@test
fn void short_circuit_consumption_is_a_real_branch() {
    let source=b"module main;resource R{cleanup{}}fn bool bad(bool gate,Result<R,i32> x){return gate && x.is_ok;}";
    let t=typing::check(fixture::resolved(source));let p=typing::problem(ref t);
    assert(p.code==46 && bytes::equal(source[p.start..p.end],b"gate && x.is_ok") && bytes::equal(source[p.move_start..p.move_end],b"x"));
    let safe=hir::retained(b"module main;resource R{i32 n;cleanup{}}fn bool good(bool gate,R r){return gate && r.n==0;}");
    assert(h::function_count(ref safe)==1);
}
