module tests::mir_option_flow;
import tests::hir_nominal as fixture;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;

@test
fn void early_option_payload_return_leaves_no_parent_edges() {
    let p=switch(lower::lower(fixture::retained(b"module main;fn i32 main(){let Option<i32> v=.some({return 6;7});return switch(v){case .some(n):{n}case .none:{0}};}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0 && m::edge_count(ref p)==0);var usize i=1;
    while(i<=m::operation_count(ref p)){assert(m::operation_at(ref p,i).kind!=21);i+=1;}
    var bytes::Buffer out=bytes::empty();assert(c::render(ref p,ref mut out)==0);
}
@test
fn void literal_assert_option_arm_has_no_value_join_edge() {
    let p=switch(lower::lower(fixture::retained(b"module main;fn i32 main(){let Option<i32> v=.some(7);return switch(v){case .none:{assert(false);}case .some(n):{n}};}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);var bytes::Buffer out=bytes::empty();assert(c::render(ref p,ref mut out)==0);
}
