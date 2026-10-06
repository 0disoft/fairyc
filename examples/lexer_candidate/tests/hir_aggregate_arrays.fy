module tests::hir_aggregate_arrays;
import tests::hir_nominal as support;
import hir_model as h;
import hir_shapes as shapes;
import array_types as arrays;
import hir_output as output;
import std::bytes as bytes;

@test
fn void canonical_rank_two_and_owned_array_fields() {
    let u32 row=arrays::make(3,2);let u32 grid=arrays::make(row,3);
    assert(arrays::valid(grid) && arrays::element(grid)==row && arrays::length(grid)==3);
    assert(arrays::make(grid,1)==0 && arrays::make(row,65)==0 && !arrays::valid(5120));
    let u32 largest=arrays::make(arrays::make(6,64),64);assert(arrays::valid(largest) && arrays::length(largest)==64);
    let h::Program p=support::retained(b"module main;struct B{u8[2][3] grid;i64[1] big;}struct A{B child;}fn B id(B b){return b;}fn i32 main(){var a=A{.child=B{.big=[4000000000],.grid=[[1,2],[3,4],[5,6]]}};a.child.grid[1][0]+=2;let b=id(a.child);assert(b.grid.len==3 && b.grid[0].len==2);return 0;}");
    let ref shapes::Table table=h::nominal(ref p);assert(shapes::count(table)==2 && shapes::field_count(table)==3);
    assert(shapes::field_at(table,1).type==grid && shapes::field_at(table,2).type==arrays::make(2,1));
    var usize constructors=0;var usize indexes=0;var usize id=1;
    while(id<=h::expression_count(ref p)){let e=h::expression_at(ref p,id);let u32 kind=h::expression_code(e.kind);
        if(kind==12){constructors+=1;let u32 type=h::type_code(e.type);assert(arrays::valid(type));var usize j=0;while(j<e.edge_count){assert(h::type_code(h::expression_at(ref p,h::edge_at(ref p,e.edge_start+j)).type)==arrays::element(type));j+=1;}}
        else if(kind==13){indexes+=1;assert(e.left<id && e.right<id);}id+=1;
    }
    assert(constructors==5 && indexes==3);var bytes::Buffer out=bytes::empty();assert(output::render(ref p,ref mut out)==0);
}
