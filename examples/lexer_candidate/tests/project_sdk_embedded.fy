module tests::project_sdk_embedded;
import project_sdk_embedded as embedded;
import project_sdk_bundle as sdk;
import project_module_visibility as visibility;
import project_workspace_graph as graph;
import project_source_set as sources;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);return switch(files::load(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;return switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn void line(ref mut bytes::Buffer output,ref u8[] value){assert(bytes::append(output,value).is_ok && bytes::append_byte(output,10).is_ok);}
fn usize number(ref u8[] input,usize at){assert(input.len==40 && at<5);var usize value=0;var usize index=at*8;while(index<at*8+8){value=value*256+(input[index] as usize);index+=1;}return value;}
@effects(filesystem)
fn void accepted(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);let counts=read(bytes::view(ref path),b"counts.bin");let flags=read(bytes::view(ref path),b"candidate.txt");let bool candidate=bytes::equal(bytes::view(ref flags),b"true");let selection=read(bytes::view(ref path),b"selection.txt");let usize profile=(bytes::view(ref selection)[0]-48) as usize;var sources::Budget budget=sources::Budget{.files=4096-number(bytes::view(ref counts),0),.bytes=67108864-number(bytes::view(ref counts),1)};var usize meter=0;
    let loaded=switch(if(candidate){visibility::load_embedded(bytes::view(ref path),true,ref mut budget,ref mut meter)}else{visibility::load(bytes::view(ref path),ref mut budget,ref mut meter)}){case .ok(value):{value}case .err(_):{assert(false);return;}};let g=visibility::workspace_graph(ref loaded);assert(budget.files==4096 && budget.bytes==67108864 && graph::node_count(g)==number(bytes::view(ref counts),2) && graph::edge_count(g)==number(bytes::view(ref counts),3) && visibility::lookup_count(ref loaded)==number(bytes::view(ref counts),4) && graph::has_sdk(g)==(profile!=0));
    var bytes::Buffer nodes=bytes::empty();var bytes::Buffer edges=bytes::empty();var bytes::Buffer lookups=bytes::empty();var usize at=0;while(at<graph::node_count(g)){line(ref mut nodes,graph::node_id(g,at));line(ref mut nodes,if(graph::is_sdk(g,at)){b"standard-library"}else if(at==0){b"root"}else{b"dependency"});if(graph::is_sdk(g,at)){let library=graph::standard_library(g);assert(sdk::candidate(library)==candidate && sdk::count(library)==if(profile>=2){8}else{7});var usize index=0;while(index<sdk::count(library)){let expected=read(root,sdk::path(index));assert(bytes::equal(graph::module_source(g,at,index),bytes::view(ref expected)) && bytes::equal(graph::module_file(g,at,index),sdk::path(index)) && bytes::equal(graph::module_name(g,at,index),sdk::name(index)));index+=1;}}at+=1;}let expected_nodes=read(bytes::view(ref path),b"nodes.txt");assert(bytes::equal(bytes::view(ref nodes),bytes::view(ref expected_nodes)));
    at=0;while(at<graph::edge_count(g)){let e=graph::edge(g,at);line(ref mut edges,graph::node_id(g,e.owner));line(ref mut edges,graph::binding(g,at));line(ref mut edges,graph::node_id(g,e.target));at+=1;}let expected_edges=read(bytes::view(ref path),b"edges.txt");assert(bytes::equal(bytes::view(ref edges),bytes::view(ref expected_edges)));
    at=0;while(at<visibility::lookup_count(ref loaded)){let l=visibility::lookup(ref loaded,at);line(ref mut lookups,graph::node_id(g,l.owner));line(ref mut lookups,graph::module_name(g,l.owner,l.source_module));line(ref mut lookups,visibility::import_path(ref loaded,at));line(ref mut lookups,graph::node_id(g,l.provider));line(ref mut lookups,graph::module_name(g,l.provider,l.provider_module));at+=1;}let expected_lookups=read(bytes::view(ref path),b"lookups.txt");assert(bytes::equal(bytes::view(ref lookups),bytes::view(ref expected_lookups)));
}
@test
@effects(filesystem)
fn void default_loader_uses_exact_embedded_sdk_for_reachable_imports(){let owned=root();let path=bytes::view(ref owned);accepted(path,b"None");accepted(path,b"CandidateUnused");accepted(path,b"Core");accepted(path,b"AliasCore");accepted(path,b"Vector");accepted(path,b"CandidateCore");accepted(path,b"ChildOnly");accepted(path,b"Both");accepted(path,b"ChildVector");accepted(path,b"UnreachableVector");accepted(path,b"FeatureSplit");accepted(path,b"InstanceAliases");accepted(path,b"LongTrivia");accepted(path,b"SimilarNamespaces");}
@effects(filesystem)
fn void rejected(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;let e=switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(_):{assert(false);return;}case .err(value):{value}};assert(budget.files==17 && budget.bytes==31 && visibility::load_phase(ref e)==2);let failed=visibility::visibility_error(ref e);let expected=read(bytes::view(ref path),b"path.txt");assert(visibility::error(failed).code==1 && bytes::equal(visibility::error_path(failed),bytes::view(ref expected)) && graph::has_sdk(visibility::error_workspace_graph(failed)));}
@test
@effects(filesystem)
fn void unknown_embedded_sdk_imports_preserve_local_counter_rollback(){let owned=root();rejected(bytes::view(ref owned),b"UnknownStd");rejected(bytes::view(ref owned),b"BareStd");}
@test
fn void embedded_factory_is_pure_and_rejects_invalid_profiles_and_work(){var usize meter=0;assert(switch(embedded::load(0,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==2}});assert(switch(embedded::load(4,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==2}});assert(meter==0);meter=16777216;assert(switch(embedded::load(1,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==5 && p.file==0}});}
