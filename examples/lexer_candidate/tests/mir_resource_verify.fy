module tests::mir_resource_verify;
import mir_model as m;
import mir_state as state;
import mir_verify as verify;
import hir_shapes as shapes;
import std::bytes as bytes;
import std::vec as v;

fn m::Operation op(u32 kind,usize result,usize left){var m::Operation o=state::operation(kind,result,98304,98304,0,left,0,0,1);o.function=1;o.block=1;return o;}
fn u32 program(u32 mutation,bool body){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"rf").is_ok);
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=98304,.binding=1,.field_start=0,.field_count=0,.public=false}).is_ok);
    let shapes::Table table=shapes::assemble(records,v::empty<shapes::Field>());
    var v::Vec<m::Target> targets=v::empty<m::Target>();var v::Vec<m::Signature> signatures=v::empty<m::Signature>();
    assert(v::push<m::Target>(ref mut targets,m::Target{ .unsafe_call=false,.effects=0,.binding=2,.module_index=0,.declaration=2,.result=7,.signature_start=0,.signature_count=0,.start=1,.end=2,.external=false}).is_ok);
    if(body){assert(v::push<m::Target>(ref mut targets,m::Target{ .unsafe_call=false,.effects=0,.binding=3,.module_index=0,.declaration=1,.result=7,.signature_start=0,.signature_count=1,.start=0,.end=1,.external=false}).is_ok);assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=98304}).is_ok);}
    var v::Vec<m::Function> functions=v::empty<m::Function>();var v::Vec<m::Local> locals=v::empty<m::Local>();var v::Vec<m::Block> blocks=v::empty<m::Block>();var v::Vec<usize> edges=v::empty<usize>();
    let usize count=if(body){2}else{3};var usize i=0;
    while(i<count){assert(v::push<m::Local>(ref mut locals,m::Local{.function=1,.binding=0,.type=98304,.parameter=false,.mutable=false,.start=0,.end=1}).is_ok);i+=1;}
    assert(v::push<m::Function>(ref mut functions,m::Function{.target=1,.entry=1,.local_start=0,.local_count=count,.block_start=0,.block_count=if(body){2}else{1},.parameter_start=0,.parameter_count=0,.start=0,.end=2,.public=false}).is_ok);
    if(body){assert(v::push<m::Local>(ref mut locals,m::Local{.function=2,.binding=4,.type=98304,.parameter=true,.mutable=true,.start=0,.end=1}).is_ok);assert(v::push<usize>(ref mut edges,3).is_ok);assert(v::push<usize>(ref mut edges,2).is_ok);assert(v::push<m::Function>(ref mut functions,m::Function{.target=2,.entry=3,.local_start=2,.local_count=1,.block_start=2,.block_count=1,.parameter_start=0,.parameter_count=1,.start=0,.end=1,.public=mutation==10}).is_ok);}
    var v::Vec<m::Operation> operations=v::empty<m::Operation>();var m::Operation make=op(6,1,0);make.next=2;assert(v::push<m::Operation>(ref mut operations,make).is_ok);
    var m::Operation move=op(if(mutation==4){2}else{24},2,1);if(!body && mutation!=1){move.next=3;}assert(v::push<m::Operation>(ref mut operations,move).is_ok);
    if(!body && mutation!=1){var m::Operation drop=op(25,0,2);if(mutation==2 || mutation==3 || mutation==9){drop.next=4;}if(mutation==5){drop.type=98305;drop.owner_type=98305;}assert(v::push<m::Operation>(ref mut operations,drop).is_ok);}
    if(!body && (mutation==2 || mutation==3 || mutation==9)){assert(v::push<m::Operation>(ref mut operations,if(mutation==2){op(25,0,2)}else if(mutation==3){op(24,3,1)}else{op(6,2,0)}).is_ok);}
    var m::Block entry=state::block(1,0,1);entry.first=1;entry.last=v::len<m::Operation>(ref operations);entry.kind=if(body){5}else{1};
    if(body){entry.callee=2;entry.type=if(mutation==11){0}else{98304};entry.argument_start=1;entry.argument_count=1;entry.target=2;}
    assert(v::push<m::Block>(ref mut blocks,entry).is_ok);
    if(body){var m::Block exit=state::block(1,0,1);exit.kind=1;assert(v::push<m::Block>(ref mut blocks,exit).is_ok);var m::Block drop_body=state::block(2,0,1);drop_body.kind=1;
        if(mutation==12){var m::Operation recur=op(25,0,3);recur.function=2;recur.block=3;assert(v::push<m::Operation>(ref mut operations,recur).is_ok);drop_body.first=3;drop_body.last=3;}
        assert(v::push<m::Block>(ref mut blocks,drop_body).is_ok);}
    var v::Vec<m::Drop> drops=v::empty<m::Drop>();let m::Drop d=m::Drop{.abilities=1,.type=98304,.target=if(body){2}else{0},.nonempty=body || mutation==6};
    if(mutation!=7){assert(v::push<m::Drop>(ref mut drops,d).is_ok);}if(mutation==8){assert(v::push<m::Drop>(ref mut drops,d).is_ok);}
    let p=switch(m::assemble_resources(source,bytes::empty(),targets,signatures,functions,locals,blocks,operations,edges,0,table,drops)){case .ok(value):{value}case .err(e):{return e.code;}};
    return verify::check(ref p).code;
}
@test fn void root_obligations_reject_missing_duplicate_drop_and_donor_reuse(){
    assert(program(0,false)==0);assert(program(1,false)==1104);assert(program(2,false)==1101);assert(program(3,false)==1101);assert(program(4,false)==1001);assert(program(5,false)==1001);
    assert(program(6,false)==4);assert(program(7,false)==4);assert(program(8,false)==4);
}
@test fn void cleanup_call_and_nonowning_receiver_keep_exact_type(){
    assert(program(0,true)==0);assert(program(10,true)==1001);assert(program(11,true)==1001);assert(program(12,true)==1001);
}
