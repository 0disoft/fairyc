module tests::project_instance_graph;
import project_instance_graph as graph;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_source_set as sources;
import project_source_modules as modules;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::text as text;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(console)
fn void require(bool condition,ref u8[] leaf,ref u8[] stage){if(!condition){discard io::write_stderr_bytes(leaf);discard io::write_stderr(": ");discard io::write_stderr_bytes(stage);discard io::write_stderr("\n");assert(false);}}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);return switch(files::manifest(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem,console)
fn void oracle(ref u8[] path,bool accepted,ref str reason){let expected=read(path,b"oracle.txt");var bytes::Buffer actual=bytes::empty();assert(bytes::append(ref mut actual,if(accepted){b"true\n"}else{b"false\n"}).is_ok && bytes::append(ref mut actual,text::as_bytes(reason)).is_ok && bytes::append_byte(ref mut actual,10).is_ok);require(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)),path,b"Go acceptance/reason");}
fn usize position(ref graph::Graph graph,usize node){var usize at=0;while(at<graph::node_count(graph)){if(graph::build_order(graph,at)==node){return at+1;}at+=1;}return 0;}
@effects(filesystem,console)
fn void accepted(ref u8[] root,ref u8[] leaf,usize files_count,usize bytes_count,usize nodes,usize edges){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=4096-files_count,.bytes=67108864-bytes_count};var usize meter=0;
    let loaded=switch(graph::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(graph::load_phase(ref e));require(false,leaf,b"unexpected error");return;}};require(graph::node_count(ref loaded)==nodes && graph::edge_count(ref loaded)==edges && budget.files==4096 && budget.bytes==67108864,leaf,b"counts/budget");oracle(bytes::view(ref path),true,"");
    var bytes::Buffer record=bytes::empty();var usize at=0;while(at<nodes){let n=graph::node(ref loaded,at);let input=graph::input(ref loaded);assert(bytes::append(ref mut record,graph::node_id(ref loaded,at)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,catalog::path(workspace::local_catalog(input),n.package_index)).is_ok && bytes::append_byte(ref mut record,10).is_ok);if(n.feature_edge!=0){let usize edge=n.feature_edge-1;var usize id=workspace::feature_first(input,edge);var bool first=true;while(id!=0){if(!first){assert(bytes::append_byte(ref mut record,44).is_ok);}assert(bytes::append(ref mut record,workspace::feature(input,edge,id)).is_ok);first=false;id=workspace::next_feature(input,edge,id);}}assert(bytes::append_byte(ref mut record,10).is_ok);require(position(ref loaded,at)>0,leaf,b"build-order coverage");at+=1;}
    at=0;while(at<edges){let e=graph::edge(ref loaded,at);assert(bytes::append(ref mut record,graph::node_id(ref loaded,e.owner)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,graph::binding(ref loaded,at)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,graph::node_id(ref loaded,e.target)).is_ok && bytes::append_byte(ref mut record,10).is_ok);require(position(ref loaded,e.target)<position(ref loaded,e.owner),leaf,b"dependency build order");at+=1;}
    let expected=read(bytes::view(ref path),b"records.txt");require(bytes::equal(bytes::view(ref record),bytes::view(ref expected)),leaf,b"Go node/edge/feature identities");if(bytes::equal(leaf,b"Aliases")){require(graph::edge(ref loaded,0).target==graph::edge(ref loaded,1).target,leaf,b"exact identity aliases");}if(bytes::equal(leaf,b"UnreachableCycle")){require(catalog::count(workspace::local_catalog(graph::input(ref loaded)))==3 && workspace::edge_count(graph::input(ref loaded))==2,leaf,b"retained unreachable declarations");}
}
@effects(filesystem,console)
fn void rejected(ref u8[] root,ref u8[] leaf,usize count){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=13,.bytes=27};var usize meter=0;
    let failure=switch(graph::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(_):{require(false,leaf,b"unexpected pass");return;}case .err(value):{value}};require(budget.files==13 && budget.bytes==27,leaf,b"failed graph budget rollback");
    if(count==0){require(graph::load_phase(ref failure)==1,leaf,b"input error ownership");let input=graph::input_error(ref failure);require(workspace::error(input).code==6 && workspace::error_input(input).len>0,leaf,b"original declaration failure");oracle(bytes::view(ref path),false,workspace::reason(workspace::error(input).code));return;}
    require(graph::load_phase(ref failure)==2,leaf,b"graph error ownership");let error=graph::graph_error(ref failure);let p=graph::error(error);require(p.code==5 && graph::cycle_count(error)==count,leaf,b"cycle rejection");oracle(bytes::view(ref path),false,graph::reason(p.code));
    var bytes::Buffer actual=bytes::empty();var usize at=0;while(at<count){if(at>0){assert(bytes::append_byte(ref mut actual,10).is_ok);}assert(bytes::append(ref mut actual,graph::cycle_id(error,at)).is_ok);at+=1;}let expected=read(bytes::view(ref path),b"cycle.txt");require(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)),leaf,b"cycle feature-instance IDs");
    let input=graph::error_workspace(error);let local=workspace::local_catalog(input);require(catalog::count(local)>=2 && modules::count(catalog::input(local,1))==1 && sources::manifest_input(modules::source_set(catalog::input(local,1))).len>0,leaf,b"retained cycle source/manifest owner");
}
@test
@effects(filesystem,console)
fn void expands_exact_feature_instances_and_rejects_reachable_cycles(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;let root=switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref u8[] path=bytes::view(ref root);
    accepted(path,b"Single",1,13,1,0);accepted(path,b"Chain",3,39,3,2);accepted(path,b"Aliases",2,25,2,2);accepted(path,b"FeatureSplit",2,25,3,2);accepted(path,b"FeatureDiamond",3,39,4,4);accepted(path,b"UnreachableCycle",3,39,1,0);accepted(path,b"LongChain",71,853,71,70);rejected(path,b"SelfCycle",2);rejected(path,b"TwoCycle",3);rejected(path,b"FeatureLiftCycle",2);rejected(path,b"VersionError",0);
}
