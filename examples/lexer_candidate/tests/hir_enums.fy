module tests::hir_enums;
import tests::hir_nominal as support;
import hir_model as h;
import hir_shapes as shapes;
import hir_output as output;
import std::bytes as bytes;

@test
fn void enum_layout_context_and_written_switch_order_survive_inputs() {
    let p=support::retained(b"module main;enum State{Cold;Ready;Done;}struct Pair{i32 value;}fn State next(State s){return switch(s){case .Done:{.Cold}case .Cold:{.Ready}case .Ready:{.Done}};}fn i32 main(){var State s=.Cold;s=next(s);return switch(s){case .Done:{2}case .Ready:{1}case .Cold:{0}};}");
    let ref shapes::Table table=h::nominal(ref p);assert(shapes::has_enums(table));assert(shapes::count(table)==2 && shapes::field_count(table)==4);
    let layout=shapes::at(table,0);assert(layout.type==131073 && layout.field_count==3);
    assert(shapes::field_at(table,1).type==0 && shapes::field_at(table,4).type==1);
    var usize i=1;var usize variants=0;var usize switches=0;
    while(i<=h::expression_count(ref p)){let e=h::expression_at(ref p,i);let u32 kind=h::expression_code(e.kind);
        if(kind==18){assert(h::type_code(e.type)==layout.type && e.binding>=1 && e.binding<=3);variants+=1;}
        if(kind==19){switches+=1;assert(e.edge_count==3 && e.left!=0);let first=h::expression_at(ref p,h::edge_at(ref p,e.edge_start));assert(h::expression_code(first.kind)==20 && first.binding==3);}
        i+=1;
    }assert(variants==4 && switches==2);
    var bytes::Buffer a=bytes::empty();var bytes::Buffer b=bytes::empty();assert(output::render(ref p,ref mut a)==0 && output::render(ref p,ref mut b)==0);assert(bytes::equal(bytes::view(ref a),bytes::view(ref b)));
}
