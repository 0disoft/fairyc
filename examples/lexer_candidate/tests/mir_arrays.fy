module tests::mir_arrays;
import mir_model as m;
import mir_state as state;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Program fixture(u32 mutation) {
    var bytes::Buffer source=bytes::empty(); assert(bytes::append(ref mut source,b"main abc").is_ok);
    var bytes::Buffer payload=bytes::empty(); assert(bytes::append(ref mut payload,b"10").is_ok);
    var v::Vec<m::Target> targets=v::empty<m::Target>();
    assert(v::push<m::Target>(ref mut targets,m::Target { .effects = 0, .binding=1,.module_index=0,.declaration=1,.result=1,.signature_start=0,.signature_count=0,.start=0,.end=4,.external=false }).is_ok);
    var v::Vec<m::Function> functions=v::empty<m::Function>();
    assert(v::push<m::Function>(ref mut functions,m::Function { .target=1,.entry=1,.local_start=0,.local_count=6,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=8,.public=false }).is_ok);
    var v::Vec<m::Local> locals=v::empty<m::Local>(); var usize i=1;
    while(i<=6) { assert(v::push<m::Local>(ref mut locals,m::Local { .function=1,.binding=0,.type=if(i==3){1033}else if(i==2 || i==4){5}else{1},.parameter=false,.mutable=false,.start=0,.end=4 }).is_ok); i+=1; }
    var v::Vec<usize> edges=v::empty<usize>(); assert(v::push<usize>(ref mut edges,if(mutation==5){2}else{1}).is_ok);
    var v::Vec<m::Operation> operations=v::empty<m::Operation>(); i=1;
    while(i<=6) {
        var m::Operation o=state::operation(1,i,1,1,0,0,0,0,4); o.function=1;o.block=1;o.next=if(i==6){0}else{i+1};
        if(i==1){o.payload_end=1;}
        else if(i==2){o.type=5;o.owner_type=5;o.payload_start=1;o.payload_end=2;}
        else if(i==3){o.kind=9;o.type=1033;o.owner_type=1033;o.argument_count=if(mutation==4){0}else{1};}
        else if(i==4){o.kind=12;o.type=5;o.owner_type=if(mutation==2){1041}else if(mutation==9){1032}else{1033};o.left=2;
            if(mutation==6){o.result=2;} if(mutation==7){o.kind=2;o.owner_type=5;}
        }
        else if(i==5){o.kind=10;o.owner_type=1033;o.left=if(mutation==3){1}else{3};o.right=if(mutation==1){2}else{4};}
        else{o.kind=2;o.left=5;if(mutation==8){o.result=4;o.type=5;o.owner_type=5;o.left=2;}}
        assert(v::push<m::Operation>(ref mut operations,o).is_ok); i+=1;
    }
    var m::Block done=state::block(1,0,8);done.kind=1;done.value=5;done.first=1;done.last=6;
    var v::Vec<m::Block> blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return switch(m::assemble(source,payload,targets,v::empty<m::Signature>(),functions,locals,blocks,operations,edges,0)){case .ok(p):{p}case .err(_):{assert(false);return fixture(0);}};
}
@test
fn void checked_index_provenance_and_failed_render_preserve_output() {
    let m::Program good=fixture(0);assert(verify::check(ref good).code==0);
    var bytes::Buffer unit=bytes::empty();assert(c::render(ref good,ref mut unit)==0);
    var u32 mutation=1;
    while(mutation<=9) {
        let m::Program p=fixture(mutation);
        assert(verify::check(ref p).code==if(mutation==1 || mutation==2 || mutation==7 || mutation==8){1501}else{1001});
        var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);
        assert(c::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;
    }
}
