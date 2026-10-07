module resource_flow;
import result_flow_graph as graph;
import name_resolution as names;
import type_state as state;
import std::vec as v;

// Reachable inputs are Live=1, Dead=2 or their conservative union=3. All
// roots start Dead; parameter/initializer production installs a live value.
// Reachable branch joins and each normal/break/continue loop edge require equal
// states. A loop baseline is after its condition; exited locals are already dead.
pub struct Cell { u8 states; usize move; }
fn void put(ref mut v::Vec<Cell> values,usize index,Cell value){let ref mut Cell slot=v::at_mut<Cell>(values,index);*slot=value;}
fn void byte(ref mut v::Vec<u8> values,usize index,u8 value){let ref mut u8 slot=v::at_mut<u8>(values,index);*slot=value;}
fn usize earlier(ref graph::Graph g,usize a,usize b){if(a==0){return b;}if(b==0){return a;}let x=graph::at(g,a);let y=graph::at(g,b);return if(x.start<y.start || x.start==y.start && x.end<=y.end){a}else{b};}
fn usize loop_error(ref graph::Graph g,usize a,usize b,usize old_move,usize new_move){
    if(a!=0 && old_move==new_move){let x=graph::at(g,a);let y=graph::at(g,b);
        // One inner mismatch can contaminate an enclosing edge's fixed point.
        // Preserve the smallest responsible loop for that same transfer site.
        if(x.start<=y.start && y.end<=x.end){return b;}if(y.start<=x.start && x.end<=y.end){return a;}}
    return earlier(g,a,b);
}
fn state::Problem analyze(ref graph::Graph g){
    let usize count=graph::count(g);let usize width=graph::width(g);if(width==0 || count==0){return state::ok();}
    if(count>262144/width){return state::fail(10,0,0,0,0);}
    var v::Vec<Cell> input=v::empty<Cell>();var v::Vec<Cell> output=v::empty<Cell>();var v::Vec<usize> queue=v::empty<usize>();
    var v::Vec<u8> seen=v::empty<u8>();var v::Vec<u8> queued=v::empty<u8>();var usize i=0;
    if(!v::reserve<Cell>(ref mut input,count*width).is_ok){return state::fail(12,0,0,0,0);}
    while(i<count*width){if(!v::push<Cell>(ref mut input,Cell{.states=2,.move=0}).is_ok){return state::fail(12,0,0,0,0);}i+=1;}
    i=0;while(i<width){if(!v::push<Cell>(ref mut output,Cell{.states=2,.move=0}).is_ok){return state::fail(12,0,0,0,0);}i+=1;}
    i=0;while(i<count){if(!v::push<u8>(ref mut seen,0).is_ok || !v::push<u8>(ref mut queued,0).is_ok){return state::fail(12,0,0,0,0);}i+=1;}
    if(!v::push<usize>(ref mut queue,1).is_ok){return state::fail(12,0,0,0,0);}byte(ref mut seen,0,1);byte(ref mut queued,0,1);var usize work=0;
    while(v::len<usize>(ref queue)!=0){
        let usize id=switch(v::pop<usize>(ref mut queue)){case .some(n):{n}case .none:{0}};let n=graph::at(g,id);byte(ref mut queued,id-1,0);
        i=0;while(i<width){let old=*v::at<Cell>(ref input,(id-1)*width+i);put(ref mut output,i,old);i+=1;}
        if(n.column!=0 && (n.kind==1 || n.kind==2 || n.kind==4)){put(ref mut output,n.column-1,Cell{.states=if(n.kind==2){1}else{2},.move=if(n.kind==1){id}else{0}});}
        var usize side=0;while(side<2){let usize successor=if(side==0){n.next}else{n.alternative};side+=1;if(successor==0){continue;}
            let bool first=*v::at<u8>(ref seen,successor-1)==0;var bool changed=first;byte(ref mut seen,successor-1,1);
            i=0;while(i<width){if(work>=16777216){return state::fail(10,n.start,n.end,0,0);}work+=1;
                let usize index=(successor-1)*width+i;let before=*v::at<Cell>(ref input,index);let incoming=*v::at<Cell>(ref output,i);
                let after=Cell{.states=if(first){incoming.states}else{before.states|incoming.states},.move=if(first){incoming.move}else{earlier(g,before.move,incoming.move)}};
                if(after.states!=before.states || after.move!=before.move){put(ref mut input,index,after);changed=true;}i+=1;
            }
            if(changed && *v::at<u8>(ref queued,successor-1)==0){if(!v::push<usize>(ref mut queue,successor).is_ok){return state::fail(12,n.start,n.end,0,0);}byte(ref mut queued,successor-1,1);}
        }
    }
    var usize error=0;var usize move=0;var usize header=0;var u32 code=45;i=1;
    while(i<=count){let n=graph::at(g,i);
        if(*v::at<u8>(ref seen,i-1)!=0 && n.column!=0 && (n.kind==1 || n.kind==7)){
            let cell=*v::at<Cell>(ref input,(i-1)*width+n.column-1);
            if(cell.states!=1){let chosen=earlier(g,error,i);if(chosen==i){error=i;move=cell.move;header=0;code=45;}}
        }
        if(*v::at<u8>(ref seen,i-1)!=0 && (n.kind==8 || n.kind==9)){var usize col=0;while(col<width){let cell=*v::at<Cell>(ref input,(i-1)*width+col);
            let before=if(n.kind==9){*v::at<Cell>(ref input,(n.column-1)*width+col)}else{cell};
            if(cell.states==3 || before.states==3 || cell.states!=before.states){let usize transfer=earlier(g,cell.move,before.move);let chosen=if(n.kind==9 && code==47){loop_error(g,error,i,move,transfer)}else{earlier(g,error,i)};if(chosen==i){error=i;move=transfer;header=if(n.kind==9){n.column}else{0};code=if(n.kind==9){47}else{46};}}col+=1;}}
        i+=1;
    }
    if(error==0){return state::ok();}let use=graph::at(g,error);var problem=state::fail(code,use.start,use.end,0,0);
    if(move!=0){let at=graph::at(g,move);problem.move_start=at.start;problem.move_end=at.end;}
    if(header!=0){let at=graph::at(g,header);problem.loop_start=at.start;problem.loop_end=at.end;}return problem;
}
pub fn state::Problem check(ref names::Resolved resolved,ref v::Vec<state::Slot> types){
    var usize i=1;var state::Problem first=state::ok();
    while(i<=names::symbol_count(resolved)){let s=names::symbol_at(resolved,i);if(s.kind==1 || s.kind==9){
        let g=switch(graph::build_resources(resolved,types,i)){case .ok(g):{g}case .err(p):{return p;}};
        let p=analyze(ref g);if(p.code!=0 && (first.code==0 || p.start<first.start || p.start==first.start && p.end<first.end)){first=p;}
    }i+=1;}return first;
}
