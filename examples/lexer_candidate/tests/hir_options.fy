module tests::hir_options;
import tests::hir_nominal as fixture;
import hir_model as h;
import hir_shapes as shapes;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;
import std::bytes as bytes;

@test
fn void options_own_canonical_payloads_and_lower_to_tag_operations() {
    let ref u8[] source=b"module main;fn Option<i64> f(){return .some(4000000000);}fn i32 main(){let Option<bool> empty=.none;let i64 n=switch(f()){case .some(n):{n}case .none:{0}};assert(n==4000000000);return switch(empty){case .none:{0}case .some(_):{1}};}";
    let p=fixture::retained(source);let ref shapes::Table table=h::nominal(ref p);
    assert(shapes::has_options(table) && shapes::count(table)==2 && shapes::field_count(table)==6);
    assert(shapes::at(table,0).type==196610 && shapes::at(table,0).binding==0 && shapes::at(table,1).type==196614);
    assert(shapes::field_at(table,1).type==2 && shapes::field_at(table,4).type==6);
    var usize i=1;var usize variants=0;var usize captures=0;
    while(i<=h::expression_count(ref p)){let e=h::expression_at(ref p,i);if(h::expression_code(e.kind)==18){variants+=1;assert(e.binding==2 || e.binding==6);}if(h::expression_code(e.kind)==21){captures+=1;}i+=1;}
    assert(variants==2 && captures==2);
    let mir=switch(lower::lower(fixture::retained(source))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref mir).code==0);i=1;var usize takes=0;
    while(i<=m::operation_count(ref mir)){let o=m::operation_at(ref mir,i);if(o.kind==21){assert(o.owner_type==196610 || o.owner_type==196614);}if(o.kind==23){takes+=1;assert(o.owner_type==196610 && o.type==2);}i+=1;}
    assert(takes==1);
}
