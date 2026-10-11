module project_instance_graph;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_source_set as sources;
import project_source_modules as modules;
import project_import_path as imports;
import project_instance_identity as identity;
import project_path as work;
import sdk_authority as sdk;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// feature_edge is zero for the root or a workspace declaration index plus one.
pub struct Node{usize package_index;usize feature_edge;usize hash;usize next;usize first;usize count;}
// declaration is the original workspace edge index, preserving its binding.
pub struct Edge{usize owner;usize target;usize declaration;}
struct Range{usize first;usize count;}
struct Frame{usize node;usize next;}
pub struct Problem{u32 code;u32 reason;usize node;usize declaration;}
// 0 no reachable SDK import; 1 core; 2 vector; 3 explicit vector candidate.
// Owners are exact local instance indices, in graph order, without duplicates.
pub resource SDKSelection{u32 profile;v::Vec<usize> owners;cleanup{}}
pub fn u32 sdk_profile(ref SDKSelection selected){return selected.profile;}
pub fn usize sdk_owner_count(ref SDKSelection selected){return v::len<usize>(ref selected.owners);}
pub fn usize sdk_owner(ref SDKSelection selected,usize at){assert(at<sdk_owner_count(selected));return *v::at<usize>(ref selected.owners,at);}
// 1 identity failure, 2 digest collision, 3 allocation, 4 work bound,
// 5 dependency cycle, 6 invalid declaration shape.
pub resource Graph{workspace::Workspace input;v::Vec<identity::Identity> identities;v::Vec<Node> nodes;v::Vec<Edge> edges;v::Vec<usize> order;cleanup{}}
pub resource Error{workspace::Workspace input;v::Vec<identity::Identity> identities;v::Vec<Node> nodes;v::Vec<Edge> edges;v::Vec<usize> cycle;Problem problem;cleanup{}}
// Phase 1 moves a workspace-input error, phase 2 a graph error, phase 3 allocation.
pub resource LoadError{u32 phase;v::Vec<workspace::Error> input;v::Vec<Error> graph;cleanup{}}
fn Problem fail(u32 code,u32 reason,usize node,usize declaration){return Problem{.code,.reason,.node,.declaration};}
pub fn ref str reason(u32 code){if(code==5){return "package-dependency-cycle";}if(code==2){return "package-instance-id-collision";}if(code==3){return "allocation-failed";}if(code==4){return "implementation-bound";}return "invalid-package-graph";}
pub fn Problem error(ref Error error){return error.problem;}
pub fn ref workspace::Workspace error_workspace(ref Error error){return ref error.input;}
pub fn usize cycle_count(ref Error error){return v::len<usize>(ref error.cycle);}
pub fn ref u8[] cycle_id(ref Error error,usize at){assert(at<cycle_count(error));return identity::digest(v::at<identity::Identity>(ref error.identities,*v::at<usize>(ref error.cycle,at)));}
pub fn u32 load_phase(ref LoadError error){return error.phase;}
pub fn ref workspace::Error input_error(ref LoadError error){assert(error.phase==1);return v::at<workspace::Error>(ref error.input,0);}
pub fn ref Error graph_error(ref LoadError error){assert(error.phase==2);return v::at<Error>(ref error.graph,0);}
pub fn ref workspace::Workspace input(ref Graph graph){return ref graph.input;}
pub fn usize node_count(ref Graph graph){return v::len<Node>(ref graph.nodes);}
pub fn Node node(ref Graph graph,usize at){assert(at<node_count(graph));return *v::at<Node>(ref graph.nodes,at);}
pub fn ref u8[] node_id(ref Graph graph,usize at){assert(at<node_count(graph));return identity::digest(v::at<identity::Identity>(ref graph.identities,at));}
pub fn usize edge_count(ref Graph graph){return v::len<Edge>(ref graph.edges);}
pub fn Edge edge(ref Graph graph,usize at){assert(at<edge_count(graph));return *v::at<Edge>(ref graph.edges,at);}
pub fn ref u8[] binding(ref Graph graph,usize at){return workspace::binding(ref graph.input,edge(graph,at).declaration);}
pub fn usize build_order(ref Graph graph,usize at){assert(at<node_count(graph));return *v::at<usize>(ref graph.order,at);}
// Selection uses only root-reachable graph instances and verified import tokens.
// Unknown std::* still selects a bundle; module resolution must reject it later.
pub fn Result<SDKSelection,Problem> select_sdk(ref Graph graph,bool candidate,ref mut usize meter){
    var v::Vec<usize> owners=v::empty<usize>();var bool vector=false;var usize node=0;
    while(node<node_count(graph)){if(!work::charge(meter,1)){return .err(fail(4,0,node,0));}let p=catalog::input(workspace::local_catalog(input(graph)),v::at<Node>(ref graph.nodes,node).package_index);var bool needed=false;var usize ordinal=0;
        while(ordinal<modules::count(p)){let tree=modules::parsed(p,modules::ordered(p,ordinal));var usize declaration=1;
            while(declaration<syntax::declaration_count(tree)){if(!work::charge(meter,1)){return .err(fail(4,0,node,declaration));}if(syntax::declaration_at(tree,declaration).kind==2){let path=switch(imports::read(tree,declaration,meter)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==2){3}else if(code==3){4}else{6},code,node,declaration));}};needed=needed || sdk::reserved(imports::name(ref path));vector=vector || bytes::equal(imports::name(ref path),b"std::vec");}declaration+=1;
            }ordinal+=1;
        }if(needed && !v::push<usize>(ref mut owners,node).is_ok){return .err(fail(3,0,node,0));}node+=1;
    }
    return .ok(SDKSelection{.profile=if(v::len<usize>(ref owners)==0){0}else if(candidate){3}else if(vector){2}else{1},.owners});
}
fn void number(ref mut v::Vec<usize> values,usize at,usize value){let ref mut usize slot=v::at_mut<usize>(values,at);*slot=value;}
fn void set_node(ref mut v::Vec<Node> nodes,usize at,Node value){let ref mut Node slot=v::at_mut<Node>(nodes,at);*slot=value;}
fn void set_range(ref mut v::Vec<Range> ranges,usize at,Range value){let ref mut Range slot=v::at_mut<Range>(ranges,at);*slot=value;}
fn void color(ref mut v::Vec<u32> colors,usize at,u32 value){let ref mut u32 slot=v::at_mut<u32>(colors,at);*slot=value;}
fn void frame(ref mut v::Vec<Frame> frames,usize at,Frame value){let ref mut Frame slot=v::at_mut<Frame>(frames,at);*slot=value;}
fn usize fingerprint(ref u8[] key){var usize value=0;var usize at=0;while(at<key.len){value=(value*33+(key[at] as usize))%16777213;at+=1;}return value;}
fn u32 capacity(ref mut v::Vec<Node> nodes,ref mut v::Vec<usize> buckets,ref mut usize meter){
    let usize before=v::len<usize>(buckets);var usize target=if(before==0){64}else{before};while(v::len<Node>(nodes)+1>target/2){target*=2;}if(target==before){return 0;}if(!work::charge(meter,target+v::len<Node>(nodes))){return 4;}if(!v::reserve<usize>(buckets,target).is_ok){return 3;}
    while(v::len<usize>(buckets)<target){if(!v::push<usize>(buckets,0).is_ok){return 3;}}var usize at=0;while(at<target){number(buckets,at,0);at+=1;}at=0;while(at<v::len<Node>(nodes)){var Node n=*v::at<Node>(nodes,at);let usize bucket=n.hash%target;n.next=*v::at<usize>(buckets,bucket);set_node(nodes,at,n);number(buckets,bucket,at+1);at+=1;}return 0;
}
fn Result<usize,Problem> intern(identity::Identity candidate,usize package_index,usize feature_edge,ref mut v::Vec<identity::Identity> identities,ref mut v::Vec<Node> nodes,ref mut v::Vec<usize> buckets,ref mut usize meter){
    if(!work::charge(meter,identity::digest(ref candidate).len+1)){return .err(fail(4,0,0,feature_edge));}let usize hash=fingerprint(identity::digest(ref candidate));var usize found=if(v::len<usize>(buckets)==0){0}else{*v::at<usize>(buckets,hash%v::len<usize>(buckets))};var usize steps=0;
    while(found!=0){let usize at=found-1;if(at>=v::len<Node>(nodes) || steps>=v::len<Node>(nodes)){return .err(fail(6,0,at,feature_edge));}let ref identity::Identity existing=v::at<identity::Identity>(identities,at);if(!work::charge(meter,identity::digest(existing).len+1)){return .err(fail(4,0,at,feature_edge));}
        if(bytes::equal(identity::digest(existing),identity::digest(ref candidate))){if(!work::charge(meter,identity::canonical(existing).len+identity::canonical(ref candidate).len+1)){return .err(fail(4,0,at,feature_edge));}if(identity::compare(existing,ref candidate)!=1){return .err(fail(2,0,at,feature_edge));}return .ok(found);}found=v::at<Node>(nodes,at).next;steps+=1;
    }
    let u32 code=capacity(nodes,buckets,meter);if(code!=0){return .err(fail(code,0,0,feature_edge));}let usize at=v::len<Node>(nodes);let usize bucket=hash%v::len<usize>(buckets);let Node n=Node{.package_index,.feature_edge,.hash,.next=*v::at<usize>(buckets,bucket),.first=0,.count=0};
    if(!v::push<identity::Identity>(identities,candidate).is_ok){return .err(fail(3,0,at,feature_edge));}if(!v::push<Node>(nodes,n).is_ok){return .err(fail(3,0,at,feature_edge));}number(buckets,bucket,at+1);return .ok(at+1);
}
fn Problem expand(ref workspace::Workspace input,ref mut v::Vec<identity::Identity> identities,ref mut v::Vec<Node> nodes,ref mut v::Vec<Edge> edges,ref mut usize meter){
    var v::Vec<usize> buckets=v::empty<usize>();let identity::Identity root=switch(identity::root(input,meter)){case .ok(value):{value}case .err(code):{return fail(if(code==6){4}else{1},code,0,0);}};switch(intern(root,0,0,identities,nodes,ref mut buckets,meter)){case .ok(_):{}case .err(p):{return p;}};
    var v::Vec<Range> ranges=v::empty<Range>();var v::Vec<usize> targets=v::empty<usize>();let usize packages=catalog::count(workspace::local_catalog(input));if(!work::charge(meter,packages+workspace::edge_count(input))){return fail(4,0,0,0);}var usize at=0;
    while(at<packages){if(!v::push<Range>(ref mut ranges,Range{.first=0,.count=0}).is_ok){return fail(3,0,0,0);}at+=1;}at=0;var usize previous_owner=0;
    while(at<workspace::edge_count(input)){let e=workspace::edge(input,at);if(e.owner>=packages || e.target>=packages || e.owner<previous_owner){return fail(6,0,0,at);}var Range r=*v::at<Range>(ref ranges,e.owner);if(r.count==0){r.first=at;}r.count+=1;set_range(ref mut ranges,e.owner,r);if(!v::push<usize>(ref mut targets,0).is_ok){return fail(3,0,0,at);}previous_owner=e.owner;at+=1;}
    var usize owner=0;while(owner<v::len<Node>(nodes)){let Node n=*v::at<Node>(nodes,owner);let Range r=*v::at<Range>(ref ranges,n.package_index);let usize first=v::len<Edge>(edges);var usize offset=0;
        while(offset<r.count){let usize declaration=r.first+offset;let e=workspace::edge(input,declaration);if(!work::charge(meter,32)){return fail(4,0,owner,declaration);}var usize target=*v::at<usize>(ref targets,declaration);
            if(target==0){let identity::Identity key=switch(identity::dependency(input,declaration,meter)){case .ok(value):{value}case .err(code):{return fail(if(code==6){4}else{1},code,owner,declaration);}};target=switch(intern(key,e.target,declaration+1,identities,nodes,ref mut buckets,meter)){case .ok(value):{value}case .err(p):{return p;}};number(ref mut targets,declaration,target);}
            if(!v::push<Edge>(edges,Edge{.owner,.target=target-1,.declaration}).is_ok){return fail(3,0,owner,declaration);}offset+=1;
        }var Node retained=*v::at<Node>(nodes,owner);retained.first=first;retained.count=r.count;set_node(nodes,owner,retained);owner+=1;
    }return fail(0,0,0,0);
}
fn Problem acyclic(ref v::Vec<Node> nodes,ref v::Vec<Edge> edges,ref mut v::Vec<usize> order,ref mut v::Vec<usize> cycle,ref mut usize meter){
    var v::Vec<u32> colors=v::empty<u32>();var v::Vec<Frame> frames=v::empty<Frame>();let usize count=v::len<Node>(nodes);if(!work::charge(meter,count)){return fail(4,0,0,0);}var usize at=0;while(at<count){if(!v::push<u32>(ref mut colors,0).is_ok){return fail(3,0,at,0);}at+=1;}at=0;
    while(at<count){if(*v::at<u32>(ref colors,at)!=0){at+=1;continue;}color(ref mut colors,at,1);if(!v::push<Frame>(ref mut frames,Frame{.node=at,.next=0}).is_ok){return fail(3,0,at,0);}
        while(v::len<Frame>(ref frames)>0){if(!work::charge(meter,1)){return fail(4,0,at,0);}let usize top=v::len<Frame>(ref frames)-1;var Frame f=*v::at<Frame>(ref frames,top);let Node n=*v::at<Node>(nodes,f.node);
            if(f.next>=n.count){color(ref mut colors,f.node,2);if(!v::push<usize>(order,f.node).is_ok){return fail(3,0,f.node,0);}discard v::truncate<Frame>(ref mut frames,top);continue;}
            let Edge e=*v::at<Edge>(edges,n.first+f.next);f.next+=1;frame(ref mut frames,top,f);let u32 state=*v::at<u32>(ref colors,e.target);
            if(state==1){var usize start=0;while(start<v::len<Frame>(ref frames) && v::at<Frame>(ref frames,start).node!=e.target){start+=1;}while(start<v::len<Frame>(ref frames)){if(!v::push<usize>(cycle,v::at<Frame>(ref frames,start).node).is_ok){return fail(3,0,f.node,e.declaration);}start+=1;}if(!v::push<usize>(cycle,e.target).is_ok){return fail(3,0,f.node,e.declaration);}return fail(5,0,f.node,e.declaration);}
            if(state==0){color(ref mut colors,e.target,1);if(!v::push<Frame>(ref mut frames,Frame{.node=e.target,.next=0}).is_ok){return fail(3,0,e.target,e.declaration);}}
        }at+=1;
    }return fail(0,0,0,0);
}
pub fn Result<Graph,Error> bind(workspace::Workspace input,ref mut usize meter){
    var v::Vec<identity::Identity> identities=v::empty<identity::Identity>();var v::Vec<Node> nodes=v::empty<Node>();var v::Vec<Edge> edges=v::empty<Edge>();var v::Vec<usize> order=v::empty<usize>();var v::Vec<usize> cycle=v::empty<usize>();var Problem p=expand(ref input,ref mut identities,ref mut nodes,ref mut edges,meter);if(p.code==0){p=acyclic(ref nodes,ref edges,ref mut order,ref mut cycle,meter);}
    if(p.code!=0){return .err(Error{.input,.identities,.nodes,.edges,.cycle,.problem=p});}return .ok(Graph{.input,.identities,.nodes,.edges,.order});
}
fn LoadError input_failure(workspace::Error e){var v::Vec<workspace::Error> errors=v::empty<workspace::Error>();let bool ok=v::push<workspace::Error>(ref mut errors,e).is_ok;return LoadError{.phase=if(ok){1}else{3},.input=errors,.graph=v::empty<Error>()};}
fn LoadError graph_failure(Error e){var v::Vec<Error> errors=v::empty<Error>();let bool ok=v::push<Error>(ref mut errors,e).is_ok;return LoadError{.phase=if(ok){2}else{3},.input=v::empty<workspace::Error>(),.graph=errors};}
@effects(filesystem)
pub fn Result<Graph,LoadError> load(ref u8[] root,ref mut sources::Budget budget,ref mut usize meter){
    var sources::Budget staged=*budget;let workspace::Workspace input=switch(workspace::load(root,ref mut staged,meter)){case .ok(value):{value}case .err(e):{return .err(input_failure(e));}};let Graph graph=switch(bind(input,meter)){case .ok(value):{value}case .err(e):{return .err(graph_failure(e));}};*budget=staged;return .ok(graph);
}
