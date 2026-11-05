module project_atlas_traps;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_lower as lower;
import module_unit_cfg_model as cfg;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// A sorted set of Go MIR trap categories, not source-site occurrences.
// These masks convey observations only; Complete owns source identity.
pub resource Collection{v::Vec<u32> frames;cleanup{}}
pub fn usize frame_count(ref Collection p){return v::len<u32>(ref p.frames);}
pub fn u32 at(ref Collection p,usize index){return *v::at<u32>(ref p.frames,index);}
pub fn ref u8[] category(usize index){
    assert(index<6);let names=b"assertion-failed|bounds|division-by-zero|integer-overflow|invalid-shift|signed-division-overflow";
    var usize start=0;var usize i=0;while(i<index){while(names[start]!=124){start+=1;}start+=1;i+=1;}
    var usize end=start;while(end<names.len && names[end]!=124){end+=1;}return names[start..end];
}
pub fn bool contains(u32 mask,usize index){assert(index<6);let u32 bit=if(index==0){1}else if(index==1){2}else if(index==2){4}else if(index==3){8}else if(index==4){16}else{32};return mask & bit!=0;}
fn u32 arithmetic(ref u8[] op,u32 scalar,bool unary){
    if(scalar<1 || scalar>5){return 0;}
    if(unary){return if(bytes::equal(op,b"-")){8}else{0};}
    if(bytes::equal(op,b"/") || bytes::equal(op,b"%")){return if(scalar==1 || scalar==2){36}else{4};}
    if(bytes::equal(op,b"+") || bytes::equal(op,b"-") || bytes::equal(op,b"*")){return 8;}
    return if(bytes::equal(op,b"<<")){24}else if(bytes::equal(op,b">>")){16}else{0};
}
pub fn Result<u32,u32> collect(ref ir::Program program,usize frame,usize maximum,ref mut usize meter){
    if(frame==0 || frame>ir::count(program) || maximum>6){return .err(9);}
    if(!work::charge(meter,1)){return .err(3);}
    let graph=switch(lower::lower(program,frame)){case .ok(p):{p}case .err(e):{return .err(if(e.code==1002){3}else if(e.code==1003){2}else{9});}};
    return visit(program,frame,ref graph,maximum,meter);
}
// No public entry accepts an unverified graph or a caller-supplied category set.
pub fn Result<Collection,u32> collect_all(ref ir::Program program,usize maximum,ref mut usize meter){
    if(maximum>6){return .err(9);}if(!work::charge(meter,1)){return .err(3);}
    let graphs=switch(lower::lower_all(program)){case .ok(p):{p}case .err(e):{return .err(if(e.code==1002){3}else if(e.code==1003){2}else{9});}};
    if(v::len<cfg::Graph>(ref graphs)!=ir::count(program)){return .err(9);}var frames=v::empty<u32>();var usize i=0;
    while(i<v::len<cfg::Graph>(ref graphs)){
        let mask=visit(program,i+1,v::at<cfg::Graph>(ref graphs,i),maximum,meter)?;
        if(!work::charge(meter,1)){return .err(3);}if(!v::push<u32>(ref mut frames,mask).is_ok){return .err(2);}i+=1;
    }return .ok(Collection{.frames});
}
fn Result<u32,u32> visit(ref ir::Program program,usize frame,ref cfg::Graph graph,usize maximum,ref mut usize meter){
    if(!work::charge(meter,cfg::work(graph))){return .err(3);}
    let body=ir::at(program,frame);let types=ir::cleanup_ir(program);var seen=v::empty<u8>();var usize i=0;
    while(i<=cfg::count(graph)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(2);}i+=1;}
    var pending=v::empty<usize>();discard schedule(ref mut pending,ref mut seen,cfg::entry(graph),meter)?;i=0;var u32 mask=0;
    while(i<v::len<usize>(ref pending)){
        if(!work::charge(meter,1)){return .err(3);}let n=cfg::at(graph,*v::at<usize>(ref pending,i));i+=1;
        if(n.kind==8){
            mask|=1;
            // Go's direct literal-false assertion terminates lowering here.
            let e=ir::node_at(body,n.source-1).value;if(e.kind==1 && bytes::equal(ir::node_text(body,n.source-1),b"false")){continue;}
        }
        // Go expands iteration into checked element access and checked usize
        // increment even though those operations are implicit in this CFG node.
        if(n.kind==21){mask|=10;}
        if(n.kind==1){let e=ir::node_at(body,n.source-1).value;
            if(e.kind==9 || e.kind==26){mask|=2;}
            else if(e.kind==4 && !e.minimum || e.kind==5){
                let t=clean::type_at(types,e.type);let op=ir::node_text(body,n.source-1);if(!work::charge(meter,op.len)){return .err(3);}
                mask|=arithmetic(op,if(t.kind==1){t.scalar}else{0},e.kind==4);
            }
        }else if(n.kind==2){let statement=ir::statement_at(body,n.target-1).value;
            if(statement.kind==4){let op=ir::assignment_operator(body,n.target-1);if(!work::charge(meter,op.len)){return .err(3);}
                if(op.len>1){let target=ir::node_at(body,ir::node(body,statement.target)-1).value;let t=clean::type_at(types,target.type);mask|=arithmetic(op[..op.len-1],if(t.kind==1){t.scalar}else{0},false);}
            }
        }
        if(n.first!=0){discard schedule(ref mut pending,ref mut seen,n.first,meter)?;}if(n.second!=0){discard schedule(ref mut pending,ref mut seen,n.second,meter)?;}
    }
    var usize count=0;i=0;while(i<6){if(!work::charge(meter,1)){return .err(3);}if(contains(mask,i)){count+=1;}i+=1;}
    if(count>maximum){return .err(3);}return .ok(mask);
}
fn Result<bool,u32> schedule(ref mut v::Vec<usize> pending,ref mut v::Vec<u8> seen,usize id,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}if(id==0 || id>=v::len<u8>(seen)){return .err(9);}if(*v::at<u8>(seen,id)!=0){return .ok(false);}
    if(!v::push<usize>(pending,id).is_ok){return .err(2);}{let slot=v::at_mut<u8>(seen,id);*slot=1;}return .ok(true);
}
