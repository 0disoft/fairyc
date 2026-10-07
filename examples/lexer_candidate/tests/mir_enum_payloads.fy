module tests::mir_enum_payloads;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_verify as verify;
import mir_output as output;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Local local(u32 type) {
    return m::Local{.function=1,.binding=0,.type,.parameter=false,.mutable=true,.start=0,.end=1};
}
fn m::Program forged(u32 mutation) {
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"main").is_ok);
    var bytes::Buffer payload=bytes::empty();assert(bytes::append(ref mut payload,b"4").is_ok);
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=131073,.binding=1,.field_start=0,.field_count=4,.public=false}).is_ok);
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=131074,.binding=2,.field_start=4,.field_count=2,.public=false}).is_ok);
    var usize i=0;while(i<3){let u32 owner=if(i<2){131073}else{131074};
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner,.type=1,.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner,.type=0,.start=1,.end=2,.payload_start=i*2,.payload_count=1}).is_ok);i+=1;
    }
    var v::Vec<m::Target> targets=v::empty<m::Target>();assert(v::push<m::Target>(ref mut targets,m::Target{ .foreign=false,.link_start=0,.link_end=0, .unsafe_call=false,.effects=0,.binding=3,.module_index=0,.declaration=1,.result=1,.signature_start=0,.signature_count=0,.start=0,.end=4,.external=false}).is_ok);
    var v::Vec<m::Function> functions=v::empty<m::Function>();assert(v::push<m::Function>(ref mut functions,m::Function{.target=1,.entry=1,.local_start=0,.local_count=4,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=4,.public=false}).is_ok);
    var v::Vec<m::Local> locals=v::empty<m::Local>();assert(v::push<m::Local>(ref mut locals,local(1)).is_ok);assert(v::push<m::Local>(ref mut locals,local(131073)).is_ok);assert(v::push<m::Local>(ref mut locals,local(1)).is_ok);assert(v::push<m::Local>(ref mut locals,local(1)).is_ok);
    var m::Operation value=state::operation(1,1,1,1,0,0,0,0,1);value.function=1;value.block=1;value.payload_end=1;value.next=2;
    var m::Operation make=state::operation(21,2,131073,131073,0,0,0,0,1);make.function=1;make.block=1;make.argument_count=1;make.next=3;
    var m::Operation take=state::operation(23,3,1,131073,1,2,0,0,1);take.function=1;take.block=1;
    if(mutation==1){make.argument_count=0;}if(mutation==2){make.operator=2;}if(mutation==3){take.operator=0;}if(mutation==4){take.operator=2;}
    if(mutation==5){take.operator=5;}if(mutation==6){take.owner_type=131074;}if(mutation==7){take.argument_count=1;}if(mutation==8){take.right=1;}
    if(mutation==9){take.type=6;}if(mutation==10){take.left=1;}if(mutation==11){make.operator=1;}
    var v::Vec<usize> edges=v::empty<usize>();assert(v::push<usize>(ref mut edges,if(mutation==12){2}else if(mutation==13){4}else{1}).is_ok);
    var v::Vec<m::Operation> ops=v::empty<m::Operation>();assert(v::push<m::Operation>(ref mut ops,value).is_ok);assert(v::push<m::Operation>(ref mut ops,make).is_ok);assert(v::push<m::Operation>(ref mut ops,take).is_ok);
    var m::Block done=state::block(1,0,4);done.kind=1;done.value=3;done.first=1;done.last=3;var v::Vec<m::Block> blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return switch(m::assemble_nominal(source,payload,targets,v::empty<m::Signature>(),functions,locals,blocks,ops,edges,0,shapes::assemble(records,fields))){case .ok(p):{p}case .err(_):{assert(false);return forged(0);}};
}
@test
fn void payload_operations_reject_malformed_edges_fields_and_reads() {
    let good=forged(0);assert(verify::check(ref good).code==0);
    var bytes::Buffer out=bytes::empty();assert(c::render(ref good,ref mut out)==0);
    var u32 mutation=1;while(mutation<=13){
        let p=forged(mutation);
        if(mutation==11){
            // Structurally valid extraction from another active member is guarded
            // dynamically before union access; it is not a type-verifier error.
            assert(verify::check(ref p).code==0);var bytes::Buffer code=bytes::empty();assert(c::render(ref p,ref mut code)==0);
        }else{
            assert(verify::check(ref p).code==if(mutation==13){1002}else{1001});
            var bytes::Buffer code=bytes::empty();assert(bytes::append(ref mut code,b"prefix").is_ok);
            assert(output::render(ref p,ref mut code)!=0 && bytes::equal(bytes::view(ref code),b"prefix"));
            assert(c::render(ref p,ref mut code)!=0 && bytes::equal(bytes::view(ref code),b"prefix"));
        }mutation+=1;
    }
}
