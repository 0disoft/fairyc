module mir_verify_resources;
import mir_model as m;
import hir_shapes as shapes;
import std::bytes as bytes;
import std::vec as v;

// Exact owned-root state: 0 Dead, 1 Live, 2 compiler-owned dropping receiver.
// All arena temporaries are included; source validation cannot hide a leaked
// constructor or a donor reused by a forged MIR. No runtime drop flags are added.
fn u32 fail(u32 code){return code;}
fn bool owned(ref m::Program p,usize id){return id!=0 && shapes::needs_drop(m::local_at(p,id).type);}
fn bool read(ref m::Program p,ref bytes::Buffer state,usize base,usize id){return !owned(p,id) || bytes::view(state)[id-base-1]!=0;}
fn bool consume(ref m::Program p,ref mut bytes::Buffer state,usize base,usize id){
    if(!owned(p,id)){return true;}
    let ref mut u8[] data=bytes::view_mut(state);let usize at=id-base-1;if(data[at]!=1){return false;}data[at]=0;return true;
}
fn bool create(ref m::Program p,ref mut bytes::Buffer state,usize base,usize id){
    if(!owned(p,id)){return true;}
    let ref mut u8[] data=bytes::view_mut(state);let usize at=id-base-1;if(data[at]!=0){return false;}data[at]=1;return true;
}
fn u32 transfer(ref m::Program p,m::Block b,usize base,ref mut bytes::Buffer state){
    var usize op=b.first;
    while(op!=0){let o=m::operation_at(p,op);
        if(!read(p,state,base,o.left) || !read(p,state,base,o.right)){return fail(1101);}
        var usize j=0;while(j<o.argument_count){if(!read(p,state,base,m::edge_at(p,o.argument_start+j))){return fail(1101);}j+=1;}
        if(o.kind==6 || o.kind==21){var usize j=0;while(j<o.argument_count){if(!consume(p,state,base,m::edge_at(p,o.argument_start+j))){return fail(1101);}j+=1;}}
        if(o.kind==23 && shapes::needs_drop(o.owner_type)){if(!consume(p,state,base,o.left)){return fail(1101);}}
        if(o.kind==24){if(!consume(p,state,base,o.left) || !create(p,state,base,o.result)){return fail(1101);}}
        else if(o.kind==25 || o.kind==26){if(!consume(p,state,base,o.left)){return fail(1101);}}
        else if(owned(p,o.result) && o.kind!=8){if(!create(p,state,base,o.result)){return fail(1101);}}
        op=o.next;
    }
    if(!read(p,state,base,b.value) || !read(p,state,base,b.left) || !read(p,state,base,b.right)){return fail(1101);}
    if(b.kind==5){var usize j=0;while(j<b.argument_count){if(!consume(p,state,base,m::edge_at(p,b.argument_start+j))){return fail(1101);}j+=1;}
        if(!create(p,state,base,b.destination)){return fail(1101);}
    }
    if(b.kind==1){
        if(!consume(p,state,base,b.value)){return fail(1101);}
        var usize i=0;while(i<bytes::len(state)){if(bytes::view(state)[i]==1){return fail(1104);}i+=1;}
    }
    // Trap/unreachable have no successor and discharge no obligations.
    return 0;
}
fn bool enqueue(ref mut v::Vec<usize> queue,ref mut bytes::Buffer pending,usize id){
    if(bytes::view(pending)[id]!=0){return true;}
    if(!v::push<usize>(queue,id).is_ok){return false;}let ref mut u8[] data=bytes::view_mut(pending);data[id]=1;return true;
}
pub fn m::Problem function(ref m::Program p,usize fid){
    let f=m::function_at(p,fid);var bool present=false;var usize i=f.local_start+1;
    while(i<=f.local_start+f.local_count){present=present || owned(p,i);i+=1;}if(!present){return m::Problem{.code=0,.start=0,.end=0};}
    if(f.local_count!=0 && f.block_count>8388608/f.local_count){return m::Problem{.code=2,.start=f.start,.end=f.end};}
    var bytes::Buffer states=bytes::empty();var bytes::Buffer scratch=bytes::empty();var bytes::Buffer seen=bytes::empty();var bytes::Buffer pending=bytes::empty();var v::Vec<usize> queue=v::empty<usize>();
    if(!bytes::resize_zeroed(ref mut states,f.local_count*f.block_count).is_ok || !bytes::resize_zeroed(ref mut scratch,f.local_count).is_ok || !bytes::resize_zeroed(ref mut seen,m::block_count(p)+1).is_ok || !bytes::resize_zeroed(ref mut pending,m::block_count(p)+1).is_ok){return m::Problem{.code=3,.start=f.start,.end=f.end};}
    let usize entry=(f.entry-f.block_start-1)*f.local_count;let usize receiver=m::drop_receiver(p,fid);i=0;
    while(i<f.parameter_count){let usize id=m::edge_at(p,f.parameter_start+i);if(owned(p,id)){let ref mut u8[] data=bytes::view_mut(ref mut states);data[entry+id-f.local_start-1]=if(id==receiver){2}else{1};}i+=1;}
    {let ref mut u8[] data=bytes::view_mut(ref mut seen);data[f.entry]=1;}
    if(!enqueue(ref mut queue,ref mut pending,f.entry)){return m::Problem{.code=3,.start=f.start,.end=f.end};}
    var usize work=0;
    while(v::len<usize>(ref queue)!=0){
        let usize id=switch(v::pop<usize>(ref mut queue)){case .some(value):{value}case .none:{0}};
        {let ref mut u8[] data=bytes::view_mut(ref mut pending);data[id]=0;}
        let b=m::block_at(p,id);let usize offset=(id-f.block_start-1)*f.local_count;
        if(work>67108864 || f.local_count>67108864-work){return m::Problem{.code=2,.start=b.start,.end=b.end};}work+=f.local_count;
        {let ref mut u8[] data=bytes::view_mut(ref mut scratch);i=0;while(i<f.local_count){data[i]=bytes::view(ref states)[offset+i];i+=1;}}
        let u32 code=transfer(p,b,f.local_start,ref mut scratch);if(code!=0){return m::Problem{.code,.start=b.start,.end=b.end};}
        var usize edge=0;while(edge<2){let usize next=if(edge==0){b.target}else{b.alternative};
            if(next!=0){let bool visited=bytes::view(ref seen)[next]!=0;let usize into=(next-f.block_start-1)*f.local_count;
                {let ref mut u8[] data=bytes::view_mut(ref mut states);i=0;while(i<f.local_count){let u8 value=bytes::view(ref scratch)[i];if(visited && data[into+i]!=value){return m::Problem{.code=1104,.start=b.start,.end=b.end};}data[into+i]=value;i+=1;}}
                if(!visited){{let ref mut u8[] data=bytes::view_mut(ref mut seen);data[next]=1;}if(!enqueue(ref mut queue,ref mut pending,next)){return m::Problem{.code=3,.start=b.start,.end=b.end};}}
            }edge+=1;
        }
    }
    return m::Problem{.code=0,.start=0,.end=0};
}
