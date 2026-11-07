module mir_closure_lower;
import hir_model as h;
import mir_model as m;
import mir_state as state;
import mir_lower as lowering;
import type_state as ids;
import std::bytes as bytes;
import std::vec as v;

// No deferred source body or environment pointer crosses this boundary. The
// ordinary lifted function and all capture operands remain explicit MIR facts.
pub fn usize expression(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut bytes::Buffer payload,ref mut state::Context c,h::Expression e,usize depth){
    if(h::expression_code(e.kind)==31){
        if(e.binding==0 || e.binding>1152 || c.targets[e.binding]==0 || e.edge_count>4){return state::fail(c,4,e.start,e.end);}
        var v::Vec<usize> captures=v::empty<usize>();var usize i=0;
        while(i<e.edge_count){let value=lowering::expression(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,e.edge_start+i),depth+1);if(c.current==0 || c.problem.code!=0){return 0;}if(!v::push<usize>(ref mut captures,value).is_ok){return state::fail(c,3,e.start,e.end);}i+=1;}
        let first=v::len<usize>(edges);i=0;while(i<v::len<usize>(ref captures)){if(!state::edge(edges,*v::at<usize>(ref captures,i),c)){return 0;}i+=1;}
        let result=state::local(locals,c,13,0,false,false,e.start,e.end);
        var op=state::operation(40,result,13,13,ids::small_id(c.targets[e.binding]),0,0,e.start,e.end);op.argument_start=if(e.edge_count==0){0}else{first};op.argument_count=e.edge_count;
        return if(state::emit(operations,blocks,c,op)!=0){result}else{0};
    }
    if(h::expression_code(e.kind)!=32){return state::fail(c,4,e.start,e.end);}
    let source=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}
    // Freeze the callable before evaluating a possibly effectful argument.
    let callable=state::local(locals,c,13,0,false,false,e.start,e.end);if(state::emit(operations,blocks,c,state::operation(2,callable,13,13,0,source,0,e.start,e.end))==0){return 0;}
    let argument=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}
    let result=state::local(locals,c,1,0,false,false,e.start,e.end);let next=state::new_block(blocks,c,e.start,e.end);
    var term=state::block(c.function,e.start,e.end);term.kind=8;term.type=1;term.value=callable;term.left=argument;term.destination=result;term.target=next;
    if(!state::terminate(blocks,c,term)){return 0;}c.current=next;return result;
}

// Select receipts from emitted construction operations, after MIR lowering has
// removed dead continuations. A dead source literal cannot leave orphan facts.
pub fn Result<v::Vec<m::ClosureTarget>,m::Problem> targets(ref v::Vec<m::Operation> operations){
    var v::Vec<m::ClosureTarget> result=v::empty<m::ClosureTarget>();var usize i=0;
    while(i<v::len<m::Operation>(operations)){let op=*v::at<m::Operation>(operations,i);if(op.kind==40){
        let target=op.operator as usize;
        if(target==0 || target>512 || op.argument_count>4){return .err(m::Problem{.code=4,.start=op.start,.end=op.end});}
        var bool found=false;var usize j=0;
        while(j<v::len<m::ClosureTarget>(ref result)){let old=*v::at<m::ClosureTarget>(ref result,j);if(old.target==target){if(old.captures!=op.argument_count){return .err(m::Problem{.code=4,.start=op.start,.end=op.end});}found=true;}j+=1;}
        if(!found && !v::push<m::ClosureTarget>(ref mut result,m::ClosureTarget{.target,.captures=op.argument_count}).is_ok){return .err(m::Problem{.code=3,.start=op.start,.end=op.end});}
    }i+=1;}return .ok(result);
}
