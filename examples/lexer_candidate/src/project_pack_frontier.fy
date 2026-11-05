module project_pack_frontier;
import project_pack_graph as graph;
import project_path as work;
import std::vec as v;

pub struct Cursor{usize rank;bool incoming;}
fn Result<bool,u32> push(ref mut v::Vec<Cursor> heap,Cursor value,ref mut usize meter){
    if(!v::push<Cursor>(heap,value).is_ok){return .err(2);}var usize at=v::len<Cursor>(heap)-1;
    while(at>0){if(!work::charge(meter,1)){return .err(3);}let parent=(at-1)/2;let p=*v::at<Cursor>(heap,parent);if(p.rank<=value.rank){break;}{let slot=v::at_mut<Cursor>(heap,at);*slot=p;}at=parent;}
    {let slot=v::at_mut<Cursor>(heap,at);*slot=value;}return .ok(true);
}
pub fn Result<v::Vec<Cursor>,u32> start(ref graph::Graph g,ref v::Vec<usize> nodes,u32 direction,ref mut usize meter){
    var heap=v::empty<Cursor>();var usize i=0;while(i<v::len<usize>(nodes)){if(!work::charge(meter,1)){return .err(3);}let node=*v::at<usize>(nodes,i);
        if(direction!=1){let first=graph::first(g,node,false);if(first!=0){discard push(ref mut heap,Cursor{.rank=first-1,.incoming=false},meter)?;}}
        if(direction!=0){let first=graph::first(g,node,true);if(first!=0){discard push(ref mut heap,Cursor{.rank=first-1,.incoming=true},meter)?;}}i+=1;
    }return .ok(heap);
}
fn Result<Cursor,u32> pop(ref mut v::Vec<Cursor> heap,ref mut usize meter){
    let n=v::len<Cursor>(heap);let result=*v::at<Cursor>(heap,0);let last=*v::at<Cursor>(heap,n-1);if(v::truncate<Cursor>(heap,n-1)!=1){return .err(2);}if(n==1){return .ok(result);}
    var usize at=0;let size=n-1;while(at<size/2){if(!work::charge(meter,1)){return .err(3);}let left=at*2+1;var usize child=left;
        if(left+1<size && v::at<Cursor>(heap,left+1).rank<v::at<Cursor>(heap,left).rank){child=left+1;}
        let value=*v::at<Cursor>(heap,child);if(last.rank<=value.rank){break;}{let slot=v::at_mut<Cursor>(heap,at);*slot=value;}at=child;
    }{let slot=v::at_mut<Cursor>(heap,at);*slot=last;}return .ok(result);
}
// Lazy canonical k-way merge. Adjacent duplicate IDs from self-calls or both
// frontier endpoints are consumed once; distinct equal observations stay distinct.
pub fn Result<usize,u32> next(ref mut v::Vec<Cursor> heap,ref graph::Graph g,usize mask,ref v::Vec<usize> seen,ref mut usize previous,ref mut usize meter){
    while(v::len<Cursor>(heap)>0){if(!work::charge(meter,1)){return .err(3);}let item=pop(heap,meter)?;let after=graph::next(g,item.rank,item.incoming);
        if(after!=0){discard push(heap,Cursor{.rank=after-1,.incoming=item.incoming},meter)?;}
        let repeated=*previous==item.rank;*previous=item.rank;
        if(repeated || v::len<usize>(seen)>0 && *v::at<usize>(seen,item.rank)!=0){continue;}
        if((mask & (1<<graph::row(g,item.rank).kind))!=0){return .ok(item.rank);}
    }return .ok(graph::count(g));
}
