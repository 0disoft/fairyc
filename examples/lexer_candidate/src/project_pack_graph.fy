module project_pack_graph;
import project_pack_relations as relations;
import project_pack_relation_order as ordering;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub resource Graph{v::Vec<relations::Relation> rows;v::Vec<usize> order;v::Vec<usize> sources;v::Vec<usize> targets;v::Vec<usize> outgoing;v::Vec<usize> incoming;v::Vec<usize> next_out;v::Vec<usize> next_in;cleanup{}}
pub fn usize count(ref Graph g){return v::len<usize>(ref g.order);}
pub fn usize nodes(ref Graph g){return v::len<usize>(ref g.outgoing);}
pub fn relations::Relation row(ref Graph g,usize rank){return *v::at<relations::Relation>(ref g.rows,*v::at<usize>(ref g.order,rank));}
pub fn usize source(ref Graph g,usize rank){return *v::at<usize>(ref g.sources,rank);}
pub fn usize target(ref Graph g,usize rank){return *v::at<usize>(ref g.targets,rank);}
pub fn usize first(ref Graph g,usize node,bool incoming){return if(incoming){*v::at<usize>(ref g.incoming,node)}else{*v::at<usize>(ref g.outgoing,node)};}
pub fn usize next(ref Graph g,usize rank,bool incoming){return if(incoming){*v::at<usize>(ref g.next_in,rank)}else{*v::at<usize>(ref g.next_out,rank)};}
pub fn Result<usize,u32> resolve(ref validate::Partition p,ref u8[] symbol,ref mut usize meter){
    let visible=wire::hex(symbol,b"fysym1-sha256-128-",32,false);if(!visible && !wire::hex(symbol,b"fysym1-sha256-256-",64,false)){return .err(9);}let t=validate::tree(p);var usize low=0;var usize high=validate::symbol_count(p);
    while(low<high){let middle=low+(high-low)/2;let id=wire::string(t,validate::symbol(p,middle),if(visible){b"visible"}else{b"full"});
        if(!work::charge(meter,id.len+symbol.len+1)){return .err(3);}if(bytes::equal(id,symbol)){return .ok(middle);}if(model::before(id,symbol)){low=middle+1;}else{high=middle;}
    }return .err(9);
}
pub fn Result<v::Vec<usize>,u32> zeros(usize n,ref mut usize meter){var result=v::empty<usize>();var usize i=0;while(i<n){if(!work::charge(meter,1)){return .err(3);}if(!v::push<usize>(ref mut result,0).is_ok){return .err(2);}i+=1;}return .ok(result);}
pub fn Result<Graph,u32> build(ref current::Ready ready,ref mut usize meter){
    let p=current::partition(ready);let t=validate::tree(p);let rows=relations::build(ready,meter)?;let order=ordering::sort(p,ref rows,meter)?;let n=validate::symbol_count(p);let r=v::len<usize>(ref order);
    var sources=v::empty<usize>();var targets=v::empty<usize>();var outgoing=zeros(n,meter)?;var incoming=zeros(n,meter)?;var next_out=zeros(r,meter)?;var next_in=zeros(r,meter)?;var usize i=0;
    while(i<r){if(!work::charge(meter,1)){return .err(3);}let row=*v::at<relations::Relation>(ref rows,*v::at<usize>(ref order,i));let source=resolve(p,wire::string(t,row.source,b"full"),meter)?;
        let target=if(row.kind==3 || row.kind==4){let id=wire::string(t,row.target,b"full");if(id.len>0){resolve(p,id,meter)?}else{n}}else{n};
        if(!v::push<usize>(ref mut sources,source).is_ok || !v::push<usize>(ref mut targets,target).is_ok){return .err(2);}i+=1;
    }
    // Zero means end-of-list; links store canonical rank + 1. Reverse insertion
    // creates ascending adjacency without sorting/copying each source list.
    i=r;while(i>0){if(!work::charge(meter,1)){return .err(3);}i-=1;let source=*v::at<usize>(ref sources,i);let target=*v::at<usize>(ref targets,i);let previous=*v::at<usize>(ref outgoing,source);
        {let slot=v::at_mut<usize>(ref mut next_out,i);*slot=previous;}{let slot=v::at_mut<usize>(ref mut outgoing,source);*slot=i+1;}
        if(target<n){let previous=*v::at<usize>(ref incoming,target);{let slot=v::at_mut<usize>(ref mut next_in,i);*slot=previous;}{let slot=v::at_mut<usize>(ref mut incoming,target);*slot=i+1;}}
    }
    return .ok(Graph{.rows,.order,.sources,.targets,.outgoing,.incoming,.next_out,.next_in});
}
