module tests::mir_literals;
import mir_model as m;
import mir_state as state;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Program fixture(u32 mutation) {
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"main abc").is_ok);
    var bytes::Buffer payload=bytes::empty();assert(bytes::append(ref mut payload,b"\xff0").is_ok);
    var v::Vec<m::Target> targets=v::empty<m::Target>();assert(v::push<m::Target>(ref mut targets,m::Target { .effects = 0, .binding=1,.module_index=0,.declaration=1,.result=1,.signature_start=0,.signature_count=0,.start=0,.end=4,.external=false }).is_ok);
    var v::Vec<m::Function> functions=v::empty<m::Function>();assert(v::push<m::Function>(ref mut functions,m::Function { .target=1,.entry=1,.local_start=0,.local_count=6,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=8,.public=false }).is_ok);
    var v::Vec<m::Local> locals=v::empty<m::Local>();var usize i=1;
    while(i<=6){assert(v::push<m::Local>(ref mut locals,m::Local { .function=1,.binding=0,.type=if(i==1 || i==6){if(mutation==7){9}else{10}}else if(i==2 || i==3){5}else if(i==4){3}else{1},.parameter=false,.mutable=false,.start=0,.end=4 }).is_ok);i+=1;}
    var v::Vec<m::Operation> operations=v::empty<m::Operation>();i=1;
    while(i<=6){
        var m::Operation o=state::operation(1,i,1,1,0,0,0,0,4);o.function=1;o.block=1;o.next=if(i==6){0}else{i+1};
        if(i==1){o.kind=14;o.type=if(mutation==7){9}else{10};o.owner_type=o.type;o.payload_end=1;if(mutation==5){o.boolean=true;}}
        else if(i==2){o.type=5;o.owner_type=5;o.payload_start=1;o.payload_end=2;}
        else if(i==3){o.kind=16;o.type=5;o.owner_type=10;o.left=1;o.right=2;if(mutation==2){o.kind=2;o.owner_type=5;o.left=2;o.right=0;}if(mutation==6){o.boolean=true;}}
        else if(i==4){o.kind=17;o.type=3;o.owner_type=10;o.left=if(mutation==3){6}else{1};o.right=if(mutation==1){2}else{3};}
        else if(i==5){o.payload_start=1;o.payload_end=2;}
        else{o.kind=2;o.type=if(mutation==7){9}else{10};o.owner_type=o.type;o.left=1;if(mutation==4){o.result=1;}}
        assert(v::push<m::Operation>(ref mut operations,o).is_ok);i+=1;
    }
    var m::Block done=state::block(1,0,8);done.kind=1;done.value=5;done.first=1;done.last=6;
    var v::Vec<m::Block> blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return switch(m::assemble(source,payload,targets,v::empty<m::Signature>(),functions,locals,blocks,operations,v::empty<usize>(),0)){case .ok(p):{p}case .err(_):{assert(false);return fixture(0);}};
}
@test
fn void literal_utf8_and_exact_frozen_owner_proof_before_c_output() {
    let m::Program good=fixture(0);assert(verify::check(ref good).code==0);
    var bytes::Buffer unit=bytes::empty();assert(c::render(ref good,ref mut unit)==0);
    var u32 mutation=1;while(mutation<=7){
        let m::Program p=fixture(mutation);assert(verify::check(ref p).code==if(mutation<=4){1501}else{1001});
        var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);
        assert(c::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;
    }
}
