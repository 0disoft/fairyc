module generic_source_closure;
import generic_source_dependency_model as model;
import generic_source_dependency_json as declarations;
import generic_source_catalog as catalog;
import generic_source_json as json;
import generic_identity as identity;
import module_state_layout as budget;
import std::bytes as bytes;
import std::vec as v;

// A visited set describes unused recursive templates finitely. Executable
// specialization admission remains the planner's separate responsibility.
// Catalog order is canonical DeclarationID order, independent of source owners.
pub fn Result<v::Vec<bool>,u32> reachable(ref model::Graph graph,usize root,ref mut usize work){
    let templates=model::templates(graph);let total=catalog::count(templates);let data=model::data(graph);
    if(root>=total || model::range_count(data)!=total){return .err(9);}if(total>4096){return .err(10);}
    var v::Vec<bool> seen=v::empty<bool>();var v::Vec<usize> pending=v::empty<usize>();var usize i=0;
    while(i<total){if(!budget::charge(work,1)){return .err(10);}if(!v::push<bool>(ref mut seen,false).is_ok){return .err(11);}i+=1;}
    let first=v::at_mut<bool>(ref mut seen,root);*first=true;if(!v::push<usize>(ref mut pending,root).is_ok){return .err(11);}
    while(v::len<usize>(ref pending)!=0){if(!budget::charge(work,1)){return .err(10);}let at=*v::at<usize>(ref pending,v::len<usize>(ref pending)-1);discard v::pop<usize>(ref mut pending);let range=model::range_at(data,at);
        if(range.count>32 || range.first>model::edge_count(data) || range.count>model::edge_count(data)-range.first){return .err(9);}i=0;
        while(i<range.count){if(!budget::charge(work,1)){return .err(10);}let edge=model::edge_at(data,range.first+i);if(edge.target==0 || edge.target>total){return .err(9);}let next=edge.target-1;
            if(!*v::at<bool>(ref seen,next)){let slot=v::at_mut<bool>(ref mut seen,next);*slot=true;if(!v::push<usize>(ref mut pending,next).is_ok){return .err(11);}}i+=1;
        }
    }return .ok(seen);
}
fn bool document(ref model::Graph graph,usize root,ref v::Vec<bool> seen,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let templates=model::templates(graph);let total=catalog::count(templates);if(root>=total || v::len<bool>(seen)!=total){return json::reject(work,9);}let row=catalog::template_at(templates,root);
    if(!json::raw(out,b"{\"Root\":",work) || !json::quoted(out,catalog::text(templates,row.id),work) || !json::raw(out,b",\"Declarations\":[",work)){return false;}var bool first=true;var usize i=0;
    while(i<total){if(!json::charge(work,1)){return false;}if(*v::at<bool>(seen,i)){if(!first && !json::raw(out,b",",work)){return false;}if(!declarations::declaration(graph,i,out,work)){return false;}first=false;}i+=1;}return json::raw(out,b"]}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref model::Graph graph,usize root,ref mut usize work){
    let seen=switch(reachable(graph,root,work)){case .ok(value):{value}case .err(code):{return .err(code);}};var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=document(graph,root,ref seen,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);
}
pub fn Result<bytes::Buffer,u32> digest(ref model::Graph graph,usize root,ref mut usize work){
    let encoded=switch(encode(graph,root,work)){case .ok(value):{value}case .err(code):{return .err(code);}};return identity::digest_text(b"source-public-template-closure-v0",bytes::view(ref encoded),work);
}
