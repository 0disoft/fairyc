module tests::mir_enums;
import tests::hir_nominal as fixture;
import hir_shapes as shapes;
import mir_model as m;
import mir_lower as lower;
import mir_state as state;
import mir_verify as verify;
import mir_output as output;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

@test
fn void enum_switch_lowers_to_owned_tags_and_checked_branches() {
    let p = switch(lower::lower(fixture::retained(b"module main;enum E{A;B;}fn E id(E e){return e;}fn i32 main(){var E e=.A;e=id(.B);return switch(e){case .B:{7}case .A:{4}};}"))) { case .ok(p): { p } case .err(_): { assert(false); return; } };
    assert(verify::check(ref p).code == 0 && shapes::has_enums(m::nominal(ref p)));
    var usize i = 1; var usize values = 0; var usize tests = 0;
    while (i <= m::operation_count(ref p)) { let o = m::operation_at(ref p,i); if(o.kind == 21) { values += 1; } if(o.kind == 22) { tests += 1; assert(o.owner_type == 131073 && o.type == 6 && o.operator == 1); } i += 1; }
    assert(values == 2 && tests == 1);
    var bytes::Buffer text = bytes::empty(); assert(output::render(ref p,ref mut text) == 0 && bytes::equal(bytes::view(ref text)[..19],b"fairyc.enum-mir/v1\n"));
    var bytes::Buffer code = bytes::empty(); assert(c::render(ref p,ref mut code) == 0);
}
fn m::Local local(u32 type) { return m::Local { .function=1,.binding=0,.type,.parameter=false,.mutable=true,.start=0,.end=1 }; }
fn m::Program mutated(u32 mutation) {
    var bytes::Buffer source=bytes::empty(); assert(bytes::append(ref mut source,b"fx").is_ok);
    var bytes::Buffer payload=bytes::empty(); assert(bytes::append(ref mut payload,b"0").is_ok);
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>(); var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=131073,.binding=1,.field_start=0,.field_count=1,.public=false}).is_ok);
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=131074,.binding=2,.field_start=1,.field_count=1,.public=false}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=131073,.type=0,.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=131074,.type=0,.start=1,.end=2,.payload_start=0,.payload_count=0}).is_ok);
    var v::Vec<m::Target> targets=v::empty<m::Target>(); assert(v::push<m::Target>(ref mut targets,m::Target{.binding=3,.module_index=0,.declaration=1,.result=1,.signature_start=0,.signature_count=0,.start=0,.end=1,.external=false}).is_ok);
    var v::Vec<m::Function> functions=v::empty<m::Function>(); assert(v::push<m::Function>(ref mut functions,m::Function{.target=1,.entry=1,.local_start=0,.local_count=5,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=2,.public=false}).is_ok);
    var v::Vec<m::Local> locals=v::empty<m::Local>(); assert(v::push<m::Local>(ref mut locals,local(131073)).is_ok); assert(v::push<m::Local>(ref mut locals,local(6)).is_ok); assert(v::push<m::Local>(ref mut locals,local(1)).is_ok); assert(v::push<m::Local>(ref mut locals,local(131073)).is_ok); assert(v::push<m::Local>(ref mut locals,local(131074)).is_ok);
    var m::Operation value=state::operation(21,1,131073,131073,0,0,0,0,1); value.function=1;value.block=1;value.next=2;
    var m::Operation test=state::operation(22,2,6,131073,0,1,0,0,1); test.function=1;test.block=1;test.next=3;
    var m::Operation zero=state::operation(1,3,1,1,0,0,0,0,1);zero.function=1;zero.block=1;zero.payload_end=1;
    if(mutation==1){value.operator=1;}if(mutation==2){value.owner_type=131074;}if(mutation==3){value.kind=2;value.left=4;}if(mutation==4){test.left=5;}if(mutation==5){test.right=1;}if(mutation==6){test.type=1;}if(mutation==7){value.boolean=true;}if(mutation==8){value.payload_end=1;}if(mutation==9){value.kind=2;value.left=5;}if(mutation==10){value.type=131074;}
    var v::Vec<m::Operation> ops=v::empty<m::Operation>();assert(v::push<m::Operation>(ref mut ops,value).is_ok);assert(v::push<m::Operation>(ref mut ops,test).is_ok);assert(v::push<m::Operation>(ref mut ops,zero).is_ok);
    var m::Block done=state::block(1,0,2);done.kind=1;done.value=3;done.first=1;done.last=3;var v::Vec<m::Block> blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return switch(m::assemble_nominal(source,payload,targets,v::empty<m::Signature>(),functions,locals,blocks,ops,v::empty<usize>(),0,shapes::assemble(records,fields))){case .ok(p):{p}case .err(_):{assert(false);return mutated(0);}};
}
@test
fn void enum_mir_rejects_bad_tags_types_and_uninitialized_values() {
    let good=mutated(0);assert(verify::check(ref good).code==0);var u32 mutation=1;
    while(mutation<=10){let p=mutated(mutation);assert(verify::check(ref p).code==if(mutation==3){1002}else{1001});var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(output::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));assert(c::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;}
}
