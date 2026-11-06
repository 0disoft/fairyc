module project_atlas_reachability;
import module_unit_body_ir_model as ir;
import module_unit_cfg_lower as lower;
import module_unit_cfg_model as cfg;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Only a locally verified CFG grants completed-expression authority. In
// particular, checking a call's arguments does not prove the call is reached.
pub resource Observations{v::Vec<usize> offsets;v::Vec<u8> completed;cleanup{}}
pub fn usize frame_count(ref Observations p){return v::len<usize>(ref p.offsets)-1;}
pub fn Result<bool,u32> completed(ref Observations p,usize frame,usize expression,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}
    if(frame==0 || frame>frame_count(p) || expression==0){return .err(9);}
    let start=*v::at<usize>(ref p.offsets,frame-1);let end=*v::at<usize>(ref p.offsets,frame);
    if(expression>end-start){return .err(9);}return .ok(*v::at<u8>(ref p.completed,start+expression-1)!=0);
}
fn Result<bool,u32> schedule(ref mut v::Vec<usize> pending,ref mut v::Vec<u8> seen,usize id,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}if(id==0 || id>=v::len<u8>(seen)){return .err(9);}
    if(*v::at<u8>(seen,id)!=0){return .ok(false);}if(!v::push<usize>(pending,id).is_ok){return .err(2);}
    {let slot=v::at_mut<u8>(seen,id);*slot=1;}return .ok(true);
}
fn Result<bool,u32> visit(ref ir::Body body,ref cfg::Graph graph,usize start,ref mut v::Vec<u8> completed,ref mut usize meter){
    if(!work::charge(meter,cfg::work(graph))){return .err(3);}var seen=v::empty<u8>();var usize i=0;
    while(i<=cfg::count(graph)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(2);}i+=1;}
    var pending=v::empty<usize>();discard schedule(ref mut pending,ref mut seen,cfg::entry(graph),meter)?;i=0;
    while(i<v::len<usize>(ref pending)){
        if(!work::charge(meter,1)){return .err(3);}let n=cfg::at(graph,*v::at<usize>(ref pending,i));i+=1;
        if(n.kind==8){let e=ir::node_at(body,n.source-1).value;if(e.kind==1 && bytes::equal(ir::node_text(body,n.source-1),b"false")){continue;}}
        // A deferred invocation executes at its exit node, not at capture time.
        if(n.kind==1 || n.kind==14){if(n.source==0 || n.source>ir::node_count(body)){return .err(9);}let slot=v::at_mut<u8>(completed,start+n.source-1);*slot=1;}
        if(n.first!=0){discard schedule(ref mut pending,ref mut seen,n.first,meter)?;}if(n.second!=0){discard schedule(ref mut pending,ref mut seen,n.second,meter)?;}
    }return .ok(true);
}
pub fn Result<Observations,u32> collect_all(ref ir::Program program,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}
    let graphs=switch(lower::lower_all(program)){case .ok(p):{p}case .err(e):{return .err(if(e.code==1002){3}else if(e.code==1003){2}else{9});}};
    if(v::len<cfg::Graph>(ref graphs)!=ir::count(program)){return .err(9);}var offsets=v::empty<usize>();var completed=v::empty<u8>();
    if(!v::push<usize>(ref mut offsets,0).is_ok){return .err(2);}var usize i=0;
    while(i<v::len<cfg::Graph>(ref graphs)){
        let body=ir::at(program,i+1);let start=v::len<u8>(ref completed);var usize j=0;
        while(j<ir::node_count(body)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut completed,0).is_ok){return .err(2);}j+=1;}
        discard visit(body,v::at<cfg::Graph>(ref graphs,i),start,ref mut completed,meter)?;
        if(!work::charge(meter,1)){return .err(3);}if(!v::push<usize>(ref mut offsets,v::len<u8>(ref completed)).is_ok){return .err(2);}i+=1;
    }return .ok(Observations{.offsets,.completed});
}
