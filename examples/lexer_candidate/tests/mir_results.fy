module tests::mir_results;
import tests::hir_nominal as fixture;
import tagged_types as tagged;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_lower as lower;
import mir_verify as verify;
import mir_output as output;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Local local(u32 type) { return m::Local{.function=1,.binding=0,.type,.parameter=false,.mutable=true,.start=0,.end=1}; }
fn shapes::Table layout() {
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    var usize i=0;while(i<2){let u32 type=if(i==0){tagged::make_result(1,6)}else{tagged::make_result(2,1)};let usize base=i*4;
        assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type,.binding=0,.field_start=base,.field_count=4,.public=false}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=tagged::success(type),.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=0,.start=1,.end=2,.payload_start=base,.payload_count=1}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=tagged::failure(type),.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=0,.start=2,.end=3,.payload_start=base+2,.payload_count=1}).is_ok);i+=1;
    }return shapes::assemble(records,fields);
}
// Construct, extract and test either alternative with different scalar types.
// Mutations keep arena indexes valid so rejection tests the executable facts.
fn m::Program forged(u32 mutation,bool error) {
    let u32 type=tagged::make_result(1,6);let u32 other=tagged::make_result(2,1);
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"main").is_ok);var bytes::Buffer payload=bytes::empty();assert(bytes::append(ref mut payload,b"4").is_ok);
    var v::Vec<m::Target> targets=v::empty<m::Target>();assert(v::push<m::Target>(ref mut targets,m::Target{ .unsafe_call=false,.effects=0,.binding=1,.module_index=0,.declaration=1,.result=1,.signature_start=0,.signature_count=0,.start=0,.end=4,.external=false}).is_ok);
    var v::Vec<m::Function> functions=v::empty<m::Function>();assert(v::push<m::Function>(ref mut functions,m::Function{.target=1,.entry=1,.local_start=0,.local_count=8,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=4,.public=false}).is_ok);
    var v::Vec<m::Local> locals=v::empty<m::Local>();
    assert(v::push<m::Local>(ref mut locals,local(1)).is_ok);assert(v::push<m::Local>(ref mut locals,local(6)).is_ok);
    assert(v::push<m::Local>(ref mut locals,local(type)).is_ok);assert(v::push<m::Local>(ref mut locals,local(1)).is_ok);assert(v::push<m::Local>(ref mut locals,local(6)).is_ok);
    assert(v::push<m::Local>(ref mut locals,local(other)).is_ok);assert(v::push<m::Local>(ref mut locals,local(6)).is_ok);assert(v::push<m::Local>(ref mut locals,local(type)).is_ok);
    var m::Operation value=state::operation(1,1,1,1,0,0,0,0,1);value.function=1;value.block=1;value.payload_end=1;value.next=2;
    var m::Operation boolean=state::operation(1,2,6,6,0,0,0,0,1);boolean.function=1;boolean.block=1;boolean.boolean=true;boolean.next=3;
    var m::Operation make=state::operation(21,3,type,type,if(error){1}else{0},0,0,0,1);make.function=1;make.block=1;make.argument_count=1;make.next=4;
    var m::Operation take=state::operation(23,if(error){5}else{4},if(error){6}else{1},type,if(error){3}else{1},3,0,0,1);take.function=1;take.block=1;take.next=5;
    var m::Operation test=state::operation(22,7,6,type,if(error){1}else{0},3,0,0,1);test.function=1;test.block=1;
    if(mutation==1){make.owner_type=other;}if(mutation==2){make.type=other;}if(mutation==3){make.operator=2;}if(mutation==4){make.operator=if(error){0}else{1};}
    if(mutation==5){make.argument_count=0;}if(mutation==6){take.owner_type=other;}if(mutation==7){take.operator=5;}if(mutation==8){take.operator=2;}
    if(mutation==9){take.type=2;}if(mutation==10){take.left=6;}if(mutation==11){take.argument_count=1;}if(mutation==14){take.right=1;}if(mutation==15){take.left=8;}
    if(mutation==16){test.operator=2;}if(mutation==17){test.type=1;}if(mutation==18){test.owner_type=other;}
    var v::Vec<usize> edges=v::empty<usize>();assert(v::push<usize>(ref mut edges,if(mutation==12){3}else if(mutation==13){if(error){5}else{4}}else if(error){2}else{1}).is_ok);
    var v::Vec<m::Operation> ops=v::empty<m::Operation>();assert(v::push<m::Operation>(ref mut ops,value).is_ok);assert(v::push<m::Operation>(ref mut ops,boolean).is_ok);assert(v::push<m::Operation>(ref mut ops,make).is_ok);assert(v::push<m::Operation>(ref mut ops,take).is_ok);assert(v::push<m::Operation>(ref mut ops,test).is_ok);
    var m::Block done=state::block(1,0,4);done.kind=1;done.value=1;done.first=1;done.last=5;var v::Vec<m::Block> blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return switch(m::assemble_nominal(source,payload,targets,v::empty<m::Signature>(),functions,locals,blocks,ops,edges,0,layout())){case .ok(p):{p}case .err(_):{assert(false);return forged(0,false);}};
}
@test
fn void result_mir_rejects_wrong_instances_tags_payloads_and_initialization() {
    var usize alternative=0;while(alternative<2){let bool error=alternative==1;let good=forged(0,error);assert(verify::check(ref good).code==0);
        var bytes::Buffer accepted=bytes::empty();assert(c::render(ref good,ref mut accepted)==0);
        var u32 mutation=1;while(mutation<=18){let p=forged(mutation,error);assert(verify::check(ref p).code==if(mutation==13 || mutation==15){1002}else{1001});
            var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);
            assert(output::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));
            assert(c::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;
        }alternative+=1;
    }
}
@test
fn void early_result_payload_return_leaves_no_parent_edges() {
    let p=switch(lower::lower(fixture::retained(b"module main;fn i32 main(){let Result<i32,bool> r=.ok({return 6;7});return switch(r){case .ok(n):{n}case .err(_):{0}};}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0 && m::edge_count(ref p)==0);var usize i=1;
    while(i<=m::operation_count(ref p)){assert(m::operation_at(ref p,i).kind!=21);i+=1;}
    var bytes::Buffer out=bytes::empty();assert(c::render(ref p,ref mut out)==0);
}
