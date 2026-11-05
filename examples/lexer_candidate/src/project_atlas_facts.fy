module project_atlas_facts;
import project_atlas_write as w;
import project_atlas_model as model;
import project_atlas_wire as wire;
import project_json_tree as json;
import project_lock_document as locks;
import project_workspace_graph as graph;
import project_compiler_input as input;
import module_project_interfaces as interfaces;
import module_generic_interface as generic;
import module_public_type_text as coordinates;
import module_unit_body_ir_model as ir;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub resource Facts{json::Tree expected;bytes::Buffer modules;usize files;cleanup{}}
pub fn ref json::Tree expected(ref Facts p){return ref p.expected;}
pub fn ref u8[] value(ref Facts p,ref u8[] name){return wire::string(ref p.expected,0,name);}
pub fn ref u8[] array(ref Facts p,ref u8[] name){return model::raw(ref p.expected,wire::field(ref p.expected,0,name));}
pub fn ref u8[] modules(ref Facts p){return bytes::view(ref p.modules);}
pub fn usize files(ref Facts p){return p.files;}
pub fn Result<v::Vec<usize>,u32> order(ref graph::Graph g,ref mut usize meter){var v::Vec<usize> result=v::empty<usize>();var usize i=0;while(i<graph::node_count(g)){if(!v::push<usize>(ref mut result,i).is_ok){return .err(2);}var usize j=i;while(j>0){if(!work::charge(meter,256)){return .err(3);}let old=*v::at<usize>(ref result,j-1);if(model::before(graph::node_id(g,old),graph::node_id(g,i))){break;}{let slot=v::at_mut<usize>(ref mut result,j);*slot=old;}{let slot=v::at_mut<usize>(ref mut result,j-1);*slot=i;}j-=1;}i+=1;}return .ok(result);}
fn bool dependency(ref graph::Graph g,usize owner,usize target,ref mut usize meter){var usize i=0;while(i<graph::edge_count(g)){if(!work::charge(meter,1)){return false;}let edge=graph::edge(g,i);if(edge.owner==owner && edge.target==target){return true;}i+=1;}return false;}
fn usize lock_node(ref locks::Document l,ref u8[] instance,ref mut usize meter){var usize i=0;while(i<locks::package_count(l)){if(!work::charge(meter,256)){return locks::package_count(l);}if(bytes::equal(locks::package_field(l,i,0),instance)){return i;}i+=1;}return locks::package_count(l);}
fn Result<v::Vec<usize>,u32> order_modules(ref graph::Graph g,usize owner,bool file_names,ref mut usize meter){
 var order=v::empty<usize>();var usize i=0;
 while(i<graph::module_count(g,owner)){
  if(!v::push<usize>(ref mut order,i).is_ok){return .err(2);}var usize at=i;
  while(at>0){let prior=*v::at<usize>(ref order,at-1);let a=if(file_names){graph::module_file(g,owner,i)}else{graph::module_name(g,owner,i)};let b=if(file_names){graph::module_file(g,owner,prior)}else{graph::module_name(g,owner,prior)};
   if(!work::charge(meter,a.len+b.len+1)){return .err(3);}if(!model::before(a,b)){break;}{let slot=v::at_mut<usize>(ref mut order,at);*slot=prior;}{let slot=v::at_mut<usize>(ref mut order,at-1);*slot=i;}at-=1;
  }i+=1;
 }
 return .ok(order);
}
fn Result<bytes::Buffer,u32> modules_sorted(ref graph::Graph g,usize owner,ref mut usize meter){
 let order=order_modules(g,owner,false,meter)?;var out=bytes::empty();discard w::raw(ref mut out,b"[",meter)?;var usize i=0;
 while(i<v::len<usize>(ref order)){
  let index=*v::at<usize>(ref order,i);if(i>0){discard w::raw(ref mut out,b",",meter)?;}
  discard w::raw(ref mut out,b"{",meter)?;discard w::pair(ref mut out,b"package_instance",graph::node_id(g,owner),meter)?;discard w::raw(ref mut out,b",",meter)?;
  discard w::pair(ref mut out,b"path",graph::module_name(g,owner,index),meter)?;discard w::raw(ref mut out,b",",meter)?;
  discard w::pair(ref mut out,b"source",graph::module_file(g,owner,index),meter)?;discard w::raw(ref mut out,b"}",meter)?;i+=1;
 }discard w::raw(ref mut out,b"]",meter)?;return .ok(out);
}
pub fn Result<Facts,u32> collect(ref graph::Graph g,ref locks::Document l,ref coordinates::Coordinates table,ref interfaces::Records checked,ref ir::Program program,ref u8[] compiler,usize owner,ref mut usize meter){
 let lock=lock_node(l,graph::node_id(g,owner),meter);if(lock==locks::package_count(l)){return .err(9);}var usize record=0;while(record<interfaces::count(checked) && interfaces::owner(interfaces::at(checked,record))!=owner){if(!work::charge(meter,1)){return .err(3);}record+=1;}if(record==interfaces::count(checked)){return .err(9);}let retained=interfaces::at(checked,record);
 // Generic hashing owns a 1 MiB sub-operation allowance. The enclosing project
 // has a 16 MiB ledger; prior source work must not consume the hash's allowance.
 // Charge the actual operation cost back even when the hash itself rejects.
 var usize hash_work=0;let generic_hash=switch(generic::hash(ir::state_layouts(program),owner,interfaces::descriptor(retained),interfaces::templates(retained),ref mut hash_work)){
  case .ok(p):{if(!work::charge(meter,hash_work)){return .err(3);}p}
  case .err(code):{if(!work::charge(meter,hash_work)){return .err(3);}return .err(if(code==10){3}else if(code==11){2}else{7});}
 };
 var bytes::Buffer out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;
 let names=b"compiler_build|interface_hash|generic_interface_hash|language|package_uid|package_instance|package_coordinate|package_source|package_source_kind|runtime_abi";var usize start=0;var usize field=0;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let name=names[start..end];if(field>0){discard w::raw(ref mut out,b",",meter)?;}discard w::pair(ref mut out,name,if(field==0){compiler}else if(field==1){interfaces::ordinary_hash(retained)}else if(field==2){bytes::view(ref generic_hash)}else if(field==3){locks::header(l,3)}else if(field==4){locks::package_field(l,lock,2)}else if(field==5){graph::node_id(g,owner)}else if(field==6){coordinates::coordinate_text(table,owner)}else if(field==7){locks::package_field(l,lock,8)}else if(field==8){locks::package_field(l,lock,3)}else{locks::package_field(l,lock,6)},meter)?;start=end+1;field+=1;}
 discard w::raw(ref mut out,b",\"features\":[",meter)?;var usize i=0;while(i<locks::feature_count(l,lock)){if(i>0){discard w::raw(ref mut out,b",",meter)?;}discard w::string(ref mut out,locks::feature(l,lock,i),meter)?;i+=1;}discard w::raw(ref mut out,b"],\"dependencies\":[",meter)?;
 let ordered=order(g,meter)?;i=0;var bool first=true;while(i<v::len<usize>(ref ordered)){let node=*v::at<usize>(ref ordered,i);if(dependency(g,owner,node,meter)){if(!first){discard w::raw(ref mut out,b",",meter)?;}discard w::string(ref mut out,graph::node_id(g,node),meter)?;first=false;}i+=1;}
 discard w::raw(ref mut out,b"],\"package_references\":[",meter)?;i=0;first=true;while(i<v::len<usize>(ref ordered)){let node=*v::at<usize>(ref ordered,i);if(node==owner || dependency(g,owner,node,meter)){let target=lock_node(l,graph::node_id(g,node),meter);if(target==locks::package_count(l)){return .err(9);}if(!first){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"{",meter)?;discard w::pair(ref mut out,b"uid",locks::package_field(l,target,2),meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"instance",graph::node_id(g,node),meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"coordinate",coordinates::coordinate_text(table,node),meter)?;discard w::raw(ref mut out,b"}",meter)?;first=false;}i+=1;}
 discard w::raw(ref mut out,b"],\"files\":[",meter)?;i=0;
 let file_order=order_modules(g,owner,true,meter)?;while(i<graph::module_count(g,owner)){let index=*v::at<usize>(ref file_order,i);let path=graph::module_file(g,owner,index);let source=graph::module_source(g,owner,index);let digest=w::source_hash(source,false,meter)?;let lines=w::source_hash(source,true,meter)?;if(i>0){discard w::raw(ref mut out,b",",meter)?;}
 discard w::raw(ref mut out,b"{",meter)?;discard w::pair(ref mut out,b"package_instance",graph::node_id(g,owner),meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"path",path,meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"digest",bytes::view(ref digest),meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"line_map",bytes::view(ref lines),meter)?;discard w::raw(ref mut out,b",\"kind\":\"source\"}",meter)?;
 i+=1;
 }
 discard w::raw(ref mut out,b"]}",meter)?;let modules=modules_sorted(g,owner,meter)?;let expected=switch(json::parse(out,meter)){case .ok(p):{p}case .err(_):{return .err(9);}};return .ok(Facts{.expected,.modules,.files=i});
}
