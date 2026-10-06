module tests::mir_mutable;
import mir_model as m;
import mir_state as state;
import mir_verify as verify;
import array_types as arrays;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn m::Program fixture(u32 mutation) {
    let u32 array=arrays::make(3,1);
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"main a b").is_ok);
    var bytes::Buffer payload=bytes::empty();assert(bytes::append(ref mut payload,b"07").is_ok);
    var v::Vec<m::Target> targets=v::empty<m::Target>();assert(v::push<m::Target>(ref mut targets,m::Target { .binding=1,.module_index=0,.declaration=1,.result=if(mutation==6){11}else{1},.signature_start=0,.signature_count=0,.start=0,.end=4,.external=false }).is_ok);
    var v::Vec<m::Function> functions=v::empty<m::Function>();assert(v::push<m::Function>(ref mut functions,m::Function { .target=1,.entry=1,.local_start=0,.local_count=8,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=8,.public=false }).is_ok);
    var v::Vec<m::Local> locals=v::empty<m::Local>();var usize i=1;
    while(i<=8){assert(v::push<m::Local>(ref mut locals,m::Local { .function=1,.binding=if(i==1){1}else{0},.type=if(i==1){array}else if(i==3){11}else if(i==4 || i==5){5}else if(i==8){1}else{3},.parameter=false,.mutable=i==1 && mutation!=1,.start=0,.end=4 }).is_ok);i+=1;}
    var v::Vec<usize> edges=v::empty<usize>();assert(v::push<usize>(ref mut edges,2).is_ok);assert(v::push<usize>(ref mut edges,if(mutation==4){8}else{6}).is_ok);
    var v::Vec<m::Operation> operations=v::empty<m::Operation>();i=1;
    while(i<=9){
        var m::Operation o=state::operation(1,2,3,3,0,0,0,0,4);o.function=1;o.block=1;o.next=if(i==9){0}else{i+1};o.payload_end=1;
        if(i==2){o.kind=9;o.result=1;o.type=array;o.owner_type=array;o.payload_end=0;o.argument_count=1;}
        else if(i==3){o.kind=18;o.result=3;o.type=11;o.owner_type=array;o.left=1;o.payload_end=0;}
        else if(i==4){o.result=4;o.type=5;o.owner_type=5;}
        else if(i==5){o.kind=16;o.result=5;o.type=5;o.owner_type=11;o.left=3;o.right=4;o.payload_end=0;}
        else if(i==6){o.result=6;o.payload_start=1;o.payload_end=2;if(mutation==8){o.kind=2;o.result=3;o.type=11;o.owner_type=11;o.left=3;o.payload_start=0;o.payload_end=0;}}
        else if(i==7){o.kind=20;o.result=0;o.type=3;o.owner_type=if(mutation==7){10}else{11};o.left=if(mutation==2){2}else{3};o.right=if(mutation==3){4}else{5};o.payload_end=0;o.argument_start=1;o.argument_count=1;}
        else if(i==8){o.kind=17;o.result=7;o.type=3;o.owner_type=11;o.left=3;o.right=5;o.payload_end=0;}
        else if(i==9){o.result=8;o.type=1;o.owner_type=1;}
        if(mutation==5){
            if(i==5){o.kind=1;o.result=6;o.type=3;o.owner_type=3;o.left=0;o.right=0;o.payload_start=1;o.payload_end=2;}
            else if(i==6){o.kind=20;o.result=0;o.type=3;o.owner_type=11;o.left=3;o.right=5;o.payload_start=0;o.payload_end=0;o.argument_start=1;o.argument_count=1;}
            else if(i==7){o.kind=16;o.result=5;o.type=5;o.owner_type=11;o.left=3;o.right=4;o.argument_start=0;o.argument_count=0;}
        }
        assert(v::push<m::Operation>(ref mut operations,o).is_ok);i+=1;
    }
    var m::Block done=state::block(1,0,8);done.kind=1;done.value=if(mutation==6){3}else{8};done.first=1;done.last=9;
    var v::Vec<m::Block> blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return switch(m::assemble(source,payload,targets,v::empty<m::Signature>(),functions,locals,blocks,operations,edges,0)){case .ok(p):{p}case .err(_):{assert(false);return fixture(0);}};
}
@test
fn void mutable_store_shape_bounds_origin_and_output_preservation() {
    let good=fixture(0);assert(verify::check(ref good).code==0);var bytes::Buffer unit=bytes::empty();assert(c::render(ref good,ref mut unit)==0);
    var u32 mutation=1;while(mutation<=8){let p=fixture(mutation);let u32 expected=if(mutation==3 || mutation==8){1501}else if(mutation==5){1002}else if(mutation==6){2102}else{1001};assert(verify::check(ref p).code==expected);var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref p,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));mutation+=1;}
}
