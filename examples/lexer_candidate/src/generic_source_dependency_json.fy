module generic_source_dependency_json;
import generic_source_dependency_model as model;
import generic_source_catalog as catalog;
import generic_package_bindings as bindings;
import generic_source_json as json;
import std::bytes as bytes;

// Exact canonical genericplan.Declaration JSON. Source syntax has no capability
// constraints; the existing scalar requirement expands through known abilities.
pub fn bool declaration(ref model::Graph graph,usize index,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let c=model::templates(graph);if(!json::charge(work,1)){return false;}if(index>=catalog::count(c)){return json::reject(work,9);}let row=catalog::template_at(c,index);let packages=catalog::packages(c);let owner=bindings::module_at(packages,row.module_index);let data=model::data(graph);let edges=model::range_at(data,index);
    if(!json::raw(out,b"{\"ID\":",work) || !json::quoted(out,catalog::text(c,row.id),work) || !json::raw(out,b",\"Owner\":",work) || !json::quoted(out,bindings::text(packages,owner.namespace),work) || !json::raw(out,b",\"Parameters\":[",work)){return false;}var usize i=0;
    while(i<row.count){if(!json::charge(work,1)){return false;}let p=catalog::parameter_at(c,row.first+i);if(i!=0 && !json::raw(out,b",",work)){return false;}if(!json::raw(out,b"{\"Name\":",work) || !json::quoted(out,catalog::text(c,p.name),work) || !json::raw(out,if(p.scalar){b",\"Abilities\":[\"copy\",\"drop\",\"inert\",\"scalar\",\"store\"]}"}else{b",\"Abilities\":[]}"},work)){return false;}i+=1;}
    if(!json::raw(out,b"],\"Constraints\":[],\"Dependencies\":[",work)){return false;}i=0;while(i<edges.count){if(!json::charge(work,1)){return false;}let edge=model::edge_at(data,edges.first+i);let target=catalog::template_at(c,edge.target-1);if(i!=0 && !json::raw(out,b",",work)){return false;}if(!json::raw(out,b"{\"Declaration\":",work) || !json::quoted(out,catalog::text(c,target.id),work) || !json::raw(out,b",\"Arguments\":[",work)){return false;}var usize j=0;
        while(j<edge.count){if(!json::charge(work,1)){return false;}let arg=model::argument_at(data,edge.first+j);if(j!=0 && !json::raw(out,b",",work)){return false;}if(!json::raw(out,b"{\"Parameter\":",work) || !json::quoted(out,if(arg.parameter){model::text(data,arg.name)}else{b""},work) || !json::raw(out,b",\"Concrete\":",work) || !json::quoted(out,if(arg.parameter){b""}else{model::text(data,arg.name)},work) || !json::raw(out,b"}",work)){return false;}j+=1;
        }if(!json::raw(out,b"]}",work)){return false;}i+=1;
    }if(!json::raw(out,b"],\"Definition\":[",work)){return false;}let hex=catalog::text(c,row.definition);i=0;while(i<32){if(!json::charge(work,1)){return false;}if(i!=0 && !json::raw(out,b",",work)){return false;}let a=hex[i*2];let b=hex[i*2+1];let value=(if(a<=57){a-48}else{a-87}) as usize *16+(if(b<=57){b-48}else{b-87}) as usize;if(!json::number(out,value,work)){return false;}i+=1;}return json::raw(out,b"]}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref model::Graph graph,usize index,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=declaration(graph,index,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
