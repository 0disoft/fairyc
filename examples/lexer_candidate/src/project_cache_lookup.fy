module project_cache_lookup;
import project_cache_environment as environment;
import project_cache_profile as profile;
import project_cache_key as key;
import project_cache_encoding as budget;
import project_module_visibility as visibility;
import project_workspace_graph as graph;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

// A pre-compilation routing hint, never a cache hit or artifact authority.
// Exact instance IDs include UID/version/features and path-and-byte source
// identity. Root, resolved binding edges and selected SDK are also retained.
// Only project_cache_artifact.restore can admit the selected saved program.
pub resource Identity{bytes::Buffer canonical;bytes::Buffer digest;cleanup{}}
pub fn ref u8[] canonical(ref Identity p){return bytes::view(ref p.canonical);}
pub fn ref u8[] digest(ref Identity p){return bytes::view(ref p.digest);}
fn u32 mapped(u32 code){return if(code==5){10}else if(code==4){11}else if(code==0){0}else{9};}
fn u32 field(ref mut bytes::Buffer out,ref u8[] value,ref mut usize work){return mapped(budget::string(out,value,work));}
fn u32 number(ref mut bytes::Buffer out,usize value,ref mut usize work){return mapped(budget::length(out,value,work));}
fn void set(ref mut v::Vec<usize> values,usize index,usize value){let ref mut usize slot=v::at_mut<usize>(values,index);*slot=value;}
fn Result<v::Vec<usize>,u32> nodes(ref graph::Graph g,ref mut usize work){
    let count=graph::node_count(g);if(count==0 || count>512){return .err(9);}var v::Vec<usize> order=v::empty<usize>();var usize i=0;
    while(i<count){if(!budget::charge(work,1)){return .err(10);}if(!v::push<usize>(ref mut order,i).is_ok){return .err(11);}var usize j=i;
        while(j>0){let previous=*v::at<usize>(ref order,j-1);let a=graph::node_id(g,i);let b=graph::node_id(g,previous);if(!budget::charge(work,a.len+b.len+1)){return .err(10);}if(!budget::before(a,b)){if(bytes::equal(a,b)){return .err(9);}break;}set(ref mut order,j,previous);j-=1;}set(ref mut order,j,i);i+=1;
    }return .ok(order);
}
fn Result<v::Vec<usize>,u32> edges(ref graph::Graph g,usize owner,ref mut usize work){
    let n=graph::node(g,owner);if(n.first>graph::edge_count(g) || n.count>graph::edge_count(g)-n.first){return .err(9);}var v::Vec<usize> order=v::empty<usize>();var usize i=0;
    while(i<n.count){let at=n.first+i;let e=graph::edge(g,at);if(e.owner!=owner || e.target>=graph::node_count(g)){return .err(9);}if(!budget::charge(work,1)){return .err(10);}if(!v::push<usize>(ref mut order,at).is_ok){return .err(11);}var usize j=i;
        while(j>0){let previous=*v::at<usize>(ref order,j-1);let a=graph::binding(g,at);let b=graph::binding(g,previous);if(!budget::charge(work,a.len+b.len+1)){return .err(10);}if(!budget::before(a,b)){if(bytes::equal(a,b)){return .err(9);}break;}set(ref mut order,j,previous);j-=1;}set(ref mut order,j,at);i+=1;
    }return .ok(order);
}
fn u32 package(ref mut bytes::Buffer out,ref graph::Graph g,usize owner,ref mut usize work){
    var u32 code=field(out,graph::node_id(g,owner),work);if(code!=0){return code;}code=number(out,if(graph::is_sdk(g,owner)){1}else{0},work);if(code!=0){return code;}
    let count=graph::module_count(g,owner);code=number(out,count,work);if(code!=0){return code;}var usize i=0;
    while(i<count){let module_index=graph::module_order(g,owner,i);code=field(out,graph::module_name(g,owner,module_index),work);if(code!=0){return code;}code=field(out,graph::module_file(g,owner,module_index),work);if(code!=0){return code;}i+=1;}
    let order=switch(edges(g,owner,work)){case .ok(p):{p}case .err(e):{return e;}};code=number(out,v::len<usize>(ref order),work);if(code!=0){return code;}i=0;
    while(i<v::len<usize>(ref order)){let at=*v::at<usize>(ref order,i);code=field(out,graph::binding(g,at),work);if(code!=0){return code;}code=field(out,graph::node_id(g,graph::edge(g,at).target),work);if(code!=0){return code;}i+=1;}return 0;
}
pub fn Result<Identity,u32> collect(ref visibility::Resolved loaded,ref environment::Environment env,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}let g=visibility::workspace_graph(loaded);let order=switch(nodes(g,work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let fields=switch(environment::key_environment(env,work)){case .ok(p):{p}case .err(e):{return .err(e);}};var bytes::Buffer out=bytes::empty();var u32 code=mapped(budget::raw(ref mut out,b"fairyc.candidate-project-lookup\x00\x00",work));if(code!=0){return .err(code);}var usize i=0;
    while(i<8){code=field(ref mut out,key::environment_field(ref fields,i),work);if(code!=0){return .err(code);}i+=1;}
    code=field(ref mut out,profile::target(),work);if(code!=0){return .err(code);}code=field(ref mut out,environment::standard_library(env),work);if(code!=0){return .err(code);}code=field(ref mut out,graph::node_id(g,0),work);if(code!=0){return .err(code);}
    code=number(ref mut out,v::len<usize>(ref order),work);if(code!=0){return .err(code);}i=0;while(i<v::len<usize>(ref order)){code=package(ref mut out,g,*v::at<usize>(ref order,i),work);if(code!=0){return .err(code);}i+=1;}
    if(!budget::charge(work,bytes::len(ref out)+81)){return .err(10);}var sha::State state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref out))){return .err(10);}var bytes::Buffer digest=bytes::empty();if(!bytes::append(ref mut digest,b"fylookup0-sha256-").is_ok || !sha::append_hex(ref state,ref mut digest)){return .err(11);}return .ok(Identity{.canonical=out,.digest});
}
