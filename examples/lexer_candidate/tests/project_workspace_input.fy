module tests::project_workspace_input;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_source_set as sources;
import project_package_input as packages;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::text as text;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,root).is_ok && bytes::append_byte(ref mut output,47).is_ok && bytes::append(ref mut output,leaf).is_ok);return output;}
@effects(console)
fn void require(bool condition,ref u8[] leaf,ref u8[] stage){if(!condition){discard io::write_stderr_bytes(leaf);discard io::write_stderr(": ");discard io::write_stderr_bytes(stage);discard io::write_stderr("\n");assert(false);}}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);return switch(files::manifest(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem,console)
fn void oracle(ref u8[] root,bool accepted,ref str reason){let expected=read(root,b"oracle.txt");var bytes::Buffer actual=bytes::empty();assert(bytes::append(ref mut actual,if(accepted){b"true\n"}else{b"false\n"}).is_ok && bytes::append(ref mut actual,text::as_bytes(reason)).is_ok && bytes::append_byte(ref mut actual,10).is_ok);require(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)),root,b"Go acceptance/reason");}
@effects(filesystem,console)
fn void accepted(ref u8[] root,ref u8[] leaf,usize files_count,usize bytes_count,usize edges_count){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=4096-files_count,.bytes=67108864-bytes_count};var usize meter=0;
    let loaded=switch(workspace::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(workspace::error(ref e).code);require(false,leaf,b"unexpected error");return;}};require(budget.files==4096 && budget.bytes==67108864 && workspace::edge_count(ref loaded)==edges_count,leaf,b"budget/edges");oracle(bytes::view(ref path),true,"");
    var bytes::Buffer record=bytes::empty();var usize at=0;while(at<edges_count){let e=workspace::edge(ref loaded,at);let local=workspace::local_catalog(ref loaded);assert(bytes::append(ref mut record,catalog::path(local,e.owner)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,workspace::binding(ref loaded,at)).is_ok && bytes::append_byte(ref mut record,10).is_ok && bytes::append(ref mut record,catalog::path(local,e.target)).is_ok && bytes::append_byte(ref mut record,10).is_ok);var usize id=workspace::feature_first(ref loaded,at);var bool first=true;while(id!=0){if(!first){assert(bytes::append_byte(ref mut record,44).is_ok);}assert(bytes::append(ref mut record,workspace::feature(ref loaded,at,id)).is_ok);first=false;id=workspace::next_feature(ref loaded,at,id);}assert(bytes::append_byte(ref mut record,10).is_ok);at+=1;}
    let expected=read(bytes::view(ref path),b"edges.txt");require(bytes::equal(bytes::view(ref record),bytes::view(ref expected)),leaf,b"Go edge/feature records");if(edges_count==3){let a=workspace::edge(ref loaded,0);let b=workspace::edge(ref loaded,1);require(a.owner==0 && b.owner==0 && a.target==1 && b.target==1 && bytes::equal(workspace::binding(ref loaded,0),b"a-lane") && bytes::equal(workspace::binding(ref loaded,1),b"b-lane"),leaf,b"independent feature edges");}
}
@effects(filesystem,console)
fn void rejected(ref u8[] root,ref u8[] leaf,u32 code,usize owner,u32 scope){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=11,.bytes=23};var usize meter=0;
    let e=switch(workspace::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(_):{require(false,leaf,b"unexpected success");return;}case .err(value):{value}};let p=workspace::error(ref e);require(p.code==code && p.package_index==owner && p.scope==scope && budget.files==11 && budget.bytes==23,leaf,b"error/rollback");
    if(code==1){require(workspace::has_catalog_error(ref e),leaf,b"catalog error owner");let nested=workspace::nested_catalog_error(ref e);require(catalog::has_input_error(nested) && packages::error(catalog::input_error(nested)).phase==0,leaf,b"original missing-manifest error");oracle(bytes::view(ref path),false,"invalid-project-manifest");return;}
    oracle(bytes::view(ref path),false,workspace::reason(code));let ref u8[] package_path=workspace::error_package(ref e);let physical=if(bytes::equal(package_path,b".")){switch(bytes::copy(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}}}else{join(bytes::view(ref path),package_path)};
    let source_root=if(scope==1){join(bytes::view(ref physical),b"src")}else{switch(bytes::copy(bytes::view(ref physical))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}}};let original=read(bytes::view(ref source_root),workspace::error_file(ref e));require(bytes::equal(bytes::view(ref original),workspace::error_input(ref e)) && p.start<p.end && p.end<=workspace::error_input(ref e).len,leaf,b"owned original bytes/span");
    if(code==6){require(p.field==5 && bytes::equal(workspace::error_context(ref e),b"shared") && workspace::error_target(ref e).len>0,leaf,b"version error metadata");}if(code==5){require(p.field==11 && bytes::equal(workspace::error_target(ref e),b"ghost"),leaf,b"member path metadata");}if(scope==1){require(p.start==7 && bytes::equal(workspace::error_input(ref e)[p.start..p.end],workspace::error_context(ref e)),leaf,b"original module name span");}
}
@test
@effects(filesystem,console)
fn void retains_dependency_edges_features_and_original_error_inputs(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;let root=switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};
    accepted(bytes::view(ref root),b"Simple",2,25,1);accepted(bytes::view(ref root),b"Features",3,39,3);accepted(bytes::view(ref root),b"NearSDKNames",2,29,1);
    rejected(bytes::view(ref root),b"UnknownRoot",5,0,0);rejected(bytes::view(ref root),b"UnknownMember",5,1,0);rejected(bytes::view(ref root),b"Version",6,0,0);rejected(bytes::view(ref root),b"BuildVersion",6,0,0);rejected(bytes::view(ref root),b"UnreachableVersion",6,1,0);rejected(bytes::view(ref root),b"ReservedUID",2,0,0);rejected(bytes::view(ref root),b"ReservedUIDMember",2,1,0);rejected(bytes::view(ref root),b"ReservedModule",3,0,1);rejected(bytes::view(ref root),b"ReservedNamespace",3,0,1);rejected(bytes::view(ref root),b"ReservedModuleMember",3,1,1);rejected(bytes::view(ref root),b"ReservedBinding",4,0,0);rejected(bytes::view(ref root),b"CatalogFailure",1,0,2);
}
