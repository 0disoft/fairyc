module result_flow;
import result_flow_graph as graph;
import name_resolution as names;
import type_state as state;
import std::vec as v;

fn void put(ref mut v::Vec<usize> values,usize at,usize value){let ref mut usize slot=v::at_mut<usize>(values,at);*slot=value;}
fn void byte(ref mut v::Vec<u8> values,usize at,u8 value){let ref mut u8 slot=v::at_mut<u8>(values,at);*slot=value;}
fn usize earlier(ref graph::Graph g,usize a,usize b){if(a==0){return b;}if(b==0){return a;}let x=graph::at(g,a);let y=graph::at(g,b);return if(x.start<y.start || x.start==y.start && x.end<=y.end){a}else{b};}
pub fn state::Problem analyze(ref graph::Graph g){
    // A dummy column preserves reachability for bare unhandled Result calls.
    let usize width=if(graph::width(g)==0){1}else{graph::width(g)};let usize count=graph::count(g);
    if(count>262144/width){return state::fail(10,0,0,0,0);}
    var v::Vec<usize> input=v::empty<usize>();var v::Vec<usize> output=v::empty<usize>();var v::Vec<usize> queue=v::empty<usize>();
    var v::Vec<u8> seen=v::empty<u8>();var v::Vec<u8> queued=v::empty<u8>();var usize i=0;
    if(!v::reserve<usize>(ref mut input,count*width).is_ok){return state::fail(12,0,0,0,0);}
    while(i<count*width){if(!v::push<usize>(ref mut input,0).is_ok){return state::fail(12,0,0,0,0);}i+=1;}
    i=0;while(i<width){if(!v::push<usize>(ref mut output,0).is_ok){return state::fail(12,0,0,0,0);}i+=1;}
    i=0;while(i<count){if(!v::push<u8>(ref mut seen,0).is_ok || !v::push<u8>(ref mut queued,0).is_ok){return state::fail(12,0,0,0,0);}i+=1;}
    if(!v::push<usize>(ref mut queue,1).is_ok){return state::fail(12,0,0,0,0);}byte(ref mut seen,0,1);byte(ref mut queued,0,1);var usize work=0;
    while(v::len<usize>(ref queue)!=0){
        let usize id=switch(v::pop<usize>(ref mut queue)){case .some(n):{n}case .none:{0}};let n=graph::at(g,id);byte(ref mut queued,id-1,0);
        i=0;while(i<width){put(ref mut output,i,*v::at<usize>(ref input,(id-1)*width+i));i+=1;}
        if(n.column!=0 && (n.kind==1 || n.kind==4 || n.kind==2)){put(ref mut output,n.column-1,if(n.kind==2){id}else{0});}
        var usize side=0;while(side<2){let usize successor=if(side==0){n.next}else{n.alternative};side+=1;if(successor==0){continue;}
            var bool changed=*v::at<u8>(ref seen,successor-1)==0;byte(ref mut seen,successor-1,1);
            i=0;while(i<width){
                if(work>=16777216){return state::fail(10,n.start,n.end,0,0);}work+=1;
                let usize at=(successor-1)*width+i;let usize before=*v::at<usize>(ref input,at);let usize after=earlier(g,before,*v::at<usize>(ref output,i));
                if(after!=before){put(ref mut input,at,after);changed=true;}i+=1;
            }
            if(changed && *v::at<u8>(ref queued,successor-1)==0){if(!v::push<usize>(ref mut queue,successor).is_ok){return state::fail(12,n.start,n.end,0,0);}byte(ref mut queued,successor-1,1);}
        }
    }
    var usize error=0;i=1;
    while(i<=count){if(*v::at<u8>(ref seen,i-1)==0){i+=1;continue;}let n=graph::at(g,i);
        if(n.kind==6){error=earlier(g,error,i);}
        if(n.kind==3 || n.kind==4){error=earlier(g,error,*v::at<usize>(ref input,(i-1)*width+n.column-1));}
        if(n.kind==5){var usize col=0;while(col<width){error=earlier(g,error,*v::at<usize>(ref input,(i-1)*width+col));col+=1;}}
        i+=1;
    }
    if(error==0){return state::ok();}let n=graph::at(g,error);return state::fail(42,n.start,n.end,0,0);
}
pub fn state::Problem check(ref names::Resolved resolved,ref v::Vec<state::Slot> types){
    var usize i=1;var state::Problem first=state::ok();
    while(i<=names::symbol_count(resolved)){
        let s=names::symbol_at(resolved,i);if(s.kind==1 || s.kind==9){
            let g=switch(graph::build(resolved,types,i)){case .ok(g):{g}case .err(p):{return p;}};
            let p=analyze(ref g);if(p.code!=0 && (first.code==0 || p.start<first.start || p.start==first.start && p.end<first.end)){first=p;}
        }i+=1;
    }return first;
}
