module project_pack_around;
import project_pack_query as query;
import project_pack_node as nodes;
import project_pack_relations as relations;
import project_pack_graph as graph;
import project_pack_frontier as frontier;
import project_pack_context as context;
import project_catalog_json as w;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

fn Result<bytes::Buffer,u32> node(ref current::Ready ready,usize index,usize depth,ref mut usize meter){
    let fact=nodes::encode(ready,index,meter)?;var out=bytes::empty();discard w::raw(ref mut out,b"{\"depth\":",meter)?;discard w::number(ref mut out,depth,meter)?;
    discard w::raw(ref mut out,b",\"node\":",meter)?;discard w::raw(ref mut out,bytes::view(ref fact),meter)?;discard w::raw(ref mut out,b"}",meter)?;return .ok(out);
}
// Sort the next owned node frontier by full-SymbolId position, independently
// of the order in which the preceding relation stream first discovered nodes.
fn Result<v::Vec<usize>,u32> ordered(v::Vec<usize> input,ref mut usize meter){
    var rows=input;let n=v::len<usize>(ref rows);var spare=graph::zeros(n,meter)?;var usize width=1;
    while(width<n){var usize start=0;while(start<n){let middle=if(start+width<n){start+width}else{n};let end=if(middle+width<n){middle+width}else{n};var usize left=start;var usize right=middle;var usize out=start;
        while(out<end){if(!work::charge(meter,1)){return .err(3);}var usize selected=0;
            if(left<middle && (right==end || *v::at<usize>(ref rows,left)<=*v::at<usize>(ref rows,right))){selected=*v::at<usize>(ref rows,left);left+=1;}else{selected=*v::at<usize>(ref rows,right);right+=1;}
            {let slot=v::at_mut<usize>(ref mut spare,out);*slot=selected;}out+=1;
        }start=end;
    }var usize i=0;while(i<n){if(!work::charge(meter,1)){return .err(3);}let value=*v::at<usize>(ref spare,i);{let slot=v::at_mut<usize>(ref mut rows,i);*slot=value;}i+=1;}width*=2;}
    return .ok(rows);
}
pub fn Result<query::Selection,u32> select(ref current::Ready ready,ref u8[] symbol,query::Policy policy,ref mut usize meter){
    if(!query::valid(policy)){return .err(9);}let p=current::partition(ready);let root=graph::resolve(p,symbol,meter)?;let c=context::build(p,policy.mask,meter)?;let root_node=node(ready,root,0,meter)?;
    var mandatory=bytes::empty();discard w::raw(ref mut mandatory,b"{\"context\":",meter)?;discard w::raw(ref mut mandatory,context::json(ref c),meter)?;
    discard w::raw(ref mut mandatory,b",\"root\":",meter)?;discard w::raw(ref mut mandatory,bytes::view(ref root_node),meter)?;discard w::raw(ref mut mandatory,b"}",meter)?;
    var query::Usage used=query::Usage{.records=0,.relations=0,.depth=0,.selection=0};if(query::reserve(ref mut used,policy,bytes::len(ref mandatory)+1,false)!=0){return .err(3);}
    let g=graph::build(ready,meter)?;let n=graph::nodes(ref g);let r=graph::count(ref g);var seen_nodes=graph::zeros(n,meter)?;{let slot=v::at_mut<usize>(ref mut seen_nodes,root);*slot=1;}
    var seen_relations=graph::zeros(r,meter)?;var membership=graph::zeros(n,meter)?;var pending=v::empty<usize>();if(!v::push<usize>(ref mut pending,root).is_ok){return .err(2);}
    var facts=root_node;var out=bytes::empty();var omitted=bytes::empty();discard query::filter(ref mut omitted,policy.mask,meter)?;var usize depth=0;var usize unjoined=0;var bool exhausted=false;
    while(depth<policy.depth && v::len<usize>(ref pending)>0 && !exhausted){
        var usize i=0;while(i<v::len<usize>(ref pending)){if(!work::charge(meter,1)){return .err(3);}let at=*v::at<usize>(ref pending,i);{let slot=v::at_mut<usize>(ref mut membership,at);*slot=depth+1;}i+=1;}
        var iterator=frontier::start(ref g,ref pending,policy.direction,meter)?;var usize previous=r;var next=v::empty<usize>();var usize rank=frontier::next(ref mut iterator,ref g,policy.mask,ref seen_relations,ref mut previous,meter)?;
        while(rank<r){let row=graph::row(ref g,rank);let encoded=relations::encode(p,row,meter)?;let stopped=query::reserve(ref mut used,policy,bytes::len(ref encoded)+1,true);
            if(stopped!=0){discard query::omission(ref mut omitted,query::refusal(stopped),b"relations",0,0,false,meter)?;exhausted=true;break;}
            {let slot=v::at_mut<usize>(ref mut seen_relations,rank);*slot=1;}if(used.relations>1){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,bytes::view(ref encoded),meter)?;
            let source=graph::source(ref g,rank);let target=graph::target(ref g,rank);if((row.kind==3 || row.kind==4) && target==n){unjoined+=1;}
            let outgoing=if(policy.direction!=1 && target<n && *v::at<usize>(ref membership,source)==depth+1){target}else{n};
            let incoming=if(policy.direction!=0 && target<n && *v::at<usize>(ref membership,target)==depth+1){source}else{n};
            let first=if(outgoing<incoming){outgoing}else{incoming};let second=if(outgoing<incoming){incoming}else{outgoing};i=0;
            while(i<2){let neighbor=if(i==0){first}else{second};i+=1;if(neighbor==n || i==2 && first==second || *v::at<usize>(ref seen_nodes,neighbor)!=0){continue;}
                let value=node(ready,neighbor,depth+1,meter)?;let stopped=query::reserve(ref mut used,policy,bytes::len(ref value)+1,false);
                if(stopped!=0){discard query::omission(ref mut omitted,query::refusal(stopped),b"nodes",0,0,false,meter)?;exhausted=true;break;}
                {let slot=v::at_mut<usize>(ref mut seen_nodes,neighbor);*slot=1;}if(!v::push<usize>(ref mut next,neighbor).is_ok){return .err(2);}
                discard w::raw(ref mut facts,b",",meter)?;discard w::raw(ref mut facts,bytes::view(ref value),meter)?;used.depth=depth+1;
            }if(exhausted){break;}rank=frontier::next(ref mut iterator,ref g,policy.mask,ref seen_relations,ref mut previous,meter)?;
        }pending=ordered(next,meter)?;depth+=1;
    }
    if(!exhausted && v::len<usize>(ref pending)>0){var probe=frontier::start(ref g,ref pending,policy.direction,meter)?;var usize previous=r;
        if(frontier::next(ref mut probe,ref g,policy.mask,ref seen_relations,ref mut previous,meter)?<r){discard query::omission(ref mut omitted,b"depth-limit",b"relations",0,0,false,meter)?;}
    }discard query::joins(ref mut omitted,policy,unjoined,meter)?;
    let identity=switch(bytes::copy(wire::string(validate::tree(p),validate::symbol(p,root),b"visible"))){case .ok(p):{p}case .err(_):{return .err(2);}};
    let complete=switch(bytes::copy(context::completeness(ref c))){case .ok(p):{p}case .err(_):{return .err(2);}};
    return .ok(query::assemble(identity,facts,complete,out,omitted,used));
}
