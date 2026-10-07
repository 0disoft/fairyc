module tests::mir_options;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_verify as verify;
import mir_output as output;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Local local(u32 type) { return m::Local{.function=1,.binding=0,.type,.parameter=false,.mutable=true,.start=0,.end=1}; }
fn shapes::Table layout(u32 mutation) {
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();
    var usize i=0;while(i<2){let u32 type=if(i==0){196609}else{196610};let usize base=i*3;
        assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type,.binding=if(mutation==1){1}else{0},.field_start=base,.field_count=3,.public=mutation==2}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=if(mutation==3){6}else if(i==0){1}else{2},.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=0,.start=1,.end=2,.payload_start=if(mutation==4){base+1}else{base},.payload_count=1}).is_ok);
        assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=0,.start=2,.end=3,.payload_start=0,.payload_count=if(mutation==5){1}else{0}}).is_ok);i+=1;
    }return shapes::assemble(records,fields);
}
fn m::Program forged(u32 mutation) {
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"main").is_ok);var bytes::Buffer payload=bytes::empty();assert(bytes::append(ref mut payload,b"4").is_ok);
    var v::Vec<m::Target> targets=v::empty<m::Target>();assert(v::push<m::Target>(ref mut targets,m::Target{ .unsafe_call=false,.effects=0,.binding=1,.module_index=0,.declaration=1,.result=1,.signature_start=0,.signature_count=0,.start=0,.end=4,.external=false}).is_ok);
    var v::Vec<m::Function> functions=v::empty<m::Function>();assert(v::push<m::Function>(ref mut functions,m::Function{.target=1,.entry=1,.local_start=0,.local_count=5,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=4,.public=false}).is_ok);
    var v::Vec<m::Local> locals=v::empty<m::Local>();assert(v::push<m::Local>(ref mut locals,local(1)).is_ok);assert(v::push<m::Local>(ref mut locals,local(196609)).is_ok);assert(v::push<m::Local>(ref mut locals,local(1)).is_ok);assert(v::push<m::Local>(ref mut locals,local(1)).is_ok);assert(v::push<m::Local>(ref mut locals,local(196610)).is_ok);
    var m::Operation value=state::operation(1,1,1,1,0,0,0,0,1);value.function=1;value.block=1;value.payload_end=1;value.next=2;
    var m::Operation make=state::operation(21,2,196609,196609,0,0,0,0,1);make.function=1;make.block=1;make.argument_count=1;make.next=3;
    var m::Operation take=state::operation(23,3,1,196609,1,2,0,0,1);take.function=1;take.block=1;
    if(mutation==1){make.owner_type=196610;}if(mutation==2){make.type=196610;}if(mutation==3){make.operator=2;}if(mutation==4){make.operator=1;}
    if(mutation==5){make.argument_count=0;}if(mutation==6){take.owner_type=196610;}if(mutation==7){take.operator=4;}if(mutation==8){take.operator=2;}
    if(mutation==9){take.type=2;}if(mutation==10){take.left=5;}if(mutation==11){take.argument_count=1;}if(mutation==14){take.right=1;}if(mutation==15){take.kind=2;take.owner_type=1;take.operator=0;take.left=4;}
    var v::Vec<usize> edges=v::empty<usize>();assert(v::push<usize>(ref mut edges,if(mutation==12){2}else if(mutation==13){4}else{1}).is_ok);
    var v::Vec<m::Operation> ops=v::empty<m::Operation>();assert(v::push<m::Operation>(ref mut ops,value).is_ok);assert(v::push<m::Operation>(ref mut ops,make).is_ok);assert(v::push<m::Operation>(ref mut ops,take).is_ok);
    var m::Block done=state::block(1,0,4);done.kind=1;done.value=3;done.first=1;done.last=3;var v::Vec<m::Block> blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return switch(m::assemble_nominal(source,payload,targets,v::empty<m::Signature>(),functions,locals,blocks,ops,edges,0,layout(0))){case .ok(p):{p}case .err(_):{assert(false);return forged(0);}};
}
@test
fn void option_layouts_cannot_forge_payload_type_or_members() {
    let good=layout(0);assert(shapes::valid(ref good,4,1));var u32 mutation=1;
    while(mutation<=5){let p=layout(mutation);assert(!shapes::valid(ref p,4,1));mutation+=1;}
}
@test
fn void option_mir_rejects_wrong_instances_tags_payloads_and_initialization() {
    let good=forged(0);assert(verify::check(ref good).code==0);var u32 mutation=1;
    while(mutation<=15){let p=forged(mutation);assert(verify::check(ref p).code==if(mutation==13 || mutation==15){1002}else{1001});
        var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);
        assert(output::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));
        assert(c::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;
    }
}
