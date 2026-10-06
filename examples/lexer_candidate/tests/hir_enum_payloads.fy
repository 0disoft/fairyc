module tests::hir_enum_payloads;
import tests::hir_nominal as support;
import hir_model as h;
import hir_shapes as shapes;
import hir_output as output;
import mir_lower as mir;
import std::bytes as bytes;
import std::vec as v;

@test
fn void owned_payloads_keep_types_order_bindings_and_discard() {
    let ref u8[] source=b"module main;enum E{None;Data{i64 n;bool ready;};}fn E make(){return .Data{.ready=true,.n=4000000000};}fn i64 main(){return switch(make()){case .Data{.ready=_,.n=value}:{value}case .None:{0}};}";
    let p=support::retained(source);
    let ref shapes::Table table=h::nominal(ref p);
    assert(shapes::has_payloads(table) && shapes::member_count(table,131073)==2);
    assert(shapes::field_count(table)==4 && shapes::field_at(table,4).payload_start==1 && shapes::field_at(table,4).payload_count==2);
    assert(shapes::field_at(table,2).type==2 && shapes::field_at(table,3).type==6);
    var usize i=1;var usize captures=0;var usize discards=0;var usize variants=0;
    while(i<=h::expression_count(ref p)){let e=h::expression_at(ref p,i);let u32 kind=h::expression_code(e.kind);
        if(kind==18){assert(e.binding==4 && e.edge_count==2);let first=h::expression_at(ref p,h::edge_at(ref p,e.edge_start));assert(first.binding==3 && h::expression_code(first.kind)==11);variants+=1;}
        if(kind==21){captures+=1;if(e.edge_count==0){discards+=1;assert(e.binding==3);}else{let binding=h::binding_at(ref p,h::edge_at(ref p,e.edge_start));assert(h::binding_code(binding.kind)==3 && h::type_code(binding.type)==2 && bytes::equal(h::binding_text(ref p,h::edge_at(ref p,e.edge_start)),b"value"));}}
        if(kind==20 && e.binding==4){assert(e.edge_count==2);let first=h::expression_at(ref p,h::edge_at(ref p,e.edge_start));assert(first.binding==3 && first.edge_count==0);}
        i+=1;
    }assert(captures==2 && discards==1 && variants==1);
    var bytes::Buffer a=bytes::empty();var bytes::Buffer b=bytes::empty();assert(output::render(ref p,ref mut a)==0 && output::render(ref p,ref mut b)==0);assert(bytes::equal(bytes::view(ref a),bytes::view(ref b)));
    // Until the execution slice lands, do not silently lower away payloads.
    assert(switch(mir::lower(support::retained(source))){case .ok(_):{false}case .err(problem):{problem.code==4}});
}

fn shapes::Table forged(usize start,usize count,u32 type) {
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=131073,.binding=1,.field_start=0,.field_count=2,.public=false}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=131073,.type,.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=131073,.type=0,.start=1,.end=2,.payload_start=start,.payload_count=count}).is_ok);
    return shapes::assemble(records,fields);
}
@test
fn void malformed_payload_ranges_and_orphans_fail_before_indexing() {
    let good=forged(0,1,2);assert(shapes::valid(ref good,2,1));
    let wrong_start=forged(1,1,2);assert(!shapes::valid(ref wrong_start,2,1));
    let too_many=forged(0,2,2);assert(!shapes::valid(ref too_many,2,1));
    let orphan=forged(0,0,2);assert(!shapes::valid(ref orphan,2,1));
    let bad_type=forged(0,1,9);assert(!shapes::valid(ref bad_type,2,1));
}
