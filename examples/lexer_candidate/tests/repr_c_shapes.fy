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
import c_emit as c;
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
fn m::Program without_abi(ref m::Program p){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,m::source_view(p)).is_ok);
    var bytes::Buffer payload=bytes::empty();var v::Vec<m::Operation> ops=v::empty<m::Operation>();var usize i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.payload_end>o.payload_start){assert(bytes::append(ref mut payload,m::literal_text(p,i)).is_ok);}assert(v::push<m::Operation>(ref mut ops,o).is_ok);i+=1;}
    var v::Vec<m::Target> targets=v::empty<m::Target>();i=1;while(i<=m::target_count(p)){assert(v::push<m::Target>(ref mut targets,m::target_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();i=0;while(i<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,i)}).is_ok);i+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();i=1;while(i<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();i=1;while(i<=m::local_count(p)){assert(v::push<m::Local>(ref mut locals,m::local_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();i=1;while(i<=m::block_count(p)){assert(v::push<m::Block>(ref mut blocks,m::block_at(p,i)).is_ok);i+=1;}
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<m::edge_count(p)){assert(v::push<usize>(ref mut edges,m::edge_at(p,i)).is_ok);i+=1;}
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();i=0;while(i<shapes::count(m::nominal(p))){assert(v::push<shapes::Shape>(ref mut records,shapes::at(m::nominal(p),i)).is_ok);i+=1;}
    var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();i=1;while(i<=shapes::field_count(m::nominal(p))){assert(v::push<shapes::Field>(ref mut fields,shapes::field_at(m::nominal(p),i)).is_ok);i+=1;}
    return switch(m::assemble_nominal(source,payload,targets,signatures,functions,locals,blocks,ops,edges,0,shapes::assemble(records,fields))){case .ok(p):{p}case .err(_):{assert(false);return without_abi(p);}};
}
@test fn void repr_c_foreign_result_requires_owned_permission(){
    let p=switch(lower::lower(owned(b"module main;@repr(C)struct Packet{u32 code;}unsafe extern C fn Packet fixture_packet();fn i32 main(){let Packet p=unsafe{fixture_packet()};return 7;}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);
    var bytes::Buffer good=bytes::empty();assert(c::render(ref p,ref mut good)==0);
    let bad=without_abi(ref p);assert(verify::check(ref bad).code==5106);
    var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);
    assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));
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
