module tests::tagged_fields;
import tests::hir_nominal as fixture;
import tests::result_types as types;
import type_check as typing;
import hir_model as h;
import hir_shapes as shapes;
import tagged_types as tagged;
import mir_model as m;
import mir_lower as lower;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;

@test
fn void tagged_fields_keep_owned_layouts_and_lower_reads_and_writes() {
    let p=fixture::retained(b"module main;struct Box{Result<i32,bool> value;i32 marker;}fn i32 main(){var b=Box{.value=.ok(1),.marker=2};b.value={b.marker=9;.ok(7)};assert(b.marker==9);return switch(b.value){case .ok(n):{n}case .err(_):{0}};}");
    let ref shapes::Table table=h::nominal(ref p);assert(shapes::count(table)==2);
    assert(shapes::field_at(table,1).type==tagged::make_result(1,6));
    let ir=switch(lower::lower(p)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(verify::check(ref ir).code==0);var usize i=1;var bool write=false;var bool read=false;
    while(i<=m::operation_count(ref ir)){let o=m::operation_at(ref ir,i);
        if(o.kind==8 && o.operator==1){write=true;}if(o.kind==7 && o.operator==1){read=true;}i+=1;
    }assert(write && read);var bytes::Buffer out=bytes::empty();assert(c::render(ref ir,ref mut out)==0);
}

@test
fn void tagged_field_cycles_and_bare_result_reads_are_rejected() {
    let option=typing::check(types::resolved(b"module main;struct Node{Option<Node> next;}fn i32 main(){return 0;}"));
    assert(typing::problem(ref option).code==20);
    let result=typing::check(types::resolved(b"module main;struct Node{Result<i32,Node> next;}fn i32 main(){return 0;}"));
    assert(typing::problem(ref result).code==20);
    let indirect=typing::check(types::resolved(b"module main;struct A{Option<B> next;}struct B{A parent;}fn i32 main(){return 0;}"));
    assert(typing::problem(ref indirect).code==20);
    let unread=typing::check(types::resolved(b"module main;struct Box{Result<i32,bool> value;}fn i32 main(){let b=Box{.value=.ok(7)};b.value;return 0;}"));
    let problem=typing::problem(ref unread);assert(problem.code==42);
    assert(bytes::equal(typing::source_view(ref unread)[problem.start..problem.end],b"b.value"));
    let nested=typing::check(types::resolved(b"module main;struct Inner{Option<i32> item;}struct Outer{Inner inner;}fn i32 main(){var b=Outer{.inner=Inner{.item=.some(1)}};b.inner.item=.none;return 0;}"));
    assert(typing::problem(ref nested).code==4);
    let sibling=typing::check(types::resolved(b"module main;struct Inner{i32 item;}struct Outer{Inner inner;Option<i32> tag;}fn i32 main(){var b=Outer{.inner=Inner{.item=1},.tag=.none};b.inner.item=2;return 0;}"));
    assert(typing::problem(ref sibling).code==4);
}
