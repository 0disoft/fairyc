module tests::array_field_views;
import tests::hir_nominal as fixture;
import hir_model as h;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;
import reference_types as refs;
import array_types as arrays;
import c_emit as c;
import std::bytes as bytes;

@test
fn void owned_subrange_selects_original_nested_field_storage() {
    let h::Program hir=fixture::retained(b"module main;struct Inner{u8[4] bytes;}struct Packet{Inner inner;usize length;}fn ref u8[] data(ref Packet p){return ref p.inner.bytes[1..p.length];}fn i32 main(){return 7;}");
    var usize borrows=0;var usize slices=0;var usize i=1;
    while(i<=h::expression_count(ref hir)){
        let e=h::expression_at(ref hir,i);let u32 kind=h::expression_code(e.kind);
        if(kind==16){assert(h::type_code(e.type)==10 && h::expression_code(h::expression_at(ref hir,e.left).kind)==10);borrows+=1;}
        if(kind==17){assert(h::expression_code(h::expression_at(ref hir,e.left).kind)==16 && h::type_code(e.type)==10);slices+=1;}
        i+=1;
    }
    assert(borrows==1 && slices==1);
    let mir=switch(lower::lower(hir)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref mir).code==0);
    var usize fields=0;var usize views=0;var usize ranges=0;i=1;
    while(i<=m::operation_count(ref mir)){
        let o=m::operation_at(ref mir,i);
        if(o.kind==32){fields+=1;assert(refs::valid(o.type));}
        if(o.kind==18){views+=1;assert(refs::valid(o.owner_type) && arrays::valid(refs::inner(o.owner_type)) && arrays::length(refs::inner(o.owner_type))==4 && o.type==10);}
        if(o.kind==19){ranges+=1;assert(o.type==10 && o.owner_type==10);}
        i+=1;
    }
    assert(fields==2 && views==1 && ranges==1);
    var bytes::Buffer out=bytes::empty();assert(c::render(ref mir,ref mut out)==0 && bytes::len(ref out)!=0);
}

@test fn void mutable_record_parameter_keeps_original_array_field_storage(){
    let hir=fixture::retained(b"module main;struct Index{usize[4] slots;}fn void set(ref mut Index p,usize at,usize value){let ref mut usize[] slots=ref mut p.slots;slots[at]=value;}fn i32 main(){var Index p=Index{.slots=[0;4]};set(ref mut p,2,7);assert(p.slots[2]==7);return 7;}");
    let mir=switch(lower::lower(hir)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(verify::check(ref mir).code==0);
    var usize fields=0;var usize views=0;var usize stores=0;var usize i=1;
    while(i<=m::operation_count(ref mir)){let o=m::operation_at(ref mir,i);if(o.kind==32){assert(refs::mutable(o.type));fields+=1;}if(o.kind==18){assert(o.type==27 && refs::mutable(o.owner_type));views+=1;}if(o.kind==20){assert(o.type==5 && o.owner_type==27);stores+=1;}i+=1;}
    assert(fields==1 && views==1 && stores==1);var bytes::Buffer out=bytes::empty();assert(c::render(ref mir,ref mut out)==0 && bytes::len(ref out)!=0);
}
