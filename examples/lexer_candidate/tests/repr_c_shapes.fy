module tests::repr_c_shapes;
import scanner as scanner;
import syntax as syntax;
import name_resolution as names;
import type_check as typing;
import hir_lower as hir;
import hir_model as h;
import hir_shapes as shapes;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;
import std::bytes as bytes;
import std::vec as v;

fn h::Program owned(ref u8[] input){
    var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,input).is_ok);
    let stream=switch(scanner::collect(b,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return owned(b"module main;");}};
    let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return owned(b"module main;");}};
    let resolved=names::resolve(parsed);assert(names::problem(ref resolved).code==0);
    let typed=typing::check(resolved);assert(typing::problem(ref typed).code==0);
    return switch(hir::lower(typed,0)){case .ok(p):{p}case .err(_):{assert(false);owned(b"module main;")}};
}
fn shapes::Table forged(u32 field_type,usize marker_end){
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();var v::Vec<shapes::Abi> abi=v::empty<shapes::Abi>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=257,.binding=1,.field_start=0,.field_count=1,.public=false}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=257,.type=field_type,.start=24,.end=25,.payload_start=0,.payload_count=0}).is_ok);
    assert(v::push<shapes::Abi>(ref mut abi,shapes::Abi{.type=257,.declaration=2,.name_start=16,.name_end=17,.marker_start=0,.marker_end}).is_ok);
    return shapes::assemble_abi(records,fields,abi);
}
@test fn void owned_repr_c_layout_survives_hir_and_mir(){
    let p=owned(b"module main;@repr /*marker*/ (C) struct Packet{u32 code;u32 amount;}fn Packet make(){return Packet{.code=1,.amount=7};}fn i32 main(){return 7;}");
    assert(shapes::abi_count(h::nominal(ref p))==1 && shapes::abi_valid(h::nominal(ref p),h::source_view(ref p)));
    let proof=shapes::abi_at(h::nominal(ref p),0);
    assert(bytes::equal(h::source_view(ref p)[proof.name_start..proof.name_end],b"Packet"));
    let program=switch(lower::lower(p)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref program).code==0 && shapes::abi_count(m::nominal(ref program))==1 && shapes::abi_valid(m::nominal(ref program),m::source_view(ref program)));
    let copied=shapes::abi_at(m::nominal(ref program),0);assert(copied.type==proof.type && copied.declaration==proof.declaration && copied.marker_start==proof.marker_start && copied.marker_end==proof.marker_end);
    // The synthetic owner is ABI-shaped but its source permission is checked.
    let ref u8[] text=b"@repr(C) struct A { i32 x; }";
    let correct=forged(1,8);assert(shapes::abi_valid(ref correct,text));
    let view=forged(10,8);assert(!shapes::abi_valid(ref view,text));
    let missing=forged(1,7);assert(!shapes::abi_valid(ref missing,text));
    assert(!shapes::abi_valid(ref correct,b"@repr(R) struct A { i32 x; }"));
    assert(!shapes::abi_valid(ref correct,b"@repr(C)"));
}
