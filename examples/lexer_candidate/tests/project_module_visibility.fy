module tests::project_module_visibility;
import project_module_visibility as visibility;
import project_instance_graph as graph;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_source_modules as modules;
import project_source_set as sources;
import project_source_file as source_files;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::text as text;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(console)
fn void require(bool condition,ref u8[] leaf,ref u8[] stage){if(!condition){discard io::write_stderr_bytes(leaf);discard io::write_stderr(": ");discard io::write_stderr_bytes(stage);discard io::write_stderr("\n");assert(false);}}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){var usize meter=0;return switch(source_files::read(root,leaf,8388608,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] input,usize at){assert(input.len==32 && at<4);var usize value=0;var usize index=at*8;let usize end=index+8;while(index<end){value=value*256+(input[index] as usize);index+=1;}return value;}
@effects(filesystem,console)
fn void oracle(ref u8[] path,bool accepted,ref str reason){let expected=read(path,b"oracle.txt");var bytes::Buffer actual=bytes::empty();assert(bytes::append(ref mut actual,if(accepted){b"true\n"}else{b"false\n"}).is_ok && bytes::append(ref mut actual,text::as_bytes(reason)).is_ok && bytes::append_byte(ref mut actual,10).is_ok);require(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)),path,b"Go acceptance/reason");}
fn ref modules::Modules package(ref graph::Graph graph,usize at){return catalog::input(workspace::local_catalog(graph::input(graph)),graph::node(graph,at).package_index);}
@effects(filesystem,console)
fn void accepted(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);let counts=read(bytes::view(ref path),b"counts.bin");let ref u8[] expected_counts=bytes::view(ref counts);var sources::Budget budget=sources::Budget{.files=4096-number(expected_counts,0),.bytes=67108864-number(expected_counts,1)};var usize meter=0;
    let loaded=switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(visibility::load_phase(ref e));require(false,leaf,b"unexpected load error");return;}};let g=visibility::instance_graph(ref loaded);require(budget.files==4096 && budget.bytes==67108864 && visibility::lookup_count(ref loaded)==number(expected_counts,2) && graph::node_count(g)==number(expected_counts,3),leaf,b"counters/node/lookup count");oracle(bytes::view(ref path),true,"");
    var bytes::Buffer record=bytes::empty();var usize at=0;while(at<visibility::lookup_count(ref loaded)){let l=visibility::lookup(ref loaded,at);let source=package(g,l.owner);let target=package(g,l.provider);let tree=modules::parsed(source,l.source_module);let d=syntax::declaration_at(tree,l.declaration);require(d.kind==2 && d.start<=l.start && l.start<l.end && l.end<=d.end && bytes::equal(modules::name(target,l.provider_module),visibility::import_path(ref loaded,at)),leaf,b"source/provider coordinates");
        assert(bytes::append(ref mut record,graph::node_id(g,l.owner)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,modules::name(source,l.source_module)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,visibility::import_path(ref loaded,at)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,graph::node_id(g,l.provider)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,modules::name(target,l.provider_module)).is_ok && bytes::append_byte(ref mut record,10).is_ok);
        if(bytes::equal(leaf,b"LongTrivia")){require(l.end-l.start>70000 && bytes::equal(visibility::import_path(ref loaded,at),b"ns::lib"),leaf,b"verified-token normalization past 64 KiB");}at+=1;
    }let expected=read(bytes::view(ref path),b"lookups.txt");require(bytes::equal(bytes::view(ref record),bytes::view(ref expected)),leaf,b"Go owner/source/path/provider/module records");
    at=0;var usize cursor=0;while(at<graph::node_count(g)){let r=visibility::owner_range(ref loaded,at);require(r.first==cursor,leaf,b"lookup range adjacency");var usize index=r.first;while(index<r.first+r.count){require(visibility::lookup(ref loaded,index).owner==at,leaf,b"lookup range owner");index+=1;}cursor+=r.count;at+=1;}require(cursor==visibility::lookup_count(ref loaded),leaf,b"lookup range coverage");
}
@effects(filesystem,console)
fn void rejected(ref u8[] root,ref u8[] leaf,u32 code){let path=join(root,leaf);let counts=read(bytes::view(ref path),b"counts.bin");var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;
    let failure=switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(_):{require(false,leaf,b"unexpected success");return;}case .err(value):{value}};require(budget.files==17 && budget.bytes==31,leaf,b"failed lookup budget rollback");
    if(code==0){require(visibility::load_phase(ref failure)==1,leaf,b"graph failure priority");let nested=visibility::input_error(ref failure);require(graph::load_phase(nested)==2 && graph::error(graph::graph_error(nested)).code==5,leaf,b"retained cycle error");oracle(bytes::view(ref path),false,"invalid-package-graph");return;}
    require(visibility::load_phase(ref failure)==2,leaf,b"visibility error owner");let e=visibility::visibility_error(ref failure);let p=visibility::error(e);require(p.code==code && visibility::error_lookup_count(e)==number(bytes::view(ref counts),2) && p.start<p.end && p.end<=visibility::error_source(e).len,leaf,b"error category/prior lookups/original span");oracle(bytes::view(ref path),false,visibility::reason(p.code));let expected_path=read(bytes::view(ref path),b"path.txt");require(bytes::equal(visibility::error_path(e),bytes::view(ref expected_path)),leaf,b"Go rejected module path");
    let g=visibility::error_graph(e);let n=graph::node(g,p.owner);let local=workspace::local_catalog(graph::input(g));let ref u8[] package_path=catalog::path(local,n.package_index);let physical=if(bytes::equal(package_path,b".")){switch(bytes::copy(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}}}else{join(bytes::view(ref path),package_path)};let source_root=join(bytes::view(ref physical),b"src");let expected_source=read(bytes::view(ref source_root),visibility::error_file(e));require(bytes::equal(bytes::view(ref expected_source),visibility::error_source(e)),leaf,b"owned original source bytes");
    var bytes::Buffer subjects=bytes::empty();if(code==1){assert(bytes::append(ref mut subjects,graph::node_id(g,p.owner)).is_ok);var usize index=0;while(index<visibility::hidden_count(e)){let c=visibility::hidden(e,index);assert(bytes::append_byte(ref mut subjects,10).is_ok && bytes::append(ref mut subjects,graph::node_id(g,c.node)).is_ok);require(bytes::equal(modules::name(package(g,c.node),c.module_index),visibility::error_path(e)),leaf,b"hidden provider coordinate");index+=1;}require(visibility::provider_count(e)==0,leaf,b"missing visible provider");}
    else{var usize index=0;while(index<visibility::provider_count(e)){if(index>0){assert(bytes::append_byte(ref mut subjects,10).is_ok);}let c=visibility::provider(e,index);assert(bytes::append(ref mut subjects,graph::node_id(g,c.node)).is_ok);require(bytes::equal(modules::name(package(g,c.node),c.module_index),visibility::error_path(e)),leaf,b"ambiguous provider coordinate");index+=1;}require(visibility::provider_count(e)>=2,leaf,b"distinct ambiguous providers");}
    let expected_subjects=read(bytes::view(ref path),b"subjects.txt");require(bytes::equal(bytes::view(ref subjects),bytes::view(ref expected_subjects)),leaf,b"Go provider/hidden subjects");if(bytes::equal(leaf,b"LongMissingTrivia")){require(p.end-p.start>70000 && bytes::equal(visibility::error_path(e),b"ns::missing"),leaf,b"long original failed import span");}
}
@test
@effects(filesystem,console)
fn void resolves_direct_instance_modules_and_retains_exact_import_errors(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;let root=switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref u8[] path=bytes::view(ref root);
    accepted(path,b"Empty");accepted(path,b"Own");accepted(path,b"Direct");accepted(path,b"Namespace");accepted(path,b"LongTrivia");accepted(path,b"Aliases");accepted(path,b"DirectTransitive");accepted(path,b"UnreachableImport");accepted(path,b"ModuleOrder");rejected(path,b"LongMissingTrivia",1);rejected(path,b"Transitive",1);rejected(path,b"Ambiguous",2);rejected(path,b"OwnAmbiguous",2);rejected(path,b"MemberOwnAmbiguous",2);rejected(path,b"FeatureAmbiguous",2);rejected(path,b"ByBinding",1);rejected(path,b"CaseMismatch",1);rejected(path,b"PartialMissing",1);rejected(path,b"CycleFirst",0);
}
