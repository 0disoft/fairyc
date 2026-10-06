module tests::tagged_arrays_hir;
import tests::hir_nominal as fixture;
import tests::result_types as fixture_types;
import type_check as typing;
import hir_model as h;
import hir_shapes as shapes;
import array_types as arrays;
import tagged_types as tagged;
import mir_lower as mir;
import std::vec as v;

@test
fn void composite_array_hir_owns_element_layouts_and_keeps_scalar_ids() {
    let p=fixture::retained(b"module main;fn i32 main(){var Option<i64>[2] a=[.some(4000000000),.none];let saved=a;a[0]=.none;return switch(saved[0]){case .some(n):{assert(n==4000000000);7}case .none:{0}};}");
    let ref shapes::Table table=h::nominal(ref p);assert(shapes::count(table)==2 && shapes::has_owned_arrays(table));
    var usize i=0;var u32 array=0;
    while(i<shapes::count(table)){let s=shapes::at(table,i);if(arrays::owned(s.type)){array=s.type;}i+=1;}
    assert(array!=0 && arrays::length(array)==2 && shapes::array_element(table,array)==tagged::make_option(2));
    assert(arrays::make(1,2)==1041 && arrays::element(1041)==1);
    assert(switch(mir::lower(p)){case .ok(_):{false}case .err(problem):{problem.code==4}});
}

@test
fn void array_layout_verifier_rejects_missing_wrong_elements_and_cycles() {
    let u32 array=524288+1024*64;
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=array,.binding=0,.field_start=0,.field_count=1,.public=false}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=array,.type=1,.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    let wrong=shapes::assemble(records,fields);assert(!shapes::valid(ref wrong,1,0));
    var v::Vec<shapes::Shape> missing_records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> missing_fields=v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut missing_records,shapes::Shape{.type=array,.binding=0,.field_start=0,.field_count=1,.public=false}).is_ok);
    assert(v::push<shapes::Field>(ref mut missing_fields,shapes::Field{.owner=array,.type=tagged::make_option(1),.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    let missing=shapes::assemble(missing_records,missing_fields);assert(!shapes::valid(ref missing,1,0));
    var v::Vec<shapes::Shape> cycle_records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> cycle_fields=v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut cycle_records,shapes::Shape{.type=256,.binding=1,.field_start=0,.field_count=1,.public=false}).is_ok);
    assert(v::push<shapes::Shape>(ref mut cycle_records,shapes::Shape{.type=524288,.binding=0,.field_start=1,.field_count=1,.public=false}).is_ok);
    assert(v::push<shapes::Field>(ref mut cycle_fields,shapes::Field{.owner=256,.type=524288,.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    assert(v::push<shapes::Field>(ref mut cycle_fields,shapes::Field{.owner=524288,.type=256,.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    let forged_cycle=shapes::assemble(cycle_records,cycle_fields);assert(!shapes::valid(ref forged_cycle,1,1));
    let cycle=typing::check(fixture_types::resolved(b"module main;struct Node{Option<Node>[1] children;}fn i32 main(){return 0;}"));
    assert(typing::problem(ref cycle).code==20);
    let repeat=fixture::retained(b"module main;fn i32 main(){let Result<i32,bool>[2] a=[.ok(7);2];return switch(a[1]){case .ok(n):{n}case .err(_):{0}};}");
    assert(shapes::has_owned_arrays(h::nominal(ref repeat)));
}
