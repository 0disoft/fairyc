module module_state_graph_transport_plan;
import module_state_graph_program as program;
import module_state_graph_model as m;
import module_state_layout as state;
import project_cache_encoding as budget;
import std::vec as v;

pub struct Cell{u32[8] key;usize codec;}
// Detached framing facts only. No graph, layout, or caller-owned views escape.
pub resource Plan{usize operations;usize literal;v::Vec<Cell> cells;cleanup{}}
pub fn usize operation_bound(ref Plan plan){return plan.operations;}
pub fn usize max_literal(ref Plan plan){return plan.literal;}
pub fn usize cell_count(ref Plan plan){return v::len<Cell>(ref plan.cells);}
pub fn Cell cell_at(ref Plan plan,usize index){return *v::at<Cell>(ref plan.cells,index);}

fn ref m::Graph graph(ref program::Verified proof,usize body){if(body==0){return program::root(proof);}return program::helper(proof,body-1);}
// Bound all storage before allocating it; identities never size allocations.
fn u32 preflight(ref program::Verified proof,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}
    let helpers=program::helper_count(proof);if(helpers>32){return 1002;}
    let layouts=program::layouts(proof);let selected=m::header(program::root(proof)).layout;
    if(selected>=state::count(layouts)){return 1001;}let layout=state::at(layouts,selected);
    if(layout.field_count>1024){return 1002;}
    if(!m::range(layout.field_start,layout.field_count,state::field_count(layouts))){return 1001;}
    var usize total=0;var usize body=0;
    while(body<=helpers){
        if(!budget::charge(work,1)){return 1002;}let g=graph(proof,body);let count=v::len<m::Operation>(m::operations(g));
        if(count>1024-total || v::len<m::Block>(m::blocks(g))>64){return 1002;}total+=count;body+=1;
    }return 0;
}
// Declaration codec IDs differ from the native graph ABI for all byte types.
fn usize native_codec(u32 codec){
    if(codec>=1 && codec<=3){return codec as usize;}
    if(codec==4){return 6;}if(codec==5){return 7;}if(codec==6){return 4;}if(codec==7){return 5;}return 0;
}
// Unsigned big-endian words have exactly the physical 32-byte key order.
fn bool before(u32[8] a,u32[8] b){var usize i=0;while(i<8){if(a[i]!=b[i]){return a[i]<b[i];}i+=1;}return false;}
fn u32 cells(ref program::Verified proof,ref mut v::Vec<Cell> output,ref mut usize work){
    let layouts=program::layouts(proof);let layout=state::at(layouts,m::header(program::root(proof)).layout);var usize i=0;
    while(i<layout.field_count){
        if(!budget::charge(work,1)){return 1002;}let field=state::field_at(layouts,layout.field_start+i);i+=1;
        if(field.kind!=1){continue;}let codec=native_codec(field.value);if(codec==0){return 1001;}
        let cell=Cell{.key=field.prefix,.codec};var usize at=0;
        while(at<v::len<Cell>(output)){
            if(!budget::charge(work,8)){return 1002;}let current=*v::at<Cell>(output,at);
            // Match Go's physical-key map even if two prefixes coincide.
            if(state::equal(current.key,cell.key)){break;}if(before(cell.key,current.key)){break;}at+=1;
        }
        if(at<v::len<Cell>(output) && state::equal(v::at<Cell>(output,at).key,cell.key)){
            if(!budget::charge(work,1)){return 1002;}{let slot=v::at_mut<Cell>(output,at);*slot=cell;}continue;
        }
        if(!budget::charge(work,v::len<Cell>(output)-at+1)){return 1002;}
        if(!v::push<Cell>(output,cell).is_ok){return 1003;}var usize j=v::len<Cell>(output)-1;
        while(j>at){let previous=*v::at<Cell>(output,j-1);{let slot=v::at_mut<Cell>(output,j);*slot=previous;}j-=1;}
        {let slot=v::at_mut<Cell>(output,at);*slot=cell;}
    }return 0;
}
pub fn Result<Plan,u32> build(ref program::Verified proof,ref mut usize work){
    let code=preflight(proof,work);if(code!=0){return .err(code);}
    // Iterative memoized DFS: each body is evaluated once, but its saturated
    // bound is added at EVERY call site, including both static branch arms.
    var usize[33] bounds=[0;33];var usize[33] next=[0;33];var usize[33] stack=[0;33];var u32[33] status=[0;33];
    let count=program::helper_count(proof)+1;var usize i=0;
    while(i<count){if(!budget::charge(work,1)){return .err(1002);}let blocks=v::len<m::Block>(m::blocks(graph(proof,i)));bounds[i]=if(blocks==0){1}else{blocks};i+=1;}
    var usize depth=1;stack[0]=0;status[0]=1;var usize literal=0;
    while(depth!=0){
        if(!budget::charge(work,1)){return .err(1002);}let body=stack[depth-1];let g=graph(proof,body);
        if(next[body]==v::len<m::Operation>(m::operations(g))){status[body]=2;depth-=1;continue;}
        let op=*v::at<m::Operation>(m::operations(g),next[body]);var usize expanded=0;
        if(op.kind==11){
            if(op.callee==0 || op.callee>=count || status[op.callee]==1){return .err(1001);}
            if(status[op.callee]==0){if(depth>=33){return .err(1002);}status[op.callee]=1;stack[depth]=op.callee;depth+=1;continue;}
            expanded=bounds[op.callee];
        }
        // SSA values and terminator trap bytes are not encoded literals.
        if(op.key.input==0 && op.key.codec!=0){let size=op.key.end-op.key.start;if(size>literal){literal=size;}}
        if(op.value.input==0 && op.value.codec!=0){let size=op.value.end-op.value.start;if(size>literal){literal=size;}}
        let n=bounds[body]+1+expanded;bounds[body]=if(n>67108864){67108864}else{n};next[body]+=1;
    }
    var v::Vec<Cell> known=v::empty<Cell>();let collected=cells(proof,ref mut known,work);if(collected!=0){return .err(collected);}
    return .ok(Plan{.operations=bounds[0],.literal,.cells=known});
}
