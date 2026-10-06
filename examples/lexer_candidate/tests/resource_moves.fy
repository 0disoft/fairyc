module tests::resource_moves;
import tests::hir_nominal as fixture;
import tests::result_types as types;
import type_check as typing;
import hir_model as h;
import hir_shapes as shapes;
import resource_identity as resources;
import std::bytes as bytes;

@test
fn void whole_root_transfers_and_copy_field_inspection_are_distinct() {
    let p=fixture::retained(b"module main;resource R{i64 n;cleanup{}}fn R relay(R source){let R first=source;var R second=first;second=R{.n=9};let i64 n=second.n;return second;}fn R main(){return relay(R{.n=7});}");
    var usize id=1;var usize moves=0;var usize reads=0;
    while(id<=h::expression_count(ref p)){let e=h::expression_at(ref p,id);
        if(h::expression_code(e.kind)==24){moves+=1;assert(resources::valid(h::type_code(e.type)) && e.binding!=0 && bytes::equal(h::source_view(ref p)[e.start..e.end],h::binding_text(ref p,e.binding)));}
        if(h::expression_code(e.kind)==3 && resources::valid(h::type_code(e.type))){reads+=1;}
        id+=1;
    }
    assert(moves==3 && reads==2);
}

@test
fn void tagged_roots_and_captures_keep_consuming_identity() {
    let p=fixture::retained(b"module main;resource R{cleanup{}}fn R take(R x){return x;}fn Option<R> relay(Option<R> x){return switch(x){case .some(r):{.some(take(r))}case .none:{.none}};}fn Result<R,i32> checked(Result<R,i32> x){let R r=x?;return .ok(r);}");
    var usize id=1;var usize moves=0;
    while(id<=h::expression_count(ref p)){let e=h::expression_at(ref p,id);if(h::expression_code(e.kind)==24){assert(shapes::needs_drop(h::type_code(e.type)));moves+=1;}id+=1;}
    assert(moves==5);
}

@test
fn void partial_resource_field_moves_reject_at_the_consuming_span() {
    let source=b"module main;resource R{cleanup{}}resource Box{R child;cleanup{}}fn R bad(Box b){return b.child;}";
    let typed=typing::check(types::resolved(source));let problem=typing::problem(ref typed);
    assert(problem.code==44 && bytes::equal(source[problem.start..problem.end],b"b.child"));
    let readable=fixture::retained(b"module main;resource R{i32 n;cleanup{}}resource Box{R child;cleanup{}}fn i32 read(Box b){return b.child.n;}");
    var usize id=1;while(id<=h::expression_count(ref readable)){assert(h::expression_code(h::expression_at(ref readable,id).kind)!=24);id+=1;}
}
