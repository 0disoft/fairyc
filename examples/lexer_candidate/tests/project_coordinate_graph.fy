module tests::project_coordinate_graph;
import project_coordinate_graph as coordinates;
import project_workspace_graph as graph;
import project_source_set as sources;
import project_sdk_bundle as sdk;
import module_public_type_text as public_type;
import project_module_visibility as visibility;
import project_compiler_input as compiler_input;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::vec as v;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);return switch(files::load(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] input=ref path;return switch(files::manifest(input[..n])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn sdk::Bundle library(ref u8[] path){var v::Vec<sdk::File> input=v::empty<sdk::File>();var usize i=0;while(i<7){assert(v::push<sdk::File>(ref mut input,sdk::file(i+1,read(path,sdk::path(i)))).is_ok);i+=1;}var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;return switch(sdk::assemble(input,false,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return library(path);}};}
@effects(filesystem)
fn visibility::Resolved load(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);let bundle=library(root);var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;return switch(visibility::load_sdk(bytes::view(ref path),bundle,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return load(root,leaf);}};}
fn usize number(ref u8[] input,ref mut usize at){assert(*at<=input.len && input.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(input[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] input,ref mut usize at){let n=number(input,at);assert(*at<=input.len && n<=input.len-*at);let start=*at;*at+=n;return input[start..*at];}
fn ref u8[] coordinate(ref public_type::Coordinates values,usize owner){assert(owner<public_type::coordinate_count(values) && public_type::coordinate_at(values,owner).owner==owner);return public_type::coordinate_text(values,owner);}
@effects(filesystem)
fn void exact(ref u8[] root,ref u8[] leaf,usize expected){let resolved=load(root,leaf);let g=visibility::workspace_graph(ref resolved);var usize meter=0;let values=switch(coordinates::collect(g,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(graph::node_count(g)==expected && public_type::coordinate_count(ref values)==expected);let path=join(root,leaf);let data=read(bytes::view(ref path),b"coordinates.bin");let input=bytes::view(ref data);assert(input.len>=16 && bytes::equal(input[..8],b"FYCOORD1"));var usize at=8;assert(number(input,ref mut at)==expected);var usize i=0;var v::Vec<bool> seen=v::empty<bool>();while(i<expected){assert(v::push<bool>(ref mut seen,false).is_ok);i+=1;}i=0;
    while(i<expected){let id=field(input,ref mut at);let golden=field(input,ref mut at);var usize matches=0;var usize owner=0;while(owner<expected){if(bytes::equal(graph::node_id(g,owner),id)){assert(!*v::at<bool>(ref seen,owner) && bytes::equal(coordinate(ref values,owner),golden));let slot=v::at_mut<bool>(ref mut seen,owner);*slot=true;var usize public_work=0;assert(public_type::validate_owner(ref values,owner,ref mut public_work)==0);matches+=1;}owner+=1;}assert(matches==1);i+=1;}assert(at==input.len);
    meter=0;let projected=switch(compiler_input::project(ref resolved,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};i=0;while(i<compiler_input::count(ref projected)){let origin=compiler_input::origin(ref projected,i);assert(origin.owner<expected && bytes::equal(compiler_input::instance_id(ref projected,i),graph::node_id(g,origin.owner)));i+=1;}
}
@test
@effects(filesystem)
fn void matches_go_root_feature_instances_and_sdk_coordinates(){let path=root();exact(bytes::view(ref path),b"Base",6);exact(bytes::view(ref path),b"NoSDK",5);}
@test
@effects(filesystem)
fn void preserves_coordinates_across_source_edits_and_reordering(){let path=root();exact(bytes::view(ref path),b"Edited",6);exact(bytes::view(ref path),b"Reordered",6);let a=load(bytes::view(ref path),b"Base");let b=load(bytes::view(ref path),b"Edited");let ga=visibility::workspace_graph(ref a);let gb=visibility::workspace_graph(ref b);var usize meter=0;let ca=switch(coordinates::collect(ga,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let cb=switch(coordinates::collect(gb,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};var usize changed=0;var usize i=0;while(i<6){assert(bytes::equal(coordinate(ref ca,i),coordinate(ref cb,i)));if(!bytes::equal(graph::node_id(ga,i),graph::node_id(gb,i))){changed+=1;}i+=1;}assert(changed==2);}
@effects(filesystem)
fn public_type::Coordinates detached(ref u8[] root){let resolved=load(root,b"Base");var usize meter=0;return switch(coordinates::collect(visibility::workspace_graph(ref resolved),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return detached(root);}};}
@test
@effects(filesystem)
fn void owns_coordinates_after_workspace_closes_and_honors_budget(){let path=root();let owned=detached(bytes::view(ref path));assert(public_type::coordinate_count(ref owned)==6);var usize meter=0;var usize i=0;while(i<6){assert(public_type::validate_owner(ref owned,i,ref mut meter)==0);i+=1;}let resolved=load(bytes::view(ref path),b"Base");let g=visibility::workspace_graph(ref resolved);var usize cost=0;let first=switch(coordinates::collect(g,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};meter=16777216-cost;let fit=switch(coordinates::collect(g,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(meter==16777216);i=0;while(i<6){assert(bytes::equal(coordinate(ref first,i),coordinate(ref fit,i)));i+=1;}meter=16777216-cost+1;assert(switch(coordinates::collect(g,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==6 && p.owner<6}});assert(meter<=16777216 && graph::node_count(g)==6 && graph::edge_count(g)==5);}
@test
@effects(filesystem)
fn void keeps_feature_namespaces_distinct_and_rejects_instance_text(){let path=root();let resolved=load(bytes::view(ref path),b"Base");let g=visibility::workspace_graph(ref resolved);var usize meter=0;let table=switch(coordinates::collect(g,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};var usize first=6;var usize second=6;var usize i=0;while(i<graph::local_node_count(g)){if(bytes::equal(graph::module_name(g,i,0),b"common")){if(first==6){first=i;}else{assert(second==6);second=i;}}i+=1;}assert(first<6 && second<6 && !bytes::equal(coordinate(ref table,first),coordinate(ref table,second)) && !bytes::equal(graph::node_id(g,first),graph::node_id(g,second)));let text=switch(bytes::copy(graph::node_id(g,first))){case .ok(value):{value}case .err(_):{assert(false);return;}};var v::Vec<public_type::Coordinate> rows=v::empty<public_type::Coordinate>();assert(v::push<public_type::Coordinate>(ref mut rows,public_type::Coordinate{.owner=first,.start=0,.end=bytes::len(ref text)}).is_ok);let wrong=public_type::assemble(text,rows);meter=0;assert(public_type::validate_owner(ref wrong,first,ref mut meter)==9);}
