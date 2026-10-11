module project_workspace_graph;
import project_instance_graph as local;
import project_instance_identity as identity;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_source_modules as modules;
import project_source_set as sources;
import project_sdk_bundle as sdk;
import project_path as work;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Retain the complete local graph; resource fields are never moved or mutated.
// SDK overrides only node/edge ranges and order, retaining local instance IDs.
pub resource Graph{local::Graph local;v::Vec<sdk::Bundle> library;v::Vec<identity::Identity> sdk_id;v::Vec<local::Node> nodes;v::Vec<local::Edge> edges;v::Vec<usize> order;cleanup{}}
pub resource Error{local::Graph local;local::Problem problem;cleanup{}}
pub fn local::Problem error(ref Error error){return error.problem;}
pub fn ref local::Graph error_local_graph(ref Error error){return ref error.local;}
pub fn Error bundle_failure(local::Graph input,sdk::Problem failure){return Error{.local=input,.problem=local::Problem{.code=8,.reason=failure.code,.node=failure.file,.declaration=0}};}
pub fn ref local::Graph local_graph(ref Graph graph){return ref graph.local;}
pub fn bool has_sdk(ref Graph graph){return v::len<sdk::Bundle>(ref graph.library)==1;}
pub fn usize local_node_count(ref Graph graph){return local::node_count(ref graph.local);}
pub fn usize node_count(ref Graph graph){return local_node_count(graph)+v::len<sdk::Bundle>(ref graph.library);}
pub fn bool is_sdk(ref Graph graph,usize at){return has_sdk(graph) && at==local_node_count(graph);}
pub fn ref sdk::Bundle standard_library(ref Graph graph){assert(has_sdk(graph));return v::at<sdk::Bundle>(ref graph.library,0);}
pub fn local::Node node(ref Graph graph,usize at){assert(at<node_count(graph));if(has_sdk(graph)){return *v::at<local::Node>(ref graph.nodes,at);}return local::node(ref graph.local,at);}
pub fn ref u8[] node_id(ref Graph graph,usize at){assert(at<node_count(graph));if(is_sdk(graph,at)){return identity::digest(v::at<identity::Identity>(ref graph.sdk_id,0));}return local::node_id(ref graph.local,at);}
pub fn usize edge_count(ref Graph graph){if(has_sdk(graph)){return v::len<local::Edge>(ref graph.edges);}return local::edge_count(ref graph.local);}
pub fn local::Edge edge(ref Graph graph,usize at){assert(at<edge_count(graph));if(has_sdk(graph)){return *v::at<local::Edge>(ref graph.edges,at);}return local::edge(ref graph.local,at);}
pub fn ref u8[] binding(ref Graph graph,usize at){let e=edge(graph,at);if(is_sdk(graph,e.target)){return b"stdlib";}return workspace::binding(local::input(ref graph.local),e.declaration);}
pub fn usize build_order(ref Graph graph,usize at){assert(at<node_count(graph));if(has_sdk(graph)){return *v::at<usize>(ref graph.order,at);}return local::build_order(ref graph.local,at);}
fn ref modules::Modules package(ref Graph graph,usize at){assert(at<local_node_count(graph));return catalog::input(workspace::local_catalog(local::input(ref graph.local)),local::node(ref graph.local,at).package_index);}
pub fn usize module_count(ref Graph graph,usize at){if(is_sdk(graph,at)){return sdk::count(standard_library(graph));}return modules::count(package(graph,at));}
pub fn usize module_order(ref Graph graph,usize at,usize ordinal){if(is_sdk(graph,at)){assert(ordinal<module_count(graph,at));return ordinal;}return modules::ordered(package(graph,at),ordinal);}
pub fn ref u8[] module_name(ref Graph graph,usize at,usize index){if(is_sdk(graph,at)){return sdk::module_name(standard_library(graph),index);}return modules::name(package(graph,at),index);}
pub fn ref u8[] module_file(ref Graph graph,usize at,usize index){if(is_sdk(graph,at)){return sdk::source_path(standard_library(graph),index);}return modules::path(package(graph,at),index);}
pub fn ref syntax::Parsed module_parser(ref Graph graph,usize at,usize index){if(is_sdk(graph,at)){return sdk::parsed(standard_library(graph),index);}return modules::parsed(package(graph,at),index);}
pub fn ref u8[] module_source(ref Graph graph,usize at,usize index){return syntax::source_view(module_parser(graph,at,index));}
pub fn Result<usize,u32> find_module(ref Graph graph,usize at,ref u8[] path,ref mut usize meter){if(is_sdk(graph,at)){if(!work::charge(meter,path.len+1)){return .err(8);}return .ok(sdk::find(standard_library(graph),path));}return modules::find_metered(package(graph,at),path,meter);}
pub fn Graph without_sdk(local::Graph input){return Graph{.local=input,.library=v::empty<sdk::Bundle>(),.sdk_id=v::empty<identity::Identity>(),.nodes=v::empty<local::Node>(),.edges=v::empty<local::Edge>(),.order=v::empty<usize>()};}
fn local::Problem problem(u32 code,u32 reason,usize node){return local::Problem{.code,.reason,.node,.declaration=0};}
fn void number(ref mut v::Vec<usize> values,usize at,usize value){let ref mut usize slot=v::at_mut<usize>(values,at);*slot=value;}
// The local graph is already acyclic and the SDK has no outgoing package edges.
// Preserve local dependency order, placing SDK immediately before its first
// direct owner in the dependency-first order (rather than globally first).
fn u32 order(ref local::Graph input,ref local::SDKSelection selected,ref mut v::Vec<usize> output,ref mut usize meter){var v::Vec<usize> marks=v::empty<usize>();let usize count=local::node_count(input);var usize at=0;while(at<count){if(!v::push<usize>(ref mut marks,0).is_ok){return 3;}at+=1;}at=0;while(at<local::sdk_owner_count(selected)){number(ref mut marks,local::sdk_owner(selected,at),1);at+=1;}var bool inserted=false;at=0;
    while(at<count){if(!work::charge(meter,1)){return 4;}let usize node=local::build_order(input,at);if(!inserted && *v::at<usize>(ref marks,node)==1){if(!v::push<usize>(output,count).is_ok){return 3;}inserted=true;}if(!v::push<usize>(output,node).is_ok){return 3;}at+=1;}return if(inserted){0}else{6};
}
// 7 wrong selected profile; other errors use the existing graph problem codes.
// Failures own the untouched local graph; SDK bytes are never searched/read here.
pub fn Result<Graph,Error> inject(local::Graph input,sdk::Bundle library,ref mut usize meter){
    let local::SDKSelection selected=switch(local::select_sdk(ref input,sdk::candidate(ref library),meter)){case .ok(value):{value}case .err(p):{return .err(Error{.local=input,.problem=p});}};if(local::sdk_profile(ref selected)==0){return .ok(without_sdk(input));}if(sdk::vector(ref library)!=(local::sdk_profile(ref selected)!=1)){let p=problem(7,local::sdk_profile(ref selected),0);return .err(Error{.local=input,.problem=p});}let v::Vec<bytes::Buffer> features=v::empty<bytes::Buffer>();
    let identity::Identity key=switch(identity::compute(sdk::uid(),sdk::source_identity(ref library),sdk::version(),ref features,meter)){case .ok(value):{value}case .err(code):{let p=problem(if(code==6){4}else if(code==5){3}else{1},code,0);return .err(Error{.local=input,.problem=p});}};let usize sdk_index=local::node_count(ref input);var usize at=0;
    while(at<sdk_index){if(!work::charge(meter,identity::digest(ref key).len+1)){let p=problem(4,0,at);return .err(Error{.local=input,.problem=p});}if(bytes::equal(local::node_id(ref input,at),identity::digest(ref key))){let p=problem(2,0,at);return .err(Error{.local=input,.problem=p});}at+=1;}
    var v::Vec<local::Node> nodes=v::empty<local::Node>();var v::Vec<local::Edge> edges=v::empty<local::Edge>();var usize owner=0;var usize selected_index=0;
    while(owner<sdk_index){var local::Node n=local::node(ref input,owner);let usize first=v::len<local::Edge>(ref edges);var usize offset=0;while(offset<n.count){if(!work::charge(meter,1)){let p=problem(4,0,owner);return .err(Error{.local=input,.problem=p});}if(!v::push<local::Edge>(ref mut edges,local::edge(ref input,n.first+offset)).is_ok){let p=problem(3,0,owner);return .err(Error{.local=input,.problem=p});}offset+=1;}
        if(selected_index<local::sdk_owner_count(ref selected) && local::sdk_owner(ref selected,selected_index)==owner){if(!v::push<local::Edge>(ref mut edges,local::Edge{.owner,.target=sdk_index,.declaration=workspace::edge_count(local::input(ref input))}).is_ok){let p=problem(3,0,owner);return .err(Error{.local=input,.problem=p});}selected_index+=1;}n.first=first;n.count=v::len<local::Edge>(ref edges)-first;if(!v::push<local::Node>(ref mut nodes,n).is_ok){let p=problem(3,0,owner);return .err(Error{.local=input,.problem=p});}owner+=1;
    }
    if(!v::push<local::Node>(ref mut nodes,local::Node{.package_index=catalog::count(workspace::local_catalog(local::input(ref input))),.feature_edge=0,.hash=0,.next=0,.first=v::len<local::Edge>(ref edges),.count=0}).is_ok){let p=problem(3,0,sdk_index);return .err(Error{.local=input,.problem=p});}var v::Vec<usize> ordered=v::empty<usize>();let u32 code=order(ref input,ref selected,ref mut ordered,meter);if(code!=0){let p=problem(code,0,0);return .err(Error{.local=input,.problem=p});}
    var v::Vec<sdk::Bundle> owned=v::empty<sdk::Bundle>();if(!v::push<sdk::Bundle>(ref mut owned,library).is_ok){let p=problem(3,0,sdk_index);return .err(Error{.local=input,.problem=p});}var v::Vec<identity::Identity> sdk_id=v::empty<identity::Identity>();if(!v::push<identity::Identity>(ref mut sdk_id,key).is_ok){let p=problem(3,0,sdk_index);return .err(Error{.local=input,.problem=p});}return .ok(Graph{.local=input,.library=owned,.sdk_id,.nodes,.edges,.order=ordered});
}
@effects(filesystem)
pub fn Result<Graph,local::LoadError> load(ref u8[] root,ref mut sources::Budget budget,ref mut usize meter){let local::Graph input=switch(local::load(root,budget,meter)){case .ok(value):{value}case .err(e):{return .err(e);}};return .ok(without_sdk(input));}
