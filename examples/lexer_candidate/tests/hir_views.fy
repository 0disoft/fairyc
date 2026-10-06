module tests::hir_views;
import tests::hir_nominal as support;
import hir_model as h;
import hir_view_check as views;
import hir_output as output;
import std::bytes as bytes;

@test
fn void retained_shared_array_views_and_slices() {
    let p = support::retained(b"module main;fn ref u8[] tail(ref u8[] b){return b[1..];}fn i32 main(){let u8[3] a=[1,2,3];let ref u8[] b=ref a;let s=tail(b);assert(s.len==2 && s[0]==2);return 0;}");
    assert(views::check(ref p).code==0);
    var usize borrows=0;var usize slices=0;var usize i=1;
    while(i<=h::expression_count(ref p)){let e=h::expression_at(ref p,i);if(h::expression_code(e.kind)==16){borrows+=1;assert(h::type_code(e.type)==10 && h::expression_at(ref p,e.left).binding!=0);}if(h::expression_code(e.kind)==17){slices+=1;assert(e.left!=0 && e.right!=0 && e.alternative!=0 && e.boolean);}i+=1;}
    assert(borrows==1 && slices==1);var bytes::Buffer text=bytes::empty();assert(output::render(ref p,ref mut text)==0);
}
